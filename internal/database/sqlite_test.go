package database

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestForeignKeyEnforcement(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "fk_test.db")

	db, err := OpenSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	t.Run("inserting format with nonexistent book_id fails", func(t *testing.T) {
		_, err := db.Exec(
			`INSERT INTO formats (book_id, format) VALUES (?, ?)`,
			99999, "EPUB",
		)
		if err == nil {
			t.Fatal("expected FK violation error, got nil")
		}
	})

	t.Run("inserting book with valid series_id succeeds", func(t *testing.T) {
		_, err := db.Exec(
			`INSERT INTO series (name) VALUES (?)`, "Test Series",
		)
		if err != nil {
			t.Fatalf("failed to insert series: %v", err)
		}

		var seriesID int64
		if err := db.QueryRow("SELECT id FROM series WHERE name = ?", "Test Series").Scan(&seriesID); err != nil {
			t.Fatalf("failed to query series: %v", err)
		}

		_, err = db.Exec(
			`INSERT INTO books (uuid, title, path, series_id) VALUES (?, ?, ?, ?)`,
			"uuid-fk-test", "FK Test Book", "/books/fktest", seriesID,
		)
		if err != nil {
			t.Fatalf("expected valid insert to succeed, got: %v", err)
		}
	})

	t.Run("inserting book with nonexistent series_id fails", func(t *testing.T) {
		_, err := db.Exec(
			`INSERT INTO books (uuid, title, path, series_id) VALUES (?, ?, ?, ?)`,
			"uuid-bad-series", "Bad Series Book", "/books/badseries", 99999,
		)
		if err == nil {
			t.Fatal("expected FK violation error, got nil")
		}
	})
}

func TestEnforceStorageCapDisabled(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	storage := &Storage{log: logger}

	// Use a path that does not exist; a disabled cap must not stat the file.
	nonExistentPath := filepath.Join(t.TempDir(), "does_not_exist.db")

	t.Run("cap zero skips file stat and returns false", func(t *testing.T) {
		pruned, err := storage.EnforceStorageCap(nonExistentPath, 0)
		if err != nil {
			t.Errorf("expected no error with cap=0, got %v", err)
		}
		if pruned {
			t.Error("expected pruned=false with cap=0")
		}
	})

	t.Run("negative cap skips file stat and returns false", func(t *testing.T) {
		pruned, err := storage.EnforceStorageCap(nonExistentPath, -1)
		if err != nil {
			t.Errorf("expected no error with cap=-1, got %v", err)
		}
		if pruned {
			t.Error("expected pruned=false with cap=-1")
		}
	})

	output := buf.String()
	if !strings.Contains(output, "storage cap disabled") {
		t.Errorf("expected disabled cap log message, got: %s", output)
	}
}

func TestOpenSQLitePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "perm_test.db")

	db, err := OpenSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	db.Close()

	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("failed to stat database file: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("expected 0600 permissions, got %o", info.Mode().Perm())
	}
}
