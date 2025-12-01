package deps

import (
	"regexp"
	"strconv"
	"strings"
)

// Version represents a semantic version.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
	Original   string // Original input string before any normalization
	Normalized string // Normalized version string (without 'v' prefix)
}

// ParseVersion parses a semver string into a Version struct.
func ParseVersion(s string) (*Version, error) {
	if s == "" {
		return nil, &ResolutionError{
			Type:    ErrInvalidConstraint,
			Message: "empty version string",
		}
	}

	original := s
	// Remove leading 'v' if present
	s = strings.TrimPrefix(s, "v")

	v := &Version{Original: original, Normalized: s}

	// Extract build metadata
	if idx := strings.Index(s, "+"); idx != -1 {
		v.Build = s[idx+1:]
		s = s[:idx]
	}

	// Extract prerelease
	if idx := strings.Index(s, "-"); idx != -1 {
		v.Prerelease = s[idx+1:]
		s = s[:idx]
	}

	// Parse major.minor.patch
	parts := strings.Split(s, ".")
	if len(parts) < 1 {
		return nil, &ResolutionError{
			Type:    ErrInvalidConstraint,
			Message: "invalid version format: " + v.Normalized,
		}
	}

	var err error
	v.Major, err = strconv.Atoi(parts[0])
	if err != nil {
		return nil, &ResolutionError{
			Type:    ErrInvalidConstraint,
			Message: "invalid major version: " + parts[0],
		}
	}

	if len(parts) >= 2 {
		v.Minor, err = strconv.Atoi(parts[1])
		if err != nil {
			return nil, &ResolutionError{
				Type:    ErrInvalidConstraint,
				Message: "invalid minor version: " + parts[1],
			}
		}
	}

	if len(parts) >= 3 {
		v.Patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return nil, &ResolutionError{
				Type:    ErrInvalidConstraint,
				Message: "invalid patch version: " + parts[2],
			}
		}
	}

	return v, nil
}

// String returns the string representation of the version.
func (v *Version) String() string {
	return v.Normalized
}

// Compare compares two versions.
// Returns -1 if v < other, 0 if v == other, 1 if v > other.
func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	// Prerelease comparison: version without prerelease > version with prerelease
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1
	}
	if v.Prerelease < other.Prerelease {
		return -1
	}
	if v.Prerelease > other.Prerelease {
		return 1
	}

	return 0
}

// Constraint represents a version constraint.
type Constraint struct {
	Op      string
	Version *Version
	Raw     string
}

// constraintPattern matches version constraints like ">=1.2.0", "<2.0.0", "=1.5.0"
var constraintPattern = regexp.MustCompile(`^(>=|<=|>|<|=|~|\^)?(.+)$`)

// ParseConstraint parses a single version constraint.
func ParseConstraint(s string) (*Constraint, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return &Constraint{Op: "*", Raw: s}, nil
	}

	matches := constraintPattern.FindStringSubmatch(s)
	if matches == nil {
		return nil, &ResolutionError{
			Type:    ErrInvalidConstraint,
			Message: "invalid constraint format: " + s,
		}
	}

	op := matches[1]
	if op == "" {
		op = "="
	}

	v, err := ParseVersion(matches[2])
	if err != nil {
		return nil, err
	}

	return &Constraint{
		Op:      op,
		Version: v,
		Raw:     s,
	}, nil
}

// Matches checks if a version satisfies the constraint.
func (c *Constraint) Matches(v *Version) bool {
	if c.Op == "*" {
		return true
	}

	cmp := v.Compare(c.Version)

	switch c.Op {
	case "=":
		return cmp == 0
	case ">":
		return cmp > 0
	case ">=":
		return cmp >= 0
	case "<":
		return cmp < 0
	case "<=":
		return cmp <= 0
	case "~": // Allows patch-level changes
		return v.Major == c.Version.Major &&
			v.Minor == c.Version.Minor &&
			cmp >= 0
	case "^": // Allows minor-level changes
		if c.Version.Major == 0 {
			// For 0.x.y, ^ behaves like ~
			return v.Major == c.Version.Major &&
				v.Minor == c.Version.Minor &&
				cmp >= 0
		}
		return v.Major == c.Version.Major && cmp >= 0
	default:
		return false
	}
}

// VersionConstraints represents a set of version constraints.
type VersionConstraints struct {
	Constraints []*Constraint
	Raw         string
}

