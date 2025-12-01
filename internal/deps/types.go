// Package deps provides dependency resolution for mod compatibility.
package deps

import (
	"fmt"
	"strings"
)

// DependencyType represents the relationship type of a dependency.
type DependencyType string

const (
	// Required dependencies must be installed.
	Required DependencyType = "required"
	// Optional dependencies are installed if available.
	Optional DependencyType = "optional"
	// Incompatible mods cannot be installed together.
	Incompatible DependencyType = "incompatible"
	// Embedded dependencies are bundled within the mod.
	Embedded DependencyType = "embedded"
)

// LoaderType represents the mod loader framework.
type LoaderType string

const (
	// LoaderForge is the Forge mod loader.
	LoaderForge LoaderType = "forge"
	// LoaderFabric is the Fabric mod loader.
	LoaderFabric LoaderType = "fabric"
	// LoaderNeoForge is the NeoForge mod loader.
	LoaderNeoForge LoaderType = "neoforge"
	// LoaderSponge is the Sponge mod loader.
	LoaderSponge LoaderType = "sponge"
	// LoaderQuilt is the Quilt mod loader.
	LoaderQuilt LoaderType = "quilt"
)

// LoaderRequirement specifies a mod loader and version requirement.
type LoaderRequirement struct {
	// Loader is the required mod loader (forge, fabric, neoforge, sponge, quilt).
	Loader LoaderType `json:"loader"`
	// VersionConstraint specifies the loader version requirement (e.g., ">=47.0.0 <48.0.0").
	VersionConstraint string `json:"version_constraint,omitempty"`
}

// Dependency represents a mod dependency with version constraints.
type Dependency struct {
	// ID is the unique identifier of the dependency (mod slug or ID).
	ID string `json:"id"`
	// VersionConstraint specifies the version requirement (e.g., ">=1.2.0 <2.0.0").
	VersionConstraint string `json:"version_constraint,omitempty"`
	// Type defines the dependency relationship.
	Type DependencyType `json:"type"`
}

// ResolvedDependency represents a dependency that has been resolved to a specific version.
type ResolvedDependency struct {
	// ID is the unique identifier of the mod.
	ID string `json:"id"`
	// Version is the resolved version.
	Version string `json:"version"`
	// DownloadURL is the URL to download this version.
	DownloadURL string `json:"download_url,omitempty"`
	// Type is the dependency type.
	Type DependencyType `json:"type"`
	// Dependencies are the resolved transitive dependencies.
	Dependencies []*ResolvedDependency `json:"dependencies,omitempty"`
	// IsOptional indicates if this was an optional dependency.
	IsOptional bool `json:"is_optional,omitempty"`
}

// DependencyGraph represents the complete dependency tree.
type DependencyGraph struct {
	// Root is the root mod being resolved.
	Root *ResolvedDependency `json:"root"`
	// AllMods is a flat list of all resolved mods.
	AllMods []*ResolvedDependency `json:"all_mods"`
	// Conflicts contains any version conflicts detected.
	Conflicts []*VersionConflict `json:"conflicts,omitempty"`
	// Incompatibles contains any incompatible mod pairs.
	Incompatibles []*IncompatiblePair `json:"incompatibles,omitempty"`
	// LoaderConflicts contains any loader/framework conflicts detected.
	LoaderConflicts []*LoaderConflict `json:"loader_conflicts,omitempty"`
}

// VersionConflict represents a version conflict between dependencies.
type VersionConflict struct {
	// ModID is the ID of the mod with conflicting requirements.
	ModID string `json:"mod_id"`
	// RequiredBy lists the mods that require this dependency.
	RequiredBy []string `json:"required_by"`
	// Constraints are the conflicting version constraints.
	Constraints []string `json:"constraints"`
}

// IncompatiblePair represents two mods that cannot be installed together.
type IncompatiblePair struct {
	// ModA is the first mod ID.
	ModA string `json:"mod_a"`
	// ModB is the second mod ID.
	ModB string `json:"mod_b"`
	// Reason explains why they are incompatible.
	Reason string `json:"reason,omitempty"`
}

// LoaderConflict represents a mod requiring a different loader or version.
type LoaderConflict struct {
	// ModID is the mod with the loader requirement.
	ModID string `json:"mod_id"`
	// RequiredLoader is the loader the mod requires.
	RequiredLoader LoaderType `json:"required_loader"`
	// RequiredVersion is the loader version constraint.
	RequiredVersion string `json:"required_version,omitempty"`
	// TargetLoader is the loader being installed.
	TargetLoader LoaderType `json:"target_loader"`
	// TargetVersion is the version of the loader being installed.
	TargetVersion string `json:"target_version,omitempty"`
	// Reason explains the conflict.
	Reason string `json:"reason,omitempty"`
}

// ResolutionError represents an error during dependency resolution.
type ResolutionError struct {
	// Type is the error type.
	Type ResolutionErrorType
	// Message is a human-readable error message.
	Message string
	// Details contains additional error details.
	Details interface{}
}

// ResolutionErrorType categorizes resolution errors.
type ResolutionErrorType string

