package metadata

import (
	"testing"
)

func TestNewQuiltClient(t *testing.T) {
	// Without cache
	client := NewQuiltClient(nil)
	if client == nil {
		t.Fatal("NewQuiltClient(nil) returned nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if client.cache != nil {
		t.Error("cache should be nil")
	}
}

func TestNewQuiltClientWithCache(t *testing.T) {
	cache, err := NewCacheWithDir(t.TempDir(), DefaultCacheTTL)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	client := NewQuiltClient(cache)
	if client == nil {
		t.Fatal("NewQuiltClient(cache) returned nil")
	}
	if client.cache == nil {
		t.Error("cache should not be nil")
	}
}

func TestQuiltClient_RefreshCache(t *testing.T) {
	// Without cache
	client := NewQuiltClient(nil)
	if err := client.RefreshCache(); err != nil {
		t.Errorf("RefreshCache() without cache should not error, got %v", err)
	}

	// With cache
	cache, err := NewCacheWithDir(t.TempDir(), DefaultCacheTTL)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}
	clientWithCache := NewQuiltClient(cache)
	if err := clientWithCache.RefreshCache(); err != nil {
		t.Errorf("RefreshCache() with cache should not error, got %v", err)
	}
}

func TestQuiltLoaderResponse_Struct(t *testing.T) {
	loader := quiltLoaderResponse{
		Separator: ".",
		Build:     1,
		Maven:     "org.quiltmc:quilt-loader:0.23.0",
		Version:   "0.23.0",
	}

	if loader.Version != "0.23.0" {
		t.Errorf("Version = %v, want 0.23.0", loader.Version)
	}
}

func TestQuiltGameResponse_Struct(t *testing.T) {
	game := quiltGameResponse{
		Version: "1.20.4",
		Stable:  true,
	}

	if game.Version != "1.20.4" {
		t.Errorf("Version = %v, want 1.20.4", game.Version)
	}
	if !game.Stable {
		t.Error("Stable should be true")
	}
}
