package scanner

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupMockCalibreDB(t *testing.T) string {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "metadata.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open mock db: %v", err)
	}
	defer db.Close()

	schema := `
	CREATE TABLE books (
		id INTEGER PRIMARY KEY,
		uuid TEXT,
		title TEXT,
		sort TEXT,
		author_sort TEXT,
		timestamp TIMESTAMP,
		pubdate TIMESTAMP,
		series_index REAL,
		last_modified TIMESTAMP,
		path TEXT,
		has_cover BOOLEAN
	);
	CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT, sort TEXT);
	CREATE TABLE tags (id INTEGER PRIMARY KEY, name TEXT);
	CREATE TABLE series (id INTEGER PRIMARY KEY, name TEXT);
	CREATE TABLE comments (id INTEGER PRIMARY KEY, book INTEGER, text TEXT);
	CREATE TABLE data (id INTEGER PRIMARY KEY, book INTEGER, format TEXT, uncompressed_size INTEGER, name TEXT);
	CREATE TABLE books_authors_link (id INTEGER PRIMARY KEY, book INTEGER, author INTEGER);
	CREATE TABLE books_tags_link (id INTEGER PRIMARY KEY, book INTEGER, tag INTEGER);
	CREATE TABLE books_series_link (id INTEGER PRIMARY KEY, book INTEGER, series INTEGER);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Insert mock data
	now := time.Now().UTC().Truncate(time.Second)
	_, err = db.Exec(`INSERT INTO books (id, uuid, title, sort, author_sort, timestamp, pubdate, series_index, last_modified, path, has_cover)
		VALUES (1, 'uuid-1', 'Test Book', 'Test Book', 'Author, Test', ?, ?, 1.0, ?, 'path/1', 1)`,
		now, now, now)
	if err != nil {
		t.Fatalf("failed to insert mock book: %v", err)
	}

	_, err = db.Exec(`INSERT INTO authors (id, name, sort) VALUES (1, 'Test Author', 'Author, Test')`)
	if err != nil {
		t.Fatalf("failed to insert mock author: %v", err)
	}
	_, err = db.Exec(`INSERT INTO books_authors_link (book, author) VALUES (1, 1)`)
	if err != nil {
		t.Fatalf("failed to insert mock book_author_link: %v", err)
	}

	_, err = db.Exec(`INSERT INTO tags (id, name) VALUES (1, 'Test Tag')`)
	if err != nil {
		t.Fatalf("failed to insert mock tag: %v", err)
	}
	_, err = db.Exec(`INSERT INTO books_tags_link (book, tag) VALUES (1, 1)`)
	if err != nil {
		t.Fatalf("failed to insert mock book_tag_link: %v", err)
	}

	_, err = db.Exec(`INSERT INTO series (id, name) VALUES (1, 'Test Series')`)
	if err != nil {
		t.Fatalf("failed to insert mock series: %v", err)
	}
	_, err = db.Exec(`INSERT INTO books_series_link (book, series) VALUES (1, 1)`)
	if err != nil {
		t.Fatalf("failed to insert mock book_series_link: %v", err)
	}

	_, err = db.Exec(`INSERT INTO comments (book, text) VALUES (1, 'Test Description')`)
	if err != nil {
		t.Fatalf("failed to insert mock comment: %v", err)
	}

	_, err = db.Exec(`INSERT INTO data (book, format, uncompressed_size, name) VALUES (1, 'EPUB', 1024, 'Test Book')`)
	if err != nil {
		t.Fatalf("failed to insert mock data: %v", err)
	}

	return dbPath
}

func TestCalibreDSN(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantDSN  string
		wantMode string
	}{
		{
			name:    "plain path",
			path:    "/data/metadata.db",
			wantDSN: "file:%2Fdata%2Fmetadata.db?mode=ro",
		},
		{
			name:    "path with spaces",
			path:    "/my library/metadata.db",
			wantDSN: "file:%2Fmy%20library%2Fmetadata.db?mode=ro",
		},
		{
			name:    "path with hash",
			path:    "/data/my#library/metadata.db",
			wantDSN: "file:%2Fdata%2Fmy%23library%2Fmetadata.db?mode=ro",
		},
		{
			name:    "path with percent",
			path:    "/data/100%books/metadata.db",
			wantDSN: "file:%2Fdata%2F100%25books%2Fmetadata.db?mode=ro",
		},
		{
			name:    "path with question mark",
			path:    "/data/what?/metadata.db",
			wantDSN: "file:%2Fdata%2Fwhat%3F%2Fmetadata.db?mode=ro",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := calibreDSN(tc.path)
			if got != tc.wantDSN {
				t.Errorf("calibreDSN(%q) = %q, want %q", tc.path, got, tc.wantDSN)
			}
			if !strings.HasSuffix(got, "?mode=ro") {
				t.Errorf("calibreDSN(%q) does not end with ?mode=ro: %q", tc.path, got)
			}
		})
	}
}

func TestNewCalibreReaderWithSpacesInPath(t *testing.T) {
	// Create a temp directory with a space in its name to verify that
	// NewCalibreReader correctly handles paths containing special characters.
	parentDir := t.TempDir()
	spacedDir := filepath.Join(parentDir, "my library")
	if err := os.MkdirAll(spacedDir, 0o750); err != nil {
		t.Fatalf("failed to create directory with space: %v", err)
	}
	dbPath := filepath.Join(spacedDir, "metadata.db")

	// Create a minimal Calibre-like schema in the test DB.
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to create mock db: %v", err)
	}
	schema := `
	CREATE TABLE books (
		id INTEGER PRIMARY KEY, uuid TEXT, title TEXT, sort TEXT,
		author_sort TEXT, timestamp TIMESTAMP, pubdate TIMESTAMP,
		series_index REAL, last_modified TIMESTAMP, path TEXT, has_cover BOOLEAN
	);
	CREATE TABLE authors (id INTEGER PRIMARY KEY, name TEXT, sort TEXT);
	CREATE TABLE tags (id INTEGER PRIMARY KEY, name TEXT);
	CREATE TABLE series (id INTEGER PRIMARY KEY, name TEXT);
	CREATE TABLE comments (id INTEGER PRIMARY KEY, book INTEGER, text TEXT);
	CREATE TABLE data (id INTEGER PRIMARY KEY, book INTEGER, format TEXT, uncompressed_size INTEGER, name TEXT);
	CREATE TABLE books_authors_link (id INTEGER PRIMARY KEY, book INTEGER, author INTEGER);
	CREATE TABLE books_tags_link (id INTEGER PRIMARY KEY, book INTEGER, tag INTEGER);
	CREATE TABLE books_series_link (id INTEGER PRIMARY KEY, book INTEGER, series INTEGER);
	`
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		t.Fatalf("failed to create schema: %v", err)
	}
	db.Close()

	reader, err := NewCalibreReader(dbPath)
	if err != nil {
		t.Fatalf("NewCalibreReader with spaced path failed: %v", err)
	}
	defer reader.Close()

	ctx := context.Background()
	books, err := reader.GetChangedBooks(ctx, time.Time{})
	if err != nil {
		t.Fatalf("GetChangedBooks failed: %v", err)
	}
	if len(books) != 0 {
		t.Errorf("expected 0 books, got %d", len(books))
	}
}

func TestCalibreReader(t *testing.T) {
	dbPath := setupMockCalibreDB(t)
	defer os.RemoveAll(filepath.Dir(dbPath))

	reader, err := NewCalibreReader(dbPath)
	if err != nil {
		t.Fatalf("failed to create reader: %v", err)
	}
	defer reader.Close()

	ctx := context.Background()
	since := time.Now().Add(-1 * time.Hour)

	books, err := reader.GetChangedBooks(ctx, since)
	if err != nil {
		t.Fatalf("failed to get changed books: %v", err)
	}

	if len(books) != 1 {
		t.Fatalf("expected 1 book, got %d", len(books))
	}

	book := books[0]
	if book.Title != "Test Book" {
		t.Errorf("expected title 'Test Book', got '%s'", book.Title)
	}
	if book.Description != "Test Description" {
		t.Errorf("expected description 'Test Description', got '%s'", book.Description)
	}

	err = reader.PopulateMetadata(ctx, books)
	if err != nil {
		t.Fatalf("failed to populate metadata: %v", err)
	}

	book = books[0]
	if len(book.Authors) != 1 || book.Authors[0].Name != "Test Author" {
		t.Errorf("expected 1 author 'Test Author', got %v", book.Authors)
	}
	if len(book.Tags) != 1 || book.Tags[0].Name != "Test Tag" {
		t.Errorf("expected 1 tag 'Test Tag', got %v", book.Tags)
	}
	if book.Series == nil || book.Series.Name != "Test Series" {
		t.Errorf("expected series 'Test Series', got %v", book.Series)
	}
	if len(book.Formats) != 1 || book.Formats[0].Format != "EPUB" {
		t.Errorf("expected 1 format 'EPUB', got %v", book.Formats)
	}
}
