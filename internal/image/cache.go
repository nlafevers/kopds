package image

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/rs/zerolog/log"
)

// DiskCache implements a disk-based LRU cache for images.
type DiskCache struct {
	path     string
	maxCount int
	lru      *lru.Cache[string, struct{}]
}

// NewDiskCache creates a new DiskCache.
func NewDiskCache(path string, maxCount int) (*DiskCache, error) {
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	c := &DiskCache{
		path:     path,
		maxCount: maxCount,
	}

	onEvict := func(key string, value struct{}) {
		filePath := filepath.Join(c.path, key)
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			log.Error().Err(err).Str("path", filePath).Msg("failed to delete evicted cache file")
		}
	}

	cache, err := lru.NewWithEvict[string, struct{}](maxCount, onEvict)
	if err != nil {
		return nil, fmt.Errorf("failed to create LRU cache: %w", err)
	}
	c.lru = cache

	if err := c.loadExistingFiles(); err != nil {
		return nil, fmt.Errorf("failed to load existing cache files: %w", err)
	}

	return c, nil
}

func (c *DiskCache) loadExistingFiles() error {
	entries, err := os.ReadDir(c.path)
	if err != nil {
		return err
	}

	type fileInfo struct {
		name    string
		modTime os.FileInfo
	}

	var files []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry)
		}
	}

	// Sort by modification time to populate LRU correctly (oldest first)
	type entryWithTime struct {
		name  string
		mtime int64
	}
	var sortedEntries []entryWithTime
	for _, entry := range files {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		sortedEntries = append(sortedEntries, entryWithTime{entry.Name(), info.ModTime().Unix()})
	}

	sort.Slice(sortedEntries, func(i, j int) bool {
		return sortedEntries[i].mtime < sortedEntries[j].mtime
	})

	for _, entry := range sortedEntries {
		c.lru.Add(entry.name, struct{}{})
	}

	log.Info().Int("count", c.lru.Len()).Msg("Loaded existing image cache entries")
	return nil
}

// Get retrieves an image from the cache.
func (c *DiskCache) Get(key string) ([]byte, error) {
	// lru.Get updates the "recentness" of the key
	if _, ok := c.lru.Get(key); !ok {
		return nil, os.ErrNotExist
	}

	filePath := filepath.Join(c.path, key)
	data, err := os.ReadFile(filePath)
	if err != nil {
		// If it's missing from disk but was in LRU, remove it from LRU
		if os.IsNotExist(err) {
			c.lru.Remove(key)
		}
		return nil, err
	}

	return data, nil
}

// Put adds an image to the cache.
func (c *DiskCache) Put(key string, data []byte) error {
	filePath := filepath.Join(c.path, key)

	// Write to a temporary file first to ensure atomicity
	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary cache file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("failed to rename cache file: %w", err)
	}

	c.lru.Add(key, struct{}{})
	return nil
}
