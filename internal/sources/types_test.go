package sources

import (
	"testing"
)

func TestDetectSource(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		want       string
	}{
		{
			name:       "local file with dot prefix",
			identifier: "./modpack.mrpack",
			want:       "local",
		},
		{
			name:       "local file with slash prefix",
			identifier: "/path/to/modpack.mrpack",
			want:       "local",
		},
		{
			name:       "modrinth with prefix",
			identifier: "modrinth:some-modpack",
			want:       "modrinth",
		},
		{
			name:       "github repo",
			identifier: "alexinslc/my-modpack",
			want:       "github",
		},
		{
			name:       "recipe registry (default)",
			identifier: "atm9",
			want:       "recipe",
		},
		{
			name:       "another recipe",
			identifier: "create-above-and-beyond",
			want:       "recipe",
		},
		{
			name:       "explicit bench::recipe syntax",
			identifier: "usechunk/recipes::atm9",
			want:       "recipe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectSource(tt.identifier)
			if got != tt.want {
				t.Errorf("DetectSource(%q) = %q, want %q", tt.identifier, got, tt.want)
			}
		})
	}
}

func TestModpackStructure(t *testing.T) {
	modpack := &Modpack{
		Name:          "Test Modpack",
		Identifier:    "test-modpack",
		Description:   "A test modpack",
		MCVersion:     "1.20.1",
		Loader:        LoaderForge,
		LoaderVersion: "47.2.0",
		Author:        "testauthor",
		Source:        "chunkhub",
		Mods: []*Mod{
			{
				Name:        "JEI",
				Version:     "15.2.0.27",
				FileName:    "jei-1.20.1-15.2.0.27.jar",
				DownloadURL: "https://example.com/jei.jar",
				Side:        SideBoth,
				Required:    true,
			},
		},
		RecommendedRAM: 8,
	}

	if modpack.Name != "Test Modpack" {
		t.Errorf("Expected Name to be 'Test Modpack', got %q", modpack.Name)
	}
	if modpack.Loader != LoaderForge {
		t.Errorf("Expected Loader to be LoaderForge, got %q", modpack.Loader)
	}
	if len(modpack.Mods) != 1 {
		t.Errorf("Expected 1 mod, got %d", len(modpack.Mods))
	}
	if modpack.Mods[0].Side != SideBoth {
		t.Errorf("Expected mod side to be SideBoth, got %q", modpack.Mods[0].Side)
	}
}

func TestLoaderTypes(t *testing.T) {
	loaders := []LoaderType{LoaderForge, LoaderFabric, LoaderNeoForge}
	expected := []string{"forge", "fabric", "neoforge"}

	for i, loader := range loaders {
		if string(loader) != expected[i] {
			t.Errorf("Expected loader %q, got %q", expected[i], loader)
		}
	}
}

func TestModSides(t *testing.T) {
	sides := []ModSide{SideClient, SideServer, SideBoth}
	expected := []string{"client", "server", "both"}

	for i, side := range sides {
		if string(side) != expected[i] {
			t.Errorf("Expected side %q, got %q", expected[i], side)
		}
	}
}
