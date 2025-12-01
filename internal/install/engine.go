package install

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alexinslc/chunk/internal/converter"
	"github.com/alexinslc/chunk/internal/sources"
	"github.com/alexinslc/chunk/internal/ui"
)

// Installer handles the complete installation workflow for modpacks
type Installer struct {
	sourceManager    *sources.SourceManager
	conversionEngine *converter.ConversionEngine
	httpClient       *http.Client
	backupDir        string
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
}

// Result contains the outcome of an installation
type Result struct {
	ModpackName   string
	MCVersion     string
	Loader        sources.LoaderType
	LoaderVersion string
	ModsInstalled int
	DestDir       string
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

	ui.PrintInfo(fmt.Sprintf("Installing to: %s", absDestDir))

	// Detect source type
	sourceType := sources.DetectSource(opts.Identifier)
	ui.PrintInfo(fmt.Sprintf("Source: %s", sourceType))

	// Fetch modpack metadata
	spinner := ui.NewSpinner("Fetching modpack information...")
	spinner.Start()

	modpack, err := i.fetchModpack(opts.Identifier, sourceType)
	if err != nil {
		spinner.Error(fmt.Sprintf("Failed to fetch modpack: %v", err))
		return nil, fmt.Errorf("failed to fetch modpack: %w", err)
	}
	spinner.Success(fmt.Sprintf("Found modpack: %s", modpack.Name))

	// Display modpack info
	i.displayModpackInfo(modpack)

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
		os.RemoveAll(i.backupDir)
	}

	return &Result{
		ModpackName:   modpack.Name,
		MCVersion:     modpack.MCVersion,
		Loader:        modpack.Loader,
		LoaderVersion: modpack.LoaderVersion,
		ModsInstalled: modsInstalled,
		DestDir:       absDestDir,
	}, nil
}

// Rollback restores the previous state after a failed installation
func (i *Installer) Rollback(destDir string) error {
	if i.backupDir == "" {
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

func (i *Installer) fetchModpack(identifier string, sourceType string) (*sources.Modpack, error) {
	return i.sourceManager.Fetch(identifier)
}

func (i *Installer) displayModpackInfo(modpack *sources.Modpack) {
	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("📦 %s\n", modpack.Name)
	if modpack.Description != "" {
		fmt.Printf("   %s\n", modpack.Description)
	}
	fmt.Println()
	fmt.Printf("   Minecraft: %s\n", modpack.MCVersion)
	fmt.Printf("   Loader:    %s", modpack.Loader)
	if modpack.LoaderVersion != "" {
		fmt.Printf(" %s", modpack.LoaderVersion)
	}
	fmt.Println()
	if modpack.Author != "" {
		fmt.Printf("   Author:    %s\n", modpack.Author)
	}
	fmt.Printf("   Source:    %s\n", modpack.Source)
	if len(modpack.Mods) > 0 {
		fmt.Printf("   Mods:      %d\n", len(modpack.Mods))
	}
	if modpack.RecommendedRAM > 0 {
		fmt.Printf("   RAM:       %dGB recommended\n", modpack.RecommendedRAM)
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
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

// DownloadFile downloads a file from a URL to a destination
func (i *Installer) DownloadFile(url, destPath string) error {
	resp, err := i.httpClient.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
