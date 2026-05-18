package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/nlafevers/kopds/internal/domain"
)

type sqliteBookRepository struct {
	db *sql.DB
}

var ErrSearchQueryTooLong = errors.New("search query is too long")

// NewBookRepository creates a new SQLite book repository.
func NewBookRepository(db *sql.DB) domain.BookRepository {
	return &sqliteBookRepository{db: db}
}

func (r *sqliteBookRepository) GetByID(ctx context.Context, id int64) (*domain.Book, error) {
	query := `
		SELECT 
			b.id, b.uuid, b.title, b.sort, b.author_sort, b.timestamp, b.pub_date, 
			b.series_index, b.last_modified, b.path, b.has_cover, b.calibre_id, b.description,
			s.id, s.name
		FROM books b
		LEFT JOIN series s ON b.series_id = s.id
		WHERE b.id = ?`

	var book domain.Book
	var seriesID sql.NullInt64
	var seriesName sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&book.ID, &book.UUID, &book.Title, &book.Sort, &book.AuthorSort, &book.Timestamp, &book.PubDate,
		&book.SeriesIndex, &book.LastModified, &book.Path, &book.HasCover, &book.CalibreID, &book.Description,
		&seriesID, &seriesName,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	if seriesID.Valid {
		book.Series = &domain.Series{
			ID:   seriesID.Int64,
			Name: seriesName.String,
		}
	}

	// Fetch Authors
	authors, err := r.getAuthors(ctx, id)
	if err != nil {
		return nil, err
	}
	book.Authors = authors

	// Fetch Tags
	tags, err := r.getTags(ctx, id)
	if err != nil {
		return nil, err
	}
	book.Tags = tags

	// Fetch Formats
	formats, err := r.getFormats(ctx, id)
	if err != nil {
		return nil, err
	}
	book.Formats = formats

	return &book, nil
}

func (r *sqliteBookRepository) getAuthors(ctx context.Context, bookID int64) ([]domain.Author, error) {
	query := `
		SELECT a.id, a.name, a.sort
		FROM authors a
		JOIN books_authors_link bal ON a.id = bal.author_id
		WHERE bal.book_id = ?`
	rows, err := r.db.QueryContext(ctx, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get authors: %w", err)
	}
	defer rows.Close()

	var authors []domain.Author
	for rows.Next() {
		var a domain.Author
		if err := rows.Scan(&a.ID, &a.Name, &a.Sort); err != nil {
			return nil, err
		}
		authors = append(authors, a)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return authors, nil
}

func (r *sqliteBookRepository) getTags(ctx context.Context, bookID int64) ([]domain.Tag, error) {
	query := `
		SELECT t.id, t.name
		FROM tags t
		JOIN books_tags_link btl ON t.id = btl.tag_id
		WHERE btl.book_id = ?`
	rows, err := r.db.QueryContext(ctx, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	defer rows.Close()

	var tags []domain.Tag
	for rows.Next() {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return tags, nil
}

func (r *sqliteBookRepository) getFormats(ctx context.Context, bookID int64) ([]domain.Format, error) {
	query := `SELECT id, format, uncompressed_size, name FROM formats WHERE book_id = ?`
	rows, err := r.db.QueryContext(ctx, query, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get formats: %w", err)
	}
	defer rows.Close()

	var formats []domain.Format
	for rows.Next() {
		var f domain.Format
		if err := rows.Scan(&f.ID, &f.Format, &f.UncompressedSize, &f.Name); err != nil {
			return nil, err
		}
		formats = append(formats, f)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return formats, nil
}

func (r *sqliteBookRepository) Search(ctx context.Context, query string, limit, offset int) ([]domain.Book, int, error) {
	ftsQuery, err := buildFTSQuery(query)
	if err != nil {
		return nil, 0, err
	}
	if ftsQuery == "" {
		return nil, 0, nil
	}

	countQuery := `SELECT COUNT(*) FROM books_search WHERE books_search MATCH ?`
	var total int
	err = r.db.QueryRowContext(ctx, countQuery, ftsQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("search count failed: %w", err)
	}

	sqlQuery := `
		SELECT b.id
		FROM books_search bs
		JOIN books b ON bs.rowid = b.id
		WHERE books_search MATCH ?
		ORDER BY rank
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, sqlQuery, ftsQuery, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search failed: %w", err)
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	rows.Close()

	var books []domain.Book
	for _, id := range ids {
		book, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		if book != nil {
			books = append(books, *book)
		}
	}
	return books, total, nil
}

func buildFTSQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return "", nil
	}
	if len(query) > 200 {
		return "", ErrSearchQueryTooLong
	}

	tokens := strings.FieldsFunc(query, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})
	if len(tokens) == 0 {
		return "", nil
	}

	terms := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = strings.ToLower(token)
		if token == "" {
			continue
		}
		terms = append(terms, token+"*")
	}
	return strings.Join(terms, " "), nil
}

func (r *sqliteBookRepository) ListRecent(ctx context.Context, limit, offset int) ([]domain.Book, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count books: %w", err)
	}

	query := `SELECT id FROM books ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	books, err := r.listBooks(ctx, query, limit, offset)
	return books, total, err
}

func (r *sqliteBookRepository) ListByAuthor(ctx context.Context, authorID int64, limit, offset int) ([]domain.Book, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books_authors_link WHERE author_id = ?", authorID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count author books: %w", err)
	}

	query := `
		SELECT b.id 
		FROM books b
		JOIN books_authors_link bal ON b.id = bal.author_id
		WHERE bal.author_id = ?
		ORDER BY b.sort ASC
		LIMIT ? OFFSET ?`
	books, err := r.listBooks(ctx, query, authorID, limit, offset)
	return books, total, err
}

func (r *sqliteBookRepository) ListBySeries(ctx context.Context, seriesID int64, limit, offset int) ([]domain.Book, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books WHERE series_id = ?", seriesID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count series books: %w", err)
	}

	query := `
		SELECT id 
		FROM books 
		WHERE series_id = ?
		ORDER BY series_index ASC, sort ASC
		LIMIT ? OFFSET ?`
	books, err := r.listBooks(ctx, query, seriesID, limit, offset)
	return books, total, err
}

