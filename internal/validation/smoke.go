package validation

import (
	"fmt"
	"os"
	"path/filepath"
)

type SmokeTest struct{}

func NewSmokeTest() *SmokeTest {
	return &SmokeTest{}
}

type TestResult struct {
	Name    string
	Passed  bool
	Message string
}

type TestReport struct {
	Results []TestResult
	Passed  int
	Failed  int
}

func (s *SmokeTest) RunAll(serverDir string) *TestReport {
	report := &TestReport{
		Results: []TestResult{},
	}

	tests := []func(string) TestResult{
		s.testServerJarExists,
		s.testModsDirectoryExists,
		s.testConfigDirectoryExists,
		s.testStartScriptExists,
		s.testStartScriptExecutable,
		s.testServerPropertiesExists,
		s.testEulaExists,
		s.testChunkManifestExists,
		s.testDirectoryWriteable,
	}

	for _, test := range tests {
		result := test(serverDir)
		report.Results = append(report.Results, result)
		if result.Passed {
			report.Passed++
		} else {
			report.Failed++
		}
	}

	return report
}

func (s *SmokeTest) testServerJarExists(serverDir string) TestResult {
	patterns := []string{
		"server.jar",
		"minecraft_server.*.jar",
		"forge-*.jar",
		"fabric-server-launch.jar",
		"neoforge-*.jar",
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(filepath.Join(serverDir, pattern))
		if len(matches) > 0 {
			return TestResult{
				Name:    "Server JAR",
				Passed:  true,
				Message: fmt.Sprintf("Found: %s", filepath.Base(matches[0])),
			}
		}
	}

	return TestResult{
		Name:    "Server JAR",
		Passed:  false,
		Message: "Server JAR not found",
	}
}

func (s *SmokeTest) testModsDirectoryExists(serverDir string) TestResult {
	modsDir := filepath.Join(serverDir, "mods")
	if _, err := os.Stat(modsDir); err == nil {
		entries, _ := os.ReadDir(modsDir)
		modCount := 0
		for _, entry := range entries {
			if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jar" {
				modCount++
			}
		}
		return TestResult{
			Name:    "Mods Directory",
			Passed:  true,
			Message: fmt.Sprintf("%d mod(s) installed", modCount),
		}
	}

	return TestResult{
		Name:    "Mods Directory",
		Passed:  false,
		Message: "Mods directory not found",
	}
}

func (s *SmokeTest) testConfigDirectoryExists(serverDir string) TestResult {
	configDir := filepath.Join(serverDir, "config")
	if _, err := os.Stat(configDir); err == nil {
		return TestResult{
			Name:    "Config Directory",
			Passed:  true,
			Message: "Config directory exists",
		}
	}

	return TestResult{
		Name:    "Config Directory",
		Passed:  true,
		Message: "Config directory will be created on first run",
	}
}

func (s *SmokeTest) testStartScriptExists(serverDir string) TestResult {
	scripts := []string{"start.sh", "start.bat", "start.command"}

	for _, script := range scripts {
		scriptPath := filepath.Join(serverDir, script)
		if _, err := os.Stat(scriptPath); err == nil {
			return TestResult{
				Name:    "Start Script",
				Passed:  true,
				Message: fmt.Sprintf("Found: %s", script),
			}
		}
	}

	return TestResult{
		Name:    "Start Script",
		Passed:  false,
		Message: "No start script found",
	}
}

func (s *SmokeTest) testStartScriptExecutable(serverDir string) TestResult {
	scriptPath := filepath.Join(serverDir, "start.sh")

	info, err := os.Stat(scriptPath)
	if err != nil {
		return TestResult{
			Name:    "Script Permissions",
			Passed:  true,
			Message: "Skipped (script not found)",
		}
	}

	if info.Mode()&0111 != 0 {
		return TestResult{
			Name:    "Script Permissions",
			Passed:  true,
			Message: "Start script is executable",
		}
	}

	return TestResult{
		Name:    "Script Permissions",
		Passed:  false,
		Message: "Start script is not executable (run: chmod +x start.sh)",
	}
}

func (s *SmokeTest) testServerPropertiesExists(serverDir string) TestResult {
	propsPath := filepath.Join(serverDir, "server.properties")

	if _, err := os.Stat(propsPath); err == nil {
		return TestResult{
			Name:    "Server Properties",
			Passed:  true,
			Message: "server.properties exists",
		}
	}

	return TestResult{
		Name:    "Server Properties",
		Passed:  true,
		Message: "Will be generated on first run",
	}
}

func (s *SmokeTest) testEulaExists(serverDir string) TestResult {
	eulaPath := filepath.Join(serverDir, "eula.txt")

	if _, err := os.Stat(eulaPath); err == nil {
		data, err := os.ReadFile(eulaPath)
		if err == nil && len(data) > 0 {
			return TestResult{
				Name:    "EULA",
				Passed:  true,
				Message: "eula.txt exists",
			}
		}
	}

	return TestResult{
		Name:    "EULA",
		Passed:  true,
		Message: "Will be generated on first run (requires acceptance)",
	}
}

func (s *SmokeTest) testChunkManifestExists(serverDir string) TestResult {
	manifestPath := filepath.Join(serverDir, ".chunk.json")

	if _, err := os.Stat(manifestPath); err == nil {
		return TestResult{
			Name:    "Chunk Manifest",
			Passed:  true,
			Message: ".chunk.json exists",
		}
	}

	return TestResult{
		Name:    "Chunk Manifest",
		Passed:  false,
		Message: ".chunk.json not found (not chunk-managed)",
	}
}

func (s *SmokeTest) testDirectoryWriteable(serverDir string) TestResult {
	testFile := filepath.Join(serverDir, ".chunk-write-test")

	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		return TestResult{
			Name:    "Directory Permissions",
			Passed:  false,
			Message: fmt.Sprintf("Directory not writeable: %v", err),
		}
	}

	os.Remove(testFile)

	return TestResult{
		Name:    "Directory Permissions",
		Passed:  true,
		Message: "Directory is writeable",
	}
}

func (s *SmokeTest) PrintReport(report *TestReport) {
	fmt.Printf("\n🔍 Smoke Test Report\n")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	for _, result := range report.Results {
		icon := "✓"
		if !result.Passed {
			icon = "✗"
		}
		fmt.Printf("%s %-25s %s\n", icon, result.Name+":", result.Message)
	}

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Results: %d passed, %d failed\n\n", report.Passed, report.Failed)

	if report.Failed == 0 {
		fmt.Println("✅ All smoke tests passed! Server is ready to start.")
	} else {
		fmt.Println("⚠️  Some tests failed. Please review the issues above.")
	}
}
