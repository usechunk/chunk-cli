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
	GitHubAPIURL = "https://api.github.com"
)

type GitHubClient struct {
	httpClient *http.Client
	token      string
}

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (g *GitHubClient) SetToken(token string) {
	g.token = token
}

func (g *GitHubClient) Fetch(identifier string) (*Modpack, error) {
	owner, repo, err := parseGitHubIdentifier(identifier)
	if err != nil {
		return nil, err
	}
	
	chunkJSON, err := g.fetchChunkJSON(owner, repo)
	if err != nil {
		return nil, err
	}
	
	modpack := &Modpack{
		Name:           chunkJSON.Name,
		Identifier:     identifier,
		Description:    chunkJSON.Description,
		MCVersion:      chunkJSON.MCVersion,
		Loader:         LoaderType(chunkJSON.Loader),
		LoaderVersion:  chunkJSON.LoaderVersion,
		Author:         owner,
		Source:         "github",
		RecommendedRAM: chunkJSON.RecommendedRAMGB,
		Dependencies:   chunkJSON.Dependencies,
	}
	
	return modpack, nil
}

func (g *GitHubClient) Search(query string) ([]*ModpackSearchResult, error) {
	searchURL := fmt.Sprintf("%s/search/repositories?q=%s+.chunk.json+in:repo", 
		GitHubAPIURL, url.QueryEscape(query))
	
	resp, err := g.get(searchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: status %d", resp.StatusCode)
	}
	
	var result struct {
		Items []struct {
			FullName    string `json:"full_name"`
			Description string `json:"description"`
			Owner       struct {
				Login string `json:"login"`
			} `json:"owner"`
		} `json:"items"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	var results []*ModpackSearchResult
	for _, item := range result.Items {
		owner, repo, _ := parseGitHubIdentifier(item.FullName)
		
		chunkJSON, err := g.fetchChunkJSON(owner, repo)
		if err != nil {
			continue
		}
		
		results = append(results, &ModpackSearchResult{
			Name:        chunkJSON.Name,
			Identifier:  item.FullName,
			Description: item.Description,
			MCVersion:   chunkJSON.MCVersion,
			Loader:      LoaderType(chunkJSON.Loader),
			Source:      "github",
		})
	}
	
	return results, nil
}

func (g *GitHubClient) GetVersions(identifier string) ([]*Version, error) {
	owner, repo, err := parseGitHubIdentifier(identifier)
	if err != nil {
		return nil, err
	}
	
	tagsURL := fmt.Sprintf("%s/repos/%s/%s/tags", GitHubAPIURL, owner, repo)
	
	resp, err := g.get(tagsURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: status %d", resp.StatusCode)
	}
	
	var tags []struct {
		Name   string `json:"name"`
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, err
	}
	
	var versions []*Version
	for _, tag := range tags {
		versions = append(versions, &Version{
			Version:  tag.Name,
			IsStable: !strings.Contains(strings.ToLower(tag.Name), "beta") && 
			          !strings.Contains(strings.ToLower(tag.Name), "alpha"),
		})
	}
	
	return versions, nil
}

func (g *GitHubClient) fetchChunkJSON(owner, repo string) (*ChunkManifest, error) {
	contentURL := fmt.Sprintf("%s/repos/%s/%s/contents/.chunk.json", 
		GitHubAPIURL, owner, repo)
	
	resp, err := g.get(contentURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("no .chunk.json found in repository")
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api error: status %d", resp.StatusCode)
	}
	
	var content struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&content); err != nil {
		return nil, err
	}
	
	if content.Encoding != "base64" {
		return nil, fmt.Errorf("unexpected encoding: %s", content.Encoding)
	}
	
	decoded, err := decodeBase64(content.Content)
	if err != nil {
		return nil, err
	}
	
	var manifest ChunkManifest
	if err := json.Unmarshal([]byte(decoded), &manifest); err != nil {
		return nil, fmt.Errorf("invalid .chunk.json: %w", err)
	}
	
	return &manifest, nil
}

func (g *GitHubClient) get(endpoint string) (*http.Response, error) {
	GetGitHubRateLimiter().Wait()
	
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("User-Agent", "chunk-cli/0.1.0")
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	if g.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.token))
	}
	
	return g.httpClient.Do(req)
}

func parseGitHubIdentifier(identifier string) (owner, repo string, err error) {
	parts := strings.Split(identifier, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid github identifier: expected owner/repo format")
	}
	return parts[0], parts[1], nil
}

func decodeBase64(encoded string) (string, error) {
	encoded = strings.ReplaceAll(encoded, "\n", "")
	decoded := make([]byte, len(encoded))
	n, err := base64Decode(encoded, decoded)
	if err != nil {
		return "", err
	}
	return string(decoded[:n]), nil
}

func base64Decode(src string, dst []byte) (int, error) {
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	
	si := 0
	di := 0
	
	for si < len(src) {
		var dbuf [4]byte
		dlen := 0
		
		for dlen < 4 && si < len(src) {
			c := src[si]
			si++
			
			if c == '=' {
				break
			}
			
			if c >= 'A' && c <= 'Z' {
				dbuf[dlen] = c - 'A'
			} else if c >= 'a' && c <= 'z' {
				dbuf[dlen] = c - 'a' + 26
			} else if c >= '0' && c <= '9' {
				dbuf[dlen] = c - '0' + 52
			} else if c == '+' {
				dbuf[dlen] = 62
			} else if c == '/' {
				dbuf[dlen] = 63
			} else {
				continue
			}
			dlen++
		}
		
		if dlen >= 2 {
			dst[di] = (dbuf[0] << 2) | (dbuf[1] >> 4)
			di++
		}
		if dlen >= 3 {
			dst[di] = (dbuf[1] << 4) | (dbuf[2] >> 2)
			di++
		}
		if dlen >= 4 {
			dst[di] = (dbuf[2] << 6) | dbuf[3]
			di++
		}
	}
	
	return di, nil
}

type ChunkManifest struct {
	Name             string   `json:"name"`
	Description      string   `json:"description,omitempty"`
	MCVersion        string   `json:"mc_version"`
	Loader           string   `json:"loader"`
	LoaderVersion    string   `json:"loader_version,omitempty"`
	RecommendedRAMGB int      `json:"recommended_ram_gb"`
	Dependencies     []string `json:"dependencies,omitempty"`
}
