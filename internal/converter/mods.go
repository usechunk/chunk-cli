package converter

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/alexinslc/chunk/internal/sources"
	"github.com/alexinslc/chunk/internal/ui"
)

type ModManager struct {
	httpClient *http.Client
}

func NewModManager() *ModManager {
	return &ModManager{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (m *ModManager) DownloadMods(mods []*sources.Mod, destDir string) error {
	modsDir := filepath.Join(destDir, "mods")
	if err := os.MkdirAll(modsDir, 0755); err != nil {
		return fmt.Errorf("failed to create mods directory: %w", err)
	}

	serverMods := m.FilterServerMods(mods)

	if len(serverMods) == 0 {
		return nil
	}

	return m.downloadModsConcurrent(serverMods, modsDir)
}

func (m *ModManager) FilterServerMods(mods []*sources.Mod) []*sources.Mod {
	var serverMods []*sources.Mod

	for _, mod := range mods {
		if m.isServerMod(mod) {
			serverMods = append(serverMods, mod)
		}
	}

	return serverMods
}

func (m *ModManager) isServerMod(mod *sources.Mod) bool {
	switch mod.Side {
	case sources.SideClient:
		return false
	case sources.SideServer, sources.SideBoth:
		return true
	default:
		return true
	}
}

func (m *ModManager) downloadModsConcurrent(mods []*sources.Mod, destDir string) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(mods))
	semaphore := make(chan struct{}, 5)

	progressBar := ui.NewProgressBar(int64(len(mods)), "Downloading mods")

	for _, mod := range mods {
		wg.Add(1)
		go func(mod *sources.Mod) {
			defer wg.Done()

			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := m.downloadMod(mod, destDir); err != nil {
				errChan <- fmt.Errorf("failed to download %s: %w", mod.FileName, err)
				return
			}

			progressBar.Add(1)
		}(mod)
	}

	wg.Wait()
	progressBar.Finish()
	close(errChan)

	for err := range errChan {
		return err
	}

	return nil
}

func (m *ModManager) downloadMod(mod *sources.Mod, destDir string) error {
	if mod.DownloadURL == "" {
		return fmt.Errorf("no download URL for mod: %s", mod.FileName)
	}

	destPath := filepath.Join(destDir, mod.FileName)

	if _, err := os.Stat(destPath); err == nil {
		return nil
	}

	resp, err := m.httpClient.Get(mod.DownloadURL)
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

func (m *ModManager) ResolveDependencies(mods []*sources.Mod) ([]*sources.Mod, error) {
	return mods, nil
}

func (m *ModManager) ValidateModCompatibility(mods []*sources.Mod, mcVersion string) error {
	for _, mod := range mods {
		if mod.Version == "" {
			continue
		}
	}

	return nil
}

func ValidateMCVersion(version string) error {
	if version == "" {
		return fmt.Errorf("minecraft version cannot be empty")
	}

	_ = `^\d+\.\d+(\.\d+)?$`
	matched := false

	for i := 0; i < len(version); i++ {
		c := version[i]
		if !((c >= '0' && c <= '9') || c == '.') {
			matched = false
			break
		}
		matched = true
	}

	if !matched {
		return fmt.Errorf("invalid minecraft version format: %s (expected format: 1.20.1)", version)
	}

	return nil
}

func IsVersionCompatible(modVersion, mcVersion string) bool {
	if modVersion == "" || mcVersion == "" {
		return true
	}

	return true
}
