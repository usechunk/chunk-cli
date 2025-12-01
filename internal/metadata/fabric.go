package metadata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// FabricMetaURL is the base URL for Fabric metadata API.
	FabricMetaURL = "https://meta.fabricmc.net/v2/versions"
	// CacheKeyFabricLoader is the cache key for Fabric loader versions.
	CacheKeyFabricLoader = "fabric_loader"
	// CacheKeyFabricGame is the cache key for Fabric game versions.
	CacheKeyFabricGame = "fabric_game"
)

// FabricClient provides methods for fetching Fabric version information.
type FabricClient struct {
	httpClient *http.Client
	cache      *Cache
}

// NewFabricClient creates a new Fabric client with optional cache.
func NewFabricClient(cache *Cache) *FabricClient {
	return &FabricClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: cache,
	}
}

// fabricLoaderResponse represents a Fabric loader version from the API.
type fabricLoaderResponse struct {
	Separator string `json:"separator"`
	Build     int    `json:"build"`
	Maven     string `json:"maven"`
	Version   string `json:"version"`
	Stable    bool   `json:"stable"`
}

// fabricGameResponse represents a Fabric game version from the API.
type fabricGameResponse struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// GetVersions returns all available Fabric loader versions.
func (f *FabricClient) GetVersions() ([]LoaderVersion, error) {
	loaders, err := f.getLoaderVersions()
	if err != nil {
		return nil, err
	}

	var versions []LoaderVersion
	for _, loader := range loaders {
		versions = append(versions, LoaderVersion{
			Version:    loader.Version,
			Stable:     loader.Stable,
			LoaderType: LoaderFabric,
		})
	}

	return versions, nil
}

// GetVersionsForMC returns Fabric loader versions compatible with a specific Minecraft version.
// Note: Fabric loader versions are generally compatible with all supported MC versions.
func (f *FabricClient) GetVersionsForMC(mcVersion string) ([]LoaderVersion, error) {
	// First check if the MC version is supported by Fabric
	supported, err := f.IsMCVersionSupported(mcVersion)
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, ErrIncompatibleVersion
	}

	// Return all loader versions as they are generally compatible
	return f.GetVersions()
}

// GetLatestVersion returns the latest stable Fabric loader version.
func (f *FabricClient) GetLatestVersion(mcVersion string) (*LoaderVersion, error) {
	// Check MC version support
	supported, err := f.IsMCVersionSupported(mcVersion)
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, ErrIncompatibleVersion
	}

	loaders, err := f.getLoaderVersions()
	if err != nil {
		return nil, err
	}

	// Find the first stable version
	for _, loader := range loaders {
		if loader.Stable {
			return &LoaderVersion{
				Version:          loader.Version,
				MinecraftVersion: mcVersion,
				Stable:           true,
				LoaderType:       LoaderFabric,
			}, nil
		}
	}

	// If no stable version, return the first one
	if len(loaders) > 0 {
		return &LoaderVersion{
			Version:          loaders[0].Version,
			MinecraftVersion: mcVersion,
			Stable:           loaders[0].Stable,
			LoaderType:       LoaderFabric,
		}, nil
	}

	return nil, ErrNotFound
}

// IsVersionCompatible checks if a Fabric loader version is compatible with a Minecraft version.
func (f *FabricClient) IsVersionCompatible(loaderVersion, mcVersion string) (bool, error) {
	// Check if MC version is supported
	supported, err := f.IsMCVersionSupported(mcVersion)
	if err != nil {
		return false, err
	}
	if !supported {
		return false, nil
	}

	// Check if loader version exists
	loaders, err := f.getLoaderVersions()
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

// IsMCVersionSupported checks if a Minecraft version is supported by Fabric.
func (f *FabricClient) IsMCVersionSupported(mcVersion string) (bool, error) {
	games, err := f.getGameVersions()
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

// GetSupportedMCVersions returns all Minecraft versions supported by Fabric.
func (f *FabricClient) GetSupportedMCVersions() ([]string, error) {
	games, err := f.getGameVersions()
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, game := range games {
		versions = append(versions, game.Version)
	}

	return versions, nil
}

// getLoaderVersions fetches Fabric loader versions.
func (f *FabricClient) getLoaderVersions() ([]fabricLoaderResponse, error) {
	// Try cache first
	if f.cache != nil {
		if data, err := f.cache.Get(CacheKeyFabricLoader); err == nil {
			var loaders []fabricLoaderResponse
			if err := json.Unmarshal(data, &loaders); err == nil {
				return loaders, nil
			}
		}
	}

	// Fetch from API
	url := FabricMetaURL + "/loader"
	resp, err := f.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrNetworkError, resp.StatusCode)
	}

	var loaders []fabricLoaderResponse
	if err := json.NewDecoder(resp.Body).Decode(&loaders); err != nil {
		return nil, fmt.Errorf("failed to decode fabric loader versions: %w", err)
	}

	// Cache the result
	if f.cache != nil {
		if data, err := json.Marshal(loaders); err == nil {
			_ = f.cache.Set(CacheKeyFabricLoader, data)
		}
	}

	return loaders, nil
}

// getGameVersions fetches Fabric-supported game versions.
func (f *FabricClient) getGameVersions() ([]fabricGameResponse, error) {
	// Try cache first
	if f.cache != nil {
		if data, err := f.cache.Get(CacheKeyFabricGame); err == nil {
			var games []fabricGameResponse
			if err := json.Unmarshal(data, &games); err == nil {
				return games, nil
			}
		}
	}

	// Fetch from API
	url := FabricMetaURL + "/game"
	resp, err := f.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrNetworkError, resp.StatusCode)
	}

	var games []fabricGameResponse
	if err := json.NewDecoder(resp.Body).Decode(&games); err != nil {
		return nil, fmt.Errorf("failed to decode fabric game versions: %w", err)
	}

	// Cache the result
	if f.cache != nil {
		if data, err := json.Marshal(games); err == nil {
			_ = f.cache.Set(CacheKeyFabricGame, data)
		}
	}

	return games, nil
}

// RefreshCache clears the cached Fabric data.
func (f *FabricClient) RefreshCache() error {
	if f.cache != nil {
		if err := f.cache.Delete(CacheKeyFabricLoader); err != nil {
			return err
		}
		return f.cache.Delete(CacheKeyFabricGame)
	}
	return nil
}
