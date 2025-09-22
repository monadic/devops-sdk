package sdk

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// CubClient provides interface to ConfigHub API
type CubClient struct {
    baseURL string
    token   string
    client  *http.Client
}

// Unit represents a ConfigHub unit
type Unit struct {
    Name      string                 `json:"name"`
    Space     string                 `json:"space"`
    Data      map[string]interface{} `json:"data"`
    Labels    map[string]string      `json:"labels"`
    Target    string                 `json:"target,omitempty"`
    Namespace string                 `json:"namespace,omitempty"`
}

// Space represents a ConfigHub space
type Space struct {
    Slug        string            `json:"slug"`
    Name        string            `json:"name"`
    Parent      string            `json:"parent,omitempty"`
    Labels      map[string]string `json:"labels,omitempty"`
    Description string            `json:"description,omitempty"`
}

// NewCubClient creates a new ConfigHub API client
func NewCubClient(baseURL, token string) *CubClient {
    if baseURL == "" {
        baseURL = "https://hub.confighub.com/api/v1"
    }

    return &CubClient{
        baseURL: baseURL,
        token:   token,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// GetUnits retrieves units from a space
func (c *CubClient) GetUnits(space string) ([]Unit, error) {
    url := fmt.Sprintf("%s/spaces/%s/units", c.baseURL, space)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("send request: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read response: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
    }

    var units []Unit
    if err := json.Unmarshal(body, &units); err != nil {
        return nil, fmt.Errorf("unmarshal units: %w", err)
    }

    return units, nil
}

// GetUnit retrieves a specific unit
func (c *CubClient) GetUnit(space, name string) (*Unit, error) {
    url := fmt.Sprintf("%s/spaces/%s/units/%s", c.baseURL, space, name)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("send request: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read response: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
    }

    var unit Unit
    if err := json.Unmarshal(body, &unit); err != nil {
        return nil, fmt.Errorf("unmarshal unit: %w", err)
    }

    return &unit, nil
}

// UpdateUnit updates a unit in a space
func (c *CubClient) UpdateUnit(space string, unit Unit) error {
    url := fmt.Sprintf("%s/spaces/%s/units/%s", c.baseURL, space, unit.Name)

    jsonData, err := json.Marshal(unit)
    if err != nil {
        return fmt.Errorf("marshal unit: %w", err)
    }

    req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("send request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
    }

    return nil
}

// CreateSpace creates a new ConfigHub space
func (c *CubClient) CreateSpace(space Space) error {
    url := fmt.Sprintf("%s/spaces", c.baseURL)

    jsonData, err := json.Marshal(space)
    if err != nil {
        return fmt.Errorf("marshal space: %w", err)
    }

    req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("create request: %w", err)
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("send request: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
        body, _ := io.ReadAll(resp.Body)
        return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
    }

    return nil
}

// GetSpace retrieves space information
func (c *CubClient) GetSpace(slug string) (*Space, error) {
    url := fmt.Sprintf("%s/spaces/%s", c.baseURL, slug)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return nil, fmt.Errorf("create request: %w", err)
    }

    req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.token))
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("send request: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read response: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
    }

    var space Space
    if err := json.Unmarshal(body, &space); err != nil {
        return nil, fmt.Errorf("unmarshal space: %w", err)
    }

    return &space, nil
}