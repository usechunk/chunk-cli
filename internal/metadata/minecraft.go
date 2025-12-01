package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	// MojangVersionManifestURL is the URL for the Minecraft version manifest.
	MojangVersionManifestURL = "https://launchermeta.mojang.com/mc/game/version_manifest.json"
	// CacheKeyMinecraftManifest is the cache key for the Minecraft version manifest.
	CacheKeyMinecraftManifest = "minecraft_manifest"
)

// MinecraftClient provides methods for fetching Minecraft version information.
type MinecraftClient struct {
	httpClient *http.Client
	cache      *Cache
}

// NewMinecraftClient creates a new Minecraft client with optional cache.
func NewMinecraftClient(cache *Cache) *MinecraftClient {
	return &MinecraftClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: cache,
	}
}

// GetVersionManifest fetches the Minecraft version manifest from Mojang.
func (m *MinecraftClient) GetVersionManifest() (*MinecraftVersionManifest, error) {
	// Try to get from cache first
	if m.cache != nil {
		if data, err := m.cache.Get(CacheKeyMinecraftManifest); err == nil {
			var manifest MinecraftVersionManifest
			if err := json.Unmarshal(data, &manifest); err == nil {
				return &manifest, nil
			}
		}
	}

	// Fetch from API
	manifest, err := m.fetchVersionManifest()
	if err != nil {
		return nil, err
	}

	// Cache the result
	if m.cache != nil {
		if data, err := json.Marshal(manifest); err == nil {
			_ = m.cache.Set(CacheKeyMinecraftManifest, data)
		}
	}

	return manifest, nil
}

// fetchVersionManifest fetches the manifest from the Mojang API.
func (m *MinecraftClient) fetchVersionManifest() (*MinecraftVersionManifest, error) {
	resp, err := m.httpClient.Get(MojangVersionManifestURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrNetworkError, resp.StatusCode)
	}

	var manifest MinecraftVersionManifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("failed to decode version manifest: %w", err)
	}

	return &manifest, nil
}

// GetVersion returns details for a specific Minecraft version.
func (m *MinecraftClient) GetVersion(version string) (*MinecraftVersion, error) {
	manifest, err := m.GetVersionManifest()
	if err != nil {
		return nil, err
	}

	for _, v := range manifest.Versions {
		if v.ID == version {
			return v, nil
		}
	}

	return nil, ErrNotFound
}

// GetLatestRelease returns the latest stable release version.
func (m *MinecraftClient) GetLatestRelease() (*MinecraftVersion, error) {
	manifest, err := m.GetVersionManifest()
	if err != nil {
		return nil, err
	}

	return m.GetVersion(manifest.Latest.Release)
}

// GetLatestSnapshot returns the latest snapshot version.
func (m *MinecraftClient) GetLatestSnapshot() (*MinecraftVersion, error) {
	manifest, err := m.GetVersionManifest()
	if err != nil {
		return nil, err
	}

	return m.GetVersion(manifest.Latest.Snapshot)
}

// GetVersions returns all release versions.
func (m *MinecraftClient) GetVersions() ([]*MinecraftVersion, error) {
	manifest, err := m.GetVersionManifest()
	if err != nil {
		return nil, err
	}

	var releases []*MinecraftVersion
	for _, v := range manifest.Versions {
		if v.Type == VersionRelease {
			releases = append(releases, v)
		}
	}

	return releases, nil
}

// GetAllVersions returns all versions including snapshots.
func (m *MinecraftClient) GetAllVersions() ([]*MinecraftVersion, error) {
	manifest, err := m.GetVersionManifest()
	if err != nil {
		return nil, err
	}

	return manifest.Versions, nil
}

// GetJavaVersion returns the required Java version for a Minecraft version.
func (m *MinecraftClient) GetJavaVersion(mcVersion string) (int, error) {
	// First check the version manifest for Java version info
	v, err := m.GetVersion(mcVersion)
	if err == nil && v.JavaVersion > 0 {
		return v.JavaVersion, nil
	}

	// Fall back to the static mapping
	return getJavaVersionFromMapping(mcVersion)
}

// getJavaVersionFromMapping returns the Java version requirement based on version prefix.
func getJavaVersionFromMapping(mcVersion string) (int, error) {
	// Check for exact matches first
	for pattern, javaVersion := range JavaVersionRequirements {
		if mcVersion == pattern {
			return javaVersion, nil
		}
	}

	// Check for prefix matches (e.g., "1.20.1" matches "1.20")
	for pattern, javaVersion := range JavaVersionRequirements {
		if strings.HasPrefix(mcVersion, pattern) {
			return javaVersion, nil
		}
	}

	// Default to Java 8 for unknown versions
	return 8, nil
}

// IsVersionValid checks if a Minecraft version string is valid.
func (m *MinecraftClient) IsVersionValid(version string) (bool, error) {
	_, err := m.GetVersion(version)
	if err == ErrNotFound {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// RefreshCache clears the cached manifest data.
func (m *MinecraftClient) RefreshCache() error {
	if m.cache != nil {
		return m.cache.Delete(CacheKeyMinecraftManifest)
	}
	return nil
}
