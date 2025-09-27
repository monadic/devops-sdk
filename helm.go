package sdk

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// HelmHelper provides Helm chart management through ConfigHub
type HelmHelper struct {
	cub     *ConfigHubClient
	spaceID uuid.UUID
}

// HelmOptions contains options for Helm operations
type HelmOptions struct {
	Namespace      string            // Kubernetes namespace
	Version        string            // Chart version
	Values         []string          // --set flag values
	ValuesFiles    []string          // --values file paths
	UpdateCRDs     bool              // Update CRDs on upgrade
	SkipCRDs       bool              // Skip CRD installation
	UsePlaceholder bool              // Use confighubplaceholder
	Labels         map[string]string // Additional labels for units
}

// NewHelmHelper creates a new Helm helper
func NewHelmHelper(cub *ConfigHubClient, spaceID uuid.UUID) *HelmHelper {
	return &HelmHelper{
		cub:     cub,
		spaceID: spaceID,
	}
}

// InstallChart installs a Helm chart as ConfigHub units
// This wraps the `cub helm install` command
func (h *HelmHelper) InstallChart(release, chart string, opts HelmOptions) error {
	args := []string{"helm", "install"}

	// Add space
	args = append(args, "--space", h.spaceID.String())

	// Add namespace
	if opts.Namespace != "" {
		args = append(args, "--namespace", opts.Namespace)
	}

	// Add version
	if opts.Version != "" {
		args = append(args, "--version", opts.Version)
	}

	// Add values files
	for _, vf := range opts.ValuesFiles {
		args = append(args, "--values", vf)
	}

	// Add set values
	for _, v := range opts.Values {
		args = append(args, "--set", v)
	}

	// Add flags
	if opts.SkipCRDs {
		args = append(args, "--skip-crds")
	}
	if !opts.UsePlaceholder {
		args = append(args, "--use-placeholder=false")
	}

	// Add release and chart
	args = append(args, release, chart)

	// Execute command
	cmd := exec.Command("cub", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm install failed: %v\nStderr: %s", err, stderr.String())
	}

	// Apply additional labels if provided
	if len(opts.Labels) > 0 {
		// Update the created units with additional labels
		units, err := h.cub.ListUnits(ListUnitsParams{
			SpaceID: h.spaceID,
			Where:   fmt.Sprintf("Labels.HelmRelease = '%s'", release),
		})
		if err == nil {
			for _, unit := range units {
				for k, v := range opts.Labels {
					unit.Labels[k] = v
				}
				_, _ = h.cub.UpdateUnit(unit.UnitID, unit)
			}
		}
	}

	return nil
}

// UpgradeChart upgrades an existing Helm release
// This wraps the `cub helm upgrade` command
func (h *HelmHelper) UpgradeChart(release, chart string, opts HelmOptions) error {
	args := []string{"helm", "upgrade"}

	// Add space
	args = append(args, "--space", h.spaceID.String())

	// Add namespace
	if opts.Namespace != "" {
		args = append(args, "--namespace", opts.Namespace)
	}

	// Add version
	if opts.Version != "" {
		args = append(args, "--version", opts.Version)
	}

	// Add values files
	for _, vf := range opts.ValuesFiles {
		args = append(args, "--values", vf)
	}

	// Add set values
	for _, v := range opts.Values {
		args = append(args, "--set", v)
	}

	// Add flags
	if opts.UpdateCRDs {
		args = append(args, "--update-crds")
	}
	if opts.SkipCRDs {
		args = append(args, "--skip-crds")
	}

	// Add release and chart
	args = append(args, release, chart)

	// Execute command
	cmd := exec.Command("cub", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("helm upgrade failed: %v\nStderr: %s", err, stderr.String())
	}

	return nil
}

// ListHelmReleases lists all Helm releases in the space
func (h *HelmHelper) ListHelmReleases() ([]HelmRelease, error) {
	units, err := h.cub.ListUnits(ListUnitsParams{
		SpaceID: h.spaceID,
		Where:   "Labels.HelmRelease != ''",
	})
	if err != nil {
		return nil, err
	}

	// Group by release name
	releaseMap := make(map[string]*HelmRelease)
	for _, unit := range units {
		releaseName := unit.Labels["HelmRelease"]
		if releaseName == "" {
			continue
		}

		if release, exists := releaseMap[releaseName]; exists {
			// Update with CRDs unit if this is it
			if strings.HasSuffix(unit.Slug, "-crds") {
				release.CRDsUnitID = &unit.UnitID
			}
		} else {
			release := &HelmRelease{
				Name:       releaseName,
				Chart:      unit.Labels["HelmChart"],
				Version:    unit.Labels["HelmChartVersion"],
				Namespace:  unit.Labels["namespace"],
				MainUnitID: unit.UnitID,
				SpaceID:    h.spaceID,
			}
			if strings.HasSuffix(unit.Slug, "-crds") {
				release.CRDsUnitID = &unit.UnitID
				release.MainUnitID = uuid.Nil // Will be set when we find main unit
			}
			releaseMap[releaseName] = release
		}
	}

	// Convert map to slice
	var releases []HelmRelease
	for _, release := range releaseMap {
		releases = append(releases, *release)
	}

	return releases, nil
}

