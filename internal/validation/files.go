package validation

import (
	"fmt"
	"os"
	"path/filepath"
)

type FileValidator struct{}

func NewFileValidator() *FileValidator {
	return &FileValidator{}
}

type ValidationError struct {
	Path    string
	Issue   string
	Fixable bool
	Fix     string
}

func (f *FileValidator) ValidateServerStructure(serverDir string) []ValidationError {
	var errors []ValidationError

	requiredDirs := []string{"mods", "config"}
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(serverDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			errors = append(errors, ValidationError{
				Path:    dirPath,
				Issue:   "Required directory missing",
				Fixable: true,
				Fix:     fmt.Sprintf("mkdir -p %s", dirPath),
			})
		}
	}

	if err := f.validateServerJar(serverDir); err != nil {
		errors = append(errors, *err)
	}

	if err := f.validateModFiles(serverDir); err != nil {
		errors = append(errors, err...)
	}

	if err := f.validatePermissions(serverDir); err != nil {
		errors = append(errors, err...)
	}

	return errors
}

func (f *FileValidator) validateServerJar(serverDir string) *ValidationError {
	patterns := []string{
		"server.jar",
		"forge-*.jar",
		"fabric-server-launch.jar",
		"neoforge-*.jar",
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(serverDir, pattern))
		if len(matches) > 0 {
			return nil
		}
	}

	return &ValidationError{
		Path:    serverDir,
		Issue:   "No server JAR file found",
		Fixable: false,
		Fix:     "Run 'chunk install' to set up the server",
	}
}

func (f *FileValidator) validateModFiles(serverDir string) []ValidationError {
	var errors []ValidationError

	modsDir := filepath.Join(serverDir, "mods")
	if _, err := os.Stat(modsDir); os.IsNotExist(err) {
		return errors
	}

	entries, err := os.ReadDir(modsDir)
	if err != nil {
		return errors
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".jar" {
			errors = append(errors, ValidationError{
				Path:    filepath.Join(modsDir, entry.Name()),
				Issue:   "Non-JAR file in mods directory",
				Fixable: true,
				Fix:     "Remove or move this file",
			})
		}
	}

	return errors
}

func (f *FileValidator) validatePermissions(serverDir string) []ValidationError {
	var errors []ValidationError

	testFile := filepath.Join(serverDir, ".chunk-permission-test")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		errors = append(errors, ValidationError{
			Path:    serverDir,
			Issue:   "Directory is not writeable",
			Fixable: true,
			Fix:     fmt.Sprintf("chmod u+w %s", serverDir),
		})
	} else {
		os.Remove(testFile)
	}

	scripts := []string{"start.sh", "start.command"}
	for _, script := range scripts {
		scriptPath := filepath.Join(serverDir, script)
		info, err := os.Stat(scriptPath)
		if err == nil && info.Mode()&0111 == 0 {
			errors = append(errors, ValidationError{
				Path:    scriptPath,
				Issue:   "Start script is not executable",
				Fixable: true,
				Fix:     fmt.Sprintf("chmod +x %s", scriptPath),
			})
		}
	}

	return errors
}

func (f *FileValidator) ValidateConfigFiles(serverDir string) []ValidationError {
	var errors []ValidationError

	configDir := filepath.Join(serverDir, "config")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return errors
	}

	entries, err := os.ReadDir(configDir)
	if err != nil {
		return errors
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		configPath := filepath.Join(configDir, entry.Name())

		info, err := os.Stat(configPath)
		if err != nil {
			continue
		}

		if info.Size() == 0 {
			errors = append(errors, ValidationError{
				Path:    configPath,
				Issue:   "Config file is empty",
				Fixable: false,
				Fix:     "This file may be generated on first run",
			})
		}
	}

	return errors
}

func (f *FileValidator) AutoFix(errors []ValidationError) int {
	fixed := 0

	for _, err := range errors {
		if !err.Fixable {
			continue
		}

		switch err.Issue {
		case "Required directory missing":
			if e := os.MkdirAll(err.Path, 0755); e == nil {
				fmt.Printf("✓ Created directory: %s\n", err.Path)
				fixed++
			}

		case "Start script is not executable":
			if e := os.Chmod(err.Path, 0755); e == nil {
				fmt.Printf("✓ Made executable: %s\n", err.Path)
				fixed++
			}

		case "Non-JAR file in mods directory":
			fmt.Printf("⚠️  Found non-JAR in mods: %s (manual removal recommended)\n", err.Path)
		}
	}

	return fixed
}

func (f *FileValidator) PrintErrors(errors []ValidationError) {
	if len(errors) == 0 {
		fmt.Println("✅ All validation checks passed!")
		return
	}

	fmt.Printf("\n⚠️  Found %d validation issue(s):\n\n", len(errors))

	for i, err := range errors {
		fmt.Printf("%d. %s\n", i+1, err.Path)
		fmt.Printf("   Issue: %s\n", err.Issue)
		if err.Fixable {
			fmt.Printf("   Fix: %s\n", err.Fix)
		} else {
			fmt.Printf("   Note: %s\n", err.Fix)
		}
		fmt.Println()
	}
}
