package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	DefaultChunkHubURL = "https://api.chunkhub.io"
)

type ChunkHubClient struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
}

func NewChunkHubClient(baseURL string) *ChunkHubClient {
	if baseURL == "" {
		baseURL = DefaultChunkHubURL
	}
	
	return &ChunkHubClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *ChunkHubClient) SetAPIKey(apiKey string) {
	c.apiKey = apiKey
}

func (c *ChunkHubClient) Fetch(identifier string) (*Modpack, error) {
	endpoint := fmt.Sprintf("%s/v1/modpacks/%s", c.baseURL, url.PathEscape(identifier))
	
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chunkhub api error: status %d", resp.StatusCode)
	}
	
	var result struct {
		Data *Modpack `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	if result.Data == nil {
		return nil, ErrNotFound
	}
	
	result.Data.Source = "chunkhub"
	return result.Data, nil
}

func (c *ChunkHubClient) Search(query string) ([]*ModpackSearchResult, error) {
	endpoint := fmt.Sprintf("%s/v1/modpacks/search?q=%s", c.baseURL, url.QueryEscape(query))
	
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chunkhub api error: status %d", resp.StatusCode)
	}
	
	var result struct {
		Data []*ModpackSearchResult `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	for _, item := range result.Data {
		item.Source = "chunkhub"
	}
	
	return result.Data, nil
}

func (c *ChunkHubClient) GetVersions(identifier string) ([]*Version, error) {
	endpoint := fmt.Sprintf("%s/v1/modpacks/%s/versions", c.baseURL, url.PathEscape(identifier))
	
	resp, err := c.get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chunkhub api error: status %d", resp.StatusCode)
	}
	
	var result struct {
		Data []*Version `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return result.Data, nil
}

func (c *ChunkHubClient) get(endpoint string) (*http.Response, error) {
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("User-Agent", "chunk-cli/0.1.0")
	req.Header.Set("Accept", "application/json")
	
	if c.apiKey != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}
	
	return c.httpClient.Do(req)
}

func (c *ChunkHubClient) DownloadFile(fileURL string, dest io.Writer) error {
	resp, err := c.httpClient.Get(fileURL)
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
