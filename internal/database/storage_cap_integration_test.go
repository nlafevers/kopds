package database

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestEnforceStorageCapIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := OpenSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}
	storage := NewStorage(db, slog.Default())
	defer storage.db.Close()

	// Bloat the database with dummy sync_state records
	// We need enough data to exceed 1MB
	for i := 0; i < 20000; i++ {
		key := fmt.Sprintf("key_%d", i)
		_, err := storage.db.Exec("INSERT INTO sync_state (key, value) VALUES (?, ?)", key, "dummy_data_to_bloat_the_database_file_size_significantly_more_than_before")
		if err != nil {
			t.Fatalf("failed to insert dummy data: %v", err)
		}
	}

	info, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("failed to stat db: %v", err)
	}
	initialSize := info.Size()
    t.Logf("Initial database size: %d bytes (Cap: %d)", initialSize, 1*1024*1024)

	// Enforce 1MB cap
	pruned, err := storage.EnforceStorageCap(dbPath, 1)
	if err != nil {
		t.Fatalf("EnforceStorageCap failed: %v", err)
	}
	if !pruned {
		t.Fatal("expected pruning to occur")
	}

	// Verify pruning worked
	var count int
	err = storage.db.QueryRow("SELECT COUNT(*) FROM sync_state").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query count: %v", err)
	}
	if count >= 20000 {
		t.Errorf("expected fewer than 20000 records, got %d", count)
	}
    t.Logf("Final record count: %d", count)
}
