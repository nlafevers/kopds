package database

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/nlafevers/kopds/internal/domain"

	_ "modernc.org/sqlite"
)

func TestBookRepository_UpsertAndSearch(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "kopds-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	db, err := OpenSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	repo := NewBookRepository(db, slog.Default())
	ctx := context.Background()

	book := &domain.Book{
		UUID:        "test-uuid-1",
		Title:       "The Go Programming Language",
		Sort:        "Go Programming Language, The",
		AuthorSort:  "Kernighan, Brian W. and Donovan, Alan A. A.",
		Timestamp:   time.Now(),
		PubDate:     time.Now(),
		SeriesIndex: 1,
		Path:        "/path/to/book",
		HasCover:    true,
		CalibreID:   123,
		Description: "A great book about Go.",
		Authors: []domain.Author{
			{Name: "Alan A. A. Donovan", Sort: "Donovan, Alan A. A."},
			{Name: "Brian W. Kernighan", Sort: "Kernighan, Brian W."},
		},
		Tags: []domain.Tag{
			{Name: "Programming"},
			{Name: "Go"},
		},
		Series: &domain.Series{
			Name: "Computer Science",
		},
		Formats: []domain.Format{
			{Format: "EPUB", UncompressedSize: 1024, Name: "book.epub"},
		},
	}

	// Test Upsert
	if err := repo.Upsert(ctx, book); err != nil {
		t.Fatalf("failed to upsert book: %v", err)
	}

	if book.ID == 0 {
		t.Fatal("book ID should not be 0 after upsert")
	}

	// Test GetByID
	got, err := repo.GetByID(ctx, book.ID)
	if err != nil {
		t.Fatalf("failed to get book by ID: %v", err)
	}

	if got == nil {
		t.Fatal("expected book, got nil")
	}

	if got.Title != book.Title {
		t.Errorf("expected title %s, got %s", book.Title, got.Title)
	}

	if len(got.Authors) != 2 {
		t.Errorf("expected 2 authors, got %d", len(got.Authors))
	}

	if got.Series == nil || got.Series.Name != "Computer Science" {
		t.Errorf("expected series 'Computer Science', got %v", got.Series)
	}

	// Test Search
	books, total, err := repo.Search(ctx, "Go", 10, 0)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if len(books) == 0 {
		t.Fatal("expected search results, got none")
	}

	if total == 0 {
		t.Fatal("expected total > 0, got 0")
	}

	if books[0].Title != book.Title {
		t.Errorf("expected search result title %s, got %s", book.Title, books[0].Title)
	}

	// Test Search by Author
	books, _, err = repo.Search(ctx, "Kernighan", 10, 0)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(books) == 0 {
		t.Fatal("expected search results for author, got none")
	}

	// Test Search with punctuation that would be invalid raw FTS syntax.
	books, _, err = repo.Search(ctx, `Go: "Programming"`, 10, 0)
	if err != nil {
		t.Fatalf("search with punctuation failed: %v", err)
	}
	if len(books) == 0 {
		t.Fatal("expected search results for punctuation query, got none")
	}

	books, total, err = repo.Search(ctx, `":`, 10, 0)
	if err != nil {
		t.Fatalf("punctuation-only search should not fail: %v", err)
	}
	if len(books) != 0 || total != 0 {
		t.Fatalf("expected no results for punctuation-only search, got %d total %d", len(books), total)
	}

	// Test ListRecent
	recent, _, err := repo.ListRecent(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}
	if len(recent) == 0 {
		t.Fatal("expected recent books, got none")
	}

	// Test ListByAuthor
	byAuthor, _, err := repo.ListByAuthor(ctx, book.Authors[0].ID, 10, 0)
	if err != nil {
		t.Fatalf("ListByAuthor failed: %v", err)
	}
	if len(byAuthor) == 0 {
		t.Fatal("expected books by author, got none")
	}

	// Test ListBySeries
	bySeries, _, err := repo.ListBySeries(ctx, book.Series.ID, 10, 0)
	if err != nil {
		t.Fatalf("ListBySeries failed: %v", err)
	}
	if len(bySeries) == 0 {
		t.Fatal("expected books by series, got none")
	}

	pruned, err := repo.PruneMissingCalibreIDs(ctx, []int64{})
	if err != nil {
		t.Fatalf("PruneMissingCalibreIDs failed: %v", err)
	}
	if pruned != 1 {
		t.Fatalf("expected 1 pruned book, got %d", pruned)
	}

	got, err = repo.GetByID(ctx, book.ID)
	if err != nil {
		t.Fatalf("failed to get pruned book: %v", err)
	}
	if got != nil {
		t.Fatal("expected pruned book to be deleted")
	}
}

