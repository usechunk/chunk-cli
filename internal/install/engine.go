package install

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexinslc/chunk/internal/bench"
	"github.com/alexinslc/chunk/internal/config"
	"github.com/alexinslc/chunk/internal/converter"
	"github.com/alexinslc/chunk/internal/search"
	"github.com/alexinslc/chunk/internal/sources"
	"github.com/alexinslc/chunk/internal/tracking"
	"github.com/alexinslc/chunk/internal/ui"
)

// Installer handles the complete installation workflow for modpacks
type Installer struct {
	sourceManager    *sources.SourceManager
	conversionEngine *converter.ConversionEngine
	httpClient       *http.Client
	backupDir        string
	absDestDir       string
	skipVerify       bool
}

// NewInstaller creates a new Installer instance
func NewInstaller() *Installer {
	return &Installer{
		sourceManager:    sources.NewSourceManager(),
		conversionEngine: converter.NewConversionEngine(),
		httpClient: &http.Client{
			Timeout: 10 * time.Minute,
		},
	}
}

// Options contains configuration for the installation
type Options struct {
	Identifier   string
	DestDir      string
	PreserveData bool
	SkipVerify   bool
}

// Result contains the outcome of an installation
type Result struct {
	ModpackName   string
	MCVersion     string
	Loader        sources.LoaderType
	LoaderVersion string
	ModsInstalled int
	DestDir       string
	ModpackInfo   *ModpackDisplayInfo
	Modpack       *sources.Modpack // Full modpack info for tracking
}

// ModpackDisplayInfo contains modpack details for display
type ModpackDisplayInfo struct {
	Name           string
	Description    string
	MCVersion      string
	Loader         sources.LoaderType
	LoaderVersion  string
	Author         string
	Source         string
	ModCount       int
	RecommendedRAM int
}

