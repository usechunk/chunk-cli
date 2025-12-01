package deps

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Resolver resolves dependency trees.
type Resolver struct {
	provider ModInfoProvider
	options  *ResolutionOptions
	cache    *resolutionCache
}

// resolutionCache caches resolution results.
type resolutionCache struct {
	mu       sync.RWMutex
	resolved map[string]*ResolvedDependency
}

func newResolutionCache() *resolutionCache {
	return &resolutionCache{
		resolved: make(map[string]*ResolvedDependency),
	}
}

func (c *resolutionCache) get(key string) (*ResolvedDependency, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.resolved[key]
	return v, ok
}

func (c *resolutionCache) set(key string, value *ResolvedDependency) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.resolved[key] = value
}

// NewResolver creates a new dependency resolver.
func NewResolver(provider ModInfoProvider, options *ResolutionOptions) *Resolver {
	if options == nil {
		options = DefaultResolutionOptions()
	}
	return &Resolver{
		provider: provider,
		options:  options,
		cache:    newResolutionCache(),
	}
}

// Resolve resolves the dependency tree for a mod.
func (r *Resolver) Resolve(modID, version string) (*DependencyGraph, error) {
	// Get the root mod info
	modInfo, err := r.provider.GetModInfo(modID, version)
	if err != nil {
		return nil, &ResolutionError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("failed to get mod info for %s@%s: %v", modID, version, err),
		}
	}

	// Track visited mods to detect circular dependencies
	visited := make(map[string]bool)
	visiting := make(map[string]bool)

	// Resolve the dependency tree
	resolved, conflicts, incompatibles, loaderConflicts, err := r.resolveDependencies(modInfo, visited, visiting, 0)
	if err != nil {
		return nil, err
	}

	// Build the flat list of all mods
	allMods := r.flattenDependencies(resolved)

	return &DependencyGraph{
		Root:            resolved,
		AllMods:         allMods,
		Conflicts:       conflicts,
		Incompatibles:   incompatibles,
		LoaderConflicts: loaderConflicts,
	}, nil
}

// resolveDependencies recursively resolves dependencies.
func (r *Resolver) resolveDependencies(
	modInfo *ModInfo,
	visited map[string]bool,
	visiting map[string]bool,
	depth int,
) (*ResolvedDependency, []*VersionConflict, []*IncompatiblePair, []*LoaderConflict, error) {
	key := modInfo.ID + "@" + modInfo.Version

	// Check for circular dependency
	if visiting[key] {
		return nil, nil, nil, nil, &ResolutionError{
			Type:    ErrCircularDependency,
			Message: fmt.Sprintf("circular dependency detected: %s", key),
		}
	}

	// Check cache
	if cached, ok := r.cache.get(key); ok {
		return cached, nil, nil, nil, nil
	}

	// Check max depth
	if r.options.MaxDepth > 0 && depth >= r.options.MaxDepth {
		return &ResolvedDependency{
			ID:          modInfo.ID,
			Version:     modInfo.Version,
			DownloadURL: modInfo.DownloadURL,
		}, nil, nil, nil, nil
	}

	// Mark as currently visiting
	visiting[key] = true
	defer func() { visiting[key] = false }()

	// Check loader compatibility if target loader is specified
	var allLoaderConflicts []*LoaderConflict
	if r.options.TargetLoader != "" {
		if conflict := CheckLoaderCompatibility(modInfo, r.options.TargetLoader, r.options.TargetLoaderVersion); conflict != nil {
			allLoaderConflicts = append(allLoaderConflicts, conflict)
		}
	}

	resolved := &ResolvedDependency{
		ID:          modInfo.ID,
		Version:     modInfo.Version,
		DownloadURL: modInfo.DownloadURL,
	}

	var allConflicts []*VersionConflict
	var allIncompatibles []*IncompatiblePair

	// Resolve each dependency
	for _, dep := range modInfo.Dependencies {
		// Handle incompatible dependencies
		if dep.Type == Incompatible {
			// Check if the incompatible mod is already resolved
			if visited[dep.ID] {
				allIncompatibles = append(allIncompatibles, &IncompatiblePair{
					ModA:   modInfo.ID,
					ModB:   dep.ID,
					Reason: fmt.Sprintf("%s declares %s as incompatible", modInfo.ID, dep.ID),
				})
			}
			continue
		}

		// Skip optional dependencies if not requested
		if dep.Type == Optional && !r.options.IncludeOptional {
			continue
		}

		// Skip embedded dependencies (they're already included)
		if dep.Type == Embedded {
			resolved.Dependencies = append(resolved.Dependencies, &ResolvedDependency{
				ID:         dep.ID,
				Version:    dep.VersionConstraint,
				Type:       Embedded,
				IsOptional: false,
			})
			continue
		}

		// Find the best version matching the constraint
		depInfo, err := r.findBestVersion(dep.ID, dep.VersionConstraint)
		if err != nil {
			if dep.Type == Optional {
				// Optional dependency not found - skip it
				continue
			}
			return nil, nil, nil, nil, &ResolutionError{
				Type:    ErrNotFound,
				Message: fmt.Sprintf("dependency %s not found: %v", dep.ID, err),
			}
		}

		// Recursively resolve the dependency
		depResolved, conflicts, incompatibles, loaderConflicts, err := r.resolveDependencies(depInfo, visited, visiting, depth+1)
		if err != nil {
			if dep.Type == Optional {
				// Failed to resolve optional dependency - skip it
				continue
			}
			return nil, nil, nil, nil, err
		}

		depResolved.Type = dep.Type
		depResolved.IsOptional = dep.Type == Optional

		resolved.Dependencies = append(resolved.Dependencies, depResolved)
		allConflicts = append(allConflicts, conflicts...)
		allIncompatibles = append(allIncompatibles, incompatibles...)
		allLoaderConflicts = append(allLoaderConflicts, loaderConflicts...)
	}

	// Mark as visited
	visited[modInfo.ID] = true

	// Cache the result
	r.cache.set(key, resolved)

	return resolved, allConflicts, allIncompatibles, allLoaderConflicts, nil
}

