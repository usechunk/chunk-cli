package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	ModrinthAPIURL = "https://api.modrinth.com/v2"
)

type ModrinthClient struct {
	httpClient *http.Client
	apiKey     string
}

func NewModrinthClient() *ModrinthClient {
	return &ModrinthClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (m *ModrinthClient) SetAPIKey(apiKey string) {
	m.apiKey = apiKey
}

func (m *ModrinthClient) Fetch(identifier string) (*Modpack, error) {
	slug := strings.TrimPrefix(identifier, "modrinth:")

	projectURL := fmt.Sprintf("%s/project/%s", ModrinthAPIURL, url.PathEscape(slug))

	resp, err := m.get(projectURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("modrinth api error: status %d", resp.StatusCode)
	}

	var project struct {
		Slug         string   `json:"slug"`
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		GameVersions []string `json:"game_versions"`
		Loaders      []string `json:"loaders"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&project); err != nil {
		return nil, err
	}

	var mcVersion string
	if len(project.GameVersions) > 0 {
		mcVersion = project.GameVersions[0]
	}

	var loader LoaderType
	if len(project.Loaders) > 0 {
		loader = LoaderType(strings.ToLower(project.Loaders[0]))
	}

	modpack := &Modpack{
		Name:        project.Title,
		Identifier:  project.Slug,
		Description: project.Description,
		MCVersion:   mcVersion,
		Loader:      loader,
		Source:      "modrinth",
	}

	return modpack, nil
}

func (m *ModrinthClient) Search(query string) ([]*ModpackSearchResult, error) {
	searchURL := fmt.Sprintf("%s/search?query=%s&facets=[[\"project_type:modpack\"]]",
		ModrinthAPIURL, url.QueryEscape(query))

	resp, err := m.get(searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("modrinth api error: status %d", resp.StatusCode)
	}

	var result struct {
		Hits []struct {
			Slug         string   `json:"slug"`
			Title        string   `json:"title"`
			Description  string   `json:"description"`
			GameVersions []string `json:"game_versions"`
			Loaders      []string `json:"loaders"`
			Downloads    int      `json:"downloads"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var results []*ModpackSearchResult
	for _, hit := range result.Hits {
		var mcVersion string
		if len(hit.GameVersions) > 0 {
			mcVersion = hit.GameVersions[0]
		}

		var loader LoaderType
		if len(hit.Loaders) > 0 {
			loader = LoaderType(strings.ToLower(hit.Loaders[0]))
		}

		results = append(results, &ModpackSearchResult{
			Name:        hit.Title,
			Identifier:  fmt.Sprintf("modrinth:%s", hit.Slug),
			Description: hit.Description,
			MCVersion:   mcVersion,
			Loader:      loader,
			Source:      "modrinth",
			Downloads:   hit.Downloads,
		})
	}

	return results, nil
}

func (m *ModrinthClient) GetVersions(identifier string) ([]*Version, error) {
	slug := strings.TrimPrefix(identifier, "modrinth:")

	versionsURL := fmt.Sprintf("%s/project/%s/version", ModrinthAPIURL, url.PathEscape(slug))

	resp, err := m.get(versionsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("modrinth api error: status %d", resp.StatusCode)
	}

	var versionList []struct {
		VersionNumber string   `json:"version_number"`
		GameVersions  []string `json:"game_versions"`
		Loaders       []string `json:"loaders"`
		DatePublished string   `json:"date_published"`
		VersionType   string   `json:"version_type"`
		Files         []struct {
			URL string `json:"url"`
		} `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&versionList); err != nil {
		return nil, err
	}

	var versions []*Version
	for _, v := range versionList {
		var mcVersion string
		if len(v.GameVersions) > 0 {
			mcVersion = v.GameVersions[0]
		}

		var loader LoaderType
		if len(v.Loaders) > 0 {
			loader = LoaderType(strings.ToLower(v.Loaders[0]))
		}

		var downloadURL string
		if len(v.Files) > 0 {
			downloadURL = v.Files[0].URL
		}

		versions = append(versions, &Version{
			Version:     v.VersionNumber,
			MCVersion:   mcVersion,
			Loader:      loader,
			ReleaseDate: v.DatePublished,
			IsStable:    v.VersionType == "release",
			DownloadURL: downloadURL,
		})
	}

	return versions, nil
}

func (m *ModrinthClient) get(endpoint string) (*http.Response, error) {
	GetModrinthRateLimiter().Wait()

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "chunk-cli/0.1.0")

	if m.apiKey != "" {
		req.Header.Set("Authorization", m.apiKey)
	}

	return m.httpClient.Do(req)
}

func (m *ModrinthClient) DownloadFile(fileURL string, dest io.Writer) error {
	resp, err := m.httpClient.Get(fileURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}

	_, err = io.Copy(dest, resp.Body)
	return err
}
