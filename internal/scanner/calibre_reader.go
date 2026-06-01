package scanner

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/nlafevers/kopds/internal/domain"
	_ "modernc.org/sqlite"
)

type CalibreReader struct {
	db *sql.DB
}

// calibreDSN builds a read-only SQLite URI for the given file path.
// url.PathEscape percent-encodes characters that are special in URI paths
// (spaces, #, ?, %, etc.) so that they are not misinterpreted by the driver.
func calibreDSN(path string) string {
	return "file:" + url.PathEscape(path) + "?mode=ro"
}

// NewCalibreReader opens the Calibre metadata.db in read-only mode.
func NewCalibreReader(path string) (*CalibreReader, error) {
	// Calibre uses standard SQLite. The connection MUST be read-only.
	// Use calibreDSN to safely encode special characters in the path.
	dsn := calibreDSN(path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open calibre database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping calibre database: %w", err)
	}

	return &CalibreReader{db: db}, nil
}

func (r *CalibreReader) Close() error {
	return r.db.Close()
}

// GetChangedBooks queries Calibre's 'books' table for books modified since the given time.
func (r *CalibreReader) GetChangedBooks(ctx context.Context, since time.Time) ([]domain.Book, error) {
	query := `
		SELECT b.id, b.uuid, b.title, b.sort, b.author_sort, b.timestamp, b.pubdate, b.series_index, b.last_modified, b.path, b.has_cover, c.text as description
		FROM books b
		LEFT JOIN comments c ON b.id = c.book
		WHERE b.last_modified > ?
	`

	rows, err := r.db.QueryContext(ctx, query, since)
	if err != nil {
		return nil, fmt.Errorf("failed to query changed books: %w", err)
	}
	defer rows.Close()

	var books []domain.Book
	for rows.Next() {
		var b domain.Book
		var description sql.NullString
		var pubDate sql.NullTime
		var timestamp sql.NullTime
		var lastModified sql.NullTime

		err := rows.Scan(
			&b.CalibreID,
			&b.UUID,
			&b.Title,
			&b.Sort,
			&b.AuthorSort,
			&timestamp,
			&pubDate,
			&b.SeriesIndex,
			&lastModified,
			&b.Path,
			&b.HasCover,
			&description,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan book row: %w", err)
		}

		if description.Valid {
			b.Description = description.String
		}
		if pubDate.Valid {
			b.PubDate = pubDate.Time
		}
		if timestamp.Valid {
			b.Timestamp = timestamp.Time
		}
		if lastModified.Valid {
			b.LastModified = lastModified.Time
		}

		books = append(books, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate changed books: %w", err)
	}

	return books, nil
}

func (r *CalibreReader) GetAllBookIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT id FROM books")
	if err != nil {
		return nil, fmt.Errorf("failed to query calibre book ids: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("failed to scan calibre book id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate calibre book ids: %w", err)
	}
	return ids, nil
}

// PopulateMetadata fills in Authors, Tags, Series, and Formats for the given books.
func (r *CalibreReader) PopulateMetadata(ctx context.Context, books []domain.Book) error {
	if len(books) == 0 {
		return nil
	}

	bookIDs := make([]interface{}, len(books))
	bookMap := make(map[int64]*domain.Book, len(books))
	for i := range books {
		bookIDs[i] = books[i].CalibreID
		bookMap[books[i].CalibreID] = &books[i]
	}

	placeholders := make([]string, len(books))
	for i := range placeholders {
		placeholders[i] = "?"
	}
	placeholderStr := strings.Join(placeholders, ",")

	// 1. Authors
	if err := r.populateAuthors(ctx, bookIDs, placeholderStr, bookMap); err != nil {
		return err
	}

	// 2. Tags
	if err := r.populateTags(ctx, bookIDs, placeholderStr, bookMap); err != nil {
		return err
	}

	// 3. Series
	if err := r.populateSeries(ctx, bookIDs, placeholderStr, bookMap); err != nil {
		return err
	}

	// 4. Formats
	if err := r.populateFormats(ctx, bookIDs, placeholderStr, bookMap); err != nil {
		return err
	}

	return nil
}

func (r *CalibreReader) populateAuthors(ctx context.Context, bookIDs []interface{}, placeholderStr string, bookMap map[int64]*domain.Book) error {
	query := fmt.Sprintf(`
		SELECT bal.book, a.id, a.name, a.sort
		FROM authors a
		JOIN books_authors_link bal ON a.id = bal.author
		WHERE bal.book IN (%s)
	`, placeholderStr)

	rows, err := r.db.QueryContext(ctx, query, bookIDs...)
	if err != nil {
		return fmt.Errorf("failed to query authors: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bookID int64
		var author domain.Author
		if err := rows.Scan(&bookID, &author.ID, &author.Name, &author.Sort); err != nil {
			return fmt.Errorf("failed to scan author row: %w", err)
		}
		if b, ok := bookMap[bookID]; ok {
			b.Authors = append(b.Authors, author)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate authors: %w", err)
	}
	return nil
}

func (r *CalibreReader) populateTags(ctx context.Context, bookIDs []interface{}, placeholderStr string, bookMap map[int64]*domain.Book) error {
	query := fmt.Sprintf(`
		SELECT btl.book, t.id, t.name
		FROM tags t
		JOIN books_tags_link btl ON t.id = btl.tag
		WHERE btl.book IN (%s)
	`, placeholderStr)

	rows, err := r.db.QueryContext(ctx, query, bookIDs...)
	if err != nil {
		return fmt.Errorf("failed to query tags: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bookID int64
		var tag domain.Tag
		if err := rows.Scan(&bookID, &tag.ID, &tag.Name); err != nil {
			return fmt.Errorf("failed to scan tag row: %w", err)
		}
		if b, ok := bookMap[bookID]; ok {
			b.Tags = append(b.Tags, tag)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate tags: %w", err)
	}
	return nil
}

func (r *CalibreReader) populateSeries(ctx context.Context, bookIDs []interface{}, placeholderStr string, bookMap map[int64]*domain.Book) error {
	query := fmt.Sprintf(`
		SELECT bsl.book, s.id, s.name
		FROM series s
		JOIN books_series_link bsl ON s.id = bsl.series
		WHERE bsl.book IN (%s)
	`, placeholderStr)

	rows, err := r.db.QueryContext(ctx, query, bookIDs...)
	if err != nil {
		return fmt.Errorf("failed to query series: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bookID int64
		var series domain.Series
		if err := rows.Scan(&bookID, &series.ID, &series.Name); err != nil {
			return fmt.Errorf("failed to scan series row: %w", err)
		}
		if b, ok := bookMap[bookID]; ok {
			b.Series = &series
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate series: %w", err)
	}
	return nil
}

func (r *CalibreReader) populateFormats(ctx context.Context, bookIDs []interface{}, placeholderStr string, bookMap map[int64]*domain.Book) error {
	query := fmt.Sprintf(`
		SELECT book, id, format, uncompressed_size, name
		FROM data
		WHERE book IN (%s)
	`, placeholderStr)

	rows, err := r.db.QueryContext(ctx, query, bookIDs...)
	if err != nil {
		return fmt.Errorf("failed to query formats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bookID int64
		var format domain.Format
		if err := rows.Scan(&bookID, &format.ID, &format.Format, &format.UncompressedSize, &format.Name); err != nil {
			return fmt.Errorf("failed to scan format row: %w", err)
		}
		if b, ok := bookMap[bookID]; ok {
			b.Formats = append(b.Formats, format)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("failed to iterate formats: %w", err)
	}
	return nil
}
