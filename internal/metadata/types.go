// Package metadata provides types and functions for querying and managing
// Minecraft version data, mod loader versions, and their compatibility matrices.
package metadata

import (
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when the requested version or resource is not found.
	ErrNotFound = errors.New("not found")
	// ErrInvalidVersion is returned when a version string is invalid.
	ErrInvalidVersion = errors.New("invalid version")
	// ErrIncompatibleVersion is returned when a loader version is incompatible with the MC version.
	ErrIncompatibleVersion = errors.New("incompatible version")
	// ErrNetworkError is returned when a network request fails.
	ErrNetworkError = errors.New("network error")
	// ErrCacheExpired is returned when the cache has expired.
	ErrCacheExpired = errors.New("cache expired")
)

// LoaderType represents the type of mod loader.
type LoaderType string

const (
	LoaderForge    LoaderType = "forge"
	LoaderFabric   LoaderType = "fabric"
	LoaderQuilt    LoaderType = "quilt"
	LoaderNeoForge LoaderType = "neoforge"
)

// VersionType indicates whether a version is stable or experimental.
type VersionType string

const (
	VersionRelease  VersionType = "release"
	VersionSnapshot VersionType = "snapshot"
	VersionBeta     VersionType = "old_beta"
	VersionAlpha    VersionType = "old_alpha"
)

// MinecraftVersion represents a Minecraft version from the version manifest.
type MinecraftVersion struct {
	ID          string      `json:"id"`
	Type        VersionType `json:"type"`
	URL         string      `json:"url"`
	Time        time.Time   `json:"time"`
	ReleaseTime time.Time   `json:"releaseTime"`
	JavaVersion int         `json:"javaVersion,omitempty"`
}

// MinecraftVersionManifest represents the Mojang version manifest.
type MinecraftVersionManifest struct {
	Latest   LatestVersions      `json:"latest"`
	Versions []*MinecraftVersion `json:"versions"`
}

// LatestVersions contains the latest release and snapshot versions.
type LatestVersions struct {
	Release  string `json:"release"`
	Snapshot string `json:"snapshot"`
}

// LoaderVersion represents a mod loader version.
type LoaderVersion struct {
	Version          string     `json:"version"`
	MinecraftVersion string     `json:"minecraft_version,omitempty"`
	Stable           bool       `json:"stable"`
	LoaderType       LoaderType `json:"loader_type"`
	ReleaseDate      time.Time  `json:"release_date,omitempty"`
}

// ForgeVersion represents a Forge mod loader version.
type ForgeVersion struct {
	Version          string `json:"version"`
	MinecraftVersion string `json:"minecraft_version"`
	IsRecommended    bool   `json:"is_recommended"`
	IsLatest         bool   `json:"is_latest"`
}

// FabricVersion represents a Fabric mod loader version.
type FabricVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// FabricGameVersion represents a Minecraft version supported by Fabric.
type FabricGameVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// QuiltVersion represents a Quilt mod loader version.
type QuiltVersion struct {
	Version string `json:"version"`
}

// QuiltGameVersion represents a Minecraft version supported by Quilt.
type QuiltGameVersion struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

// NeoForgeVersion represents a NeoForge mod loader version.
type NeoForgeVersion struct {
	Version          string `json:"version"`
	MinecraftVersion string `json:"minecraft_version"`
}

// VersionProvider defines the interface for fetching version information.
type VersionProvider interface {
	// GetVersions returns all available versions.
	GetVersions() ([]LoaderVersion, error)
	// GetVersionsForMC returns versions compatible with a specific Minecraft version.
	GetVersionsForMC(mcVersion string) ([]LoaderVersion, error)
	// GetLatestVersion returns the latest stable version for a Minecraft version.
	GetLatestVersion(mcVersion string) (*LoaderVersion, error)
	// IsVersionCompatible checks if a loader version is compatible with a MC version.
	IsVersionCompatible(loaderVersion, mcVersion string) (bool, error)
}

// MinecraftProvider defines the interface for fetching Minecraft version information.
type MinecraftProvider interface {
	// GetVersionManifest returns the full version manifest.
	GetVersionManifest() (*MinecraftVersionManifest, error)
	// GetVersion returns details for a specific Minecraft version.
	GetVersion(version string) (*MinecraftVersion, error)
	// GetLatestRelease returns the latest stable release version.
	GetLatestRelease() (*MinecraftVersion, error)
	// GetLatestSnapshot returns the latest snapshot version.
	GetLatestSnapshot() (*MinecraftVersion, error)
	// GetJavaVersion returns the required Java version for a Minecraft version.
	GetJavaVersion(mcVersion string) (int, error)
}

// CacheEntry represents a cached item with TTL.
type CacheEntry struct {
	Data      []byte    `json:"data"`
	ExpiresAt time.Time `json:"expires_at"`
	CachedAt  time.Time `json:"cached_at"`
}

// IsExpired returns true if the cache entry has expired.
func (c *CacheEntry) IsExpired() bool {
	return time.Now().After(c.ExpiresAt)
}

// JavaVersionRequirements maps Minecraft version patterns to required Java versions.
// This is used when the version manifest doesn't include Java version info.
var JavaVersionRequirements = map[string]int{
	// Java 21+ (1.20.5+)
	"1.21":   21,
	"1.20.5": 21,
	"1.20.6": 21,
	// Java 17+ (1.18+)
	"1.20": 17,
	"1.19": 17,
	"1.18": 17,
	"1.17": 16, // 1.17 requires Java 16
	// Java 8+ (older versions)
	"1.16": 8,
	"1.15": 8,
	"1.14": 8,
	"1.13": 8,
	"1.12": 8,
	"1.11": 8,
	"1.10": 8,
	"1.9":  8,
	"1.8":  8,
	"1.7":  8,
}
