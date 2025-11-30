package sources

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type MRPackParser struct{}

func NewMRPackParser() *MRPackParser {
	return &MRPackParser{}
}

func (p *MRPackParser) Parse(filePath string) (*Modpack, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open mrpack: %w", err)
	}
	defer reader.Close()

	var manifestFile *zip.File
	for _, file := range reader.File {
		if file.Name == "modrinth.index.json" {
			manifestFile = file
			break
		}
	}

	if manifestFile == nil {
		return nil, fmt.Errorf("modrinth.index.json not found in mrpack")
	}

	rc, err := manifestFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	defer rc.Close()

	var manifest MRPackManifest
	if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("invalid modrinth.index.json: %w", err)
	}

	modpack := &Modpack{
		Name:        manifest.Name,
		Identifier:  filepath.Base(filePath),
		Description: manifest.Summary,
		MCVersion:   manifest.Dependencies.Minecraft,
		Source:      "local",
	}

	if manifest.Dependencies.Forge != "" {
		modpack.Loader = LoaderForge
		modpack.LoaderVersion = manifest.Dependencies.Forge
	} else if manifest.Dependencies.Fabric != "" {
		modpack.Loader = LoaderFabric
		modpack.LoaderVersion = manifest.Dependencies.FabricLoader
	} else if manifest.Dependencies.NeoForge != "" {
		modpack.Loader = LoaderNeoForge
		modpack.LoaderVersion = manifest.Dependencies.NeoForge
	}

	for _, file := range manifest.Files {
		var side ModSide = SideBoth
		if len(file.Env.Client) > 0 && file.Env.Client == "required" {
			if len(file.Env.Server) == 0 || file.Env.Server == "unsupported" {
				side = SideClient
			}
		}
		if len(file.Env.Server) > 0 && file.Env.Server == "required" {
			if len(file.Env.Client) == 0 || file.Env.Client == "unsupported" {
				side = SideServer
			}
		}

		mod := &Mod{
			FileName:    file.Path,
			DownloadURL: file.Downloads[0],
			Side:        side,
			Required:    true,
		}

		modpack.Mods = append(modpack.Mods, mod)
	}

	return modpack, nil
}

func (p *MRPackParser) Extract(filePath, destDir string) error {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return fmt.Errorf("failed to open mrpack: %w", err)
	}
	defer reader.Close()

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	for _, file := range reader.File {
		if file.Name == "modrinth.index.json" {
			continue
		}

		path := filepath.Join(destDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(path, file.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		outFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}

		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("failed to open file in archive: %w", err)
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()

		if err != nil {
			return fmt.Errorf("failed to extract file: %w", err)
		}
	}

	return nil
}

type MRPackManifest struct {
	FormatVersion int    `json:"formatVersion"`
	Game          string `json:"game"`
	VersionID     string `json:"versionId"`
	Name          string `json:"name"`
	Summary       string `json:"summary,omitempty"`
	Files         []struct {
		Path   string `json:"path"`
		Hashes struct {
			SHA1   string `json:"sha1"`
			SHA512 string `json:"sha512"`
		} `json:"hashes"`
		Env struct {
			Client string `json:"client"`
			Server string `json:"server"`
		} `json:"env"`
		Downloads []string `json:"downloads"`
		FileSize  int      `json:"fileSize"`
	} `json:"files"`
	Dependencies struct {
		Minecraft    string `json:"minecraft"`
		Forge        string `json:"forge,omitempty"`
		FabricLoader string `json:"fabric-loader,omitempty"`
		NeoForge     string `json:"neoforge,omitempty"`
		Fabric       string `json:"fabric,omitempty"`
	} `json:"dependencies"`
}