// ParseVersionConstraints parses a version constraint string.
// Supports space-separated constraints: ">=1.2.0 <2.0.0"
// Supports OR with ||: ">=1.2.0 <2.0.0 || >=3.0.0"
func ParseVersionConstraints(s string) (*VersionConstraints, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "*" {
		return &VersionConstraints{
			Constraints: []*Constraint{{Op: "*", Raw: "*"}},
			Raw:         s,
		}, nil
	}

	// For now, we only support AND (space-separated), not OR (||)
	// OR support can be added later if needed
	parts := strings.Fields(s)

	var constraints []*Constraint
	for _, part := range parts {
		c, err := ParseConstraint(part)
		if err != nil {
			return nil, err
		}
		constraints = append(constraints, c)
	}

	return &VersionConstraints{
		Constraints: constraints,
		Raw:         s,
	}, nil
}

// Matches checks if a version satisfies all constraints.
func (vc *VersionConstraints) Matches(v *Version) bool {
	for _, c := range vc.Constraints {
		if !c.Matches(v) {
			return false
		}
	}
	return true
}

// MatchesString checks if a version string satisfies all constraints.
func (vc *VersionConstraints) MatchesString(version string) bool {
	v, err := ParseVersion(version)
	if err != nil {
		return false
	}
	return vc.Matches(v)
}

// String returns the string representation of the constraints.
func (vc *VersionConstraints) String() string {
	return vc.Raw
}

// Intersect returns the intersection of two constraint sets.
// Deduplicates constraints to avoid redundancy.
func (vc *VersionConstraints) Intersect(other *VersionConstraints) *VersionConstraints {
	// Use a map to deduplicate constraints by their raw string
	seen := make(map[string]bool)
	var combined []*Constraint

	for _, c := range vc.Constraints {
		if !seen[c.Raw] {
			seen[c.Raw] = true
			combined = append(combined, c)
		}
	}
	for _, c := range other.Constraints {
		if !seen[c.Raw] {
			seen[c.Raw] = true
			combined = append(combined, c)
		}
	}

	var rawParts []string
	if vc.Raw != "" {
		rawParts = append(rawParts, vc.Raw)
	}
	if other.Raw != "" {
		rawParts = append(rawParts, other.Raw)
	}

	return &VersionConstraints{
		Constraints: combined,
		Raw:         strings.Join(rawParts, " "),
	}
}

// IsCompatible checks if two constraint sets can be satisfied simultaneously.
// This is a simplified check - returns true if constraints don't obviously conflict.
func IsCompatible(c1, c2 *VersionConstraints) bool {
	// Simple check: ensure there's no obvious conflict
	// For a more robust check, we would need to test version ranges
	combined := c1.Intersect(c2)

	// Try to find at least one version that satisfies both
	// Since we don't have the actual versions available, we do a basic check
	var minVersion, maxVersion *Version
	var minStrict, maxStrict bool // Track if bounds are strict (> or <)
	hasMin, hasMax := false, false

	for _, c := range combined.Constraints {
		if c.Op == "*" {
			continue
		}
		switch c.Op {
		case ">":
			if !hasMin || c.Version.Compare(minVersion) > 0 ||
				(c.Version.Compare(minVersion) == 0 && !minStrict) {
				minVersion = c.Version
				minStrict = true
				hasMin = true
			}
		case ">=":
			if !hasMin || c.Version.Compare(minVersion) > 0 {
				minVersion = c.Version
				minStrict = false
				hasMin = true
			}
		case "<":
			if !hasMax || c.Version.Compare(maxVersion) < 0 ||
				(c.Version.Compare(maxVersion) == 0 && !maxStrict) {
				maxVersion = c.Version
				maxStrict = true
				hasMax = true
			}
		case "<=":
			if !hasMax || c.Version.Compare(maxVersion) < 0 {
				maxVersion = c.Version
				maxStrict = false
				hasMax = true
			}
		case "=":
			// If there's an exact match, check if it's compatible with other constraints
			for _, other := range combined.Constraints {
				if other.Op != "=" && !other.Matches(c.Version) {
					return false
				}
			}
		}
	}

	if hasMin && hasMax {
		cmp := minVersion.Compare(maxVersion)
		// If min > max, definitely incompatible
		if cmp > 0 {
			return false
		}
		// If min == max, only compatible if both bounds are inclusive
		if cmp == 0 && (minStrict || maxStrict) {
			return false
		}
		return true
	}

	return true
}