func (r *sqliteBookRepository) ListAuthors(ctx context.Context, limit, offset int) ([]domain.AuthorWithCount, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM authors").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count authors: %w", err)
	}

	query := `
		SELECT a.id, a.name, a.sort, COUNT(bal.book_id) as book_count
		FROM authors a
		LEFT JOIN books_authors_link bal ON a.id = bal.author_id
		GROUP BY a.id
		ORDER BY a.sort ASC
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list authors: %w", err)
	}
	defer rows.Close()

	var authors []domain.AuthorWithCount
	for rows.Next() {
		var a domain.AuthorWithCount
		if err := rows.Scan(&a.ID, &a.Name, &a.Sort, &a.BookCount); err != nil {
			return nil, 0, err
		}
		authors = append(authors, a)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return authors, total, nil
}

func (r *sqliteBookRepository) ListSeries(ctx context.Context, limit, offset int) ([]domain.SeriesWithCount, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM series").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count series: %w", err)
	}

	query := `
		SELECT s.id, s.name, COUNT(b.id) as book_count
		FROM series s
		LEFT JOIN books b ON s.id = b.series_id
		GROUP BY s.id
		ORDER BY s.name ASC
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list series: %w", err)
	}
	defer rows.Close()

	var series []domain.SeriesWithCount
	for rows.Next() {
		var s domain.SeriesWithCount
		if err := rows.Scan(&s.ID, &s.Name, &s.BookCount); err != nil {
			return nil, 0, err
		}
		series = append(series, s)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return series, total, nil
}

func (r *sqliteBookRepository) ListTags(ctx context.Context, limit, offset int) ([]domain.TagWithCount, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM tags").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tags: %w", err)
	}

	query := `
		SELECT t.id, t.name, COUNT(btl.book_id) as book_count
		FROM tags t
		LEFT JOIN books_tags_link btl ON t.id = btl.tag_id
		GROUP BY t.id
		ORDER BY t.name ASC
		LIMIT ? OFFSET ?`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list tags: %w", err)
	}
	defer rows.Close()

	var tags []domain.TagWithCount
	for rows.Next() {
		var t domain.TagWithCount
		if err := rows.Scan(&t.ID, &t.Name, &t.BookCount); err != nil {
			return nil, 0, err
		}
		tags = append(tags, t)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return tags, total, nil
}

func (r *sqliteBookRepository) ListByTag(ctx context.Context, tagID int64, limit, offset int) ([]domain.Book, int, error) {
	var total int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books_tags_link WHERE tag_id = ?", tagID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count tag books: %w", err)
	}

	query := `
		SELECT b.id 
		FROM books b
		JOIN books_tags_link btl ON b.id = btl.book_id
		WHERE btl.tag_id = ?
		ORDER BY b.sort ASC
		LIMIT ? OFFSET ?`
	books, err := r.listBooks(ctx, query, tagID, limit, offset)
	return books, total, err
}

