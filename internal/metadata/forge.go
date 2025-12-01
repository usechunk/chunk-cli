package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	// ForgePromotionsURL is the URL for Forge promotions metadata.
	ForgePromotionsURL = "https://files.minecraftforge.net/net/minecraftforge/forge/promotions_slim.json"
	// CacheKeyForgeVersions is the cache key for Forge versions.
	CacheKeyForgeVersions = "forge_versions"
)

// ForgeClient provides methods for fetching Forge version information.
type ForgeClient struct {
	httpClient *http.Client
	cache      *Cache
}

// NewForgeClient creates a new Forge client with optional cache.
func NewForgeClient(cache *Cache) *ForgeClient {
	return &ForgeClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: cache,
	}
}

// forgePromotions represents the Forge promotions_slim.json structure.
type forgePromotions struct {
	Homepage string            `json:"homepage"`
	Promos   map[string]string `json:"promos"`
}

// GetVersions returns all available Forge versions.
func (f *ForgeClient) GetVersions() ([]LoaderVersion, error) {
	promos, err := f.getPromotions()
	if err != nil {
		return nil, err
	}

	return f.parsePromotions(promos)
}

// GetVersionsForMC returns Forge versions compatible with a specific Minecraft version.
func (f *ForgeClient) GetVersionsForMC(mcVersion string) ([]LoaderVersion, error) {
	promos, err := f.getPromotions()
	if err != nil {
		return nil, err
	}

	allVersions, err := f.parsePromotions(promos)
	if err != nil {
		return nil, err
	}

	var compatible []LoaderVersion
	for _, v := range allVersions {
		if v.MinecraftVersion == mcVersion {
			compatible = append(compatible, v)
		}
	}

	return compatible, nil
}

// GetLatestVersion returns the latest stable (recommended) Forge version for a Minecraft version.
func (f *ForgeClient) GetLatestVersion(mcVersion string) (*LoaderVersion, error) {
	promos, err := f.getPromotions()
	if err != nil {
		return nil, err
	}

	// Look for recommended version first
	recommendedKey := mcVersion + "-recommended"
	if version, ok := promos.Promos[recommendedKey]; ok {
		return &LoaderVersion{
			Version:          version,
			MinecraftVersion: mcVersion,
			Stable:           true,
			LoaderType:       LoaderForge,
		}, nil
	}

	// Fall back to latest version
	latestKey := mcVersion + "-latest"
	if version, ok := promos.Promos[latestKey]; ok {
		return &LoaderVersion{
			Version:          version,
			MinecraftVersion: mcVersion,
			Stable:           false,
			LoaderType:       LoaderForge,
		}, nil
	}

	return nil, ErrNotFound
}

// IsVersionCompatible checks if a Forge version is compatible with a Minecraft version.
func (f *ForgeClient) IsVersionCompatible(forgeVersion, mcVersion string) (bool, error) {
	versions, err := f.GetVersionsForMC(mcVersion)
	if err != nil {
		return false, err
	}

	for _, v := range versions {
		if v.Version == forgeVersion {
			return true, nil
		}
	}

	// Also check by parsing the version string (Forge versions often contain MC version)
	// Format: mcVersion-forgeVersion (e.g., "1.20.1-47.2.0")
	if strings.HasPrefix(forgeVersion, mcVersion+"-") {
		return true, nil
	}

	return false, nil
}

// getPromotions fetches the Forge promotions data.
func (f *ForgeClient) getPromotions() (*forgePromotions, error) {
	// Try to get from cache first
	if f.cache != nil {
		if data, err := f.cache.Get(CacheKeyForgeVersions); err == nil {
			var promos forgePromotions
			if err := json.Unmarshal(data, &promos); err == nil {
				return &promos, nil
			}
		}
	}

	// Fetch from API
	promos, err := f.fetchPromotions()
	if err != nil {
		return nil, err
	}

	// Cache the result
	if f.cache != nil {
		if data, err := json.Marshal(promos); err == nil {
			_ = f.cache.Set(CacheKeyForgeVersions, data)
		}
	}

	return promos, nil
}

// fetchPromotions fetches promotions from the Forge API.
func (f *ForgeClient) fetchPromotions() (*forgePromotions, error) {
	resp, err := f.httpClient.Get(ForgePromotionsURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrNetworkError, resp.StatusCode)
	}

	var promos forgePromotions
	if err := json.NewDecoder(resp.Body).Decode(&promos); err != nil {
		return nil, fmt.Errorf("failed to decode forge promotions: %w", err)
	}

	return &promos, nil
}

// parsePromotions converts Forge promotions data to LoaderVersion slice.
func (f *ForgeClient) parsePromotions(promos *forgePromotions) ([]LoaderVersion, error) {
	var versions []LoaderVersion
	versionPattern := regexp.MustCompile(`^(.+)-(recommended|latest)$`)

	seen := make(map[string]bool)

	for key, version := range promos.Promos {
		matches := versionPattern.FindStringSubmatch(key)
		if len(matches) != 3 {
			continue
		}

		mcVersion := matches[1]
		versionType := matches[2]

		// Create a unique key to avoid duplicates
		uniqueKey := mcVersion + "-" + version
		if seen[uniqueKey] {
			continue
		}
		seen[uniqueKey] = true

		stable := versionType == "recommended"

		versions = append(versions, LoaderVersion{
			Version:          version,
			MinecraftVersion: mcVersion,
			Stable:           stable,
			LoaderType:       LoaderForge,
		})
	}

	return versions, nil
}

// GetSupportedMCVersions returns all Minecraft versions that have Forge support.
func (f *ForgeClient) GetSupportedMCVersions() ([]string, error) {
	promos, err := f.getPromotions()
	if err != nil {
		return nil, err
	}

	versionPattern := regexp.MustCompile(`^(.+)-(recommended|latest)$`)
	seen := make(map[string]bool)
	var mcVersions []string

	for key := range promos.Promos {
		matches := versionPattern.FindStringSubmatch(key)
		if len(matches) != 3 {
			continue
		}

		mcVersion := matches[1]
		if !seen[mcVersion] {
			seen[mcVersion] = true
			mcVersions = append(mcVersions, mcVersion)
		}
	}

	return mcVersions, nil
}

// RefreshCache clears the cached Forge data.
func (f *ForgeClient) RefreshCache() error {
	if f.cache != nil {
		return f.cache.Delete(CacheKeyForgeVersions)
	}
	return nil
}
