package preserve

import (
	"fmt"
	"strings"

	"github.com/alexinslc/chunk/internal/config"
)

type VersionDiffer struct{}

func NewVersionDiffer() *VersionDiffer {
	return &VersionDiffer{}
}

type ModpackDiff struct {
	MCVersionChange    *VersionChange
	LoaderChange       *LoaderChange
	ModsAdded          []ModInfo
	ModsRemoved        []ModInfo
	ModsUpdated        []ModUpdate
	ModsUnchanged      []ModInfo
	HasBreakingChanges bool
	Recommendations    []string
}

type VersionChange struct {
	From string
	To   string
}

type LoaderChange struct {
	TypeFrom    string
	TypeTo      string
	VersionFrom string
	VersionTo   string
}

type ModInfo struct {
	Name    string
	Version string
	ID      string
}

type ModUpdate struct {
	Name        string
	VersionFrom string
	VersionTo   string
	ID          string
	IsBreaking  bool
}

func (d *VersionDiffer) CompareMCVersions(from, to string) *VersionChange {
	if from == to {
		return nil
	}
	return &VersionChange{From: from, To: to}
}

func (d *VersionDiffer) CompareLoaders(fromType, fromVersion, toType, toVersion string) *LoaderChange {
	if fromType == toType && fromVersion == toVersion {
		return nil
	}

	return &LoaderChange{
		TypeFrom:    fromType,
		TypeTo:      toType,
		VersionFrom: fromVersion,
		VersionTo:   toVersion,
	}
}

func (d *VersionDiffer) CompareModpacks(oldManifest, newManifest *config.ChunkManifest) *ModpackDiff {
	diff := &ModpackDiff{
		Recommendations: []string{},
	}

	diff.MCVersionChange = d.CompareMCVersions(oldManifest.MCVersion, newManifest.MCVersion)
	diff.LoaderChange = d.CompareLoaders(
		oldManifest.Loader,
		oldManifest.LoaderVersion,
		newManifest.Loader,
		newManifest.LoaderVersion,
	)

	oldMods := make(map[string]ModInfo)
	// TODO: Implement mod comparison once ChunkManifest has Mods field
	// for _, mod := range oldManifest.Mods {
	// 	oldMods[mod.ID] = ModInfo{
	// 		Name:    mod.Name,
	// 		Version: mod.Version,
	// 		ID:      mod.ID,
	// 	}
	// }

	newMods := make(map[string]ModInfo)
	// TODO: Implement mod comparison once ChunkManifest has Mods field
	// for _, mod := range newManifest.Mods {
	// 	newMods[mod.ID] = ModInfo{
	// 		Name:    mod.Name,
	// 		Version: mod.Version,
	// 		ID:      mod.ID,
	// 	}
	// }

	for id, newMod := range newMods {
		if oldMod, exists := oldMods[id]; exists {
			if oldMod.Version != newMod.Version {
				isBreaking := d.isBreakingModUpdate(oldMod.Version, newMod.Version)
				diff.ModsUpdated = append(diff.ModsUpdated, ModUpdate{
					Name:        newMod.Name,
					VersionFrom: oldMod.Version,
					VersionTo:   newMod.Version,
					ID:          id,
					IsBreaking:  isBreaking,
				})
				if isBreaking {
					diff.HasBreakingChanges = true
				}
			} else {
				diff.ModsUnchanged = append(diff.ModsUnchanged, newMod)
			}
		} else {
			diff.ModsAdded = append(diff.ModsAdded, newMod)
		}
	}

	for id, oldMod := range oldMods {
		if _, exists := newMods[id]; !exists {
			diff.ModsRemoved = append(diff.ModsRemoved, oldMod)
		}
	}

	d.generateRecommendations(diff)

	return diff
}

func (d *VersionDiffer) isBreakingModUpdate(from, to string) bool {
	fromParts := strings.Split(from, ".")
	toParts := strings.Split(to, ".")

	if len(fromParts) < 2 || len(toParts) < 2 {
		return true
	}

	if fromParts[0] != toParts[0] {
		return true
	}

	return false
}

