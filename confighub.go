package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// Real ConfigHub API types based on actual source code

// Space represents a ConfigHub space
type Space struct {
	SpaceID        uuid.UUID         `json:"SpaceID,omitempty"`
	OrganizationID uuid.UUID         `json:"OrganizationID,omitempty"`
	Slug           string            `json:"Slug"`
	DisplayName    string            `json:"DisplayName,omitempty"`
	Labels         map[string]string `json:"Labels,omitempty"`
	Annotations    map[string]string `json:"Annotations,omitempty"`
	CreatedAt      time.Time         `json:"CreatedAt,omitempty"`
	UpdatedAt      time.Time         `json:"UpdatedAt,omitempty"`
	Version        int64             `json:"Version,omitempty"`
	EntityType     string            `json:"EntityType,omitempty"`
}

// Unit represents a ConfigHub configuration unit
type Unit struct {
	UnitID          uuid.UUID         `json:"UnitID,omitempty"`
	SpaceID         uuid.UUID         `json:"SpaceID,omitempty"`
	OrganizationID  uuid.UUID         `json:"OrganizationID,omitempty"`
	Slug            string            `json:"Slug"`
	DisplayName     string            `json:"DisplayName,omitempty"`
	Data            string            `json:"Data,omitempty"`
	Labels          map[string]string `json:"Labels,omitempty"`
	Annotations     map[string]string `json:"Annotations,omitempty"`
	UpstreamUnitID  *uuid.UUID        `json:"UpstreamUnitID,omitempty"` // For upstream/downstream
	SetIDs          []uuid.UUID       `json:"SetIDs,omitempty"`          // Sets this unit belongs to
	TargetID        *uuid.UUID        `json:"TargetID,omitempty"`
	BridgeWorkerID  *uuid.UUID        `json:"BridgeWorkerID,omitempty"`
	ApplyGates      map[string]bool   `json:"ApplyGates,omitempty"`
	CreatedAt       time.Time         `json:"CreatedAt,omitempty"`
	UpdatedAt       time.Time         `json:"UpdatedAt,omitempty"`
	Version         int64             `json:"Version,omitempty"`
	EntityType      string            `json:"EntityType,omitempty"`
}

// Set represents a group of related Units (REAL ConfigHub feature)
type Set struct {
	SetID          uuid.UUID         `json:"SetID,omitempty"`
	SpaceID        uuid.UUID         `json:"SpaceID,omitempty"`
	OrganizationID uuid.UUID         `json:"OrganizationID,omitempty"`
	Slug           string            `json:"Slug"`
	DisplayName    string            `json:"DisplayName,omitempty"`
	Labels         map[string]string `json:"Labels,omitempty"`
	Annotations    map[string]string `json:"Annotations,omitempty"`
	CreatedAt      time.Time         `json:"CreatedAt,omitempty"`
	UpdatedAt      time.Time         `json:"UpdatedAt,omitempty"`
	Version        int64             `json:"Version,omitempty"`
	EntityType     string            `json:"EntityType,omitempty"`
}

// Filter represents a ConfigHub filter with WHERE clauses (REAL feature)
type Filter struct {
	FilterID       uuid.UUID         `json:"FilterID,omitempty"`
	SpaceID        uuid.UUID         `json:"SpaceID,omitempty"`
	OrganizationID uuid.UUID         `json:"OrganizationID,omitempty"`
	Slug           string            `json:"Slug"`
	DisplayName    string            `json:"DisplayName,omitempty"`
	From           string            `json:"From"`           // Entity type to filter (e.g., "Unit")
	FromSpaceID    *uuid.UUID        `json:"FromSpaceID,omitempty"`
	Where          string            `json:"Where"`          // WHERE clause
	Select         []string          `json:"Select,omitempty"`
	Labels         map[string]string `json:"Labels,omitempty"`
	Annotations    map[string]string `json:"Annotations,omitempty"`
	Hash           string            `json:"Hash,omitempty"`
	CreatedAt      time.Time         `json:"CreatedAt,omitempty"`
	UpdatedAt      time.Time         `json:"UpdatedAt,omitempty"`
	Version        int64             `json:"Version,omitempty"`
	EntityType     string            `json:"EntityType,omitempty"`
}

// LiveState represents the live deployment state (READ-ONLY)
type LiveState struct {
	UnitID        uuid.UUID `json:"UnitID"`
	SpaceID       uuid.UUID `json:"SpaceID"`
	Status        string    `json:"Status"`
	DriftDetected bool      `json:"DriftDetected"`
	LastAppliedAt time.Time `json:"LastAppliedAt"`
	LastError     string    `json:"LastError,omitempty"`
}

// Target represents a deployment target
type Target struct {
	TargetID       uuid.UUID         `json:"TargetID,omitempty"`
	OrganizationID uuid.UUID         `json:"OrganizationID,omitempty"`
	Slug           string            `json:"Slug"`
	DisplayName    string            `json:"DisplayName,omitempty"`
	TargetType     string            `json:"TargetType"` // e.g., "kubernetes"
	Config         map[string]string `json:"Config,omitempty"`
	Labels         map[string]string `json:"Labels,omitempty"`
	Annotations    map[string]string `json:"Annotations,omitempty"`
	CreatedAt      time.Time         `json:"CreatedAt,omitempty"`
	UpdatedAt      time.Time         `json:"UpdatedAt,omitempty"`
	Version        int64             `json:"Version,omitempty"`
}

