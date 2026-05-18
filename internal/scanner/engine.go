package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nlafevers/kopds/internal/domain"
	"log/slog"
)

type SyncEngine struct {
	Repo         domain.BookRepository
	CalibrePath  string
	DatabasePath string
	StorageCapMB int
	Logger       *slog.Logger
}

// NewSyncEngine creates a new synchronization engine.
func NewSyncEngine(repo domain.BookRepository, calibrePath, dbPath string, capMB int, logger *slog.Logger) *SyncEngine {
	return &SyncEngine{
		Repo:         repo,
		CalibrePath:  calibrePath,
		DatabasePath: dbPath,
		StorageCapMB: capMB,
		Logger:       logger,
	}
}

// Sync performs the synchronization between the Calibre library and the local index.
func (e *SyncEngine) Sync(ctx context.Context) error {
	dbPath := filepath.Join(e.CalibrePath, "metadata.db")
	info, err := os.Stat(dbPath)
	if err != nil {
		return fmt.Errorf("failed to stat calibre metadata.db at %s: %w", dbPath, err)
	}

	currentMtime := info.ModTime().Unix()
	currentSize := info.Size()

	lastMtimeStr, _ := e.Repo.GetSyncState(ctx, "calibre_mtime")
	lastSizeStr, _ := e.Repo.GetSyncState(ctx, "calibre_size")

	if lastMtimeStr == strconv.FormatInt(currentMtime, 10) && lastSizeStr == strconv.FormatInt(currentSize, 10) {
		e.Logger.Debug("Calibre library metadata.db unchanged, skipping sync")
		return nil
	}

	reader, err := NewCalibreReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize calibre reader: %w", err)
	}
	defer reader.Close()

	allCalibreIDs, err := reader.GetAllBookIDs(ctx)
	if err != nil {
		return fmt.Errorf("failed to read calibre book ids: %w", err)
	}

	lastModifiedStr, _ := e.Repo.GetSyncState(ctx, "last_modified_timestamp")
	var threshold time.Time
	if lastModifiedStr != "" {
		threshold, err = time.Parse(time.RFC3339, lastModifiedStr)
		if err != nil {
			e.Logger.Warn("Failed to parse last_modified_timestamp sync state, using zero time", "error", err)
			threshold = time.Time{}
		}
	}

	e.Logger.Info("Fetching changed books from Calibre", "threshold", threshold)
	books, err := reader.GetChangedBooks(ctx, threshold)
	if err != nil {
		return fmt.Errorf("failed to get changed books: %w", err)
	}

	if len(books) == 0 {
		e.Logger.Info("No new or modified books found in Calibre")
		pruned, err := e.Repo.PruneMissingCalibreIDs(ctx, allCalibreIDs)
		if err != nil {
			return fmt.Errorf("failed to prune missing books: %w", err)
		}
		if pruned > 0 {
			e.Logger.Info("Pruned books removed from Calibre", "count", pruned)
		}
		err = e.updateSyncState(ctx, currentMtime, currentSize, threshold)
		if err != nil {
			return err
		}
	} else {
		e.Logger.Info("Populating metadata for changed books", "count", len(books))
		if err := reader.PopulateMetadata(ctx, books); err != nil {
			return fmt.Errorf("failed to populate metadata: %w", err)
		}

		latestModified := threshold
		successCount := 0
		var syncErr error
		for _, book := range books {
			if err := e.Repo.Upsert(ctx, &book); err != nil {
				e.Logger.Error("Failed to upsert book", "calibre_id", book.CalibreID, "error", err)
				syncErr = fmt.Errorf("failed to upsert one or more books")
				continue
			}
			successCount++
			if book.LastModified.After(latestModified) {
				latestModified = book.LastModified
			}
		}

		e.Logger.Info("Synchronization batch completed", "total", len(books), "success", successCount)
		if syncErr != nil {
			return syncErr
		}

		pruned, err := e.Repo.PruneMissingCalibreIDs(ctx, allCalibreIDs)
		if err != nil {
			return fmt.Errorf("failed to prune missing books: %w", err)
		}
		if pruned > 0 {
			e.Logger.Info("Pruned books removed from Calibre", "count", pruned)
		}

		if err := e.updateSyncState(ctx, currentMtime, currentSize, latestModified); err != nil {
			return err
		}
	}

	// Enforce storage cap
	if pruned, err := e.Repo.EnforceStorageCap(ctx, e.DatabasePath, e.StorageCapMB); err != nil {
		e.Logger.Error("failed to enforce storage cap", "error", err)
	} else if pruned {
		e.Logger.Info("storage cap enforced: oldest records pruned", "db_path", e.DatabasePath, "cap_mb", e.StorageCapMB)
	}

	return nil
}

func (e *SyncEngine) updateSyncState(ctx context.Context, mtime int64, size int64, lastModified time.Time) error {
	if err := e.Repo.SetSyncState(ctx, "calibre_mtime", strconv.FormatInt(mtime, 10)); err != nil {
		return fmt.Errorf("failed to update calibre_mtime: %w", err)
	}
	if err := e.Repo.SetSyncState(ctx, "calibre_size", strconv.FormatInt(size, 10)); err != nil {
		return fmt.Errorf("failed to update calibre_size: %w", err)
	}
	if !lastModified.IsZero() {
		if err := e.Repo.SetSyncState(ctx, "last_modified_timestamp", lastModified.Format(time.RFC3339)); err != nil {
			return fmt.Errorf("failed to update last_modified_timestamp: %w", err)
		}
	}
	return nil
}
