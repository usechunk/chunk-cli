package metadata

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// DefaultCacheTTL is the default time-to-live for cached data (24 hours).
	DefaultCacheTTL = 24 * time.Hour
	// CacheDir is the subdirectory name for metadata cache.
	CacheDir = "metadata"
)

// Cache provides a file-based caching layer for metadata.
type Cache struct {
	baseDir string
	ttl     time.Duration
	mu      sync.RWMutex
}

// NewCache creates a new cache instance with the default cache directory.
func NewCache() (*Cache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(home, ".chunk", CacheDir)
	return NewCacheWithDir(cacheDir, DefaultCacheTTL)
}

// NewCacheWithDir creates a new cache instance with a custom directory and TTL.
func NewCacheWithDir(dir string, ttl time.Duration) (*Cache, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	return &Cache{
		baseDir: dir,
		ttl:     ttl,
	}, nil
}

// Get retrieves data from the cache.
// Returns ErrCacheExpired if the cached data has expired.
// Returns ErrNotFound if the data is not in the cache.
func (c *Cache) Get(key string) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	path := c.getPath(key)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache entry: %w", err)
	}

	if entry.IsExpired() {
		return nil, ErrCacheExpired
	}

	return entry.Data, nil
}

// Set stores data in the cache with the configured TTL.
func (c *Cache) Set(key string, data []byte) error {
	return c.SetWithTTL(key, data, c.ttl)
}

// SetWithTTL stores data in the cache with a custom TTL.
func (c *Cache) SetWithTTL(key string, data []byte, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := CacheEntry{
		Data:      data,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	entryData, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal cache entry: %w", err)
	}

	path := c.getPath(key)
	if err := os.WriteFile(path, entryData, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// Delete removes data from the cache.
func (c *Cache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	path := c.getPath(key)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete cache file: %w", err)
	}

	return nil
}

// Clear removes all cached data.
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	entries, err := os.ReadDir(c.baseDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			path := filepath.Join(c.baseDir, entry.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove cache file %s: %w", entry.Name(), err)
			}
		}
	}

	return nil
}

// Refresh forces a refresh of the cached data by marking it as expired.
func (c *Cache) Refresh(key string) error {
	return c.Delete(key)
}

// getPath returns the file path for a cache key.
func (c *Cache) getPath(key string) string {
	// Sanitize the key to create a valid filename
	safeName := sanitizeFilename(key) + ".json"
	return filepath.Join(c.baseDir, safeName)
}

// sanitizeFilename converts a cache key to a safe filename.
func sanitizeFilename(name string) string {
	// Replace characters that are not safe for filenames
	var result []byte
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			result = append(result, ch)
		} else {
			result = append(result, '_')
		}
	}
	return string(result)
}

// GetTTL returns the configured TTL.
func (c *Cache) GetTTL() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ttl
}

// SetTTL updates the TTL for future cache entries.
func (c *Cache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ttl = ttl
}
