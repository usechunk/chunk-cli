package metadata

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache, err := NewCache()
	if err != nil {
		t.Fatalf("NewCache() error = %v", err)
	}

	if cache == nil {
		t.Fatal("NewCache() returned nil")
	}

	if cache.ttl != DefaultCacheTTL {
		t.Errorf("cache.ttl = %v, want %v", cache.ttl, DefaultCacheTTL)
	}
}

func TestNewCacheWithDir(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache")
	defer os.RemoveAll(tmpDir)

	ttl := 1 * time.Hour
	cache, err := NewCacheWithDir(tmpDir, ttl)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	if cache == nil {
		t.Fatal("NewCacheWithDir() returned nil")
	}

	if cache.baseDir != tmpDir {
		t.Errorf("cache.baseDir = %v, want %v", cache.baseDir, tmpDir)
	}

	if cache.ttl != ttl {
		t.Errorf("cache.ttl = %v, want %v", cache.ttl, ttl)
	}

	// Check directory was created
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("Cache directory was not created")
	}
}

func TestCache_SetAndGet(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache-setget")
	defer os.RemoveAll(tmpDir)

	cache, err := NewCacheWithDir(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	key := "test-key"
	data := []byte(`{"test": "data"}`)

	// Set data
	if err := cache.Set(key, data); err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}

	// Get data
	got, err := cache.Get(key)
	if err != nil {
		t.Fatalf("cache.Get() error = %v", err)
	}

	if string(got) != string(data) {
		t.Errorf("cache.Get() = %v, want %v", string(got), string(data))
	}
}

func TestCache_GetNotFound(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache-notfound")
	defer os.RemoveAll(tmpDir)

	cache, err := NewCacheWithDir(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	_, err = cache.Get("nonexistent-key")
	if err != ErrNotFound {
		t.Errorf("cache.Get() error = %v, want ErrNotFound", err)
	}
}

func TestCache_GetExpired(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache-expired")
	defer os.RemoveAll(tmpDir)

	cache, err := NewCacheWithDir(tmpDir, 1*time.Millisecond)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	key := "test-key"
	data := []byte(`{"test": "data"}`)

	// Set data with very short TTL
	if err := cache.Set(key, data); err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Get should return expired error
	_, err = cache.Get(key)
	if err != ErrCacheExpired {
		t.Errorf("cache.Get() error = %v, want ErrCacheExpired", err)
	}
}

func TestCache_SetWithTTL(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache-ttl")
	defer os.RemoveAll(tmpDir)

	cache, err := NewCacheWithDir(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	key := "test-key"
	data := []byte(`{"test": "data"}`)

	// Set with custom TTL (very short)
	if err := cache.SetWithTTL(key, data, 1*time.Millisecond); err != nil {
		t.Fatalf("cache.SetWithTTL() error = %v", err)
	}

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Get should return expired error
	_, err = cache.Get(key)
	if err != ErrCacheExpired {
		t.Errorf("cache.Get() error = %v, want ErrCacheExpired", err)
	}
}

func TestCache_Delete(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache-delete")
	defer os.RemoveAll(tmpDir)

	cache, err := NewCacheWithDir(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	key := "test-key"
	data := []byte(`{"test": "data"}`)

	// Set data
	if err := cache.Set(key, data); err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}

	// Delete data
	if err := cache.Delete(key); err != nil {
		t.Fatalf("cache.Delete() error = %v", err)
	}

	// Get should return not found
	_, err = cache.Get(key)
	if err != ErrNotFound {
		t.Errorf("cache.Get() error = %v, want ErrNotFound", err)
	}
}

func TestCache_DeleteNonExistent(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache-delete-nonexistent")
	defer os.RemoveAll(tmpDir)

	cache, err := NewCacheWithDir(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	// Delete non-existent key should not error
	if err := cache.Delete("nonexistent-key"); err != nil {
		t.Errorf("cache.Delete() error = %v, want nil", err)
	}
}

func TestCache_Clear(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache-clear")
	defer os.RemoveAll(tmpDir)

	cache, err := NewCacheWithDir(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	// Set multiple keys
	keys := []string{"key1", "key2", "key3"}
	for _, key := range keys {
		if err := cache.Set(key, []byte(key)); err != nil {
			t.Fatalf("cache.Set() error = %v", err)
		}
	}

	// Clear cache
	if err := cache.Clear(); err != nil {
		t.Fatalf("cache.Clear() error = %v", err)
	}

	// All keys should be gone
	for _, key := range keys {
		_, err := cache.Get(key)
		if err != ErrNotFound {
			t.Errorf("cache.Get(%s) error = %v, want ErrNotFound", key, err)
		}
	}
}

func TestCache_Refresh(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache-refresh")
	defer os.RemoveAll(tmpDir)

	cache, err := NewCacheWithDir(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	key := "test-key"
	data := []byte(`{"test": "data"}`)

	// Set data
	if err := cache.Set(key, data); err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}

	// Refresh (delete) data
	if err := cache.Refresh(key); err != nil {
		t.Fatalf("cache.Refresh() error = %v", err)
	}

	// Get should return not found
	_, err = cache.Get(key)
	if err != ErrNotFound {
		t.Errorf("cache.Get() error = %v, want ErrNotFound", err)
	}
}

func TestCache_GetTTL(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache-getttl")
	defer os.RemoveAll(tmpDir)

	ttl := 2 * time.Hour
	cache, err := NewCacheWithDir(tmpDir, ttl)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	if cache.GetTTL() != ttl {
		t.Errorf("cache.GetTTL() = %v, want %v", cache.GetTTL(), ttl)
	}
}

func TestCache_SetTTL(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "chunk-test-cache-setttl")
	defer os.RemoveAll(tmpDir)

	cache, err := NewCacheWithDir(tmpDir, 1*time.Hour)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	newTTL := 30 * time.Minute
	cache.SetTTL(newTTL)

	if cache.GetTTL() != newTTL {
		t.Errorf("cache.GetTTL() = %v, want %v", cache.GetTTL(), newTTL)
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "simple name",
			input: "test",
			want:  "test",
		},
		{
			name:  "with underscores",
			input: "test_key_name",
			want:  "test_key_name",
		},
		{
			name:  "with hyphens",
			input: "test-key-name",
			want:  "test-key-name",
		},
		{
			name:  "with special chars",
			input: "test:key/name",
			want:  "test_key_name",
		},
		{
			name:  "with spaces",
			input: "test key name",
			want:  "test_key_name",
		},
		{
			name:  "mixed case",
			input: "TestKeyName",
			want:  "TestKeyName",
		},
		{
			name:  "with numbers",
			input: "test123key456",
			want:  "test123key456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeFilename(tt.input); got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
