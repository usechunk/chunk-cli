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
	// NeoForgeMavenURL is the base URL for NeoForge Maven repository.
	NeoForgeMavenURL = "https://maven.neoforged.net/api/maven/versions/releases/net/neoforged/neoforge"
	// CacheKeyNeoForgeVersions is the cache key for NeoForge versions.
	CacheKeyNeoForgeVersions = "neoforge_versions"
)

// neoForgeVersionPattern is a pre-compiled regex for parsing NeoForge version strings.
var neoForgeVersionPattern = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-(.+))?$`)

// NeoForgeClient provides methods for fetching NeoForge version information.
type NeoForgeClient struct {
	httpClient *http.Client
	cache      *Cache
}

// NewNeoForgeClient creates a new NeoForge client with optional cache.
func NewNeoForgeClient(cache *Cache) *NeoForgeClient {
	return &NeoForgeClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: cache,
	}
}

// neoForgeMavenResponse represents the NeoForge Maven API response.
type neoForgeMavenResponse struct {
	IsSnapshot bool     `json:"isSnapshot"`
	Versions   []string `json:"versions"`
}

// GetVersions returns all available NeoForge versions.
func (n *NeoForgeClient) GetVersions() ([]LoaderVersion, error) {
	mavenData, err := n.getMavenVersions()
	if err != nil {
		return nil, err
	}

	return n.parseVersions(mavenData.Versions)
}

// GetVersionsForMC returns NeoForge versions compatible with a specific Minecraft version.
func (n *NeoForgeClient) GetVersionsForMC(mcVersion string) ([]LoaderVersion, error) {
	allVersions, err := n.GetVersions()
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

// GetLatestVersion returns the latest NeoForge version for a Minecraft version.
func (n *NeoForgeClient) GetLatestVersion(mcVersion string) (*LoaderVersion, error) {
	versions, err := n.GetVersionsForMC(mcVersion)
	if err != nil {
		return nil, err
	}

	if len(versions) == 0 {
		return nil, ErrNotFound
	}

	// Return the first (latest) version
	return &versions[0], nil
}

// IsVersionCompatible checks if a NeoForge version is compatible with a Minecraft version.
func (n *NeoForgeClient) IsVersionCompatible(neoforgeVersion, mcVersion string) (bool, error) {
	versions, err := n.GetVersionsForMC(mcVersion)
	if err != nil {
		return false, err
	}

	for _, v := range versions {
		if v.Version == neoforgeVersion {
			return true, nil
		}
	}

	return false, nil
}

// GetSupportedMCVersions returns all Minecraft versions that have NeoForge support.
func (n *NeoForgeClient) GetSupportedMCVersions() ([]string, error) {
	allVersions, err := n.GetVersions()
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var mcVersions []string

	for _, v := range allVersions {
		if v.MinecraftVersion != "" && !seen[v.MinecraftVersion] {
			seen[v.MinecraftVersion] = true
			mcVersions = append(mcVersions, v.MinecraftVersion)
		}
	}

	return mcVersions, nil
}

// getMavenVersions fetches version data from NeoForge Maven.
func (n *NeoForgeClient) getMavenVersions() (*neoForgeMavenResponse, error) {
	// Try cache first
	if n.cache != nil {
		if data, err := n.cache.Get(CacheKeyNeoForgeVersions); err == nil {
			var mavenData neoForgeMavenResponse
			if err := json.Unmarshal(data, &mavenData); err == nil {
				return &mavenData, nil
			}
		}
	}

	// Fetch from API
	resp, err := n.httpClient.Get(NeoForgeMavenURL)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNetworkError, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrNetworkError, resp.StatusCode)
	}

	var mavenData neoForgeMavenResponse
	if err := json.NewDecoder(resp.Body).Decode(&mavenData); err != nil {
		return nil, fmt.Errorf("failed to decode neoforge versions: %w", err)
	}

	// Cache the result
	if n.cache != nil {
		if data, err := json.Marshal(mavenData); err == nil {
			_ = n.cache.Set(CacheKeyNeoForgeVersions, data)
		}
	}

	return &mavenData, nil
}

// parseVersions converts NeoForge version strings to LoaderVersion slice.
// NeoForge version format: MCMinor.MCPatch.NeoForgeVersion (e.g., "21.1.1" for MC 1.21.1)
// For MC 1.20.x: Format is 20.x.* (e.g., "20.4.237" for MC 1.20.4)
func (n *NeoForgeClient) parseVersions(versionStrings []string) ([]LoaderVersion, error) {
	var versions []LoaderVersion

	for _, v := range versionStrings {
		matches := neoForgeVersionPattern.FindStringSubmatch(v)
		if len(matches) < 4 {
			continue
		}

		major := matches[1]
		minor := matches[2]

		// Determine MC version from NeoForge version
		// NeoForge 20.x.* -> MC 1.20.x
		// NeoForge 21.x.* -> MC 1.21.x
		mcVersion := n.inferMCVersion(major, minor)

		stable := true
		if len(matches) > 4 && matches[4] != "" {
			suffix := strings.ToLower(matches[4])
			stable = !strings.Contains(suffix, "beta") && !strings.Contains(suffix, "alpha")
		}

		versions = append(versions, LoaderVersion{
			Version:          v,
			MinecraftVersion: mcVersion,
			Stable:           stable,
			LoaderType:       LoaderNeoForge,
		})
	}

	// Sort by version (newest first) - versions are typically ordered in Maven response
	// Reverse the order since Maven returns oldest first
	for i, j := 0, len(versions)-1; i < j; i, j = i+1, j-1 {
		versions[i], versions[j] = versions[j], versions[i]
	}

	return versions, nil
}

// inferMCVersion attempts to determine the Minecraft version from NeoForge version numbers.
func (n *NeoForgeClient) inferMCVersion(major, minor string) string {
	// NeoForge uses a versioning scheme where:
	// major = MC minor version (e.g., 20 for 1.20.x, 21 for 1.21.x)
	// minor = MC patch version (e.g., 4 for 1.20.4)
	return fmt.Sprintf("1.%s.%s", major, minor)
}

// RefreshCache clears the cached NeoForge data.
func (n *NeoForgeClient) RefreshCache() error {
	if n.cache != nil {
		return n.cache.Delete(CacheKeyNeoForgeVersions)
	}
	return nil
}
