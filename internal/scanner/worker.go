package scanner

import (
	"context"
	"time"

	"github.com/nlafevers/kopds/internal/domain"
	"github.com/rs/zerolog"
)

// StartWorker starts a background worker that periodically triggers the sync engine.
func StartWorker(ctx context.Context, indexer domain.Indexer, interval time.Duration, logger zerolog.Logger) {
	logger.Info().Dur("interval", interval).Msg("Starting background sync worker")

	// Initial sync
	logger.Info().Msg("Triggering initial sync")
	if err := indexer.Sync(ctx); err != nil {
		logger.Error().Err(err).Msg("Initial sync failed")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Background sync worker stopping")
			return
		case <-ticker.C:
			logger.Info().Msg("Triggering periodic sync")
			if err := indexer.Sync(ctx); err != nil {
				logger.Error().Err(err).Msg("Periodic sync failed")
			}
		}
	}
}