const (
	// ErrCircularDependency indicates a circular dependency was detected.
	ErrCircularDependency ResolutionErrorType = "circular_dependency"
	// ErrVersionConflict indicates incompatible version requirements.
	ErrVersionConflict ResolutionErrorType = "version_conflict"
	// ErrNotFound indicates a dependency could not be found.
	ErrNotFound ResolutionErrorType = "not_found"
	// ErrIncompatible indicates incompatible mods are requested.
	ErrIncompatible ResolutionErrorType = "incompatible"
	// ErrInvalidConstraint indicates an invalid version constraint.
	ErrInvalidConstraint ResolutionErrorType = "invalid_constraint"
	// ErrLoaderMismatch indicates a mod requires a different loader or version.
	ErrLoaderMismatch ResolutionErrorType = "loader_mismatch"
)

func (e *ResolutionError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// ModInfo provides metadata about a mod for resolution.
type ModInfo struct {
	// ID is the unique identifier.
	ID string `json:"id"`
	// Name is the display name.
	Name string `json:"name"`
	// Version is the mod version.
	Version string `json:"version"`
	// Dependencies are the mod's declared dependencies.
	Dependencies []*Dependency `json:"dependencies,omitempty"`
	// DownloadURL is the URL to download this version.
	DownloadURL string `json:"download_url,omitempty"`
	// LoaderRequirements specifies required mod loaders and their versions.
	LoaderRequirements []*LoaderRequirement `json:"loader_requirements,omitempty"`
	// MinecraftVersion specifies the required Minecraft version.
	MinecraftVersion string `json:"minecraft_version,omitempty"`
}

// ModInfoProvider is an interface for fetching mod information.
type ModInfoProvider interface {
	// GetModInfo fetches information about a specific mod version.
	GetModInfo(modID, version string) (*ModInfo, error)
	// GetLatestVersion returns the latest version matching a constraint.
	GetLatestVersion(modID, constraint string) (*ModInfo, error)
	// GetAllVersions returns all available versions of a mod.
	GetAllVersions(modID string) ([]*ModInfo, error)
}

// ResolutionStrategy defines how to select versions when multiple are available.
type ResolutionStrategy string

const (
	// StrategyLatest selects the latest compatible version.
	StrategyLatest ResolutionStrategy = "latest"
	// StrategyMinimal selects the minimum compatible version.
	StrategyMinimal ResolutionStrategy = "minimal"
)

// ResolutionOptions configures the resolver behavior.
type ResolutionOptions struct {
	// Strategy determines how to select versions.
	Strategy ResolutionStrategy
	// IncludeOptional whether to resolve optional dependencies.
	IncludeOptional bool
	// MaxDepth limits the recursion depth (0 = unlimited).
	MaxDepth int
	// TargetLoader is the mod loader being used (forge, fabric, neoforge, etc.).
	TargetLoader LoaderType
	// TargetLoaderVersion is the version of the target loader.
	TargetLoaderVersion string
	// MinecraftVersion is the target Minecraft version.
	MinecraftVersion string
}

// DefaultResolutionOptions returns the default resolution options.
func DefaultResolutionOptions() *ResolutionOptions {
	return &ResolutionOptions{
		Strategy:        StrategyLatest,
		IncludeOptional: true,
		MaxDepth:        0,
	}
}

// CheckLoaderCompatibility checks if a mod is compatible with the target loader.
func CheckLoaderCompatibility(mod *ModInfo, targetLoader LoaderType, targetVersion string) *LoaderConflict {
	if len(mod.LoaderRequirements) == 0 {
		return nil // No loader requirements specified
	}

	for _, req := range mod.LoaderRequirements {
		if req.Loader == targetLoader {
			// Loader matches, check version if specified
			if req.VersionConstraint != "" && targetVersion != "" {
				constraints, err := ParseVersionConstraints(req.VersionConstraint)
				if err != nil {
					return &LoaderConflict{
						ModID:           mod.ID,
						RequiredLoader:  req.Loader,
						RequiredVersion: req.VersionConstraint,
						TargetLoader:    targetLoader,
						TargetVersion:   targetVersion,
						Reason:          fmt.Sprintf("invalid loader version constraint: %v", err),
					}
				}
				if !constraints.MatchesString(targetVersion) {
					return &LoaderConflict{
						ModID:           mod.ID,
						RequiredLoader:  req.Loader,
						RequiredVersion: req.VersionConstraint,
						TargetLoader:    targetLoader,
						TargetVersion:   targetVersion,
						Reason: fmt.Sprintf("mod %s requires %s version %s, but %s is installed",
							mod.ID, req.Loader, req.VersionConstraint, targetVersion),
					}
				}
			}
			return nil // Compatible
		}
	}

	// No matching loader found - mod may not support the target loader
	var requiredLoaders []string
	for _, req := range mod.LoaderRequirements {
		requiredLoaders = append(requiredLoaders, string(req.Loader))
	}
	return &LoaderConflict{
		ModID:          mod.ID,
		RequiredLoader: mod.LoaderRequirements[0].Loader,
		TargetLoader:   targetLoader,
		TargetVersion:  targetVersion,
		Reason: fmt.Sprintf("mod %s requires one of [%s], but %s is being used",
			mod.ID, strings.Join(requiredLoaders, ", "), targetLoader),
	}
}
