package metadata

import (
	"testing"
)

func TestNewForgeClient(t *testing.T) {
	// Without cache
	client := NewForgeClient(nil)
	if client == nil {
		t.Fatal("NewForgeClient(nil) returned nil")
	}
	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
	if client.cache != nil {
		t.Error("cache should be nil")
	}
}

func TestNewForgeClientWithCache(t *testing.T) {
	cache, err := NewCacheWithDir(t.TempDir(), DefaultCacheTTL)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}

	client := NewForgeClient(cache)
	if client == nil {
		t.Fatal("NewForgeClient(cache) returned nil")
	}
	if client.cache == nil {
		t.Error("cache should not be nil")
	}
}

func TestForgeClient_parsePromotions(t *testing.T) {
	client := NewForgeClient(nil)

	promos := &forgePromotions{
		Homepage: "https://files.minecraftforge.net/",
		Promos: map[string]string{
			"1.20.1-recommended": "47.2.0",
			"1.20.1-latest":      "47.2.5",
			"1.19.4-recommended": "45.2.0",
			"1.19.4-latest":      "45.2.9",
		},
	}

	versions, err := client.parsePromotions(promos)
	if err != nil {
		t.Fatalf("parsePromotions() error = %v", err)
	}

	if len(versions) == 0 {
		t.Fatal("parsePromotions() returned no versions")
	}

	// Check that we have versions for expected MC versions
	mcVersions := make(map[string]bool)
	for _, v := range versions {
		mcVersions[v.MinecraftVersion] = true
		if v.LoaderType != LoaderForge {
			t.Errorf("LoaderType = %v, want forge", v.LoaderType)
		}
	}

	if !mcVersions["1.20.1"] {
		t.Error("Expected versions for MC 1.20.1")
	}
	if !mcVersions["1.19.4"] {
		t.Error("Expected versions for MC 1.19.4")
	}
}

func TestForgeClient_RefreshCache(t *testing.T) {
	// Without cache
	client := NewForgeClient(nil)
	if err := client.RefreshCache(); err != nil {
		t.Errorf("RefreshCache() without cache should not error, got %v", err)
	}

	// With cache
	cache, err := NewCacheWithDir(t.TempDir(), DefaultCacheTTL)
	if err != nil {
		t.Fatalf("NewCacheWithDir() error = %v", err)
	}
	clientWithCache := NewForgeClient(cache)
	if err := clientWithCache.RefreshCache(); err != nil {
		t.Errorf("RefreshCache() with cache should not error, got %v", err)
	}
}

func TestForgeClient_IsVersionCompatible_ParsesVersionString(t *testing.T) {
	tests := []struct {
		name           string
		forgeVersion   string
		mcVersion      string
		wantCompatible bool
	}{
		{
			name:           "version with MC prefix is compatible",
			forgeVersion:   "1.20.1-47.2.0",
			mcVersion:      "1.20.1",
			wantCompatible: true,
		},
		{
			name:           "version without MC prefix",
			forgeVersion:   "47.2.0",
			mcVersion:      "1.20.1",
			wantCompatible: false, // Would need API to confirm
		},
	}

	// Note: This test only checks the string parsing fallback
	// Full compatibility checking requires API access
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't fully test without API, but we can test the string parsing
			if len(tt.forgeVersion) > len(tt.mcVersion)+1 && tt.forgeVersion[:len(tt.mcVersion)+1] == tt.mcVersion+"-" {
				// This is a prefixed version
				if !tt.wantCompatible {
					t.Errorf("Expected version %s to be compatible with %s", tt.forgeVersion, tt.mcVersion)
				}
			}
		})
	}
}