// Request/Response types

type CreateSpaceRequest struct {
	Slug        string            `json:"Slug"`
	DisplayName string            `json:"DisplayName,omitempty"`
	Labels      map[string]string `json:"Labels,omitempty"`
	Annotations map[string]string `json:"Annotations,omitempty"`
}

type CreateUnitRequest struct {
	Slug           string            `json:"Slug"`
	DisplayName    string            `json:"DisplayName,omitempty"`
	Data           string            `json:"Data"`
	Labels         map[string]string `json:"Labels,omitempty"`
	Annotations    map[string]string `json:"Annotations,omitempty"`
	UpstreamUnitID *uuid.UUID        `json:"UpstreamUnitID,omitempty"`
	SetIDs         []uuid.UUID       `json:"SetIDs,omitempty"`
	TargetID       *uuid.UUID        `json:"TargetID,omitempty"`
}

type CreateSetRequest struct {
	Slug        string            `json:"Slug"`
	DisplayName string            `json:"DisplayName,omitempty"`
	Labels      map[string]string `json:"Labels,omitempty"`
	Annotations map[string]string `json:"Annotations,omitempty"`
}

type CreateFilterRequest struct {
	Slug        string            `json:"Slug"`
	DisplayName string            `json:"DisplayName,omitempty"`
	From        string            `json:"From"`  // "Unit", "Space", etc.
	Where       string            `json:"Where"` // WHERE clause
	Select      []string          `json:"Select,omitempty"`
	Labels      map[string]string `json:"Labels,omitempty"`
	Annotations map[string]string `json:"Annotations,omitempty"`
}

type ListUnitsParams struct {
	SpaceID  uuid.UUID  `json:"SpaceID,omitempty"`
	FilterID *uuid.UUID `json:"FilterID,omitempty"`
	SetID    *uuid.UUID `json:"SetID,omitempty"`
	Where    string     `json:"Where,omitempty"`
	Limit    int        `json:"Limit,omitempty"`
	Offset   int        `json:"Offset,omitempty"`
}

type BulkApplyParams struct {
	SpaceID uuid.UUID `json:"SpaceID"`
	Where   string    `json:"Where"` // e.g., "SetID = 'xxx'"
	DryRun  bool      `json:"DryRun,omitempty"`
}

type BulkPatchParams struct {
	SpaceID uuid.UUID         `json:"SpaceID"`
	Where   string            `json:"Where"`
	Patch   map[string]interface{} `json:"Patch"`
	Upgrade bool              `json:"Upgrade,omitempty"` // For push-upgrade pattern
}