func (d *VersionDiffer) generateRecommendations(diff *ModpackDiff) {
	if diff.MCVersionChange != nil {
		if d.isMajorMCVersionChange(diff.MCVersionChange.From, diff.MCVersionChange.To) {
			diff.Recommendations = append(diff.Recommendations,
				"⚠️  Major Minecraft version change detected. World backup is strongly recommended.")
		}
	}

	if diff.LoaderChange != nil && diff.LoaderChange.TypeFrom != diff.LoaderChange.TypeTo {
		diff.Recommendations = append(diff.Recommendations,
			"⚠️  Mod loader type is changing. This requires a fresh installation and may not be compatible with existing worlds.")
		diff.HasBreakingChanges = true
	}

	if len(diff.ModsRemoved) > 0 {
		diff.Recommendations = append(diff.Recommendations,
			fmt.Sprintf("ℹ️  %d mod(s) will be removed. Ensure they are not critical to your world.", len(diff.ModsRemoved)))
	}

	if diff.HasBreakingChanges {
		diff.Recommendations = append(diff.Recommendations,
			"⚠️  Breaking changes detected. Test in a backup world first.")
	}

	if len(diff.ModsAdded) > 10 {
		diff.Recommendations = append(diff.Recommendations,
			"ℹ️  Many new mods are being added. First launch may take longer than usual.")
	}
}

func (d *VersionDiffer) isMajorMCVersionChange(from, to string) bool {
	fromParts := strings.Split(from, ".")
	toParts := strings.Split(to, ".")

	if len(fromParts) < 2 || len(toParts) < 2 {
		return true
	}

	return fromParts[0] != toParts[0] || fromParts[1] != toParts[1]
}

func (d *VersionDiffer) PrintDiff(diff *ModpackDiff) {
	fmt.Println("\n📊 Modpack Difference Report")
	fmt.Println(strings.Repeat("=", 50))

	if diff.MCVersionChange != nil {
		fmt.Printf("\n🎮 Minecraft Version:\n")
		fmt.Printf("   %s → %s\n", diff.MCVersionChange.From, diff.MCVersionChange.To)
	}

	if diff.LoaderChange != nil {
		fmt.Printf("\n🔧 Mod Loader:\n")
		if diff.LoaderChange.TypeFrom != diff.LoaderChange.TypeTo {
			fmt.Printf("   Type: %s → %s\n", diff.LoaderChange.TypeFrom, diff.LoaderChange.TypeTo)
		}
		if diff.LoaderChange.VersionFrom != diff.LoaderChange.VersionTo {
			fmt.Printf("   Version: %s → %s\n", diff.LoaderChange.VersionFrom, diff.LoaderChange.VersionTo)
		}
	}

	if len(diff.ModsAdded) > 0 {
		fmt.Printf("\n➕ Added Mods (%d):\n", len(diff.ModsAdded))
		for _, mod := range diff.ModsAdded {
			fmt.Printf("   + %s (%s)\n", mod.Name, mod.Version)
		}
	}

	if len(diff.ModsRemoved) > 0 {
		fmt.Printf("\n➖ Removed Mods (%d):\n", len(diff.ModsRemoved))
		for _, mod := range diff.ModsRemoved {
			fmt.Printf("   - %s (%s)\n", mod.Name, mod.Version)
		}
	}

	if len(diff.ModsUpdated) > 0 {
		fmt.Printf("\n🔄 Updated Mods (%d):\n", len(diff.ModsUpdated))
		for _, mod := range diff.ModsUpdated {
			icon := "↑"
			if mod.IsBreaking {
				icon = "⚠️"
			}
			fmt.Printf("   %s %s: %s → %s\n", icon, mod.Name, mod.VersionFrom, mod.VersionTo)
		}
	}

	if len(diff.ModsUnchanged) > 0 {
		fmt.Printf("\n✓ Unchanged Mods: %d\n", len(diff.ModsUnchanged))
	}

	if len(diff.Recommendations) > 0 {
		fmt.Printf("\n💡 Recommendations:\n")
		for _, rec := range diff.Recommendations {
			fmt.Printf("   %s\n", rec)
		}
	}

	fmt.Println(strings.Repeat("=", 50))
}

func (d *VersionDiffer) GetKnownWorkingVersions(loader, mcVersion string) []string {
	knownVersions := map[string]map[string][]string{
		"forge": {
			"1.20.1": {"47.2.0", "47.1.0", "47.0.0"},
			"1.19.2": {"43.2.0", "43.1.0"},
			"1.18.2": {"40.2.0", "40.1.0"},
		},
		"fabric": {
			"1.20.1": {"0.15.0", "0.14.24"},
			"1.19.2": {"0.14.21", "0.14.19"},
			"1.18.2": {"0.14.6", "0.14.5"},
		},
		"neoforge": {
			"1.20.5": {"20.5.14", "20.5.0"},
			"1.20.4": {"20.4.237", "20.4.200"},
		},
	}

	if loaderVersions, ok := knownVersions[strings.ToLower(loader)]; ok {
		if versions, ok := loaderVersions[mcVersion]; ok {
			return versions
		}
	}

	return []string{}
}
