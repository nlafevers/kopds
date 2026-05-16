package scanner

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"
)

type mockIndexer struct {
	syncCount int
	mu        sync.Mutex
}

func (m *mockIndexer) Sync(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncCount++
	return nil
}

func (m *mockIndexer) getSyncCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.syncCount
}

func TestStartWorker(t *testing.T) {
	mock := &mockIndexer{}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	ctx, cancel := context.WithCancel(context.Background())

	interval := 100 * time.Millisecond

	go StartWorker(ctx, mock, interval, logger)

	// Wait a bit to allow initial sync and at least one periodic sync
	time.Sleep(250 * time.Millisecond)

	count := mock.getSyncCount()
	if count < 2 {
		t.Errorf("expected at least 2 syncs (initial + 1 periodic), got %d", count)
	}

	cancel()
	time.Sleep(50 * time.Millisecond)

	countAfterCancel := mock.getSyncCount()
	time.Sleep(200 * time.Millisecond)
	if mock.getSyncCount() > countAfterCancel {
		t.Errorf("expected no more syncs after cancel, but count increased from %d to %d", countAfterCancel, mock.getSyncCount())
	}
}
