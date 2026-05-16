package scanner

import (
	"context"
	"time"

	"github.com/nlafevers/kopds/internal/domain"
	"log/slog"
)

// StartWorker starts a background worker that periodically triggers the sync engine.
func StartWorker(ctx context.Context, indexer domain.Indexer, interval time.Duration, logger *slog.Logger) {
	logger.Info("Starting background sync worker", "interval", interval)

	// Initial sync
	logger.Info("Triggering initial sync")
	if err := indexer.Sync(ctx); err != nil {
		logger.Error("Initial sync failed", "error", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Background sync worker stopping")
			return
		case <-ticker.C:
			logger.Info("Triggering periodic sync")
			if err := indexer.Sync(ctx); err != nil {
				logger.Error("Periodic sync failed", "error", err)
			}
		}
	}
}
