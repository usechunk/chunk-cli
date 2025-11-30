package dependencies

import (
	"encoding/json"
	"strings"
)

// systemDependencies are mod IDs that represent system dependencies
// (loader, minecraft, java) which should be skipped during dependency resolution.
var systemDependencies = map[string]bool{
	"fabricloader": true,
	"fabric":       true,
	"minecraft":    true,
	"java":         true,
	"forge":        true,
	"neoforge":     true,
}

// ModrinthDependency represents a dependency in Modrinth format.
type ModrinthDependency struct {
	VersionID      string `json:"version_id"`
	ProjectID      string `json:"project_id"`
	FileName       string `json:"file_name"`
	DependencyType string `json:"dependency_type"` // "required", "optional", "incompatible", "embedded"
}

// ModrinthVersion represents version metadata from Modrinth.
type ModrinthVersion struct {
	ID            string               `json:"id"`
	ProjectID     string               `json:"project_id"`
	Name          string               `json:"name"`
	VersionNumber string               `json:"version_number"`
	GameVersions  []string             `json:"game_versions"`
	Loaders       []string             `json:"loaders"`
	Dependencies  []ModrinthDependency `json:"dependencies"`
	Files         []struct {
		URL      string `json:"url"`
		Filename string `json:"filename"`
	} `json:"files"`
}

