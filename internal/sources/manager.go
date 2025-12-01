package sources

import (
	"fmt"
)

type SourceManager struct {
	chunkhub *ChunkHubClient
	github   *GitHubClient
	modrinth *ModrinthClient
	local    *LocalClient
}

func NewSourceManager() *SourceManager {
	return &SourceManager{
		chunkhub: NewChunkHubClient(""),
		github:   NewGitHubClient(),
		modrinth: NewModrinthClient(),
		local:    NewLocalClient(),
	}
}

func (s *SourceManager) Fetch(identifier string) (*Modpack, error) {
	sourceType := DetectSource(identifier)

	switch sourceType {
	case "chunkhub":
		return s.chunkhub.Fetch(identifier)
	case "github":
		return s.github.Fetch(identifier)
	case "modrinth":
		return s.modrinth.Fetch(identifier)
	case "local":
		return s.local.Fetch(identifier)
	default:
		return nil, fmt.Errorf("unknown source type: %s", sourceType)
	}
}

func (s *SourceManager) Search(query string) ([]*ModpackSearchResult, error) {
	var allResults []*ModpackSearchResult

	chunkHubResults, err := s.chunkhub.Search(query)
	if err == nil {
		allResults = append(allResults, chunkHubResults...)
	}

	modrinthResults, err := s.modrinth.Search(query)
	if err == nil {
		allResults = append(allResults, modrinthResults...)
	}

	githubResults, err := s.github.Search(query)
	if err == nil {
		allResults = append(allResults, githubResults...)
	}

	if len(allResults) == 0 {
		return nil, fmt.Errorf("no results found for query: %s", query)
	}

	return allResults, nil
}

func (s *SourceManager) GetVersions(identifier string) ([]*Version, error) {
	sourceType := DetectSource(identifier)

	switch sourceType {
	case "chunkhub":
		return s.chunkhub.GetVersions(identifier)
	case "github":
		return s.github.GetVersions(identifier)
	case "modrinth":
		return s.modrinth.GetVersions(identifier)
	case "local":
		return s.local.GetVersions(identifier)
	default:
		return nil, fmt.Errorf("unknown source type: %s", sourceType)
	}
}

func (s *SourceManager) GetClient(sourceType string) (ModpackSource, error) {
	switch sourceType {
	case "chunkhub":
		return s.chunkhub, nil
	case "github":
		return s.github, nil
	case "modrinth":
		return s.modrinth, nil
	case "local":
		return s.local, nil
	default:
		return nil, fmt.Errorf("unknown source type: %s", sourceType)
	}
}