func (r *sqliteBookRepository) listBooks(ctx context.Context, query string, args ...interface{}) ([]domain.Book, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list books: %w", err)
	}

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()

	var books []domain.Book
	for _, id := range ids {
		book, err := r.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if book != nil {
			books = append(books, *book)
		}
	}
	return books, nil
}

func (r *sqliteBookRepository) Upsert(ctx context.Context, book *domain.Book) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Handle Series
	var seriesID sql.NullInt64
	if book.Series != nil {
		err := tx.QueryRowContext(ctx, "INSERT INTO series (name) VALUES (?) ON CONFLICT(name) DO UPDATE SET name=excluded.name RETURNING id", book.Series.Name).Scan(&seriesID)
		if err != nil {
			return fmt.Errorf("failed to upsert series: %w", err)
		}
		book.Series.ID = seriesID.Int64
	}

	// 2. Upsert Book
	query := `
		INSERT INTO books (
			uuid, title, sort, author_sort, timestamp, pub_date, series_id, series_index, 
			last_modified, path, has_cover, calibre_id, description
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(calibre_id) DO UPDATE SET
			uuid=excluded.uuid,
			title=excluded.title,
			sort=excluded.sort,
			author_sort=excluded.author_sort,
			timestamp=excluded.timestamp,
			pub_date=excluded.pub_date,
			series_id=excluded.series_id,
			series_index=excluded.series_index,
			last_modified=excluded.last_modified,
			path=excluded.path,
			has_cover=excluded.has_cover,
			description=excluded.description
		RETURNING id`

	err = tx.QueryRowContext(ctx, query,
		book.UUID, book.Title, book.Sort, book.AuthorSort, book.Timestamp, book.PubDate, seriesID, book.SeriesIndex,
		book.LastModified, book.Path, book.HasCover, book.CalibreID, book.Description,
	).Scan(&book.ID)
	if err != nil {
		return fmt.Errorf("failed to upsert book: %w", err)
	}

	// 3. Handle Authors
	// Clear existing links
	_, err = tx.ExecContext(ctx, "DELETE FROM books_authors_link WHERE book_id = ?", book.ID)
	if err != nil {
		return err
	}
	for i, author := range book.Authors {
		var authorID int64
		err := tx.QueryRowContext(ctx, "INSERT INTO authors (name, sort) VALUES (?, ?) ON CONFLICT(name) DO UPDATE SET sort=excluded.sort RETURNING id", author.Name, author.Sort).Scan(&authorID)
		if err != nil {
			return fmt.Errorf("failed to upsert author: %w", err)
		}
		book.Authors[i].ID = authorID
		_, err = tx.ExecContext(ctx, "INSERT INTO books_authors_link (book_id, author_id) VALUES (?, ?)", book.ID, authorID)
		if err != nil {
			return err
		}
	}

	// 4. Handle Tags
	// Clear existing links
	_, err = tx.ExecContext(ctx, "DELETE FROM books_tags_link WHERE book_id = ?", book.ID)
	if err != nil {
		return err
	}
	for i, tag := range book.Tags {
		var tagID int64
		err := tx.QueryRowContext(ctx, "INSERT INTO tags (name) VALUES (?) ON CONFLICT(name) DO UPDATE SET name=excluded.name RETURNING id", tag.Name).Scan(&tagID)
		if err != nil {
			return fmt.Errorf("failed to upsert tag: %w", err)
		}
		book.Tags[i].ID = tagID
		_, err = tx.ExecContext(ctx, "INSERT INTO books_tags_link (book_id, tag_id) VALUES (?, ?)", book.ID, tagID)
		if err != nil {
			return err
		}
	}

	// 5. Handle Formats
	_, err = tx.ExecContext(ctx, "DELETE FROM formats WHERE book_id = ?", book.ID)
	if err != nil {
		return err
	}
	for i, format := range book.Formats {
		var formatID int64
		err := tx.QueryRowContext(ctx, "INSERT INTO formats (book_id, format, uncompressed_size, name) VALUES (?, ?, ?, ?) RETURNING id", book.ID, format.Format, format.UncompressedSize, format.Name).Scan(&formatID)
		if err != nil {
			return fmt.Errorf("failed to insert format: %w", err)
		}
		book.Formats[i].ID = formatID
	}

	// 6. Update FTS5
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit upsert: %w", err)
	}

	return r.ReindexBook(ctx, book.ID)
}

