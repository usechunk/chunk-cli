package java

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

type JavaInstallation struct {
	Path    string
	Version string
	Major   int
	Vendor  string
}

type JavaDetector struct{}

func NewJavaDetector() *JavaDetector {
	return &JavaDetector{}
}

func (d *JavaDetector) DetectAll() ([]*JavaInstallation, error) {
	var installations []*JavaInstallation

	javaCmd, err := exec.LookPath("java")
	if err == nil {
		if install, err := d.getJavaInfo(javaCmd); err == nil {
			installations = append(installations, install)
		}
	}

	switch runtime.GOOS {
	case "windows":
		windowsInstalls := d.detectWindows()
		installations = append(installations, windowsInstalls...)
	case "darwin":
		macInstalls := d.detectMac()
		installations = append(installations, macInstalls...)
	case "linux":
		linuxInstalls := d.detectLinux()
		installations = append(installations, linuxInstalls...)
	}

	installations = d.deduplicate(installations)

	return installations, nil
}

func (d *JavaDetector) getJavaInfo(javaPath string) (*JavaInstallation, error) {
	cmd := exec.Command(javaPath, "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	versionStr := string(output)
	version, major := d.parseVersion(versionStr)
	vendor := d.parseVendor(versionStr)

	return &JavaInstallation{
		Path:    javaPath,
		Version: version,
		Major:   major,
		Vendor:  vendor,
	}, nil
}

func (d *JavaDetector) parseVersion(output string) (string, int) {
	versionPattern := regexp.MustCompile(`version "([^"]+)"`)
	matches := versionPattern.FindStringSubmatch(output)

	if len(matches) < 2 {
		return "unknown", 0
	}

	version := matches[1]

	majorPattern := regexp.MustCompile(`^1\.(\d+)\.`)
	if majorMatches := majorPattern.FindStringSubmatch(version); len(majorMatches) >= 2 {
		major := 0
		fmt.Sscanf(majorMatches[1], "%d", &major)
		return version, major
	}

	majorPattern = regexp.MustCompile(`^(\d+)\.`)
	if majorMatches := majorPattern.FindStringSubmatch(version); len(majorMatches) >= 2 {
		major := 0
		fmt.Sscanf(majorMatches[1], "%d", &major)
		return version, major
	}

	return version, 0
}

func (d *JavaDetector) parseVendor(output string) string {
	output = strings.ToLower(output)

	if strings.Contains(output, "openjdk") {
		return "OpenJDK"
	}
	if strings.Contains(output, "oracle") {
		return "Oracle"
	}
	if strings.Contains(output, "adoptium") || strings.Contains(output, "eclipse temurin") {
		return "Adoptium"
	}
	if strings.Contains(output, "corretto") {
		return "Amazon Corretto"
	}
	if strings.Contains(output, "zulu") {
		return "Azul Zulu"
	}

	return "Unknown"
}

func (d *JavaDetector) detectWindows() []*JavaInstallation {
	var installations []*JavaInstallation

	paths := []string{
		"C:\\Program Files\\Java",
		"C:\\Program Files (x86)\\Java",
		"C:\\Program Files\\Eclipse Adoptium",
		"C:\\Program Files\\Amazon Corretto",
	}

	for _, basePath := range paths {
		cmd := exec.Command("cmd", "/c", "dir", "/b", basePath)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		dirs := strings.Split(string(output), "\n")
		for _, dir := range dirs {
			dir = strings.TrimSpace(dir)
			if dir == "" {
				continue
			}

			javaPath := fmt.Sprintf("%s\\%s\\bin\\java.exe", basePath, dir)
			if install, err := d.getJavaInfo(javaPath); err == nil {
				installations = append(installations, install)
			}
		}
	}

	return installations
}

func (d *JavaDetector) detectMac() []*JavaInstallation {
	var installations []*JavaInstallation

	cmd := exec.Command("/usr/libexec/java_home", "-V")
	output, err := cmd.CombinedOutput()
	if err == nil {
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "/Library/Java") {
				parts := strings.Fields(line)
				for _, part := range parts {
					if strings.HasPrefix(part, "/") {
						javaPath := fmt.Sprintf("%s/bin/java", part)
						if install, err := d.getJavaInfo(javaPath); err == nil {
							installations = append(installations, install)
						}
						break
					}
				}
			}
		}
	}

	return installations
}

func (d *JavaDetector) detectLinux() []*JavaInstallation {
	var installations []*JavaInstallation

	paths := []string{
		"/usr/lib/jvm",
		"/usr/java",
		"/opt/java",
		"/opt/jdk",
	}

	for _, basePath := range paths {
		cmd := exec.Command("ls", basePath)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		dirs := strings.Split(string(output), "\n")
		for _, dir := range dirs {
			dir = strings.TrimSpace(dir)
			if dir == "" {
				continue
			}

			javaPath := fmt.Sprintf("%s/%s/bin/java", basePath, dir)
			if install, err := d.getJavaInfo(javaPath); err == nil {
				installations = append(installations, install)
			}
		}
	}

	return installations
}

func (d *JavaDetector) deduplicate(installations []*JavaInstallation) []*JavaInstallation {
	seen := make(map[string]bool)
	var result []*JavaInstallation

	for _, install := range installations {
		key := fmt.Sprintf("%s-%s", install.Path, install.Version)
		if !seen[key] {
			seen[key] = true
			result = append(result, install)
		}
	}

	return result
}

func (d *JavaDetector) FindCompatible(mcVersion string) (*JavaInstallation, error) {
	installations, err := d.DetectAll()
	if err != nil {
		return nil, err
	}

	requiredJava := GetRequiredJavaVersion(mcVersion)

	for _, install := range installations {
		if install.Major >= requiredJava {
			return install, nil
		}
	}

	return nil, fmt.Errorf("no compatible Java installation found (need Java %d+)", requiredJava)
}

func GetRequiredJavaVersion(mcVersion string) int {
	if strings.HasPrefix(mcVersion, "1.20.5") || strings.HasPrefix(mcVersion, "1.21") {
		return 21
	}
	if strings.HasPrefix(mcVersion, "1.20") || strings.HasPrefix(mcVersion, "1.19") {
		return 17
	}
	if strings.HasPrefix(mcVersion, "1.18") || strings.HasPrefix(mcVersion, "1.17") {
		return 17
	}
	if strings.HasPrefix(mcVersion, "1.16") {
		return 8
	}

	return 8
}