// findBestVersion finds the best version of a mod matching the constraint.
func (r *Resolver) findBestVersion(modID, constraint string) (*ModInfo, error) {
	// If no constraint, get the latest version
	if constraint == "" || constraint == "*" {
		return r.provider.GetLatestVersion(modID, "*")
	}

	// Parse the constraint
	constraints, err := ParseVersionConstraints(constraint)
	if err != nil {
		return nil, err
	}

	// Get all versions
	versions, err := r.provider.GetAllVersions(modID)
	if err != nil {
		return nil, err
	}

	// Filter versions that match the constraint
	var matching []*ModInfo
	for _, v := range versions {
		if constraints.MatchesString(v.Version) {
			matching = append(matching, v)
		}
	}

	if len(matching) == 0 {
		return nil, &ResolutionError{
			Type:    ErrNotFound,
			Message: fmt.Sprintf("no version of %s matches constraint %s", modID, constraint),
		}
	}

	// Sort versions based on strategy
	sort.Slice(matching, func(i, j int) bool {
		vi, err1 := ParseVersion(matching[i].Version)
		vj, err2 := ParseVersion(matching[j].Version)
		// Handle parsing errors: place unparsable versions last
		if err1 != nil && err2 != nil {
			return false // maintain order
		}
		if err1 != nil {
			return false // i is unparsable, so j < i
		}
		if err2 != nil {
			return true // j is unparsable, so i < j
		}
		cmp := vi.Compare(vj)
		if r.options.Strategy == StrategyLatest {
			return cmp > 0 // Latest first
		}
		return cmp < 0 // Minimal first
	})

	return matching[0], nil
}

// flattenDependencies returns a flat list of all resolved dependencies.
func (r *Resolver) flattenDependencies(resolved *ResolvedDependency) []*ResolvedDependency {
	seen := make(map[string]bool)
	var result []*ResolvedDependency

	var flatten func(dep *ResolvedDependency)
	flatten = func(dep *ResolvedDependency) {
		key := dep.ID + "@" + dep.Version
		if seen[key] {
			return
		}
		seen[key] = true
		result = append(result, dep)

		for _, child := range dep.Dependencies {
			flatten(child)
		}
	}

	flatten(resolved)
	return result
}

// ValidateDependencies checks a list of dependencies for conflicts and issues.
func (r *Resolver) ValidateDependencies(deps []*Dependency) ([]*ValidationResult, error) {
	var results []*ValidationResult

	// Track version requirements per mod
	requirements := make(map[string][]string)  // modID -> list of constraints
	incompatibles := make(map[string][]string) // modID -> list of mods that mark it incompatible

	for _, dep := range deps {
		if dep.Type == Incompatible {
			incompatibles[dep.ID] = append(incompatibles[dep.ID], "root")
		} else if dep.VersionConstraint != "" {
			requirements[dep.ID] = append(requirements[dep.ID], dep.VersionConstraint)
		}
	}

	// Check for version conflicts
	for modID, constraints := range requirements {
		if len(constraints) > 1 {
			// Check if constraints are compatible
			compatible := true
			for i := 0; i < len(constraints); i++ {
				c1, err := ParseVersionConstraints(constraints[i])
				if err != nil {
					continue
				}
				for j := i + 1; j < len(constraints); j++ {
					c2, err := ParseVersionConstraints(constraints[j])
					if err != nil {
						continue
					}
					if !IsCompatible(c1, c2) {
						compatible = false
						break
					}
				}
				if !compatible {
					break
				}
			}

			if !compatible {
				results = append(results, &ValidationResult{
					Type:    ValidationConflict,
					ModID:   modID,
					Message: fmt.Sprintf("conflicting version constraints: %s", strings.Join(constraints, ", ")),
				})
			}
		}
	}

	// Check for incompatible mods
	for modID := range requirements {
		if blockers, ok := incompatibles[modID]; ok {
			results = append(results, &ValidationResult{
				Type:    ValidationIncompatible,
				ModID:   modID,
				Message: fmt.Sprintf("incompatible with: %s", strings.Join(blockers, ", ")),
			})
		}
	}

	return results, nil
}

