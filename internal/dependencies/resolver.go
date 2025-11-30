package dependencies

import (
	"fmt"
)

// ModRegistry provides access to available mod versions.
type ModRegistry interface {
	// GetMod retrieves a mod by ID and version expression.
	// Returns nil if no matching version is found.
	GetMod(modID string, versionExpr string) (*ModInfo, error)

	// GetAvailableVersions returns all available versions for a mod.
	GetAvailableVersions(modID string) ([]*ModInfo, error)

	// GetLatestVersion returns the latest stable version of a mod.
	GetLatestVersion(modID string) (*ModInfo, error)
}

// Resolver handles dependency resolution for mods.
type Resolver struct {
	registry      ModRegistry
	resolved      map[string]*ResolvedMod
	resolving     map[string]bool // Track mods currently being resolved (for cycle detection)
	conflicts     []Conflict
	missingDeps   []Dependency
	resolutionErr []error // Track resolution errors
}

// NewResolver creates a new dependency resolver.
func NewResolver(registry ModRegistry) *Resolver {
	return &Resolver{
		registry:      registry,
		resolved:      make(map[string]*ResolvedMod),
		resolving:     make(map[string]bool),
		conflicts:     []Conflict{},
		resolutionErr: []error{},
		missingDeps:   []Dependency{},
	}
}

// Resolve resolves dependencies for a list of mods.
// Returns the resolution result with mods in topologically sorted install order.
func (r *Resolver) Resolve(mods []*ModInfo) (*ResolutionResult, error) {
	// Reset state
	r.resolved = make(map[string]*ResolvedMod)
	r.resolving = make(map[string]bool)
	r.conflicts = []Conflict{}
	r.missingDeps = []Dependency{}
	r.resolutionErr = []error{}

	// Resolve each mod
	for _, mod := range mods {
		if err := r.resolveMod(mod, ""); err != nil {
			// Track the error but continue resolving other mods to collect all issues
			r.resolutionErr = append(r.resolutionErr, fmt.Errorf("failed to resolve %s: %w", mod.ID, err))
		}
	}

	// Build result
	result := &ResolutionResult{
		ResolvedMods: make([]*ResolvedMod, 0, len(r.resolved)),
		Conflicts:    r.conflicts,
		MissingDeps:  r.missingDeps,
	}

	// Get topologically sorted order
	sortedIDs, err := r.topologicalSort()
	if err != nil {
		return nil, err
	}

	result.InstallOrder = sortedIDs

	for _, id := range sortedIDs {
		if mod, ok := r.resolved[id]; ok {
			result.ResolvedMods = append(result.ResolvedMods, mod)
		}
	}

	return result, nil
}

// resolveMod recursively resolves a mod and its dependencies.
func (r *Resolver) resolveMod(mod *ModInfo, requiredBy string) error {
	// Check for circular dependency
	if r.resolving[mod.ID] {
		return fmt.Errorf("%w: %s", ErrCircularDependency, mod.ID)
	}

	// Check if already resolved
	if existing, ok := r.resolved[mod.ID]; ok {
		// Check version compatibility
		existingVersion, err := ParseVersion(existing.ModInfo.Version)
		if err != nil {
			return err
		}
		newVersion, err := ParseVersion(mod.Version)
		if err != nil {
			return err
		}

		// If different versions, record conflict
		if existingVersion.Compare(newVersion) != 0 {
			r.addConflict(mod.ID, requiredBy, existing.InstalledBy, mod.Version, existing.ModInfo.Version)
		}
		return nil
	}

	// Mark as being resolved
	r.resolving[mod.ID] = true
	defer delete(r.resolving, mod.ID)

	// Resolve dependencies first
	for _, dep := range mod.Dependencies {
		if !dep.Required {
			continue
		}

		depMod, err := r.registry.GetMod(dep.ModID, dep.VersionExpr)
		if err != nil || depMod == nil {
			r.missingDeps = append(r.missingDeps, dep)
			continue
		}

		if err := r.resolveMod(depMod, mod.ID); err != nil {
			// Record but continue to find all issues
			continue
		}
	}

	// Add resolved mod
	r.resolved[mod.ID] = &ResolvedMod{
		ModInfo:     mod,
		InstalledBy: requiredBy,
	}

	return nil
}

