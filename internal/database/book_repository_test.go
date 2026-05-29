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

	db, err := NewSQLite(dbPath, true)
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
