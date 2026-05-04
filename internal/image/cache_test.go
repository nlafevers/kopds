package image

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDiskCache(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kopds-cache-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	maxCount := 3
	cache, err := NewDiskCache(tempDir, maxCount)
	if err != nil {
		t.Fatalf("failed to create disk cache: %v", err)
	}

	// Test Put and Get
	keys := []string{"key1", "key2", "key3"}
	data := [][]byte{
		[]byte("data1"),
		[]byte("data2"),
		[]byte("data3"),
	}

	for i, key := range keys {
		if err := cache.Put(key, data[i]); err != nil {
			t.Errorf("failed to put %s: %v", key, err)
		}
	}

	for i, key := range keys {
		got, err := cache.Get(key)
		if err != nil {
			t.Errorf("failed to get %s: %v", key, err)
		}
		if !bytes.Equal(got, data[i]) {
			t.Errorf("expected %s, got %s", data[i], got)
		}
	}

	// Test Eviction
	key4 := "key4"
	data4 := []byte("data4")
	if err := cache.Put(key4, data4); err != nil {
		t.Fatalf("failed to put key4: %v", err)
	}

	// key1 should be evicted because it was the first one added and we used 1, 2, 3 in Get (so 1 was least recent)
	// Wait, I accessed 1, then 2, then 3. So 1 is the oldest now.
	_, err = cache.Get("key1")
	if err == nil {
		t.Errorf("expected key1 to be evicted")
	}

	// Verify file is gone
	if _, err := os.Stat(filepath.Join(tempDir, "key1")); !os.IsNotExist(err) {
		t.Errorf("expected file key1 to be deleted from disk")
	}

	// Test Reload
	cache2, err := NewDiskCache(tempDir, maxCount)
	if err != nil {
		t.Fatalf("failed to reload cache: %v", err)
	}

	if cache2.lru.Len() != 3 {
		t.Errorf("expected 3 items in reloaded cache, got %d", cache2.lru.Len())
	}

	// Check if key4 is still there
	got, err := cache2.Get("key4")
	if err != nil {
		t.Errorf("failed to get key4 after reload: %v", err)
	}
	if !bytes.Equal(got, data4) {
		t.Errorf("expected %s, got %s after reload", data4, got)
	}
}
