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
	UnitID         uuid.UUID         `json:"UnitID,omitempty"`
	SpaceID        uuid.UUID         `json:"SpaceID,omitempty"`
	OrganizationID uuid.UUID         `json:"OrganizationID,omitempty"`
	Slug           string            `json:"Slug"`
	DisplayName    string            `json:"DisplayName,omitempty"`
	Data           string            `json:"Data,omitempty"`
	Labels         map[string]string `json:"Labels,omitempty"`
	Annotations    map[string]string `json:"Annotations,omitempty"`
	UpstreamUnitID *uuid.UUID        `json:"UpstreamUnitID,omitempty"` // For upstream/downstream
	SetIDs         []uuid.UUID       `json:"SetIDs,omitempty"`         // Sets this unit belongs to
	TargetID       *uuid.UUID        `json:"TargetID,omitempty"`
	BridgeWorkerID *uuid.UUID        `json:"BridgeWorkerID,omitempty"`
	ApplyGates     map[string]bool   `json:"ApplyGates,omitempty"`
	CreatedAt      time.Time         `json:"CreatedAt,omitempty"`
	UpdatedAt      time.Time         `json:"UpdatedAt,omitempty"`
	Version        int64             `json:"Version,omitempty"`
	EntityType     string            `json:"EntityType,omitempty"`
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
	From           string            `json:"From"` // Entity type to filter (e.g., "Unit")
	FromSpaceID    *uuid.UUID        `json:"FromSpaceID,omitempty"`
	Where          string            `json:"Where"` // WHERE clause
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
	ChangeSetID    *uuid.UUID        `json:"ChangeSetID,omitempty"`
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
	SpaceID uuid.UUID              `json:"SpaceID"`
	Where   string                 `json:"Where"`
	Patch   map[string]interface{} `json:"Patch"`
	Upgrade bool                   `json:"Upgrade,omitempty"` // For push-upgrade pattern
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

func (c *ConfigHubClient) DeleteSpace(spaceID uuid.UUID) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/space/%s", spaceID), nil, nil)
	return err
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

