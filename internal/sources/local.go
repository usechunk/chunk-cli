package sources

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type LocalClient struct {
	mrpackParser *MRPackParser
}

func NewLocalClient() *LocalClient {
	return &LocalClient{
		mrpackParser: NewMRPackParser(),
	}
}

func (l *LocalClient) Fetch(identifier string) (*Modpack, error) {
	if !fileExists(identifier) {
		return nil, fmt.Errorf("file not found: %s", identifier)
	}
	
	ext := strings.ToLower(filepath.Ext(identifier))
	
	switch ext {
	case ".mrpack":
		return l.mrpackParser.Parse(identifier)
	case ".zip":
		return l.parseZip(identifier)
	default:
		return nil, fmt.Errorf("unsupported file format: %s", ext)
	}
}

func (l *LocalClient) Search(query string) ([]*ModpackSearchResult, error) {
	return nil, fmt.Errorf("search not supported for local files")
}

func (l *LocalClient) GetVersions(identifier string) ([]*Version, error) {
	return nil, fmt.Errorf("version lookup not supported for local files")
}

func (l *LocalClient) parseZip(filePath string) (*Modpack, error) {
	modpack, err := l.mrpackParser.Parse(filePath)
	if err == nil {
		return modpack, nil
	}
	
	return &Modpack{
		Name:       filepath.Base(filePath),
		Identifier: filePath,
		Source:     "local",
	}, nil
}

func (l *LocalClient) Extract(filePath, destDir string) error {
	ext := strings.ToLower(filepath.Ext(filePath))
	
	switch ext {
	case ".mrpack", ".zip":
		return l.mrpackParser.Extract(filePath, destDir)
	default:
		return fmt.Errorf("unsupported file format for extraction: %s", ext)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
