package dependencies

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	ErrCircularDependency  = errors.New("circular dependency detected")
	ErrUnresolvedDep       = errors.New("unresolved dependency")
	ErrVersionConflict     = errors.New("version conflict")
	ErrInvalidVersionRange = errors.New("invalid version range")
)

// Dependency represents a mod dependency with version constraints.
type Dependency struct {
	ModID       string
	VersionExpr string // Version expression (e.g., ">=1.0.0", "1.0.0-2.0.0", "*")
	Required    bool
	Side        string // "client", "server", or "both"
}

// ModInfo contains metadata about a mod including its dependencies.
type ModInfo struct {
	ID           string
	Name         string
	Version      string
	Dependencies []Dependency
	Provides     []string // IDs this mod provides (for compatibility)
}

// ResolvedMod represents a mod that has been resolved with a specific version.
type ResolvedMod struct {
	ModInfo     *ModInfo
	DownloadURL string
	FileName    string
	InstalledBy string // Which mod required this as a dependency (empty if user-requested)
}

// Conflict represents a version conflict between dependencies.
type Conflict struct {
	ModID        string
	RequiredBy   []string
	Versions     []string
	VersionExprs []string
}

// ResolutionResult contains the result of dependency resolution.
type ResolutionResult struct {
	ResolvedMods []*ResolvedMod // Mods in topologically sorted install order
	Conflicts    []Conflict     // Detected conflicts
	MissingDeps  []Dependency   // Dependencies that could not be resolved
	InstallOrder []string       // Mod IDs in installation order
}

// VersionRange represents a parsed version constraint.
type VersionRange struct {
	MinVersion   *Version
	MaxVersion   *Version
	MinInclusive bool
	MaxInclusive bool
	ExactVersion *Version
	AnyVersion   bool
}

// Version represents a parsed semantic version.
type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Raw        string
}

// ParseVersion parses a version string into a Version struct.
func ParseVersion(versionStr string) (*Version, error) {
	if versionStr == "" {
		return nil, fmt.Errorf("empty version string")
	}

	v := &Version{Raw: versionStr}

	// Remove common prefixes
	cleaned := strings.TrimPrefix(versionStr, "v")
	cleaned = strings.TrimPrefix(cleaned, "V")

	// Handle prerelease suffix
	if idx := strings.IndexAny(cleaned, "-+"); idx != -1 {
		v.Prerelease = cleaned[idx:]
		cleaned = cleaned[:idx]
	}

	parts := strings.Split(cleaned, ".")
	if len(parts) < 1 {
		return nil, fmt.Errorf("invalid version format: %s", versionStr)
	}

	// Parse major
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid major version: %s", parts[0])
	}
	v.Major = major

	// Parse minor (optional)
	if len(parts) > 1 {
		minor, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid minor version: %s", parts[1])
		}
		v.Minor = minor
	}

	// Parse patch (optional)
	if len(parts) > 2 {
		patch, err := strconv.Atoi(parts[2])
		if err != nil {
			return nil, fmt.Errorf("invalid patch version: %s", parts[2])
		}
		v.Patch = patch
	}

	return v, nil
}

// Compare compares two versions. Returns -1 if v < other, 0 if v == other, 1 if v > other.
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

	// Handle prerelease comparison
	if v.Prerelease == "" && other.Prerelease != "" {
		return 1 // No prerelease > prerelease
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return -1 // Prerelease < no prerelease
	}
	if v.Prerelease < other.Prerelease {
		return -1
	}
	if v.Prerelease > other.Prerelease {
		return 1
	}

	return 0
}

// String returns the version as a string.
func (v *Version) String() string {
	if v.Raw != "" {
		return v.Raw
	}
	result := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		result += v.Prerelease
	}
	return result
}