// ConfigHubClient provides interface to real ConfigHub API
type ConfigHubClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewConfigHubClient creates a new ConfigHub API client
func NewConfigHubClient(baseURL, token string) *ConfigHubClient {
	if baseURL == "" {
		// Use environment variable or default to ConfigHub API
		baseURL = os.Getenv("CUB_API_URL")
		if baseURL == "" {
			baseURL = "https://hub.confighub.com/api"
		}
	}

	return &ConfigHubClient{
		baseURL: baseURL,
		token:   token,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Space operations

func (c *ConfigHubClient) CreateSpace(req CreateSpaceRequest) (*Space, error) {
	result, err := c.doRequest("POST", "/space", req, &Space{})
	if err != nil {
		return nil, err
	}
	return result.(*Space), nil
}

func (c *ConfigHubClient) GetSpace(spaceID uuid.UUID) (*Space, error) {
	result, err := c.doRequest("GET", fmt.Sprintf("/space/%s", spaceID), nil, &Space{})
	if err != nil {
		return nil, err
	}
	return result.(*Space), nil
}

func (c *ConfigHubClient) ListSpaces() ([]*Space, error) {
	var spaces []*Space
	return spaces, c.doRequestList("GET", "/space", nil, &spaces)
}

// Unit operations

func (c *ConfigHubClient) CreateUnit(spaceID uuid.UUID, req CreateUnitRequest) (*Unit, error) {
	result, err := c.doRequest("POST", fmt.Sprintf("/space/%s/unit", spaceID), req, &Unit{})
	if err != nil {
		return nil, err
	}
	return result.(*Unit), nil
}

func (c *ConfigHubClient) GetUnit(spaceID, unitID uuid.UUID) (*Unit, error) {
	result, err := c.doRequest("GET", fmt.Sprintf("/space/%s/unit/%s", spaceID, unitID), nil, &Unit{})
	if err != nil {
		return nil, err
	}
	return result.(*Unit), nil
}

func (c *ConfigHubClient) UpdateUnit(spaceID, unitID uuid.UUID, req CreateUnitRequest) (*Unit, error) {
	result, err := c.doRequest("PUT", fmt.Sprintf("/space/%s/unit/%s", spaceID, unitID), req, &Unit{})
	if err != nil {
		return nil, err
	}
	return result.(*Unit), nil
}

func (c *ConfigHubClient) ListUnits(params ListUnitsParams) ([]*Unit, error) {
	var units []*Unit
	endpoint := fmt.Sprintf("/space/%s/unit", params.SpaceID)
	if params.Where != "" {
		endpoint += fmt.Sprintf("?where=%s", params.Where)
	}
	return units, c.doRequestList("GET", endpoint, nil, &units)
}

func (c *ConfigHubClient) ApplyUnit(spaceID, unitID uuid.UUID) error {
	_, err := c.doRequest("POST", fmt.Sprintf("/space/%s/unit/%s/apply", spaceID, unitID), nil, nil)
	return err
}

func (c *ConfigHubClient) DestroyUnit(spaceID, unitID uuid.UUID) error {
	_, err := c.doRequest("POST", fmt.Sprintf("/space/%s/unit/%s/destroy", spaceID, unitID), nil, nil)
	return err
}

// Set operations (REAL)

func (c *ConfigHubClient) CreateSet(spaceID uuid.UUID, req CreateSetRequest) (*Set, error) {
	result, err := c.doRequest("POST", fmt.Sprintf("/space/%s/set", spaceID), req, &Set{})
	if err != nil {
		return nil, err
	}
	return result.(*Set), nil
}

func (c *ConfigHubClient) GetSet(spaceID, setID uuid.UUID) (*Set, error) {
	result, err := c.doRequest("GET", fmt.Sprintf("/space/%s/set/%s", spaceID, setID), nil, &Set{})
	if err != nil {
		return nil, err
	}
	return result.(*Set), nil
}

func (c *ConfigHubClient) UpdateSet(spaceID, setID uuid.UUID, req CreateSetRequest) (*Set, error) {
	result, err := c.doRequest("PUT", fmt.Sprintf("/space/%s/set/%s", spaceID, setID), req, &Set{})
	if err != nil {
		return nil, err
	}
	return result.(*Set), nil
}

func (c *ConfigHubClient) ListSets(spaceID uuid.UUID) ([]*Set, error) {
	var sets []*Set
	return sets, c.doRequestList("GET", fmt.Sprintf("/space/%s/set", spaceID), nil, &sets)
}

// Filter operations (REAL)

func (c *ConfigHubClient) CreateFilter(spaceID uuid.UUID, req CreateFilterRequest) (*Filter, error) {
	result, err := c.doRequest("POST", fmt.Sprintf("/space/%s/filter", spaceID), req, &Filter{})
	if err != nil {
		return nil, err
	}
	return result.(*Filter), nil
}

func (c *ConfigHubClient) GetFilter(spaceID, filterID uuid.UUID) (*Filter, error) {
	result, err := c.doRequest("GET", fmt.Sprintf("/space/%s/filter/%s", spaceID, filterID), nil, &Filter{})
	if err != nil {
		return nil, err
	}
	return result.(*Filter), nil
}

// Bulk operations (REAL)

func (c *ConfigHubClient) BulkApplyUnits(params BulkApplyParams) error {
	_, err := c.doRequest("POST", fmt.Sprintf("/space/%s/unit/bulk-apply", params.SpaceID), params, nil)
	return err
}

func (c *ConfigHubClient) BulkPatchUnits(params BulkPatchParams) error {
	_, err := c.doRequest("PATCH", fmt.Sprintf("/space/%s/unit/bulk-patch", params.SpaceID), params, nil)
	return err
}

// Live State (READ-ONLY)

func (c *ConfigHubClient) GetUnitLiveState(spaceID, unitID uuid.UUID) (*LiveState, error) {
	result, err := c.doRequest("GET", fmt.Sprintf("/space/%s/unit/%s/live-state", spaceID, unitID), nil, &LiveState{})
	if err != nil {
		return nil, err
	}
	return result.(*LiveState), nil
}

// Target operations

func (c *ConfigHubClient) CreateTarget(req Target) (*Target, error) {
	result, err := c.doRequest("POST", "/target", req, &Target{})
	if err != nil {
		return nil, err
	}
	return result.(*Target), nil
}

func (c *ConfigHubClient) GetTarget(targetID uuid.UUID) (*Target, error) {
	result, err := c.doRequest("GET", fmt.Sprintf("/target/%s", targetID), nil, &Target{})
	if err != nil {
		return nil, err
	}
	return result.(*Target), nil
}

// Helper methods

func (c *ConfigHubClient) doRequest(method, endpoint string, body interface{}, result interface{}) (interface{}, error) {
	url := c.baseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return nil, fmt.Errorf("unmarshal response: %w", err)
		}
		return result, nil
	}

	return nil, nil
}

func (c *ConfigHubClient) doRequestList(method, endpoint string, body interface{}, result interface{}) error {
	url := c.baseURL + endpoint

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, reqBody)
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

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	if len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}

	return nil
}