// TestListRecent_OrderPreserved verifies that ListRecent returns books in
// descending timestamp order and that all relations are hydrated correctly.
func TestListRecent_OrderPreserved(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "kopds-order-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	db, err := OpenSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	repo := NewBookRepository(db, slog.Default())
	ctx := context.Background()

	now := time.Now()

	// Insert books with different timestamps so the ORDER BY is deterministic.
	books := []*domain.Book{
		{
			UUID: "uuid-oldest", Title: "Alpha Book", Sort: "Alpha Book", CalibreID: 101,
			Timestamp: now.Add(-2 * time.Hour),
			Authors:   []domain.Author{{Name: "Author One", Sort: "One, Author"}},
			Tags:      []domain.Tag{{Name: "Science"}},
			Formats:   []domain.Format{{Format: "EPUB", UncompressedSize: 100, Name: "alpha.epub"}},
		},
		{
			UUID: "uuid-middle", Title: "Beta Book", Sort: "Beta Book", CalibreID: 102,
			Timestamp: now.Add(-1 * time.Hour),
			Authors:   []domain.Author{{Name: "Author Two", Sort: "Two, Author"}},
			Tags:      []domain.Tag{{Name: "Fiction"}},
			Formats:   []domain.Format{{Format: "EPUB", UncompressedSize: 200, Name: "beta.epub"}},
		},
		{
			UUID: "uuid-newest", Title: "Gamma Book", Sort: "Gamma Book", CalibreID: 103,
			Timestamp: now,
			Authors:   []domain.Author{{Name: "Author Three", Sort: "Three, Author"}},
			Tags:      []domain.Tag{{Name: "History"}},
			Formats:   []domain.Format{{Format: "EPUB", UncompressedSize: 300, Name: "gamma.epub"}},
		},
	}
	for _, b := range books {
		if err := repo.Upsert(ctx, b); err != nil {
			t.Fatalf("failed to upsert book %q: %v", b.Title, err)
		}
	}

	recent, total, err := repo.ListRecent(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListRecent failed: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected total=3, got %d", total)
	}
	if len(recent) != 3 {
		t.Fatalf("expected 3 books, got %d", len(recent))
	}

	// Newest first.
	expectedOrder := []string{"Gamma Book", "Beta Book", "Alpha Book"}
	for i, title := range expectedOrder {
		if recent[i].Title != title {
			t.Errorf("position %d: expected %q, got %q", i, title, recent[i].Title)
		}
	}

	// Relations must be hydrated for every book.
	for _, b := range recent {
		if len(b.Authors) == 0 {
			t.Errorf("book %q has no authors after batch hydration", b.Title)
		}
		if len(b.Tags) == 0 {
			t.Errorf("book %q has no tags after batch hydration", b.Title)
		}
		if len(b.Formats) == 0 {
			t.Errorf("book %q has no formats after batch hydration", b.Title)
		}
	}
}