// ParseVersionRange parses a version expression into a VersionRange.
// Supported formats:
// - "*" or empty: any version
// - "1.0.0": exact version
// - ">=1.0.0": greater than or equal
// - ">1.0.0": greater than
// - "<=1.0.0": less than or equal
// - "<1.0.0": less than
// - "1.0.0-2.0.0": range (inclusive)
// - ">=1.0.0,<2.0.0": combined constraints
func ParseVersionRange(expr string) (*VersionRange, error) {
	expr = strings.TrimSpace(expr)

	if expr == "" || expr == "*" {
		return &VersionRange{AnyVersion: true}, nil
	}

	vr := &VersionRange{}

	// Handle combined constraints with comma
	if strings.Contains(expr, ",") {
		parts := strings.Split(expr, ",")
		for _, part := range parts {
			subRange, err := ParseVersionRange(strings.TrimSpace(part))
			if err != nil {
				return nil, err
			}
			if subRange.MinVersion != nil {
				vr.MinVersion = subRange.MinVersion
				vr.MinInclusive = subRange.MinInclusive
			}
			if subRange.MaxVersion != nil {
				vr.MaxVersion = subRange.MaxVersion
				vr.MaxInclusive = subRange.MaxInclusive
			}
		}
		return vr, nil
	}

	// Handle range with hyphen: "1.0.0-2.0.0"
	if matched, _ := regexp.MatchString(`^\d+\.\d+\.\d+-\d+\.\d+\.\d+$`, expr); matched {
		parts := strings.SplitN(expr, "-", 2)
		if len(parts) == 2 {
			minV, err := ParseVersion(parts[0])
			if err != nil {
				return nil, err
			}
			maxV, err := ParseVersion(parts[1])
			if err != nil {
				return nil, err
			}
			vr.MinVersion = minV
			vr.MaxVersion = maxV
			vr.MinInclusive = true
			vr.MaxInclusive = true
			return vr, nil
		}
	}

	// Handle comparison operators
	if strings.HasPrefix(expr, ">=") {
		v, err := ParseVersion(strings.TrimPrefix(expr, ">="))
		if err != nil {
			return nil, err
		}
		vr.MinVersion = v
		vr.MinInclusive = true
		return vr, nil
	}

	if strings.HasPrefix(expr, ">") {
		v, err := ParseVersion(strings.TrimPrefix(expr, ">"))
		if err != nil {
			return nil, err
		}
		vr.MinVersion = v
		vr.MinInclusive = false
		return vr, nil
	}

	if strings.HasPrefix(expr, "<=") {
		v, err := ParseVersion(strings.TrimPrefix(expr, "<="))
		if err != nil {
			return nil, err
		}
		vr.MaxVersion = v
		vr.MaxInclusive = true
		return vr, nil
	}

	if strings.HasPrefix(expr, "<") {
		v, err := ParseVersion(strings.TrimPrefix(expr, "<"))
		if err != nil {
			return nil, err
		}
		vr.MaxVersion = v
		vr.MaxInclusive = false
		return vr, nil
	}

	// Exact version
	v, err := ParseVersion(expr)
	if err != nil {
		return nil, err
	}
	vr.ExactVersion = v
	return vr, nil
}

// Matches checks if a version satisfies the version range.
func (vr *VersionRange) Matches(v *Version) bool {
	if vr.AnyVersion {
		return true
	}

	if vr.ExactVersion != nil {
		return v.Compare(vr.ExactVersion) == 0
	}

	if vr.MinVersion != nil {
		cmp := v.Compare(vr.MinVersion)
		if vr.MinInclusive {
			if cmp < 0 {
				return false
			}
		} else {
			if cmp <= 0 {
				return false
			}
		}
	}

	if vr.MaxVersion != nil {
		cmp := v.Compare(vr.MaxVersion)
		if vr.MaxInclusive {
			if cmp > 0 {
				return false
			}
		} else {
			if cmp >= 0 {
				return false
			}
		}
	}

	return true
}

// String returns the version range as a string.
func (vr *VersionRange) String() string {
	if vr.AnyVersion {
		return "*"
	}

	if vr.ExactVersion != nil {
		return vr.ExactVersion.String()
	}

	var parts []string

	if vr.MinVersion != nil {
		op := ">"
		if vr.MinInclusive {
			op = ">="
		}
		parts = append(parts, op+vr.MinVersion.String())
	}

	if vr.MaxVersion != nil {
		op := "<"
		if vr.MaxInclusive {
			op = "<="
		}
		parts = append(parts, op+vr.MaxVersion.String())
	}

	return strings.Join(parts, ",")
}
