package scanner

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nlafevers/kopds/internal/database"
	"github.com/nlafevers/kopds/internal/logger"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) (*sql.DB, string) {
	tmpFile, err := os.CreateTemp("", "kopds-engine-test-*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()

	db, err := database.NewSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db, dbPath
}

func TestSyncEngine_Sync(t *testing.T) {
	// 1. Setup Calibre Mock DB
	tmpCalibreDir := t.TempDir()
	calibreDBPath := filepath.Join(tmpCalibreDir, "metadata.db")

	// Create the mock calibre DB
	cDB, err := sql.Open("sqlite", calibreDBPath)
	if err != nil {
		t.Fatalf("failed to open mock calibre db: %v", err)
	}

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
	if _, err := cDB.Exec(schema); err != nil {
		t.Fatalf("failed to create calibre schema: %v", err)
	}

	now := time.Now().UTC().Truncate(time.Second)
	_, err = cDB.Exec(`INSERT INTO books (id, uuid, title, sort, author_sort, timestamp, pubdate, series_index, last_modified, path, has_cover)
		VALUES (1, 'uuid-1', 'Test Book', 'Test Book', 'Author, Test', ?, ?, 1.0, ?, 'path/1', 1)`,
		now, now, now)
	if err != nil {
		t.Fatalf("failed to insert mock book: %v", err)
	}
	cDB.Close()

	// 2. Setup Local DB
	repoDB, repoDBPath := setupTestDB(t)
	defer os.Remove(repoDBPath)
	defer repoDB.Close()

	repo := database.NewBookRepository(repoDB, slog.Default())
	l := logger.New("debug", false, "")
	engine := NewSyncEngine(repo, tmpCalibreDir, repoDBPath, 0, l)

	ctx := context.Background()

	// 3. First Sync
	if err := engine.Sync(ctx); err != nil {
		t.Fatalf("First sync failed: %v", err)
	}

	// Verify book was upserted
	recent, _, err := repo.ListRecent(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list recent books: %v", err)
	}
	if len(recent) != 1 {
		t.Fatalf("Expected 1 book, got %d", len(recent))
	}
	if recent[0].Title != "Test Book" {
		t.Errorf("Expected title 'Test Book', got '%s'", recent[0].Title)
	}
	if recent[0].CalibreID != 1 {
		t.Errorf("Expected CalibreID 1, got %d", recent[0].CalibreID)
	}

	// Verify sync state
	mtime, _ := repo.GetSyncState(ctx, "calibre_mtime")
	if mtime == "" {
		t.Fatal("calibre_mtime sync state not set")
	}

	lastModifiedStr, _ := repo.GetSyncState(ctx, "last_modified_timestamp")
	if lastModifiedStr == "" {
		t.Fatal("last_modified_timestamp sync state not set")
	}

	// 4. Second Sync (should skip)
	if err := engine.Sync(ctx); err != nil {
		t.Fatalf("Second sync failed: %v", err)
	}

	// 5. Update Calibre DB and sync again
	cDB, _ = sql.Open("sqlite", calibreDBPath)
	later := now.Add(time.Hour)
	_, err = cDB.Exec(`UPDATE books SET title = 'Updated Title', last_modified = ? WHERE id = 1`, later)
	if err != nil {
		t.Fatalf("failed to update mock book: %v", err)
	}
	cDB.Close()

	// Force mtime change to ensure it's different from the previous one
	future := time.Now().Add(10 * time.Second)
	if err := os.Chtimes(calibreDBPath, future, future); err != nil {
		t.Fatalf("failed to change mtime: %v", err)
	}

	if err := engine.Sync(ctx); err != nil {
		t.Fatalf("Third sync failed: %v", err)
	}

	recent, _, _ = repo.ListRecent(ctx, 10, 0)
	if recent[0].Title != "Updated Title" {
		t.Errorf("Expected updated title 'Updated Title', got '%s'", recent[0].Title)
	}

	// 6. Delete Calibre book and verify local index pruning
	cDB, _ = sql.Open("sqlite", calibreDBPath)
	_, err = cDB.Exec(`DELETE FROM books WHERE id = 1`)
	if err != nil {
		t.Fatalf("failed to delete mock book: %v", err)
	}
	cDB.Close()

	future = time.Now().Add(20 * time.Second)
	if err := os.Chtimes(calibreDBPath, future, future); err != nil {
		t.Fatalf("failed to change mtime after delete: %v", err)
	}

	if err := engine.Sync(ctx); err != nil {
		t.Fatalf("delete sync failed: %v", err)
	}

	recent, _, err = repo.ListRecent(ctx, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list recent books after delete: %v", err)
	}
	if len(recent) != 0 {
		t.Fatalf("Expected deleted book to be pruned, got %d books", len(recent))
	}
}
