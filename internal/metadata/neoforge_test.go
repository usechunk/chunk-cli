package metadata

import (
	"testing"
)

func TestNewNeoForgeClient(t *testing.T) {
	// Without cache
	client := NewNeoForgeClient(nil)
	if client == nil {
		t.Fatal("NewNeoForgeClient(nil) returned nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if client.cache != nil {
		t.Error("cache should be nil")
	}
}

func TestNewNeoForgeClientWithCache(t *testing.T) {
	cache, err := NewCacheWithDir(t.TempDir(), DefaultCacheTTL)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	client := NewNeoForgeClient(cache)
	if client == nil {
		t.Fatal("NewNeoForgeClient(cache) returned nil")
	}
	if client.cache == nil {
		t.Error("cache should not be nil")
	}
}

func TestNeoForgeClient_RefreshCache(t *testing.T) {
	// Without cache
	client := NewNeoForgeClient(nil)
	if err := client.RefreshCache(); err != nil {
		t.Errorf("RefreshCache() without cache should not error, got %v", err)
	}

	// With cache
	cache, err := NewCacheWithDir(t.TempDir(), DefaultCacheTTL)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}
	clientWithCache := NewNeoForgeClient(cache)
	if err := clientWithCache.RefreshCache(); err != nil {
		t.Errorf("RefreshCache() with cache should not error, got %v", err)
	}
}

func TestNeoForgeClient_inferMCVersion(t *testing.T) {
	client := NewNeoForgeClient(nil)

	tests := []struct {
		name  string
		major string
		minor string
		want  string
	}{
		{
			name:  "MC 1.21.1",
			major: "21",
			minor: "1",
			want:  "1.21.1",
		},
		{
			name:  "MC 1.20.4",
			major: "20",
			minor: "4",
			want:  "1.20.4",
		},
		{
			name:  "MC 1.21.0",
			major: "21",
			minor: "0",
			want:  "1.21.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := client.inferMCVersion(tt.major, tt.minor)
			if got != tt.want {
				t.Errorf("inferMCVersion(%s, %s) = %v, want %v", tt.major, tt.minor, got, tt.want)
			}
		})
	}
}

func TestNeoForgeClient_parseVersions(t *testing.T) {
	client := NewNeoForgeClient(nil)

	versionStrings := []string{
		"21.1.1",
		"21.1.0",
		"21.0.5",
		"20.4.237",
		"20.4.200-beta",
	}

	versions, err := client.parseVersions(versionStrings)
	if err != nil {
		t.Fatalf("parseVersions() error = %v", err)
	}

	if len(versions) != 5 {
		t.Errorf("parseVersions() returned %d versions, want 5", len(versions))
	}

	// Check versions are reversed (newest first)
	if versions[0].Version != "20.4.200-beta" {
		t.Errorf("First version should be 20.4.200-beta (reversed), got %s", versions[0].Version)
	}

	// Check stability
	for _, v := range versions {
		if v.Version == "20.4.200-beta" && v.Stable {
			t.Error("Beta version should not be stable")
		}
		if v.LoaderType != LoaderNeoForge {
			t.Errorf("LoaderType = %v, want neoforge", v.LoaderType)
		}
	}

	// Check MC version inference
	found := false
	for _, v := range versions {
		if v.Version == "21.1.1" && v.MinecraftVersion == "1.21.1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected version 21.1.1 to have MC version 1.21.1")
	}
}

func TestNeoForgeMavenResponse_Struct(t *testing.T) {
	resp := neoForgeMavenResponse{
		IsSnapshot: false,
		Versions:   []string{"21.1.1", "21.1.0", "20.4.237"},
	}

	if resp.IsSnapshot {
		t.Error("IsSnapshot should be false")
	}
	if len(resp.Versions) != 3 {
		t.Errorf("len(Versions) = %v, want 3", len(resp.Versions))
	}
}