func (r *sqliteBookRepository) ReindexBook(ctx context.Context, bookID int64) error {
	// First, remove existing entry if any
	_, err := r.db.ExecContext(ctx, "DELETE FROM books_search WHERE rowid = ?", bookID)
	if err != nil {
		return err
	}

	// Update books_search using a single INSERT SELECT
	query := `
		INSERT INTO books_search (rowid, title, series, authors, tags)
		SELECT 
			b.id, 
			b.title, 
			IFNULL(s.name, ''),
			(SELECT IFNULL(GROUP_CONCAT(a.name, ' '), '') FROM authors a JOIN books_authors_link bal ON a.id = bal.author_id WHERE bal.book_id = b.id),
			(SELECT IFNULL(GROUP_CONCAT(t.name, ' '), '') FROM tags t JOIN books_tags_link btl ON t.id = btl.tag_id WHERE btl.book_id = b.id)
		FROM books b 
		LEFT JOIN series s ON b.series_id = s.id 
		WHERE b.id = ?`

	_, err = r.db.ExecContext(ctx, query, bookID)
	if err != nil {
		return fmt.Errorf("failed to reindex book: %w", err)
	}

	return nil
}

// ReindexBook updates the FTS5 search table for a given book.
func ReindexBook(tx *sql.Tx, bookID int64) error {
	_, err := tx.Exec("DELETE FROM books_search WHERE rowid = ?", bookID)
	if err != nil {
		return err
	}
	query := `
		INSERT INTO books_search (rowid, title, series, authors, tags)
		SELECT 
			b.id, 
			b.title, 
			IFNULL(s.name, ''),
			(SELECT IFNULL(GROUP_CONCAT(a.name, ' '), '') FROM authors a JOIN books_authors_link bal ON a.id = bal.author_id WHERE bal.book_id = b.id),
			(SELECT IFNULL(GROUP_CONCAT(t.name, ' '), '') FROM tags t JOIN books_tags_link btl ON t.id = btl.tag_id WHERE btl.book_id = b.id)
		FROM books b 
		LEFT JOIN series s ON b.series_id = s.id 
		WHERE b.id = ?`
	_, err = tx.Exec(query, bookID)
	return err
}

func (r *sqliteBookRepository) PruneMissingCalibreIDs(ctx context.Context, keepIDs []int64) (int64, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "CREATE TEMP TABLE IF NOT EXISTS sync_keep_calibre_ids (id INTEGER PRIMARY KEY)"); err != nil {
		return 0, fmt.Errorf("failed to create sync keep table: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM sync_keep_calibre_ids"); err != nil {
		return 0, fmt.Errorf("failed to clear sync keep table: %w", err)
	}

	if len(keepIDs) > 0 {
		stmt, err := tx.PrepareContext(ctx, "INSERT OR IGNORE INTO sync_keep_calibre_ids (id) VALUES (?)")
		if err != nil {
			return 0, fmt.Errorf("failed to prepare sync keep insert: %w", err)
		}
		for _, id := range keepIDs {
			if _, err := stmt.ExecContext(ctx, id); err != nil {
				stmt.Close()
				return 0, fmt.Errorf("failed to insert sync keep id: %w", err)
			}
		}
		if err := stmt.Close(); err != nil {
			return 0, fmt.Errorf("failed to close sync keep insert: %w", err)
		}
	}

	result, err := tx.ExecContext(ctx, `
		DELETE FROM books
		WHERE calibre_id IS NOT NULL
		  AND calibre_id NOT IN (SELECT id FROM sync_keep_calibre_ids)`)
	if err != nil {
		return 0, fmt.Errorf("failed to prune missing books: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM books_search WHERE rowid NOT IN (SELECT id FROM books)"); err != nil {
		return 0, fmt.Errorf("failed to prune stale search rows: %w", err)
	}

	pruned, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to read prune count: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return pruned, nil
}

func (r *sqliteBookRepository) GetSyncState(ctx context.Context, key string) (string, error) {
	var value string
	err := r.db.QueryRowContext(ctx, "SELECT value FROM sync_state WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("failed to get sync state: %w", err)
	}
	return value, nil
}

func (r *sqliteBookRepository) SetSyncState(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, "INSERT INTO sync_state (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, value)
	if err != nil {
		return fmt.Errorf("failed to set sync state: %w", err)
	}
	return nil
}

func (r *sqliteBookRepository) EnforceStorageCap(ctx context.Context, path string, capMB int) (bool, error) {
	storage := &Storage{db: r.db}
	return storage.EnforceStorageCap(path, capMB)
}
