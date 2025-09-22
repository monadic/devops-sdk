package sdk

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DeploymentHelper assists with ConfigHub-based deployments
type DeploymentHelper struct {
	Cub         *ConfigHubClient
	ProjectName string
	AppName     string
}

// NewDeploymentHelper creates a deployment helper for a DevOps app
func NewDeploymentHelper(cub *ConfigHubClient, appName string) (*DeploymentHelper, error) {
	// Use ConfigHub's new-prefix to generate unique names (like "chubby-paws")
	// This would call: cub space new-prefix
	prefix, err := cub.GetNewSpacePrefix()
	if err != nil {
		// Fallback to timestamp if API call fails
		prefix = fmt.Sprintf("prefix-%d", time.Now().Unix())
	}

	// Project name format: prefix-appname (e.g., "chubby-paws-drift-detector")
	projectName := fmt.Sprintf("%s-%s", prefix, appName)

	return &DeploymentHelper{
		Cub:         cub,
		ProjectName: projectName,
		AppName:     appName,
	}, nil
}

// SetupBaseSpace creates the base ConfigHub structure
func (d *DeploymentHelper) SetupBaseSpace() error {
	// Create main space
	_, err := d.Cub.CreateSpace(CreateSpaceRequest{
		Slug:        d.ProjectName,
		DisplayName: fmt.Sprintf("%s DevOps App", d.AppName),
		Labels: map[string]string{
			"app":     d.AppName,
			"type":    "devops-app",
			"project": d.ProjectName,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create main space: %w", err)
	}

	// Create base space for base configurations
	_, err = d.Cub.CreateSpace(CreateSpaceRequest{
		Slug:        fmt.Sprintf("%s-base", d.ProjectName),
		DisplayName: fmt.Sprintf("%s Base Configurations", d.AppName),
		Labels: map[string]string{
			"base":    "true",
			"project": d.ProjectName,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create base space: %w", err)
	}

	// Create filters space
	_, err = d.Cub.CreateSpace(CreateSpaceRequest{
		Slug:        fmt.Sprintf("%s-filters", d.ProjectName),
		DisplayName: fmt.Sprintf("%s Filters", d.AppName),
		Labels: map[string]string{
			"type":    "filters",
			"project": d.ProjectName,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create filters space: %w", err)
	}

	return nil
}

// CreateStandardFilters creates common filters for DevOps apps
func (d *DeploymentHelper) CreateStandardFilters() error {
	filtersSpaceID := d.getSpaceID(fmt.Sprintf("%s-filters", d.ProjectName))

	// All project units filter
	_, err := d.Cub.CreateFilter(filtersSpaceID, CreateFilterRequest{
		Slug:        "all",
		DisplayName: "All Project Units",
		From:        "Unit",
		Where:       fmt.Sprintf("Space.Labels.project = '%s'", d.ProjectName),
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create all filter: %w", err)
	}

	// App-specific filter
	_, err = d.Cub.CreateFilter(filtersSpaceID, CreateFilterRequest{
		Slug:        d.AppName,
		DisplayName: fmt.Sprintf("%s Units", d.AppName),
		From:        "Unit",
		Where:       fmt.Sprintf("Labels.app = '%s'", d.AppName),
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create app filter: %w", err)
	}

	// Critical services filter
	_, err = d.Cub.CreateFilter(filtersSpaceID, CreateFilterRequest{
		Slug:        "critical",
		DisplayName: "Critical Services",
		From:        "Unit",
		Where:       "Labels.tier = 'critical'",
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("create critical filter: %w", err)
	}

	return nil
}

// LoadBaseConfigurations loads K8s manifests as ConfigHub units
func (d *DeploymentHelper) LoadBaseConfigurations(configPath string) error {
	baseSpaceID := d.getSpaceID(fmt.Sprintf("%s-base", d.ProjectName))

	// Standard files to load
	configs := []struct {
		name     string
		file     string
		unitType string
		tier     string
	}{
		{"namespace", "namespace.yaml", "infrastructure", "critical"},
		{fmt.Sprintf("%s-rbac", d.AppName), fmt.Sprintf("%s-rbac.yaml", d.AppName), "devops-app", "critical"},
		{fmt.Sprintf("%s-deployment", d.AppName), fmt.Sprintf("%s-deployment.yaml", d.AppName), "devops-app", "critical"},
		{fmt.Sprintf("%s-service", d.AppName), fmt.Sprintf("%s-service.yaml", d.AppName), "devops-app", "critical"},
	}

	for _, cfg := range configs {
		filePath := filepath.Join(configPath, cfg.file)
		// In real implementation, would read file content
		_, err := d.Cub.CreateUnit(baseSpaceID, CreateUnitRequest{
			Slug:        cfg.name,
			DisplayName: fmt.Sprintf("%s Configuration", cfg.name),
			Data:        fmt.Sprintf("# Content from %s", filePath),
			Labels: map[string]string{
				"type": cfg.unitType,
				"app":  d.AppName,
				"tier": cfg.tier,
			},
		})
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("create unit %s: %w", cfg.name, err)
		}
	}

	return nil
}

// CreateEnvironmentHierarchy sets up dev → staging → prod
func (d *DeploymentHelper) CreateEnvironmentHierarchy() error {
	baseSpaceID := d.getSpaceID(fmt.Sprintf("%s-base", d.ProjectName))

	// Create dev environment
	devSpaceID, err := d.createEnvironment("dev", &baseSpaceID)
	if err != nil {
		return fmt.Errorf("create dev environment: %w", err)
	}

	// Create staging environment (downstream from dev)
	stagingSpaceID, err := d.createEnvironment("staging", &devSpaceID)
	if err != nil {
		return fmt.Errorf("create staging environment: %w", err)
	}

	// Create prod environment (downstream from staging)
	_, err = d.createEnvironment("prod", &stagingSpaceID)
	if err != nil {
		return fmt.Errorf("create prod environment: %w", err)
	}

	return nil
}

// ApplyToEnvironment applies all units to a specific environment
func (d *DeploymentHelper) ApplyToEnvironment(environment string) error {
	spaceID := d.getSpaceID(fmt.Sprintf("%s-%s", d.ProjectName, environment))

	// Apply units in correct order
	units := []string{
		"namespace",
		fmt.Sprintf("%s-rbac", d.AppName),
		fmt.Sprintf("%s-service", d.AppName),
		fmt.Sprintf("%s-deployment", d.AppName),
	}

	for _, unit := range units {
		// Get unit ID by slug
		unitList, err := d.Cub.ListUnits(ListUnitsParams{
			SpaceID: spaceID,
			Where:   fmt.Sprintf("Slug = '%s'", unit),
		})
		if err != nil {
			return fmt.Errorf("list units for %s: %w", unit, err)
		}

		if len(unitList) > 0 {
			err = d.Cub.ApplyUnit(spaceID, unitList[0].UnitID)
			if err != nil {
				return fmt.Errorf("apply unit %s: %w", unit, err)
			}
		}
	}

	// Alternative: Use bulk apply
	err := d.Cub.BulkApplyUnits(BulkApplyParams{
		SpaceID: spaceID,
		Where:   fmt.Sprintf("Labels.app = '%s'", d.AppName),
		DryRun:  false,
	})
	if err != nil {
		return fmt.Errorf("bulk apply: %w", err)
	}

	return nil
}

// PromoteEnvironment promotes changes from one environment to another
func (d *DeploymentHelper) PromoteEnvironment(from, to string) error {
	fromSpaceID := d.getSpaceID(fmt.Sprintf("%s-%s", d.ProjectName, from))
	toSpaceID := d.getSpaceID(fmt.Sprintf("%s-%s", d.ProjectName, to))

	// Use push-upgrade pattern
	err := d.Cub.BulkPatchUnits(BulkPatchParams{
		SpaceID: toSpaceID,
		Where:   fmt.Sprintf("UpstreamSpaceID = '%s'", fromSpaceID),
		Patch:   map[string]interface{}{},
		Upgrade: true, // Push-upgrade
	})
	if err != nil {
		return fmt.Errorf("promote from %s to %s: %w", from, to, err)
	}

	return nil
}

// Helper functions

func (d *DeploymentHelper) createEnvironment(env string, upstreamSpaceID *uuid.UUID) (uuid.UUID, error) {
	spaceName := fmt.Sprintf("%s-%s", d.ProjectName, env)

	space, err := d.Cub.CreateSpace(CreateSpaceRequest{
		Slug:        spaceName,
		DisplayName: fmt.Sprintf("%s %s Environment", d.AppName, strings.Title(env)),
		Labels: map[string]string{
			"project":     d.ProjectName,
			"environment": env,
		},
	})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return uuid.UUID{}, fmt.Errorf("create space: %w", err)
	}

	// Clone units from upstream
	if upstreamSpaceID != nil {
		err = d.cloneUnitsFromUpstream(*upstreamSpaceID, space.SpaceID, env)
		if err != nil {
			return uuid.UUID{}, fmt.Errorf("clone units: %w", err)
		}
	}

	return space.SpaceID, nil
}

func (d *DeploymentHelper) cloneUnitsFromUpstream(fromSpaceID, toSpaceID uuid.UUID, env string) error {
	// List units in upstream space
	units, err := d.Cub.ListUnits(ListUnitsParams{
		SpaceID: fromSpaceID,
	})
	if err != nil {
		return fmt.Errorf("list upstream units: %w", err)
	}

	// Clone each unit with upstream relationship
	for _, unit := range units {
		_, err = d.Cub.CreateUnit(toSpaceID, CreateUnitRequest{
			Slug:           unit.Slug,
			DisplayName:    unit.DisplayName,
			Data:           unit.Data,
			Labels:         mergeLabels(unit.Labels, map[string]string{"environment": env}),
			UpstreamUnitID: &unit.UnitID,
		})
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("clone unit %s: %w", unit.Slug, err)
		}
	}

	return nil
}

func (d *DeploymentHelper) getSpaceID(spaceName string) uuid.UUID {
	// In real implementation, would fetch from ConfigHub
	// For now, generate deterministic UUID from name
	return uuid.NewSHA1(uuid.NameSpaceDNS, []byte(spaceName))
}

func mergeLabels(base, additional map[string]string) map[string]string {
	result := make(map[string]string)
	for k, v := range base {
		result[k] = v
	}
	for k, v := range additional {
		result[k] = v
	}
	return result
}

// QuickDeploy performs a complete deployment setup
// Example usage:
//   helper, err := NewDeploymentHelper(cub, "drift-detector")
//   if err != nil { ... }
//   err = helper.QuickDeploy("confighub/base")
func (d *DeploymentHelper) QuickDeploy(configPath string) error {
	// 1. Setup base spaces
	if err := d.SetupBaseSpace(); err != nil {
		return fmt.Errorf("setup base space: %w", err)
	}

	// 2. Create standard filters
	if err := d.CreateStandardFilters(); err != nil {
		return fmt.Errorf("create filters: %w", err)
	}

	// 3. Load base configurations
	if err := d.LoadBaseConfigurations(configPath); err != nil {
		return fmt.Errorf("load configs: %w", err)
	}

	// 4. Create environment hierarchy
	if err := d.CreateEnvironmentHierarchy(); err != nil {
		return fmt.Errorf("create environments: %w", err)
	}

	// 5. Apply to dev environment
	if err := d.ApplyToEnvironment("dev"); err != nil {
		return fmt.Errorf("apply to dev: %w", err)
	}

	return nil
}