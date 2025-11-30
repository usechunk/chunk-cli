package java

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/alexinslc/chunk/internal/ui"
)

type JavaInstaller struct {
	validator *JavaValidator
}

func NewJavaInstaller() *JavaInstaller {
	return &JavaInstaller{
		validator: NewJavaValidator(),
	}
}

func (i *JavaInstaller) GuideInstallation(mcVersion string) error {
	requiredVersion := GetRequiredJavaVersion(mcVersion)
	
	fmt.Printf("\n🔧 Java Installation Guide\n\n")
	fmt.Printf("Minecraft %s requires Java %d or higher.\n\n", mcVersion, requiredVersion)
	
	recommendation := i.validator.GetRecommendedJava(mcVersion)
	fmt.Printf("Recommended: %s\n\n", recommendation)
	
	switch runtime.GOOS {
	case "windows":
		i.printWindowsInstructions(requiredVersion)
	case "darwin":
		i.printMacInstructions(requiredVersion)
	case "linux":
		i.printLinuxInstructions(requiredVersion)
	}
	
	fmt.Printf("\nAfter installation:\n")
	fmt.Printf("1. Restart your terminal\n")
	fmt.Printf("2. Run: java -version\n")
	fmt.Printf("3. Verify Java %d or higher is installed\n", requiredVersion)
	fmt.Printf("4. Run 'chunk install' again\n\n")
	
	return nil
}

func (i *JavaInstaller) printWindowsInstructions(version int) {
	fmt.Printf("Windows Installation:\n")
	fmt.Printf("1. Visit https://adoptium.net/\n")
	fmt.Printf("2. Click 'Download' for Java %d\n", version)
	fmt.Printf("3. Run the .msi installer\n")
	fmt.Printf("4. Follow the installation wizard\n")
	fmt.Printf("5. Ensure 'Add to PATH' is checked\n")
}

func (i *JavaInstaller) printMacInstructions(version int) {
	fmt.Printf("macOS Installation:\n")
	fmt.Printf("\nOption 1 - Homebrew (Recommended):\n")
	fmt.Printf("  brew install openjdk@%d\n", version)
	fmt.Printf("  sudo ln -sfn $(brew --prefix)/opt/openjdk@%d/libexec/openjdk.jdk /Library/Java/JavaVirtualMachines/openjdk-%d.jdk\n", version, version)
	fmt.Printf("\nOption 2 - Manual Download:\n")
	fmt.Printf("1. Visit https://adoptium.net/\n")
	fmt.Printf("2. Download Java %d for macOS\n", version)
	fmt.Printf("3. Open the .pkg file\n")
	fmt.Printf("4. Follow the installation wizard\n")
}

func (i *JavaInstaller) printLinuxInstructions(version int) {
	fmt.Printf("Linux Installation:\n")
	fmt.Printf("\nDebian/Ubuntu:\n")
	fmt.Printf("  sudo apt update\n")
	fmt.Printf("  sudo apt install openjdk-%d-jdk\n", version)
	fmt.Printf("\nFedora/RHEL:\n")
	fmt.Printf("  sudo dnf install java-%d-openjdk\n", version)
	fmt.Printf("\nArch Linux:\n")
	fmt.Printf("  sudo pacman -S jdk-openjdk\n")
	fmt.Printf("\nOr download from: https://adoptium.net/\n")
}

func (i *JavaInstaller) CheckAndGuide(mcVersion string) error {
	err := i.validator.ValidateForMCVersion(mcVersion, 0)
	if err != nil {
		fmt.Println(err)
		fmt.Printf("\n")
		return i.GuideInstallation(mcVersion)
	}
	
	return nil
}
