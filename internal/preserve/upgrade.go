package preserve

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alexinslc/chunk/internal/converter"
)

type UpgradeManager struct {
	preserver *DataPreserver
	converter *converter.ConversionEngine
}

func NewUpgradeManager() *UpgradeManager {
	return &UpgradeManager{
		preserver: NewDataPreserver(),
		converter: converter.NewConversionEngine(),
	}
}

func (u *UpgradeManager) UpgradeModpack(serverDir, newPackSource string, preserveData bool) error {
	if !preserveData {
		fmt.Println("⚠️  Skipping data preservation (--preserve-data=false)")
		return u.performUpgrade(serverDir, newPackSource)
	}

	criticalFiles := u.preserver.GetCriticalFiles(serverDir)

	if len(criticalFiles) == 0 {
		fmt.Println("ℹ️  No existing data found, proceeding with fresh install")
		return u.performUpgrade(serverDir, newPackSource)
	}

	fmt.Printf("\n📦 Found existing server data:\n")
	for _, file := range criticalFiles {
		fmt.Printf("  • %s\n", file)
	}

	fmt.Println("\n💾 Creating backup before upgrade...")
	backupDir, err := u.preserver.BackupBeforeUpgrade(serverDir)
	if err != nil {
		return fmt.Errorf("backup failed: %w", err)
	}

	fmt.Printf("✓ Backup created at: %s\n\n", backupDir)

	fmt.Println("🔄 Upgrading modpack...")
	if err := u.performUpgrade(serverDir, newPackSource); err != nil {
		fmt.Println("\n❌ Upgrade failed, restoring from backup...")
		if restoreErr := u.preserver.RestoreFromBackup(serverDir, backupDir); restoreErr != nil {
			return fmt.Errorf("upgrade failed and restore failed: %w, restore error: %v", err, restoreErr)
		}
		return fmt.Errorf("upgrade failed, data restored: %w", err)
	}

	fmt.Println("\n📁 Restoring world data...")
	worldPaths := []string{"world", "world_nether", "world_the_end"}
	for _, worldPath := range worldPaths {
		srcPath := filepath.Join(backupDir, worldPath)
		dstPath := filepath.Join(serverDir, worldPath)

		if _, err := os.Stat(srcPath); err == nil {
			if err := u.preserver.copyDir(srcPath, dstPath); err != nil {
				fmt.Printf("⚠️  Warning: Failed to restore %s: %v\n", worldPath, err)
			} else {
				fmt.Printf("✓ Restored: %s\n", worldPath)
			}
		}
	}

	fmt.Println("\n📄 Restoring server configuration...")
	configFiles := []string{
		"server.properties",
		"whitelist.json",
		"ops.json",
		"banned-players.json",
		"banned-ips.json",
	}

	for _, configFile := range configFiles {
		srcPath := filepath.Join(backupDir, configFile)
		dstPath := filepath.Join(serverDir, configFile)

		if _, err := os.Stat(srcPath); err == nil {
			if err := u.preserver.copyFile(srcPath, dstPath); err != nil {
				fmt.Printf("⚠️  Warning: Failed to restore %s: %v\n", configFile, err)
			} else {
				fmt.Printf("✓ Restored: %s\n", configFile)
			}
		}
	}

	fmt.Println("\n✅ Upgrade complete with data preserved!")
	fmt.Printf("📦 Backup is available at: %s\n", backupDir)

	return nil
}

func (u *UpgradeManager) performUpgrade(serverDir, newPackSource string) error {
	modsDir := filepath.Join(serverDir, "mods")
	if err := os.RemoveAll(modsDir); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old mods: %w", err)
	}

	return nil
}

func (u *UpgradeManager) CheckUpgradeAvailable(currentVersion, latestVersion string) bool {
	return currentVersion != latestVersion
}

func (u *UpgradeManager) GetUpgradeInfo(serverDir string) (*UpgradeInfo, error) {
	chunkFile := filepath.Join(serverDir, ".chunk.json")

	if _, err := os.Stat(chunkFile); err != nil {
		return nil, fmt.Errorf("no .chunk.json found, not a chunk-managed server")
	}

	return &UpgradeInfo{
		CurrentVersion: "unknown",
		HasWorldData:   len(u.preserver.GetCriticalFiles(serverDir)) > 0,
		ServerDir:      serverDir,
	}, nil
}

type UpgradeInfo struct {
	CurrentVersion string
	LatestVersion  string
	HasWorldData   bool
	ServerDir      string
}
