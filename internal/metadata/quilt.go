package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// QuiltMetaURL is the base URL for Quilt metadata API.
	QuiltMetaURL = "https://meta.quiltmc.org/v3/versions"
	// CacheKeyQuiltLoader is the cache key for Quilt loader versions.
	CacheKeyQuiltLoader = "quilt_loader"
	// CacheKeyQuiltGame is the cache key for Quilt game versions.
	CacheKeyQuiltGame = "quilt_game"
)

// QuiltClient provides methods for fetching Quilt version information.
type QuiltClient struct {
	httpClient *http.Client
	cache      *Cache
}

// NewQuiltClient creates a new Quilt client with optional cache.
func NewQuiltClient(cache *Cache) *QuiltClient {
	return &QuiltClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: cache,
	}
}

// quiltLoaderResponse represents a Quilt loader version from the API.
type quiltLoaderResponse struct {
	Separator string `json:"separator"`
	Build     int    `json:"build"`
	Maven     string `json:"maven"`
	Version   string `json:"version"`
}

// quiltGameResponse represents a Quilt game version from the API.
type quiltGameResponse struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// GetVersions returns all available Quilt loader versions.
func (q *QuiltClient) GetVersions() ([]LoaderVersion, error) {
	loaders, err := q.getLoaderVersions()
	if err != nil {
		return nil, err
	}

	var versions []LoaderVersion
	for _, loader := range loaders {
		versions = append(versions, LoaderVersion{
			Version:    loader.Version,
			Stable:     true, // Quilt API doesn't indicate stability, assume all are stable
			LoaderType: LoaderQuilt,
		})
	}

	return versions, nil
}

// GetVersionsForMC returns Quilt loader versions for a specific Minecraft version.
// Note: Quilt loader versions are generally compatible with all supported MC versions.
func (q *QuiltClient) GetVersionsForMC(mcVersion string) ([]LoaderVersion, error) {
	// First check if the MC version is supported by Quilt
	supported, err := q.IsMCVersionSupported(mcVersion)
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, ErrIncompatibleVersion
	}

	// Return all loader versions as they are generally compatible
	return q.GetVersions()
}

// GetLatestVersion returns the latest Quilt loader version.
func (q *QuiltClient) GetLatestVersion(mcVersion string) (*LoaderVersion, error) {
	// Check MC version support
	supported, err := q.IsMCVersionSupported(mcVersion)
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, ErrIncompatibleVersion
	}

	loaders, err := q.getLoaderVersions()
	if err != nil {
		return nil, err
	}

	if len(loaders) == 0 {
		return nil, ErrNotFound
	}

	// Return the first (latest) version
	return &LoaderVersion{
		Version:          loaders[0].Version,
		MinecraftVersion: mcVersion,
		Stable:           true,
		LoaderType:       LoaderQuilt,
	}, nil
}

// IsVersionCompatible checks if a Quilt loader version is compatible with a Minecraft version.
func (q *QuiltClient) IsVersionCompatible(loaderVersion, mcVersion string) (bool, error) {
	// Check if MC version is supported
	supported, err := q.IsMCVersionSupported(mcVersion)
	if err != nil {
		return false, err
	}
	if !supported {
		return false, nil
	}

	// Check if loader version exists
	loaders, err := q.getLoaderVersions()
	if err != nil {
		return false, err
	}

	for _, loader := range loaders {
		if loader.Version == loaderVersion {
			return true, nil
		}
	}

	return false, nil
}

// IsMCVersionSupported checks if a Minecraft version is supported by Quilt.
func (q *QuiltClient) IsMCVersionSupported(mcVersion string) (bool, error) {
	games, err := q.getGameVersions()
	if err != nil {
		return false, err
	}

	for _, game := range games {
		if game.Version == mcVersion {
			return true, nil
		}
	}

	return false, nil
}

// GetSupportedMCVersions returns all Minecraft versions supported by Quilt.
func (q *QuiltClient) GetSupportedMCVersions() ([]string, error) {
	games, err := q.getGameVersions()
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, game := range games {
		versions = append(versions, game.Version)
	}

	return versions, nil
}

// getLoaderVersions fetches Quilt loader versions.
func (q *QuiltClient) getLoaderVersions() ([]quiltLoaderResponse, error) {
	// Try cache first
	if q.cache != nil {
		if data, err := q.cache.Get(CacheKeyQuiltLoader); err == nil {
			var loaders []quiltLoaderResponse
			if err := json.Unmarshal(data, &loaders); err == nil {
				return loaders, nil
			}
		}
	}

	// Fetch from API
	url := QuiltMetaURL + "/loader"
	resp, err := q.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrNetworkError, resp.StatusCode)
	}

	var loaders []quiltLoaderResponse
	if err := json.NewDecoder(resp.Body).Decode(&loaders); err != nil {
		return nil, fmt.Errorf("failed to decode quilt loader versions: %w", err)
	}

	// Cache the result
	if q.cache != nil {
		if data, err := json.Marshal(loaders); err == nil {
			_ = q.cache.Set(CacheKeyQuiltLoader, data)
		}
	}

	return loaders, nil
}

// getGameVersions fetches Quilt-supported game versions.
func (q *QuiltClient) getGameVersions() ([]quiltGameResponse, error) {
	// Try cache first
	if q.cache != nil {
		if data, err := q.cache.Get(CacheKeyQuiltGame); err == nil {
			var games []quiltGameResponse
			if err := json.Unmarshal(data, &games); err == nil {
				return games, nil
			}
		}
	}

	// Fetch from API
	url := QuiltMetaURL + "/game"
	resp, err := q.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrNetworkError, resp.StatusCode)
	}

	var games []quiltGameResponse
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("failed to decode quilt game versions: %w", err)
	}

	// Cache the result
	if q.cache != nil {
		if data, err := json.Marshal(games); err == nil {
			_ = q.cache.Set(CacheKeyQuiltGame, data)
		}
	}

	return games, nil
}

// RefreshCache clears the cached Quilt data.
func (q *QuiltClient) RefreshCache() error {
	if q.cache != nil {
		if err := q.cache.Delete(CacheKeyQuiltLoader); err != nil {
			return err
		}
		return q.cache.Delete(CacheKeyQuiltGame)
	}
	return nil
}
