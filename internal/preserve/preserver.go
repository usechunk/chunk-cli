package preserve

import (
	"fmt"
	"os"
	"path/filepath"
)

type DataPreserver struct{}

func NewDataPreserver() *DataPreserver {
	return &DataPreserver{}
}

func (p *DataPreserver) PreserveData(serverDir string) error {
	criticalPaths := []string{
		"world",
		"world_nether",
		"world_the_end",
		"server.properties",
		"whitelist.json",
		"ops.json",
		"banned-players.json",
		"banned-ips.json",
		"usercache.json",
	}

	for _, path := range criticalPaths {
		fullPath := filepath.Join(serverDir, path)
		if _, err := os.Stat(fullPath); err == nil {
			fmt.Printf("✓ Found: %s\n", path)
		}
	}

	return nil
}

func (p *DataPreserver) BackupBeforeUpgrade(serverDir string) (string, error) {
	backupDir := filepath.Join(serverDir, ".chunk-backup")

	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	criticalPaths := []string{
		"world",
		"world_nether",
		"world_the_end",
		"server.properties",
		"whitelist.json",
		"ops.json",
		"banned-players.json",
		"banned-ips.json",
	}

	for _, path := range criticalPaths {
		srcPath := filepath.Join(serverDir, path)
		if _, err := os.Stat(srcPath); err != nil {
			continue
		}

		dstPath := filepath.Join(backupDir, path)

		info, err := os.Stat(srcPath)
		if err != nil {
			continue
		}

		if info.IsDir() {
			if err := p.CopyDir(srcPath, dstPath); err != nil {
				return "", fmt.Errorf("failed to backup %s: %w", path, err)
			}
		} else {
			if err := p.CopyFile(srcPath, dstPath); err != nil {
				return "", fmt.Errorf("failed to backup %s: %w", path, err)
			}
		}

		fmt.Printf("✓ Backed up: %s\n", path)
	}

	return backupDir, nil
}

func (p *DataPreserver) RestoreFromBackup(serverDir, backupDir string) error {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	for _, entry := range entries {
		srcPath := filepath.Join(backupDir, entry.Name())
		dstPath := filepath.Join(serverDir, entry.Name())

		if entry.IsDir() {
			if err := p.CopyDir(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to restore %s: %w", entry.Name(), err)
			}
		} else {
			if err := p.CopyFile(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to restore %s: %w", entry.Name(), err)
			}
		}

		fmt.Printf("✓ Restored: %s\n", entry.Name())
	}

	return nil
}

// CopyFile copies a file from src to dst (exported for use in upgrade command)
func (p *DataPreserver) CopyFile(src, dst string) error {
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, 0644)
}

// CopyDir recursively copies a directory from src to dst (exported for use in upgrade command)
func (p *DataPreserver) CopyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := p.CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := p.CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *DataPreserver) GetCriticalFiles(serverDir string) []string {
	var existingFiles []string

	criticalPaths := []string{
		"world",
		"world_nether",
		"world_the_end",
		"server.properties",
		"whitelist.json",
		"ops.json",
		"banned-players.json",
		"banned-ips.json",
	}

	for _, path := range criticalPaths {
		fullPath := filepath.Join(serverDir, path)
		if _, err := os.Stat(fullPath); err == nil {
			existingFiles = append(existingFiles, path)
		}
	}

	return existingFiles
}
