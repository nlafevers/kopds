package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/nlafevers/kopds/internal/domain"
	"github.com/rs/zerolog"
)

type SyncEngine struct {
	Repo        domain.BookRepository
	CalibrePath string
	Logger      zerolog.Logger
}

// NewSyncEngine creates a new synchronization engine.
func NewSyncEngine(repo domain.BookRepository, calibrePath string, logger zerolog.Logger) *SyncEngine {
	return &SyncEngine{
		Repo:        repo,
		CalibrePath: calibrePath,
		Logger:      logger,
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
		e.Logger.Debug().Msg("Calibre library metadata.db unchanged, skipping sync")
		return nil
	}

	reader, err := NewCalibreReader(dbPath)
	if err != nil {
		return fmt.Errorf("failed to initialize calibre reader: %w", err)
	}
	defer reader.Close()

	lastModifiedStr, _ := e.Repo.GetSyncState(ctx, "last_modified_timestamp")
	var threshold time.Time
	if lastModifiedStr != "" {
		threshold, err = time.Parse(time.RFC3339, lastModifiedStr)
		if err != nil {
			e.Logger.Warn().Err(err).Msg("Failed to parse last_modified_timestamp sync state, using zero time")
			threshold = time.Time{}
		}
	}

	e.Logger.Info().Time("threshold", threshold).Msg("Fetching changed books from Calibre")
	books, err := reader.GetChangedBooks(ctx, threshold)
	if err != nil {
		return fmt.Errorf("failed to get changed books: %w", err)
	}

	if len(books) == 0 {
		e.Logger.Info().Msg("No new or modified books found in Calibre")
		return e.updateSyncState(ctx, currentMtime, currentSize, threshold)
	}

	e.Logger.Info().Int("count", len(books)).Msg("Populating metadata for changed books")
	if err := reader.PopulateMetadata(ctx, books); err != nil {
		return fmt.Errorf("failed to populate metadata: %w", err)
	}

	latestModified := threshold
	successCount := 0
	for _, book := range books {
		if err := e.Repo.Upsert(ctx, &book); err != nil {
			e.Logger.Error().Err(err).Int64("calibre_id", book.CalibreID).Msg("Failed to upsert book")
			continue
		}
		successCount++
		if book.LastModified.After(latestModified) {
			latestModified = book.LastModified
		}
	}

	e.Logger.Info().Int("total", len(books)).Int("success", successCount).Msg("Synchronization batch completed")

	return e.updateSyncState(ctx, currentMtime, currentSize, latestModified)
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