// Install performs the complete installation workflow
func (i *Installer) Install(opts *Options) (*Result, error) {
	// Normalize destination directory
	destDir := opts.DestDir
	if destDir == "" {
		destDir = "./server"
	}

	// Make destination path absolute
	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve destination path: %w", err)
	}

	// Store options for later use
	i.absDestDir = absDestDir
	i.skipVerify = opts.SkipVerify

	ui.PrintInfo(fmt.Sprintf("Installing to: %s", absDestDir))

	// Detect source type
	sourceType := sources.DetectSource(opts.Identifier)
	ui.PrintInfo(fmt.Sprintf("Source: %s", sourceType))

	// Fetch modpack metadata
	spinner := ui.NewSpinner("Fetching modpack information...")
	spinner.Start()

	modpack, err := i.fetchModpack(opts.Identifier)
	if err != nil {
		spinner.Error(fmt.Sprintf("Failed to fetch modpack: %v", err))
		return nil, fmt.Errorf("failed to fetch modpack: %w", err)
	}
	spinner.Success(fmt.Sprintf("Found modpack: %s", modpack.Name))

	// Build modpack display info for the command layer to display
	modpackInfo := &ModpackDisplayInfo{
		Name:           modpack.Name,
		Description:    modpack.Description,
		MCVersion:      modpack.MCVersion,
		Loader:         modpack.Loader,
		LoaderVersion:  modpack.LoaderVersion,
		Author:         modpack.Author,
		Source:         modpack.Source,
		ModCount:       len(modpack.Mods),
		RecommendedRAM: modpack.RecommendedRAM,
	}

	// Create backup if directory exists and has content
	if err := i.createBackup(absDestDir); err != nil {
		ui.PrintWarning(fmt.Sprintf("Could not create backup: %v", err))
	}

	// Prepare installation directory
	spinner = ui.NewSpinner("Preparing installation directory...")
	spinner.Start()
	if err := i.prepareDirectory(absDestDir, opts.PreserveData); err != nil {
		spinner.Error(fmt.Sprintf("Failed to prepare directory: %v", err))
		return nil, fmt.Errorf("failed to prepare directory: %w", err)
	}
	spinner.Success("Directory prepared")

	// For recipes, download and extract the modpack
	if sourceType == "recipe" {
		spinner = ui.NewSpinner("Downloading modpack from recipe...")
		spinner.Start()
		if err := i.downloadAndExtractRecipe(opts.Identifier, modpack, absDestDir); err != nil {
			spinner.Error(fmt.Sprintf("Failed to download modpack: %v", err))
			return nil, fmt.Errorf("failed to download modpack: %w", err)
		}
		spinner.Success("Modpack downloaded and extracted")
	}

	// For local files, extract them first
	if sourceType == "local" {
		spinner = ui.NewSpinner("Extracting modpack files...")
		spinner.Start()
		if err := i.extractLocalModpack(opts.Identifier, absDestDir); err != nil {
			spinner.Error(fmt.Sprintf("Failed to extract modpack: %v", err))
			return nil, fmt.Errorf("failed to extract modpack: %w", err)
		}
		spinner.Success("Modpack files extracted")
	}

	// Install mod loader
	spinner = ui.NewSpinner(fmt.Sprintf("Installing %s loader...", modpack.Loader))
	spinner.Start()
	if err := i.installLoader(modpack, absDestDir); err != nil {
		spinner.Error(fmt.Sprintf("Failed to install loader: %v", err))
		return nil, fmt.Errorf("failed to install mod loader: %w", err)
	}
	spinner.Success(fmt.Sprintf("%s loader installed", modpack.Loader))

	// Download mods
	modsInstalled := 0
	if len(modpack.Mods) > 0 {
		ui.PrintInfo(fmt.Sprintf("Downloading %d mods (filtering server-side only)...", len(modpack.Mods)))
		installed, err := i.downloadMods(modpack.Mods, absDestDir)
		if err != nil {
			return nil, fmt.Errorf("failed to download mods: %w", err)
		}
		modsInstalled = installed
		ui.PrintSuccess(fmt.Sprintf("Downloaded %d server-side mods", modsInstalled))
	} else {
		ui.PrintInfo("No mods to download")
	}

	// Generate configuration files
	spinner = ui.NewSpinner("Generating server configuration...")
	spinner.Start()
	if err := i.generateConfigs(modpack, absDestDir); err != nil {
		spinner.Error(fmt.Sprintf("Failed to generate configs: %v", err))
		return nil, fmt.Errorf("failed to generate configs: %w", err)
	}
	spinner.Success("Server configuration generated")

	// Generate start scripts
	spinner = ui.NewSpinner("Creating start scripts...")
	spinner.Start()
	if err := i.generateScripts(modpack, absDestDir); err != nil {
		spinner.Error(fmt.Sprintf("Failed to generate scripts: %v", err))
		return nil, fmt.Errorf("failed to generate scripts: %w", err)
	}
	spinner.Success("Start scripts created")

	// Clean up backup if successful
	if i.backupDir != "" {
		if err := os.RemoveAll(i.backupDir); err != nil {
			ui.PrintWarning(fmt.Sprintf("Failed to clean up backup: %v", err))
		}
	}

	return &Result{
		ModpackName:   modpack.Name,
		MCVersion:     modpack.MCVersion,
		Loader:        modpack.Loader,
		LoaderVersion: modpack.LoaderVersion,
		ModsInstalled: modsInstalled,
		DestDir:       absDestDir,
		ModpackInfo:   modpackInfo,
		Modpack:       modpack,
	}, nil
}

