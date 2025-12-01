package converter

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexinslc/chunk/internal/sources"
)

type ConversionEngine struct {
	loaderInstaller *LoaderInstaller
	modManager      *ModManager
	configGenerator *ConfigGenerator
	scriptGenerator *ScriptGenerator
}

func NewConversionEngine() *ConversionEngine {
	return &ConversionEngine{
		loaderInstaller: NewLoaderInstaller(),
		modManager:      NewModManager(),
		configGenerator: NewConfigGenerator(),
		scriptGenerator: NewScriptGenerator(),
	}
}

type ConversionOptions struct {
	DestDir        string
	ModpackName    string
	MCVersion      string
	Loader         sources.LoaderType
	LoaderVersion  string
	RecommendedRAM int
	PreserveData   bool
}

func (e *ConversionEngine) Convert(modpack *sources.Modpack, destDir string) error {
	opts := &ConversionOptions{
		DestDir:        destDir,
		ModpackName:    modpack.Name,
		MCVersion:      modpack.MCVersion,
		Loader:         modpack.Loader,
		LoaderVersion:  modpack.LoaderVersion,
		RecommendedRAM: modpack.RecommendedRAM,
		PreserveData:   false,
	}

	return e.ConvertWithOptions(modpack, opts)
}

func (e *ConversionEngine) ConvertWithOptions(modpack *sources.Modpack, opts *ConversionOptions) error {
	if err := e.validateModpack(modpack); err != nil {
		return fmt.Errorf("invalid modpack: %w", err)
	}

	if err := e.prepareDirectory(opts.DestDir, opts.PreserveData); err != nil {
		return fmt.Errorf("failed to prepare directory: %w", err)
	}

	if err := e.loaderInstaller.Install(opts); err != nil {
		return fmt.Errorf("failed to install mod loader: %w", err)
	}

	serverMods := e.filterServerMods(modpack.Mods)

	if err := e.modManager.DownloadMods(serverMods, opts.DestDir); err != nil {
		return fmt.Errorf("failed to download mods: %w", err)
	}

	if err := e.configGenerator.Generate(opts); err != nil {
		return fmt.Errorf("failed to generate configs: %w", err)
	}

	if err := e.scriptGenerator.Generate(opts); err != nil {
		return fmt.Errorf("failed to generate start scripts: %w", err)
	}

	return nil
}

func (e *ConversionEngine) validateModpack(modpack *sources.Modpack) error {
	if modpack.MCVersion == "" {
		return fmt.Errorf("minecraft version not specified")
	}

	if modpack.Loader == "" {
		return fmt.Errorf("mod loader not specified")
	}

	if modpack.Loader != sources.LoaderForge &&
		modpack.Loader != sources.LoaderFabric &&
		modpack.Loader != sources.LoaderNeoForge {
		return fmt.Errorf("unsupported mod loader: %s", modpack.Loader)
	}

	return nil
}

func (e *ConversionEngine) prepareDirectory(destDir string, preserveData bool) error {
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}
	} else if !preserveData {
		modsDir := filepath.Join(destDir, "mods")
		if err := os.RemoveAll(modsDir); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	requiredDirs := []string{"mods", "config", "logs"}
	for _, dir := range requiredDirs {
		path := filepath.Join(destDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (e *ConversionEngine) filterServerMods(mods []*sources.Mod) []*sources.Mod {
	var serverMods []*sources.Mod

	for _, mod := range mods {
		if mod.Side == sources.SideClient {
			continue
		}
		serverMods = append(serverMods, mod)
	}

	return serverMods
}
