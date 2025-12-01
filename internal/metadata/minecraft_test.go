package metadata

import (
	"testing"
)

func TestGetJavaVersionFromMapping(t *testing.T) {
	tests := []struct {
		name      string
		mcVersion string
		want      int
	}{
		{
			name:      "exact match 1.21",
			mcVersion: "1.21",
			want:      21,
		},
		{
			name:      "exact match 1.20.5",
			mcVersion: "1.20.5",
			want:      21,
		},
		{
			name:      "exact match 1.20",
			mcVersion: "1.20",
			want:      17,
		},
		{
			name:      "prefix match 1.20.4",
			mcVersion: "1.20.4",
			want:      17,
		},
		{
			name:      "prefix match 1.19.4",
			mcVersion: "1.19.4",
			want:      17,
		},
		{
			name:      "prefix match 1.18.2",
			mcVersion: "1.18.2",
			want:      17,
		},
		{
			name:      "1.17 requires Java 16",
			mcVersion: "1.17",
			want:      16,
		},
		{
			name:      "1.17.1 requires Java 16",
			mcVersion: "1.17.1",
			want:      16,
		},
		{
			name:      "1.16.5 requires Java 8",
			mcVersion: "1.16.5",
			want:      8,
		},
		{
			name:      "1.12.2 requires Java 8",
			mcVersion: "1.12.2",
			want:      8,
		},
		{
			name:      "unknown version defaults to Java 8",
			mcVersion: "1.0.0",
			want:      8,
		},
		{
			name:      "snapshot version defaults to Java 8",
			mcVersion: "24w05a",
			want:      8,
		},
		// Edge cases for version prefix matching - ensure 1.2 doesn't match 1.20
		{
			name:      "1.20.1 should match 1.20 not 1.2",
			mcVersion: "1.20.1",
			want:      17, // Should match 1.20 (Java 17), not 1.2 (doesn't exist)
		},
		{
			name:      "1.21.1 should match 1.21 for Java 21",
			mcVersion: "1.21.1",
			want:      21, // Should match 1.21
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getJavaVersionFromMapping(tt.mcVersion)
			if err != nil {
				t.Fatalf("getJavaVersionFromMapping(%q) error = %v", tt.mcVersion, err)
			}
			if got != tt.want {
				t.Errorf("getJavaVersionFromMapping(%q) = %v, want %v", tt.mcVersion, got, tt.want)
			}
		})
	}
}

func TestNewMinecraftClient(t *testing.T) {
	// Without cache
	client := NewMinecraftClient(nil)
	if client == nil {
		t.Fatal("NewMinecraftClient(nil) returned nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if client.cache != nil {
		t.Error("cache should be nil")
	}
}

func TestNewMinecraftClientWithCache(t *testing.T) {
	cache, err := NewCacheWithDir(t.TempDir(), DefaultCacheTTL)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	client := NewMinecraftClient(cache)
	if client == nil {
		t.Fatal("NewMinecraftClient(cache) returned nil")
	}
	if client.cache == nil {
		t.Error("cache should not be nil")
	}
}

func TestMinecraftClient_RefreshCache(t *testing.T) {
	// Without cache
	client := NewMinecraftClient(nil)
	if err := client.RefreshCache(); err != nil {
		t.Errorf("RefreshCache() without cache should not error, got %v", err)
	}

	// With cache
	cache, err := NewCacheWithDir(t.TempDir(), DefaultCacheTTL)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}
	clientWithCache := NewMinecraftClient(cache)
	if err := clientWithCache.RefreshCache(); err != nil {
		t.Errorf("RefreshCache() with cache should not error, got %v", err)
	}
}