// Rollback restores the previous state after a failed installation
func (i *Installer) Rollback() error {
	if i.backupDir == "" {
		return nil
	}

	// Use the stored absolute path to ensure correct rollback
	destDir := i.absDestDir
	if destDir == "" {
		return nil
	}

	ui.PrintWarning("Rolling back installation...")

	// Remove failed installation
	if err := os.RemoveAll(destDir); err != nil {
		return fmt.Errorf("failed to remove failed installation: %w", err)
	}

	// Restore backup
	if err := os.Rename(i.backupDir, destDir); err != nil {
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	ui.PrintSuccess("Rollback complete")
	return nil
}

func (i *Installer) fetchModpack(identifier string) (*sources.Modpack, error) {
	return i.sourceManager.Fetch(identifier)
}

func (i *Installer) createBackup(destDir string) error {
	if _, err := os.Stat(destDir); os.IsNotExist(err) {
		return nil // Nothing to back up
	}

	// Check if directory has content
	entries, err := os.ReadDir(destDir)
	if err != nil || len(entries) == 0 {
		return nil
	}

	// Create backup directory
	i.backupDir = fmt.Sprintf("%s.backup.%d", destDir, time.Now().Unix())
	return os.Rename(destDir, i.backupDir)
}

func (i *Installer) prepareDirectory(destDir string, preserveData bool) error {
	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}

	// Create required subdirectories
	dirs := []string{"mods", "config", "logs"}
	for _, dir := range dirs {
		path := filepath.Join(destDir, dir)
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	return nil
}

func (i *Installer) extractLocalModpack(filePath, destDir string) error {
	localClient := sources.NewLocalClient()
	return localClient.Extract(filePath, destDir)
}

func (i *Installer) downloadAndExtractRecipe(identifier string, modpack *sources.Modpack, destDir string) error {
	// Get the recipe client
	recipeClient := sources.NewRecipeClient()

	// Parse identifier to get recipe info
	benchName := ""
	recipeName := identifier
	if strings.Contains(identifier, "::") {
		parts := strings.SplitN(identifier, "::", 2)
		if len(parts) == 2 {
			benchName = parts[0]
			recipeName = parts[1]
		}
	}

	// Find the recipe to get checksum
	recipe, err := i.findRecipe(recipeName, benchName)
	if err != nil {
		return fmt.Errorf("failed to find recipe: %w", err)
	}

	// Create a temp file for the download
	tmpFile, err := os.CreateTemp("", "chunk-download-*.mrpack")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download with progress
	ui.PrintInfo(fmt.Sprintf("Downloading from: %s", modpack.ManifestURL))
	
	err = recipeClient.DownloadFile(modpack.ManifestURL, tmpFile, func(downloaded, total int64) {
		if total > 0 {
			percent := float64(downloaded) / float64(total) * 100
			fmt.Printf("\rProgress: %.1f%% (%d MB / %d MB)", percent, downloaded/(1024*1024), total/(1024*1024))
		}
	})
	fmt.Println() // New line after progress
	
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Verify checksum if not skipped
	if !i.skipVerify && recipe.SHA256 != "" {
		ui.PrintInfo("Verifying checksum...")
		if err := sources.VerifyChecksum(tmpFile.Name(), recipe.SHA256); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		ui.PrintSuccess("Checksum verified")
	} else if recipe.SHA256 == "" {
		ui.PrintWarning("No checksum provided in recipe, skipping verification")
	}

	// Extract the archive
	ui.PrintInfo("Extracting modpack...")
	if err := sources.ExtractArchive(tmpFile.Name(), destDir); err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Save recipe snapshot
	if err := sources.SaveRecipeSnapshot(recipe, destDir); err != nil {
		ui.PrintWarning(fmt.Sprintf("Failed to save recipe snapshot: %v", err))
		// Don't fail the installation if snapshot fails
	}

	return nil
}

func (i *Installer) findRecipe(recipeName string, benchFilter string) (*search.Recipe, error) {
	manager, err := bench.NewManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create bench manager: %w", err)
	}

	benches := manager.List()
	if len(benches) == 0 {
		return nil, fmt.Errorf("no benches installed")
	}

	// Filter to specific bench if requested
	var searchBenches []config.Bench
	if benchFilter != "" {
		for _, b := range benches {
			if b.Name == benchFilter {
				searchBenches = []config.Bench{b}
				break
			}
		}
		if len(searchBenches) == 0 {
			return nil, fmt.Errorf("bench '%s' not found", benchFilter)
		}
	} else {
		searchBenches = benches
	}

	// Search for recipe in benches
	for _, bench := range searchBenches {
		recipes, err := search.LoadRecipesFromBench(bench.Path, bench.Name)
		if err != nil {
			continue
		}

		for _, recipe := range recipes {
			if recipe.Slug == recipeName {
				return recipe, nil
			}
		}
	}

	return nil, fmt.Errorf("recipe '%s' not found", recipeName)
}

func (i *Installer) installLoader(modpack *sources.Modpack, destDir string) error {
	opts := &converter.ConversionOptions{
		DestDir:        destDir,
		ModpackName:    modpack.Name,
		MCVersion:      modpack.MCVersion,
		Loader:         modpack.Loader,
		LoaderVersion:  modpack.LoaderVersion,
		RecommendedRAM: modpack.RecommendedRAM,
		PreserveData:   false,
	}

	loaderInstaller := converter.NewLoaderInstaller()
	return loaderInstaller.Install(opts)
}