// GetHelmRelease gets details of a specific Helm release
func (h *HelmHelper) GetHelmRelease(releaseName string) (*HelmRelease, error) {
	releases, err := h.ListHelmReleases()
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		if release.Name == releaseName {
			return &release, nil
		}
	}

	return nil, fmt.Errorf("helm release %s not found", releaseName)
}

// DeleteHelmRelease deletes a Helm release and its units
func (h *HelmHelper) DeleteHelmRelease(releaseName string) error {
	release, err := h.GetHelmRelease(releaseName)
	if err != nil {
		return err
	}

	// Delete CRDs unit first if it exists
	if release.CRDsUnitID != nil && *release.CRDsUnitID != uuid.Nil {
		if err := h.cub.DeleteUnit(*release.CRDsUnitID); err != nil {
			return fmt.Errorf("failed to delete CRDs unit: %w", err)
		}
	}

	// Delete main unit
	if release.MainUnitID != uuid.Nil {
		if err := h.cub.DeleteUnit(release.MainUnitID); err != nil {
			return fmt.Errorf("failed to delete main unit: %w", err)
		}
	}

	return nil
}

// HelmRelease represents a Helm release in ConfigHub
type HelmRelease struct {
	Name       string     // Release name
	Chart      string     // Chart name
	Version    string     // Chart version
	Namespace  string     // Kubernetes namespace
	MainUnitID uuid.UUID  // Main resources unit
	CRDsUnitID *uuid.UUID // CRDs unit (optional)
	SpaceID    uuid.UUID  // ConfigHub space
}

// GetLatestChartVersion gets the latest version of a Helm chart
// This requires helm CLI to be installed
func (h *HelmHelper) GetLatestChartVersion(chart string) (string, error) {
	cmd := exec.Command("helm", "search", "repo", chart, "--output", "json")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to search chart: %w", err)
	}

	// Parse JSON output to get version
	// For simplicity, we'll just extract the version string
	// In production, use proper JSON parsing
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, `"version"`) {
			parts := strings.Split(line, `"`)
			if len(parts) >= 4 {
				return parts[3], nil
			}
		}
	}

	return "", fmt.Errorf("could not parse chart version")
}

// CompareChartVersions checks if a newer version is available
func (h *HelmHelper) CompareChartVersions(release *HelmRelease) (bool, string, error) {
	latest, err := h.GetLatestChartVersion(release.Chart)
	if err != nil {
		return false, "", err
	}

	if latest != release.Version {
		return true, latest, nil
	}

	return false, "", nil
}

// GenerateUpgradeCommand generates the CLI command to upgrade a release
func (h *HelmHelper) GenerateUpgradeCommand(release *HelmRelease, newVersion string) string {
	return fmt.Sprintf("cub helm upgrade --space %s --namespace %s %s %s --version %s",
		h.spaceID.String(),
		release.Namespace,
		release.Name,
		release.Chart,
		newVersion)
}

// ValidateHelmValues validates Helm values using ConfigHub functions
func (h *HelmHelper) ValidateHelmValues(values map[string]interface{}) error {
	// Convert values to YAML for validation
	yamlBytes, err := yaml.Marshal(values)
	if err != nil {
		return fmt.Errorf("failed to marshal values: %w", err)
	}

	// Create temporary unit for validation
	tempUnit, err := h.cub.CreateUnit(CreateUnitRequest{
		SpaceID: h.spaceID,
		Slug:    fmt.Sprintf("helm-values-validation-%d", time.Now().Unix()),
		Data:    string(yamlBytes),
		Labels: map[string]string{
			"temporary": "true",
			"purpose":   "helm-values-validation",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create validation unit: %w", err)
	}
	defer h.cub.DeleteUnit(tempUnit.UnitID)

	// Validate no placeholders
	valid, message, err := h.cub.ValidateNoPlaceholders(h.spaceID, tempUnit.UnitID)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("values contain placeholders: %s", message)
	}

	// Validate YAML structure
	valid, message, err = h.cub.ValidateYAML(h.spaceID, tempUnit.UnitID)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("invalid YAML in values: %s", message)
	}

	return nil
}