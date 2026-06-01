package database

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnforceStorageCapHelper(t *testing.T) {
	missingPath := filepath.Join(t.TempDir(), "missing.db")

	t.Run("disabled cap skips file stat and callbacks", func(t *testing.T) {
		pruned, err := enforceStorageCap(missingPath, 0, func() (int64, error) {
			t.Fatal("prune should not run when cap is disabled")
			return 0, nil
		}, func() error {
			t.Fatal("vacuum should not run when cap is disabled")
			return nil
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if pruned {
			t.Fatal("expected no pruning when cap is disabled")
		}
	})

	t.Run("missing file returns error", func(t *testing.T) {
		pruned, err := enforceStorageCap(missingPath, 1, func() (int64, error) {
			t.Fatal("prune should not run when stat fails")
			return 0, nil
		}, func() error {
			t.Fatal("vacuum should not run when stat fails")
			return nil
		})
		if err == nil {
			t.Fatal("expected stat error")
		}
		if pruned {
			t.Fatal("expected no pruning when stat fails")
		}
	})

	t.Run("below cap skips callbacks", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "small.db")
		if err := os.WriteFile(path, []byte("small"), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		pruned, err := enforceStorageCap(path, 1, func() (int64, error) {
			t.Fatal("prune should not run below cap")
			return 0, nil
		}, func() error {
			t.Fatal("vacuum should not run below cap")
			return nil
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if pruned {
			t.Fatal("expected no pruning below cap")
		}
	})

	t.Run("at cap runs prune before vacuum", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "large.db")
		if err := os.WriteFile(path, make([]byte, 1024*1024), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		var calls []string
		pruned, err := enforceStorageCap(path, 1, func() (int64, error) {
			calls = append(calls, "prune")
			return 1, nil
		}, func() error {
			calls = append(calls, "vacuum")
			return nil
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !pruned {
			t.Fatal("expected pruning at cap")
		}
		if len(calls) != 2 || calls[0] != "prune" || calls[1] != "vacuum" {
			t.Fatalf("expected prune then vacuum, got %v", calls)
		}
	})

	t.Run("prune error skips vacuum", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "large.db")
		if err := os.WriteFile(path, make([]byte, 1024*1024), 0600); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		expectedErr := errors.New("prune failed")
		pruned, err := enforceStorageCap(path, 1, func() (int64, error) {
			return 0, expectedErr
		}, func() error {
			t.Fatal("vacuum should not run after prune error")
			return nil
		})
		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected prune error, got %v", err)
		}
		if pruned {
			t.Fatal("expected no pruning result after prune error")
		}
	})
}

func TestStorageCapLogsMaintenance(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	previous := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() {
		slog.SetDefault(previous)
	})

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "cap_logging.db")
	db, err := OpenSQLite(dbPath, true)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	if err := Migrate(db); err != nil {
		t.Fatalf("failed to migrate db: %v", err)
	}

	storage := &Storage{db: db, log: logger}
	for i := 0; i < 10; i++ {
		_, err := db.Exec("INSERT INTO sync_state (key, value) VALUES (?, ?)", fmt.Sprintf("key-%d", i), "value")
		if err != nil {
			t.Fatalf("failed to insert sync_state row: %v", err)
		}
	}

	if err := os.Truncate(dbPath, 2*1024*1024); err != nil {
		t.Fatalf("failed to enlarge database file: %v", err)
	}

	if _, err := storage.EnforceStorageCap(dbPath, 1); err != nil {
		t.Fatalf("EnforceStorageCap failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "checking storage cap") {
		t.Fatalf("expected storage cap check log, got %s", output)
	}
	if !strings.Contains(output, "storage cap exceeded") {
		t.Fatalf("expected storage cap exceeded log, got %s", output)
	}
	if !strings.Contains(output, "pruning storage cap records") {
		t.Fatalf("expected pruning log, got %s", output)
	}
	if !strings.Contains(output, "storage cap records pruned") {
		t.Fatalf("expected pruned summary log, got %s", output)
	}
	if !strings.Contains(output, "database vacuum completed") {
		t.Fatalf("expected vacuum completion log, got %s", output)
	}
}
