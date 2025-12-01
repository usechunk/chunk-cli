package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type ChunkManifest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	MCVersion        string   `json:"mc_version"`
	Loader           string   `json:"loader"`
	LoaderVersion    string   `json:"loader_version,omitempty"`
	RecommendedRAMGB int      `json:"recommended_ram_gb"`
	Dependencies     []string `json:"dependencies,omitempty"`
	JavaVersion      int      `json:"java_version,omitempty"`
}

func LoadChunkManifest(path string) (*ChunkManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read .chunk.json: %w", err)
	}

	var manifest ChunkManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("invalid .chunk.json: %w", err)
	}

	if err := validateChunkManifest(&manifest); err != nil {
		return nil, err
	}

	return &manifest, nil
}

func validateChunkManifest(manifest *ChunkManifest) error {
	if manifest.Name == "" {
		return fmt.Errorf("name is required")
	}

	if manifest.MCVersion == "" {
		return fmt.Errorf("mc_version is required")
	}

	if manifest.Loader == "" {
		return fmt.Errorf("loader is required")
	}

	validLoaders := map[string]bool{
		"forge":    true,
		"fabric":   true,
		"neoforge": true,
	}

	if !validLoaders[manifest.Loader] {
		return fmt.Errorf("invalid loader: %s (must be forge, fabric, or neoforge)", manifest.Loader)
	}

	if manifest.RecommendedRAMGB < 0 {
		return fmt.Errorf("recommended_ram_gb must be positive")
	}

	if manifest.JavaVersion != 0 && (manifest.JavaVersion < 8 || manifest.JavaVersion > 21) {
		return fmt.Errorf("java_version must be between 8 and 21")
	}

	return nil
}

func SaveChunkManifest(path string, manifest *ChunkManifest) error {
	if err := validateChunkManifest(manifest); err != nil {
		return err
	}

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write .chunk.json: %w", err)
	}

	return nil
}