// ValidationResult represents a validation issue.
type ValidationResult struct {
	Type    ValidationType
	ModID   string
	Message string
}

// ValidationType categorizes validation results.
type ValidationType string

const (
	ValidationConflict     ValidationType = "conflict"
	ValidationIncompatible ValidationType = "incompatible"
	ValidationMissing      ValidationType = "missing"
	ValidationWarning      ValidationType = "warning"
)

// GenerateGraph returns a DOT format graph for visualization.
func (g *DependencyGraph) GenerateGraph() string {
	var sb strings.Builder
	sb.WriteString("digraph dependencies {\n")
	sb.WriteString("  rankdir=TB;\n")
	sb.WriteString("  node [shape=box];\n\n")

	// Add nodes and edges
	var addNode func(dep *ResolvedDependency, parent string)
	seen := make(map[string]bool)

	addNode = func(dep *ResolvedDependency, parent string) {
		nodeID := strings.ReplaceAll(dep.ID, "-", "_")
		if !seen[dep.ID] {
			seen[dep.ID] = true
			label := fmt.Sprintf("%s\\n%s", dep.ID, dep.Version)
			color := "black"
			if dep.IsOptional {
				color = "gray"
			}
			if dep.Type == Embedded {
				color = "blue"
			}
			sb.WriteString(fmt.Sprintf("  %s [label=\"%s\" color=\"%s\"];\n", nodeID, label, color))
		}

		if parent != "" {
			parentID := strings.ReplaceAll(parent, "-", "_")
			style := "solid"
			if dep.IsOptional {
				style = "dashed"
			}
			sb.WriteString(fmt.Sprintf("  %s -> %s [style=\"%s\"];\n", parentID, nodeID, style))
		}

		for _, child := range dep.Dependencies {
			addNode(child, dep.ID)
		}
	}

	if g.Root != nil {
		addNode(g.Root, "")
	}

	// Add conflict nodes
	for _, conflict := range g.Conflicts {
		nodeID := strings.ReplaceAll(conflict.ModID, "-", "_") + "_conflict"
		sb.WriteString(fmt.Sprintf("  %s [label=\"CONFLICT: %s\" color=\"red\" style=\"filled\" fillcolor=\"#ffcccc\"];\n",
			nodeID, conflict.ModID))
	}

	// Add incompatible edges
	for _, pair := range g.Incompatibles {
		modA := strings.ReplaceAll(pair.ModA, "-", "_")
		modB := strings.ReplaceAll(pair.ModB, "-", "_")
		sb.WriteString(fmt.Sprintf("  %s -> %s [color=\"red\" style=\"dashed\" label=\"incompatible\"];\n", modA, modB))
	}

	sb.WriteString("}\n")
	return sb.String()
}

// HasErrors returns true if the graph has any conflicts or incompatibilities.
func (g *DependencyGraph) HasErrors() bool {
	return len(g.Conflicts) > 0 || len(g.Incompatibles) > 0 || len(g.LoaderConflicts) > 0
}

// GetErrors returns a list of error messages.
func (g *DependencyGraph) GetErrors() []string {
	var errors []string

	for _, conflict := range g.Conflicts {
		errors = append(errors, fmt.Sprintf("Version conflict for %s: required by %s with constraints %s",
			conflict.ModID, strings.Join(conflict.RequiredBy, ", "), strings.Join(conflict.Constraints, ", ")))
	}

	for _, pair := range g.Incompatibles {
		errors = append(errors, fmt.Sprintf("Incompatible mods: %s and %s - %s",
			pair.ModA, pair.ModB, pair.Reason))
	}

	for _, loader := range g.LoaderConflicts {
		errors = append(errors, fmt.Sprintf("Loader conflict for %s: %s",
			loader.ModID, loader.Reason))
	}

	return errors
}
