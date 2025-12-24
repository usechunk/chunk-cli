package sources

import (
	"errors"
	"strings"
)

var (
	ErrInvalidSource   = errors.New("invalid modpack source")
	ErrNotFound        = errors.New("modpack not found")
	ErrDownloadFailed  = errors.New("download failed")
	ErrInvalidManifest = errors.New("invalid manifest")
)

type ModpackSource interface {
	Fetch(identifier string) (*Modpack, error)
	Search(query string) ([]*ModpackSearchResult, error)
	GetVersions(identifier string) ([]*Version, error)
}

type Modpack struct {
	Name           string
	Identifier     string
	Description    string
	MCVersion      string
	Loader         LoaderType
	LoaderVersion  string
	Author         string
	Source         string
	Mods           []*Mod
	Dependencies   []string
	RecommendedRAM int
	ManifestURL    string
}

type ModpackSearchResult struct {
	Name        string
	Identifier  string
	Description string
	MCVersion   string
	Loader      LoaderType
	Source      string
	Downloads   int
}

type Version struct {
	Version     string
	MCVersion   string
	Loader      LoaderType
	ReleaseDate string
	IsStable    bool
	DownloadURL string
	SHA256      string
	SHA512      string
}

type Mod struct {
	Name        string
	Version     string
	FileName    string
	DownloadURL string
	Side        ModSide
	Required    bool
	SHA256      string
	SHA512      string
}

type LoaderType string

const (
	LoaderForge    LoaderType = "forge"
	LoaderFabric   LoaderType = "fabric"
	LoaderNeoForge LoaderType = "neoforge"
)

type ModSide string

const (
	SideClient ModSide = "client"
	SideServer ModSide = "server"
	SideBoth   ModSide = "both"
)

func DetectSource(identifier string) string {
	// Local file paths
	if len(identifier) > 0 && identifier[0] == '.' || identifier[0] == '/' {
		return "local"
	}

	// Modrinth prefix
	if len(identifier) > 9 && identifier[:9] == "modrinth:" {
		return "modrinth"
	}

	// Explicit bench::recipe syntax (e.g., usechunk/recipes::atm9)
	if strings.Contains(identifier, "::") {
		return "recipe"
	}

	// GitHub repository (contains / but not ::)
	if len(identifier) > 0 && (identifier[0] >= 'a' && identifier[0] <= 'z' || identifier[0] >= 'A' && identifier[0] <= 'Z') {
		for i, ch := range identifier {
			if ch == '/' {
				// If it contains / and no ::, it's GitHub
				return "github"
			}
			if i > 50 {
				break
			}
		}
	}

	// Default to recipe (searches local benches first)
	// This replaces chunkhub as the default
	return "recipe"
}