func (i *Installer) downloadMods(mods []*sources.Mod, destDir string) (int, error) {
	modManager := converter.NewModManager()
	modManager.SkipVerify = i.skipVerify
	serverMods := modManager.FilterServerMods(mods)

	if len(serverMods) == 0 {
		return 0, nil
	}

	if err := modManager.DownloadMods(serverMods, destDir); err != nil {
		return 0, err
	}

	return len(serverMods), nil
}

func (i *Installer) generateConfigs(modpack *sources.Modpack, destDir string) error {
	opts := &converter.ConversionOptions{
		DestDir:        destDir,
		ModpackName:    modpack.Name,
		MCVersion:      modpack.MCVersion,
		Loader:         modpack.Loader,
		LoaderVersion:  modpack.LoaderVersion,
		RecommendedRAM: modpack.RecommendedRAM,
	}

	configGen := converter.NewConfigGenerator()
	return configGen.Generate(opts)
}

func (i *Installer) generateScripts(modpack *sources.Modpack, destDir string) error {
	opts := &converter.ConversionOptions{
		DestDir:        destDir,
		ModpackName:    modpack.Name,
		MCVersion:      modpack.MCVersion,
		Loader:         modpack.Loader,
		LoaderVersion:  modpack.LoaderVersion,
		RecommendedRAM: modpack.RecommendedRAM,
	}

	scriptGen := converter.NewScriptGenerator()
	return scriptGen.Generate(opts)
}

// createRecipeSnapshot converts modpack data to a recipe snapshot for tracking
func createRecipeSnapshot(modpack *sources.Modpack) map[string]interface{} {
	snapshot := map[string]interface{}{
		"name":           modpack.Name,
		"identifier":     modpack.Identifier,
		"description":    modpack.Description,
		"mc_version":     modpack.MCVersion,
		"loader":         string(modpack.Loader),
		"loader_version": modpack.LoaderVersion,
		"author":         modpack.Author,
		"source":         modpack.Source,
		"recommended_ram": modpack.RecommendedRAM,
		"manifest_url":   modpack.ManifestURL,
	}

	if len(modpack.Dependencies) > 0 {
		snapshot["dependencies"] = modpack.Dependencies
	}

	if len(modpack.Mods) > 0 {
		mods := make([]map[string]interface{}, 0, len(modpack.Mods))
		for _, mod := range modpack.Mods {
			modData := map[string]interface{}{
				"name":     mod.Name,
				"version":  mod.Version,
				"filename": mod.FileName,
				"side":     string(mod.Side),
				"required": mod.Required,
			}
			if mod.DownloadURL != "" {
				modData["download_url"] = mod.DownloadURL
			}
			if mod.SHA256 != "" {
				modData["sha256"] = mod.SHA256
			}
			if mod.SHA512 != "" {
				modData["sha512"] = mod.SHA512
			}
			mods = append(mods, modData)
		}
		snapshot["mods"] = mods
	}

	return snapshot
}

// TrackInstallation records the installation in the tracking registry
func TrackInstallation(result *Result, identifier string) error {
	if result == nil || result.Modpack == nil {
		return fmt.Errorf("result and modpack data required for tracking")
	}

	tracker, err := tracking.NewTracker()
	if err != nil {
		return fmt.Errorf("failed to initialize tracker: %w", err)
	}

	// Extract slug and version from identifier or modpack
	slug := result.Modpack.Identifier
	if slug == "" {
		slug = identifier
	}

	// Determine version - use MC version + loader combination as default
	// since modpack struct doesn't have a version field yet
	version := fmt.Sprintf("%s-%s", result.Modpack.MCVersion, result.Modpack.Loader)
	if result.Modpack.LoaderVersion != "" {
		version = fmt.Sprintf("%s-%s-%s", result.Modpack.MCVersion, result.Modpack.Loader, result.Modpack.LoaderVersion)
	}

	// Determine bench (source repository)
	bench := "unknown"
	if result.Modpack.Source != "" {
		bench = result.Modpack.Source
	}

	installation := &tracking.Installation{
		Slug:           slug,
		Version:        version,
		Bench:          bench,
		Path:           result.DestDir,
		InstalledAt:    time.Now().UTC(),
		RecipeSnapshot: createRecipeSnapshot(result.Modpack),
	}

	if err := tracker.AddInstallation(installation); err != nil {
		return fmt.Errorf("failed to track installation: %w", err)
	}

	return nil
}