// addConflict records a version conflict.
func (r *Resolver) addConflict(modID, requiredBy1, requiredBy2, version1, version2 string) {
	// Check if conflict already exists
	for i := range r.conflicts {
		if r.conflicts[i].ModID == modID {
			// Add to existing conflict
			r.conflicts[i].RequiredBy = append(r.conflicts[i].RequiredBy, requiredBy1)
			r.conflicts[i].Versions = append(r.conflicts[i].Versions, version1)
			return
		}
	}

	// Create new conflict
	r.conflicts = append(r.conflicts, Conflict{
		ModID:      modID,
		RequiredBy: []string{requiredBy1, requiredBy2},
		Versions:   []string{version1, version2},
	})
}

// topologicalSort returns mod IDs in topologically sorted order (dependencies first).
func (r *Resolver) topologicalSort() ([]string, error) {
	// Build adjacency list
	inDegree := make(map[string]int)
	graph := make(map[string][]string)

	// Initialize
	for id := range r.resolved {
		inDegree[id] = 0
		graph[id] = []string{}
	}

	// Build edges: dependency -> dependent
	for id, mod := range r.resolved {
		for _, dep := range mod.ModInfo.Dependencies {
			if _, ok := r.resolved[dep.ModID]; ok {
				graph[dep.ModID] = append(graph[dep.ModID], id)
				inDegree[id]++
			}
		}
	}

	// Kahn's algorithm
	var queue []string
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	var sorted []string
	for len(queue) > 0 {
		// Pop from queue
		current := queue[0]
		queue = queue[1:]
		sorted = append(sorted, current)

		// Process neighbors
		for _, neighbor := range graph[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// Check for cycles
	if len(sorted) != len(r.resolved) {
		return nil, ErrCircularDependency
	}

	return sorted, nil
}

// HasConflicts returns true if there are any version conflicts.
func (r *Resolver) HasConflicts() bool {
	return len(r.conflicts) > 0
}

// HasMissingDeps returns true if there are any unresolved dependencies.
func (r *Resolver) HasMissingDeps() bool {
	return len(r.missingDeps) > 0
}

// GetResolutionErrors returns any errors encountered during resolution.
func (r *Resolver) GetResolutionErrors() []error {
	return r.resolutionErr
}

// FindBestVersion finds the best version that satisfies multiple version expressions.
func FindBestVersion(versions []*ModInfo, constraints []string) (*ModInfo, error) {
	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions available")
	}

	// Parse all constraints
	var ranges []*VersionRange
	for _, expr := range constraints {
		vr, err := ParseVersionRange(expr)
		if err != nil {
			return nil, fmt.Errorf("invalid constraint %q: %w", expr, err)
		}
		ranges = append(ranges, vr)
	}

	// Find versions that satisfy all constraints
	var compatible []*ModInfo
	for _, mod := range versions {
		v, err := ParseVersion(mod.Version)
		if err != nil {
			continue
		}

		satisfiesAll := true
		for _, vr := range ranges {
			if !vr.Matches(v) {
				satisfiesAll = false
				break
			}
		}

		if satisfiesAll {
			compatible = append(compatible, mod)
		}
	}

	if len(compatible) == 0 {
		return nil, fmt.Errorf("no version satisfies all constraints")
	}

	// Return the latest compatible version
	best := compatible[0]
	bestV, _ := ParseVersion(best.Version)

	for _, mod := range compatible[1:] {
		v, err := ParseVersion(mod.Version)
		if err != nil {
			continue
		}
		if v.Compare(bestV) > 0 {
			best = mod
			bestV = v
		}
	}

	return best, nil
}
