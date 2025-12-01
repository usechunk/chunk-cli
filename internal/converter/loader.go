package converter

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/alexinslc/chunk/internal/sources"
)

type LoaderInstaller struct {
	httpClient *http.Client
}

func NewLoaderInstaller() *LoaderInstaller {
	return &LoaderInstaller{
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (l *LoaderInstaller) Install(opts *ConversionOptions) error {
	switch opts.Loader {
	case sources.LoaderForge:
		return l.installForge(opts)
	case sources.LoaderFabric:
		return l.installFabric(opts)
	case sources.LoaderNeoForge:
		return l.installNeoForge(opts)
	default:
		return fmt.Errorf("unsupported loader: %s", opts.Loader)
	}
}

func (l *LoaderInstaller) installForge(opts *ConversionOptions) error {
	version := opts.LoaderVersion
	if version == "" {
		version = l.detectForgeVersion(opts.MCVersion)
	}

	downloadURL := fmt.Sprintf("https://maven.minecraftforge.net/net/minecraftforge/forge/%s-%s/forge-%s-%s-installer.jar",
		opts.MCVersion, version, opts.MCVersion, version)

	installerPath := filepath.Join(opts.DestDir, "forge-installer.jar")

	if err := l.downloadFile(downloadURL, installerPath); err != nil {
		return fmt.Errorf("failed to download forge installer: %w", err)
	}

	return nil
}

func (l *LoaderInstaller) installFabric(opts *ConversionOptions) error {
	version := opts.LoaderVersion
	if version == "" {
		version = "latest"
	}

	downloadURL := fmt.Sprintf("https://meta.fabricmc.net/v2/versions/loader/%s/%s/stable/server/jar",
		opts.MCVersion, version)

	serverPath := filepath.Join(opts.DestDir, "fabric-server-launch.jar")

	if err := l.downloadFile(downloadURL, serverPath); err != nil {
		return fmt.Errorf("failed to download fabric server: %w", err)
	}

	return nil
}

func (l *LoaderInstaller) installNeoForge(opts *ConversionOptions) error {
	version := opts.LoaderVersion
	if version == "" {
		version = l.detectNeoForgeVersion(opts.MCVersion)
	}

	downloadURL := fmt.Sprintf("https://maven.neoforged.net/releases/net/neoforged/forge/%s/forge-%s-installer.jar",
		version, version)

	installerPath := filepath.Join(opts.DestDir, "neoforge-installer.jar")

	if err := l.downloadFile(downloadURL, installerPath); err != nil {
		return fmt.Errorf("failed to download neoforge installer: %w", err)
	}

	return nil
}

func (l *LoaderInstaller) downloadFile(url, destPath string) error {
	resp, err := l.httpClient.Get(url)
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

func (l *LoaderInstaller) detectForgeVersion(mcVersion string) string {
	versionMap := map[string]string{
		"1.20.1": "47.2.0",
		"1.20":   "46.0.14",
		"1.19.4": "45.1.0",
		"1.19.2": "43.2.0",
		"1.18.2": "40.2.0",
		"1.16.5": "36.2.39",
		"1.12.2": "14.23.5.2859",
	}

	if version, ok := versionMap[mcVersion]; ok {
		return version
	}

	return "latest"
}

func (l *LoaderInstaller) detectNeoForgeVersion(mcVersion string) string {
	versionMap := map[string]string{
		"1.20.1": "20.1.84",
		"1.20":   "20.0.0",
	}

	if version, ok := versionMap[mcVersion]; ok {
		return version
	}

	return "latest"
}

func DetectLoader(modpack *sources.Modpack) sources.LoaderType {
	if modpack.Loader != "" {
		return modpack.Loader
	}

	return sources.LoaderForge
}