// TestSearch_OrderPreserved verifies that Search returns books in FTS rank order
// and that all relations are hydrated correctly.
func TestSearch_OrderPreserved(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "kopds-search-order-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	db, err := OpenSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	repo := NewBookRepository(db, slog.Default())
	ctx := context.Background()

	insertBooks := []*domain.Book{
		{
			UUID: "uuid-s1", Title: "Golang Patterns", Sort: "Golang Patterns", CalibreID: 201,
			Authors: []domain.Author{{Name: "Dev Writer", Sort: "Writer, Dev"}},
			Tags:    []domain.Tag{{Name: "Programming"}},
			Formats: []domain.Format{{Format: "PDF", UncompressedSize: 400, Name: "golang.pdf"}},
		},
		{
			UUID: "uuid-s2", Title: "Rust Systems", Sort: "Rust Systems", CalibreID: 202,
			Authors: []domain.Author{{Name: "Systems Expert", Sort: "Expert, Systems"}},
			Tags:    []domain.Tag{{Name: "Systems"}},
			Formats: []domain.Format{{Format: "EPUB", UncompressedSize: 500, Name: "rust.epub"}},
		},
	}
	for _, b := range insertBooks {
		if err := repo.Upsert(ctx, b); err != nil {
			t.Fatalf("failed to upsert: %v", err)
		}
	}

	results, total, err := repo.Search(ctx, "Golang", 10, 0)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if total == 0 {
		t.Fatal("expected total > 0")
	}
	if len(results) == 0 {
		t.Fatal("expected search results, got none")
	}
	if results[0].Title != "Golang Patterns" {
		t.Errorf("expected first result 'Golang Patterns', got %q", results[0].Title)
	}

	// Relations must be hydrated.
	for _, b := range results {
		if len(b.Authors) == 0 {
			t.Errorf("book %q has no authors after batch hydration", b.Title)
		}
		if len(b.Formats) == 0 {
			t.Errorf("book %q has no formats after batch hydration", b.Title)
		}
	}
}

// TestListByAuthor_CorrectBooks is a regression test for the ListByAuthor join bug.
// Previously the join used `b.id = bal.author_id` (wrong column) instead of
// `b.id = bal.book_id`, which caused incorrect or empty results.
func TestListByAuthor_CorrectBooks(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "kopds-listbyauthor-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	db, err := OpenSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	repo := NewBookRepository(db, slog.Default())
	ctx := context.Background()

	// Author A writes book1; Author B writes book2.
	book1 := &domain.Book{
		UUID:      "uuid-book1",
		Title:     "Book One",
		Sort:      "Book One",
		CalibreID: 1,
		Authors:   []domain.Author{{Name: "Author Alpha", Sort: "Alpha, Author"}},
	}
	book2 := &domain.Book{
		UUID:      "uuid-book2",
		Title:     "Book Two",
		Sort:      "Book Two",
		CalibreID: 2,
		Authors:   []domain.Author{{Name: "Author Beta", Sort: "Beta, Author"}},
	}

	if err := repo.Upsert(ctx, book1); err != nil {
		t.Fatalf("failed to upsert book1: %v", err)
	}
	if err := repo.Upsert(ctx, book2); err != nil {
		t.Fatalf("failed to upsert book2: %v", err)
	}

	authorAlphaID := book1.Authors[0].ID
	authorBetaID := book2.Authors[0].ID

	// Author Alpha should return exactly book1, not book2.
	alphaBooks, total, err := repo.ListByAuthor(ctx, authorAlphaID, 10, 0)
	if err != nil {
		t.Fatalf("ListByAuthor for Alpha failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1 for Author Alpha, got %d", total)
	}
	if len(alphaBooks) != 1 {
		t.Fatalf("expected 1 book for Author Alpha, got %d", len(alphaBooks))
	}
	if alphaBooks[0].Title != "Book One" {
		t.Errorf("expected 'Book One' for Author Alpha, got %q", alphaBooks[0].Title)
	}

	// Author Beta should return exactly book2, not book1.
	betaBooks, total, err := repo.ListByAuthor(ctx, authorBetaID, 10, 0)
	if err != nil {
		t.Fatalf("ListByAuthor for Beta failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total=1 for Author Beta, got %d", total)
	}
	if len(betaBooks) != 1 {
		t.Fatalf("expected 1 book for Author Beta, got %d", len(betaBooks))
	}
	if betaBooks[0].Title != "Book Two" {
		t.Errorf("expected 'Book Two' for Author Beta, got %q", betaBooks[0].Title)
	}
}