// FabricModJSON represents the fabric.mod.json format.
type FabricModJSON struct {
	ID          string            `json:"id"`
	Version     string            `json:"version"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Depends     map[string]string `json:"depends"`
	Recommends  map[string]string `json:"recommends"`
	Suggests    map[string]string `json:"suggests"`
	Breaks      map[string]string `json:"breaks"`
	Conflicts   map[string]string `json:"conflicts"`
	Provides    []string          `json:"provides"`
}

// ForgeModToml represents the mods.toml format for Forge/NeoForge.
type ForgeModToml struct {
	ModLoader     string `toml:"modLoader"`
	LoaderVersion string `toml:"loaderVersion"`
	Mods          []struct {
		ModID       string `toml:"modId"`
		Version     string `toml:"version"`
		DisplayName string `toml:"displayName"`
		Description string `toml:"description"`
	} `toml:"mods"`
	Dependencies map[string][]struct {
		ModID        string `toml:"modId"`
		Mandatory    bool   `toml:"mandatory"`
		VersionRange string `toml:"versionRange"`
		Ordering     string `toml:"ordering"`
		Side         string `toml:"side"`
	} `toml:"dependencies"`
}

// ParseModrinthDependencies parses dependencies from Modrinth version metadata.
func ParseModrinthDependencies(versionData []byte) (*ModInfo, error) {
	var version ModrinthVersion
	if err := json.Unmarshal(versionData, &version); err != nil {
		return nil, err
	}

	mod := &ModInfo{
		ID:      version.ProjectID,
		Name:    version.Name,
		Version: version.VersionNumber,
	}

	for _, dep := range version.Dependencies {
		depInfo := Dependency{
			ModID:    dep.ProjectID,
			Required: dep.DependencyType == "required",
		}

		if dep.VersionID != "" {
			depInfo.VersionExpr = dep.VersionID
		} else {
			depInfo.VersionExpr = "*"
		}

		mod.Dependencies = append(mod.Dependencies, depInfo)
	}

	return mod, nil
}

// ParseFabricModJSON parses dependencies from a fabric.mod.json file.
func ParseFabricModJSON(data []byte) (*ModInfo, error) {
	var fabric FabricModJSON
	if err := json.Unmarshal(data, &fabric); err != nil {
		return nil, err
	}

	mod := &ModInfo{
		ID:       fabric.ID,
		Name:     fabric.Name,
		Version:  fabric.Version,
		Provides: fabric.Provides,
	}

	// Parse required dependencies
	for modID, versionExpr := range fabric.Depends {
		// Skip system dependencies
		if systemDependencies[modID] {
			continue
		}

		mod.Dependencies = append(mod.Dependencies, Dependency{
			ModID:       modID,
			VersionExpr: normalizeFabricVersion(versionExpr),
			Required:    true,
		})
	}

	// Parse recommended dependencies (optional)
	for modID, versionExpr := range fabric.Recommends {
		mod.Dependencies = append(mod.Dependencies, Dependency{
			ModID:       modID,
			VersionExpr: normalizeFabricVersion(versionExpr),
			Required:    false,
		})
	}

	return mod, nil
}

// normalizeFabricVersion converts Fabric version expressions to our format.
func normalizeFabricVersion(expr string) string {
	expr = strings.TrimSpace(expr)

	// Handle wildcard
	if expr == "*" {
		return "*"
	}

	// Handle Fabric's range format: >=x.y.z <a.b.c (must check this first)
	if strings.Contains(expr, " ") {
		parts := strings.Fields(expr)
		if len(parts) == 2 {
			return parts[0] + "," + parts[1]
		}
	}

	// Fabric uses >=x.y.z format which we support
	if strings.HasPrefix(expr, ">=") || strings.HasPrefix(expr, ">") ||
		strings.HasPrefix(expr, "<=") || strings.HasPrefix(expr, "<") {
		return expr
	}

	// Assume exact version
	return expr
}

// ParseForgeModDependencies parses dependencies for a Forge mod.
// Takes the mod ID and the dependency declarations.
func ParseForgeModDependencies(modID string, deps []struct {
	ModID        string
	Mandatory    bool
	VersionRange string
	Ordering     string
	Side         string
}) []Dependency {
	var dependencies []Dependency

	for _, dep := range deps {
		// Skip system dependencies
		if systemDependencies[dep.ModID] {
			continue
		}

		dependencies = append(dependencies, Dependency{
			ModID:       dep.ModID,
			VersionExpr: normalizeForgeVersion(dep.VersionRange),
			Required:    dep.Mandatory,
			Side:        dep.Side,
		})
	}

	return dependencies
}

// normalizeForgeVersion converts Forge version ranges to our format.
// Forge uses Maven version range format: [1.0,2.0) means >=1.0,<2.0
func normalizeForgeVersion(expr string) string {
	expr = strings.TrimSpace(expr)

	if expr == "" {
		return "*"
	}

	// Handle Maven-style ranges
	if strings.HasPrefix(expr, "[") || strings.HasPrefix(expr, "(") {
		return parseMavenVersionRange(expr)
	}

	return expr
}

// parseMavenVersionRange converts Maven version range format to our format.
func parseMavenVersionRange(expr string) string {
	// Remove whitespace
	expr = strings.ReplaceAll(expr, " ", "")

	if len(expr) < 2 {
		return "*"
	}

	minInclusive := expr[0] == '['
	maxInclusive := expr[len(expr)-1] == ']'

	// Remove brackets
	inner := expr[1 : len(expr)-1]

	parts := strings.Split(inner, ",")
	if len(parts) != 2 {
		// Single version constraint
		if len(parts) == 1 && parts[0] != "" {
			return parts[0]
		}
		return "*"
	}

	var constraints []string

	// Min version
	if parts[0] != "" {
		op := ">"
		if minInclusive {
			op = ">="
		}
		constraints = append(constraints, op+parts[0])
	}

	// Max version
	if parts[1] != "" {
		op := "<"
		if maxInclusive {
			op = "<="
		}
		constraints = append(constraints, op+parts[1])
	}

	return strings.Join(constraints, ",")
}

// ExtractModIDFromFilename attempts to extract a mod ID from a JAR filename.
func ExtractModIDFromFilename(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, ".jar")

	// Common patterns: modid-version.jar, modid_version.jar
	for _, sep := range []string{"-", "_"} {
		if idx := strings.LastIndex(name, sep); idx != -1 {
			potential := name[:idx]
			// Check if what follows looks like a version
			rest := name[idx+1:]
			if looksLikeVersion(rest) {
				return potential
			}
		}
	}

	return name
}

// looksLikeVersion checks if a string looks like a version number.
func looksLikeVersion(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Check if it starts with a digit
	if s[0] >= '0' && s[0] <= '9' {
		return true
	}

	// Check for common version prefixes
	prefixes := []string{"v", "V", "mc", "MC"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) && len(s) > len(prefix) {
			return s[len(prefix)] >= '0' && s[len(prefix)] <= '9'
		}
	}

	return false
}
