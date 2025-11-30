package java

import (
	"fmt"

	"github.com/alexinslc/chunk/internal/config"
)

type JavaValidator struct {
	detector *JavaDetector
}

func NewJavaValidator() *JavaValidator {
	return &JavaValidator{
		detector: NewJavaDetector(),
	}
}

func (v *JavaValidator) Validate(manifest *config.ChunkManifest) error {
	java, err := v.detector.FindCompatible(manifest.MCVersion)
	if err != nil {
		return v.createUserFriendlyError(manifest.MCVersion, manifest.JavaVersion)
	}

	requiredVersion := manifest.JavaVersion
	if requiredVersion == 0 {
		requiredVersion = GetRequiredJavaVersion(manifest.MCVersion)
	}

	if java.Major < requiredVersion {
		return v.createVersionMismatchError(java, requiredVersion, manifest.MCVersion)
	}

	return nil
}

func (v *JavaValidator) ValidateForMCVersion(mcVersion string, requiredJava int) error {
	java, err := v.detector.FindCompatible(mcVersion)
	if err != nil {
		return v.createUserFriendlyError(mcVersion, requiredJava)
	}

	if requiredJava == 0 {
		requiredJava = GetRequiredJavaVersion(mcVersion)
	}

	if java.Major < requiredJava {
		return v.createVersionMismatchError(java, requiredJava, mcVersion)
	}

	return nil
}

func (v *JavaValidator) GetRecommendedJava(mcVersion string) string {
	requiredVersion := GetRequiredJavaVersion(mcVersion)

	recommendations := map[int]string{
		21: "Eclipse Temurin 21 (https://adoptium.net/)",
		17: "Eclipse Temurin 17 (https://adoptium.net/)",
		8:  "Eclipse Temurin 8 (https://adoptium.net/)",
	}

	if rec, ok := recommendations[requiredVersion]; ok {
		return rec
	}

	return fmt.Sprintf("Java %d from Eclipse Temurin (https://adoptium.net/)", requiredVersion)
}

func (v *JavaValidator) createUserFriendlyError(mcVersion string, requiredJava int) error {
	if requiredJava == 0 {
		requiredJava = GetRequiredJavaVersion(mcVersion)
	}

	recommendation := v.GetRecommendedJava(mcVersion)

	return fmt.Errorf(`No compatible Java installation found.

Minecraft %s requires Java %d or higher.

To fix this issue:
1. Download and install: %s
2. Restart your terminal/command prompt
3. Run 'chunk install' again

For more help, visit: https://docs.chunkhub.io/java-setup`,
		mcVersion, requiredJava, recommendation)
}

func (v *JavaValidator) createVersionMismatchError(java *JavaInstallation, required int, mcVersion string) error {
	recommendation := v.GetRecommendedJava(mcVersion)

	return fmt.Errorf(`Java version mismatch detected.

Found: Java %d (%s) at %s
Required: Java %d or higher for Minecraft %s

To fix this issue:
1. Download and install: %s
2. Ensure the new Java version is in your PATH
3. Run 'java -version' to verify
4. Run 'chunk install' again

For more help, visit: https://docs.chunkhub.io/java-setup`,
		java.Major, java.Vendor, java.Path, required, mcVersion, recommendation)
}

func (v *JavaValidator) CheckInstallation() (*JavaInstallation, error) {
	installations, err := v.detector.DetectAll()
	if err != nil {
		return nil, err
	}

	if len(installations) == 0 {
		return nil, fmt.Errorf("no Java installations found")
	}

	return installations[0], nil
}

func (v *JavaValidator) ListInstallations() ([]*JavaInstallation, error) {
	return v.detector.DetectAll()
}
