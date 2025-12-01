package metadata

import (
	"testing"
)

func TestNewFabricClient(t *testing.T) {
	// Without cache
	client := NewFabricClient(nil)
	if client == nil {
		t.Fatal("NewFabricClient(nil) returned nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if client.cache != nil {
		t.Error("cache should be nil")
	}
}

func TestNewFabricClientWithCache(t *testing.T) {
	cache, err := NewCacheWithDir(t.TempDir(), DefaultCacheTTL)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	client := NewFabricClient(cache)
	if client == nil {
		t.Fatal("NewFabricClient(cache) returned nil")
	}
	if client.cache == nil {
		t.Error("cache should not be nil")
	}
}

func TestFabricClient_RefreshCache(t *testing.T) {
	// Without cache
	client := NewFabricClient(nil)
	if err := client.RefreshCache(); err != nil {
		t.Errorf("RefreshCache() without cache should not error, got %v", err)
	}

	// With cache
	cache, err := NewCacheWithDir(t.TempDir(), DefaultCacheTTL)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}
	clientWithCache := NewFabricClient(cache)
	if err := clientWithCache.RefreshCache(); err != nil {
		t.Errorf("RefreshCache() with cache should not error, got %v", err)
	}
}

func TestFabricLoaderResponse_Struct(t *testing.T) {
	loader := fabricLoaderResponse{
		Separator: ".",
		Build:     1,
		Maven:     "net.fabricmc:fabric-loader:0.14.21",
		Version:   "0.14.21",
		Stable:    true,
	}

	if loader.Version != "0.14.21" {
		t.Errorf("Version = %v, want 0.14.21", loader.Version)
	}
	if !loader.Stable {
		t.Error("Stable should be true")
	}
}

func TestFabricGameResponse_Struct(t *testing.T) {
	game := fabricGameResponse{
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