// GetNewSpacePrefix calls ConfigHub to generate a unique space prefix
// Returns something like "chubby-paws" or "whisker-tail"
func (c *ConfigHubClient) GetNewSpacePrefix() (string, error) {
	// This would typically call: cub space new-prefix
	// Since we don't have direct CLI access, we'd need to call the API endpoint
	// For now, this is a placeholder that would need the actual API endpoint

	// In practice, this would be:
	// result, err := c.doRequest("POST", "/space/new-prefix", nil, &struct{Prefix string})
	// return result.Prefix, err

	// For demonstration, generate a readable prefix
	adjectives := []string{"happy", "clever", "swift", "bright", "gentle"}
	nouns := []string{"paws", "tail", "whisker", "cloud", "star"}

	adj := adjectives[time.Now().UnixNano()%int64(len(adjectives))]
	noun := nouns[time.Now().UnixNano()%int64(len(nouns))]

	return fmt.Sprintf("%s-%s", adj, noun), nil
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

// High-level convenience helpers

// GetSpaceBySlug finds a space by its slug name
func (c *ConfigHubClient) GetSpaceBySlug(slug string) (*Space, error) {
	spaces, err := c.ListSpaces()
	if err != nil {
		return nil, fmt.Errorf("list spaces: %w", err)
	}

	// Filter by slug
	for i, space := range spaces {
		if space.Slug == slug {
			return spaces[i], nil
		}
	}

	return nil, fmt.Errorf("space not found: %s", slug)
}

// CreateSpaceWithUniquePrefix creates a space with a unique prefix + suffix
func (c *ConfigHubClient) CreateSpaceWithUniquePrefix(suffix string, displayName string, labels map[string]string) (*Space, string, error) {
	prefix, err := c.GetNewSpacePrefix()
	if err != nil {
		return nil, "", fmt.Errorf("get unique prefix: %w", err)
	}

	slug := fmt.Sprintf("%s-%s", prefix, suffix)
	space, err := c.CreateSpace(CreateSpaceRequest{
		Slug:        slug,
		DisplayName: displayName,
		Labels:      labels,
	})
	if err != nil {
		return nil, "", fmt.Errorf("create space: %w", err)
	}

	return space, slug, nil
}

// EnsureSpaceRecreated implements the delete-then-create pattern for spaces.
// If a space with the given slug exists, it deletes it completely first.
// Then creates a fresh space with the same slug.
// This ensures we always start with a clean slate and avoid stale configurations.
func (c *ConfigHubClient) EnsureSpaceRecreated(req CreateSpaceRequest) (*Space, error) {
	// First, try to find existing space by slug
	existingSpace, err := c.GetSpaceBySlug(req.Slug)
	if err == nil && existingSpace != nil {
		// Space exists, delete it first
		fmt.Printf("Deleting existing space: %s\n", req.Slug)
		if err := c.DeleteSpace(existingSpace.SpaceID); err != nil {
			return nil, fmt.Errorf("delete existing space %s: %w", req.Slug, err)
		}
		fmt.Printf("Successfully deleted space: %s\n", req.Slug)
	}

	// Now create the space (whether it's new or we just deleted the old one)
	fmt.Printf("Creating space: %s\n", req.Slug)
	space, err := c.CreateSpace(req)
	if err != nil {
		return nil, fmt.Errorf("create space %s: %w", req.Slug, err)
	}

	fmt.Printf("Successfully created space: %s\n", req.Slug)
	return space, nil
}

// CloneUnitWithUpstream creates a unit in the target space with an upstream relationship
func (c *ConfigHubClient) CloneUnitWithUpstream(sourceSpaceID, targetSpaceID uuid.UUID, unitSlug string, additionalLabels map[string]string) (*Unit, error) {
	// Get the source unit
	sourceUnits, err := c.ListUnits(ListUnitsParams{
		SpaceID: sourceSpaceID,
		Where:   fmt.Sprintf("Slug = '%s'", unitSlug),
	})
	if err != nil {
		return nil, fmt.Errorf("list source units: %w", err)
	}

	if len(sourceUnits) == 0 {
		return nil, fmt.Errorf("source unit not found: %s", unitSlug)
	}

	sourceUnit := sourceUnits[0]

	// Merge labels
	labels := make(map[string]string)
	for k, v := range sourceUnit.Labels {
		labels[k] = v
	}
	for k, v := range additionalLabels {
		labels[k] = v
	}

	// Create downstream unit with upstream relationship
	return c.CreateUnit(targetSpaceID, CreateUnitRequest{
		Slug:           sourceUnit.Slug,
		DisplayName:    sourceUnit.DisplayName,
		Data:           sourceUnit.Data,
		Labels:         labels,
		UpstreamUnitID: &sourceUnit.UnitID,
	})
}

// BulkCloneUnitsWithUpstream clones multiple units from source to target space
func (c *ConfigHubClient) BulkCloneUnitsWithUpstream(sourceSpaceID, targetSpaceID uuid.UUID, unitSlugs []string, additionalLabels map[string]string) ([]*Unit, error) {
	var clonedUnits []*Unit

	for _, slug := range unitSlugs {
		unit, err := c.CloneUnitWithUpstream(sourceSpaceID, targetSpaceID, slug, additionalLabels)
		if err != nil {
			return nil, fmt.Errorf("clone unit %s: %w", slug, err)
		}
		clonedUnits = append(clonedUnits, unit)
	}

	return clonedUnits, nil
}

// ApplyUnitsInOrder applies units in the correct dependency order
func (c *ConfigHubClient) ApplyUnitsInOrder(spaceID uuid.UUID, unitSlugs []string) error {
	for _, slug := range unitSlugs {
		units, err := c.ListUnits(ListUnitsParams{
			SpaceID: spaceID,
			Where:   fmt.Sprintf("Slug = '%s'", slug),
		})
		if err != nil {
			return fmt.Errorf("list units for %s: %w", slug, err)
		}

		if len(units) > 0 {
			err = c.ApplyUnit(spaceID, units[0].UnitID)
			if err != nil {
				return fmt.Errorf("apply unit %s: %w", slug, err)
			}
		}
	}

	return nil
}

// ListFilters lists filters in a space
// TODO: Implement when ConfigHub API supports filter listing
func (c *ConfigHubClient) ListFilters(spaceID uuid.UUID) ([]*Filter, error) {
	// Placeholder implementation - would call actual ConfigHub API
	return []*Filter{}, nil
}

// FunctionInvocationRequest represents a request to invoke a ConfigHub function
type FunctionInvocationRequest struct {
	FunctionName     string                   `json:"FunctionName"`
	ToolchainType    string                   `json:"ToolchainType"`
	Arguments        []FunctionArgument       `json:"Arguments,omitempty"`
	Where            string                   `json:"Where,omitempty"`
	FilterID         *uuid.UUID               `json:"FilterID,omitempty"`
	DryRun           bool                     `json:"DryRun"`
	ChangeSetID      *uuid.UUID               `json:"ChangeSetID,omitempty"`
}

type FunctionArgument struct {
	ParameterName string      `json:"ParameterName"`
	Value         interface{} `json:"Value"`
}

type FunctionInvocationResponse struct {
	Results []FunctionResult `json:"Results"`
}

type FunctionResult struct {
	UnitID       uuid.UUID              `json:"UnitID"`
	UnitSlug     string                 `json:"UnitSlug"`
	Success      bool                   `json:"Success"`
	Error        string                 `json:"Error,omitempty"`
	Output       interface{}            `json:"Output,omitempty"`
	Value        interface{}            `json:"Value,omitempty"`
	Passed       bool                   `json:"Passed,omitempty"` // For validation functions
}

// ExecuteFunction runs a ConfigHub function on units
func (c *ConfigHubClient) ExecuteFunction(spaceID uuid.UUID, req FunctionInvocationRequest) (*FunctionInvocationResponse, error) {
	endpoint := fmt.Sprintf("/space/%s/function/invoke", spaceID)
	var result FunctionInvocationResponse
	err := c.doRequest("POST", endpoint, req, &result)
	return &result, err
}

// SetImageVersion uses the set-image function to update container image
func (c *ConfigHubClient) SetImageVersion(spaceID, unitID uuid.UUID, containerName, image string) error {
	req := FunctionInvocationRequest{
		FunctionName:  "set-image",
		ToolchainType: "Kubernetes/YAML",
		Where:         fmt.Sprintf("UnitID = '%s'", unitID),
		Arguments: []FunctionArgument{
			{ParameterName: "container-name", Value: containerName},
			{ParameterName: "image", Value: image},
		},
	}
	_, err := c.ExecuteFunction(spaceID, req)
	return err
}

// SetReplicas uses the set-replicas function to update replica count
func (c *ConfigHubClient) SetReplicas(spaceID, unitID uuid.UUID, replicas int) error {
	req := FunctionInvocationRequest{
		FunctionName:  "set-replicas",
		ToolchainType: "Kubernetes/YAML",
		Where:         fmt.Sprintf("UnitID = '%s'", unitID),
		Arguments: []FunctionArgument{
			{ParameterName: "replicas", Value: replicas},
		},
	}
	_, err := c.ExecuteFunction(spaceID, req)
	return err
}

// ListWorkers lists workers in a space (placeholder for PRINCIPLE #1 requirement)
// TODO: Implement when ConfigHub API supports worker listing
func (c *ConfigHubClient) ListWorkers(spaceID string) ([]interface{}, error) {
	// Placeholder implementation - ConfigHub worker API not yet available
	// In production, this would call the actual ConfigHub API
	// For now, return empty to trigger health check warnings
	return []interface{}{}, nil
}

// ListTargets lists targets in a space (placeholder for PRINCIPLE #4 requirement)
// TODO: Implement when ConfigHub API supports target listing
func (c *ConfigHubClient) ListTargets(spaceID string) ([]interface{}, error) {
	// Placeholder implementation - ConfigHub target API not yet available
	// In production, this would call the actual ConfigHub API
	// For now, return empty to trigger health check warnings
	return []interface{}{}, nil
}

// ChangeSet operations for grouping related changes

type ChangeSet struct {
	ChangeSetID uuid.UUID         `json:"changeSetId"`
	SpaceID     uuid.UUID         `json:"spaceId"`
	DisplayName string            `json:"displayName"`
	Description string            `json:"description"`
	CreatedAt   string            `json:"createdAt"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type CreateChangeSetRequest struct {
	DisplayName string            `json:"displayName"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// CreateChangeSet creates a new ChangeSet for grouping related changes
func (c *ConfigHubClient) CreateChangeSet(spaceID uuid.UUID, req CreateChangeSetRequest) (*ChangeSet, error) {
	result, err := c.doRequest("POST", fmt.Sprintf("/space/%s/changeset", spaceID), req, &ChangeSet{})
	if err != nil {
		return nil, err
	}
	return result.(*ChangeSet), nil
}

// GetChangeSet retrieves a ChangeSet
func (c *ConfigHubClient) GetChangeSet(spaceID, changeSetID uuid.UUID) (*ChangeSet, error) {
	result, err := c.doRequest("GET", fmt.Sprintf("/space/%s/changeset/%s", spaceID, changeSetID), nil, &ChangeSet{})
	if err != nil {
		return nil, err
	}
	return result.(*ChangeSet), nil
}

// DeleteChangeSet deletes a ChangeSet
func (c *ConfigHubClient) DeleteChangeSet(spaceID, changeSetID uuid.UUID) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/space/%s/changeset/%s", spaceID, changeSetID), nil, nil)
	return err
}

// ApplyChangeSet applies all changes in a ChangeSet
func (c *ConfigHubClient) ApplyChangeSet(spaceID, changeSetID uuid.UUID) error {
	_, err := c.doRequest("POST", fmt.Sprintf("/space/%s/changeset/%s/apply", spaceID, changeSetID), nil, nil)
	return err
}

// UpdateUnitWithChangeSet updates a unit and associates it with a ChangeSet
func (c *ConfigHubClient) UpdateUnitWithChangeSet(spaceID, unitID, changeSetID uuid.UUID, data interface{}) (*Unit, error) {
	req := CreateUnitRequest{
		Data:        data,
		ChangeSetID: &changeSetID,
	}
	return c.UpdateUnit(spaceID, unitID, req)
}

// Validation Functions - Use ConfigHub's built-in validation

// ValidateNoPlaceholders checks if a unit has any unresolved placeholders
func (c *ConfigHubClient) ValidateNoPlaceholders(spaceID, unitID uuid.UUID) (bool, string, error) {
	req := FunctionInvocationRequest{
		FunctionName:  "no-placeholders",
		ToolchainType: "Kubernetes/YAML",
		Where:         fmt.Sprintf("UnitID = '%s'", unitID),
	}
	result, err := c.ExecuteFunction(spaceID, req)
	if err != nil {
		return false, "", err
	}

	if len(result.Results) > 0 && result.Results[0].Success {
		return result.Results[0].Passed, fmt.Sprintf("Unit %s validation", result.Results[0].UnitSlug), nil
	}
	return false, "Validation failed", nil
}

// ValidateCEL validates units against a CEL (Common Expression Language) expression
func (c *ConfigHubClient) ValidateCEL(spaceID uuid.UUID, where, expression string) ([]FunctionResult, error) {
	req := FunctionInvocationRequest{
		FunctionName:  "cel-validate",
		ToolchainType: "Kubernetes/YAML",
		Where:         where,
		Arguments: []FunctionArgument{
			{ParameterName: "expression", Value: expression},
		},
	}
	result, err := c.ExecuteFunction(spaceID, req)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// GetReplicas uses the get-replicas function to retrieve replica counts
func (c *ConfigHubClient) GetReplicas(spaceID uuid.UUID, where string) ([]FunctionResult, error) {
	req := FunctionInvocationRequest{
		FunctionName:  "get-replicas",
		ToolchainType: "Kubernetes/YAML",
		Where:         where,
	}
	result, err := c.ExecuteFunction(spaceID, req)
	if err != nil {
		return nil, err
	}
	return result.Results, nil
}

// SetIntPath sets an integer value at a specific path in the configuration
func (c *ConfigHubClient) SetIntPath(spaceID, unitID uuid.UUID, apiVersion, kind, path string, value int) error {
	req := FunctionInvocationRequest{
		FunctionName:  "set-int-path",
		ToolchainType: "Kubernetes/YAML",
		Where:         fmt.Sprintf("UnitID = '%s'", unitID),
		Arguments: []FunctionArgument{
			{ParameterName: "apiVersion", Value: apiVersion},
			{ParameterName: "kind", Value: kind},
			{ParameterName: "path", Value: path},
			{ParameterName: "value", Value: value},
		},
	}
	_, err := c.ExecuteFunction(spaceID, req)
	return err
}
