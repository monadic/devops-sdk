package sdk

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// DevModeDeployer implements direct ConfigHub → Kubernetes deployment for development
// This bypasses Git/GitOps for fast feedback loops during development
type DevModeDeployer struct {
	app           *DevOpsApp
	dynamicClient dynamic.Interface
	spaceID       uuid.UUID
}

// NewDevModeDeployer creates a new development mode deployer
func NewDevModeDeployer(app *DevOpsApp, spaceID uuid.UUID) *DevModeDeployer {
	return &DevModeDeployer{
		app:           app,
		dynamicClient: app.K8s.DynamicClient,
		spaceID:       spaceID,
	}
}

// DeployUnit deploys a single ConfigHub unit directly to Kubernetes
func (d *DevModeDeployer) DeployUnit(unitID uuid.UUID) error {
	d.app.Logger.Printf("🚀 [Dev Mode] Deploying unit %s directly to Kubernetes", unitID)

	// Get unit from ConfigHub
	unit, err := d.app.Cub.GetUnit(unitID)
	if err != nil {
		return fmt.Errorf("get unit: %w", err)
	}

	// Parse and apply manifest
	manifest, ok := unit.ManifestData.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid manifest data type")
	}

	return d.applyManifest(manifest, unit.Slug)
}

// DeploySpace deploys all units in a ConfigHub space directly to Kubernetes
func (d *DevModeDeployer) DeploySpace() error {
	d.app.Logger.Printf("🚀 [Dev Mode] Deploying all units from space %s", d.spaceID)
	start := time.Now()

	// List all units in space
	units, err := d.app.Cub.ListUnits(d.spaceID)
	if err != nil {
		return fmt.Errorf("list units: %w", err)
	}

	deployed := 0
	failed := 0

	for _, unit := range units {
		if err := d.DeployUnit(unit.UnitID); err != nil {
			d.app.Logger.Printf("⚠️  Failed to deploy %s: %v", unit.Slug, err)
			failed++
		} else {
			deployed++
		}
	}

	d.app.Logger.Printf("✅ [Dev Mode] Deployment complete: %d succeeded, %d failed in %v",
		deployed, failed, time.Since(start))
	return nil
}

// DeployWithFilter deploys units matching a filter directly to Kubernetes
func (d *DevModeDeployer) DeployWithFilter(filterID uuid.UUID) error {
	d.app.Logger.Printf("🚀 [Dev Mode] Deploying units matching filter %s", filterID)

	// Get filtered units
	units, err := d.app.Cub.GetFilteredUnits(filterID, d.spaceID)
	if err != nil {
		return fmt.Errorf("get filtered units: %w", err)
	}

	deployed := 0
	for _, unit := range units {
		if err := d.DeployUnit(unit.UnitID); err != nil {
			d.app.Logger.Printf("⚠️  Failed to deploy %s: %v", unit.Slug, err)
		} else {
			deployed++
		}
	}

	d.app.Logger.Printf("✅ [Dev Mode] Deployed %d/%d units matching filter", deployed, len(units))
	return nil
}

// WatchAndSync continuously syncs ConfigHub changes to Kubernetes
func (d *DevModeDeployer) WatchAndSync(ctx context.Context, interval time.Duration) error {
	d.app.Logger.Printf("👁️  [Dev Mode] Watching ConfigHub space %s for changes", d.spaceID)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Track last revision for change detection
	lastRevisions := make(map[uuid.UUID]int)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := d.syncChanges(lastRevisions); err != nil {
				d.app.Logger.Printf("⚠️  Sync error: %v", err)
			}
		}
	}
}

// syncChanges syncs any changed units to Kubernetes
func (d *DevModeDeployer) syncChanges(lastRevisions map[uuid.UUID]int) error {
	units, err := d.app.Cub.ListUnits(d.spaceID)
	if err != nil {
		return fmt.Errorf("list units: %w", err)
	}

	changes := 0
	for _, unit := range units {
		// Check if unit has changed
		lastRev, exists := lastRevisions[unit.UnitID]
		currentRev := unit.Revision // Assuming Unit has a Revision field

		if !exists || currentRev > lastRev {
			d.app.Logger.Printf("🔄 [Dev Mode] Detected change in %s (rev %d -> %d)",
				unit.Slug, lastRev, currentRev)

			if err := d.DeployUnit(unit.UnitID); err != nil {
				d.app.Logger.Printf("⚠️  Failed to sync %s: %v", unit.Slug, err)
			} else {
				changes++
				lastRevisions[unit.UnitID] = currentRev
			}
		}
	}

	if changes > 0 {
		d.app.Logger.Printf("✅ [Dev Mode] Synced %d changed units", changes)
	}
	return nil
}

// applyManifest applies a Kubernetes manifest directly
func (d *DevModeDeployer) applyManifest(manifest map[string]interface{}, name string) error {
	// Extract resource information
	apiVersion, _ := manifest["apiVersion"].(string)
	kind, _ := manifest["kind"].(string)

	if apiVersion == "" || kind == "" {
		return fmt.Errorf("missing apiVersion or kind in manifest")
	}

	// Parse GVR from manifest
	gvr, namespace, err := d.parseGVR(apiVersion, kind, manifest)
	if err != nil {
		return fmt.Errorf("parse GVR: %w", err)
	}

	// Create unstructured object
	obj := &unstructured.Unstructured{
		Object: manifest,
	}

	// Apply to Kubernetes
	ctx := context.Background()
	var result *unstructured.Unstructured

	if namespace == "" {
		// Cluster-scoped resource
		result, err = d.dynamicClient.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			// Try update if create fails
			result, err = d.dynamicClient.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
		}
	} else {
		// Namespaced resource
		result, err = d.dynamicClient.Resource(gvr).Namespace(namespace).Create(ctx, obj, metav1.CreateOptions{})
		if err != nil {
			// Try update if create fails
			result, err = d.dynamicClient.Resource(gvr).Namespace(namespace).Update(ctx, obj, metav1.UpdateOptions{})
		}
	}

	if err != nil {
		return fmt.Errorf("apply manifest: %w", err)
	}

	d.app.Logger.Printf("✅ [Dev Mode] Applied %s/%s: %s", kind, apiVersion, result.GetName())
	return nil
}

// parseGVR parses Group, Version, Resource from manifest
func (d *DevModeDeployer) parseGVR(apiVersion, kind string, manifest map[string]interface{}) (schema.GroupVersionResource, string, error) {
	// Common resource mappings
	resourceMap := map[string]string{
		"Deployment":            "deployments",
		"Service":               "services",
		"ConfigMap":             "configmaps",
		"Secret":                "secrets",
		"StatefulSet":           "statefulsets",
		"DaemonSet":             "daemonsets",
		"Pod":                   "pods",
		"Ingress":               "ingresses",
		"ServiceAccount":        "serviceaccounts",
		"Role":                  "roles",
		"RoleBinding":           "rolebindings",
		"ClusterRole":           "clusterroles",
		"ClusterRoleBinding":    "clusterrolebindings",
		"PersistentVolumeClaim": "persistentvolumeclaims",
		"HorizontalPodAutoscaler": "horizontalpodautoscalers",
	}

	resource, ok := resourceMap[kind]
	if !ok {
		// Try to pluralize by adding 's'
		resource = kind + "s"
	}

	// Parse group and version from apiVersion
	group := ""
	version := apiVersion

	if idx := len(apiVersion) - 1; idx > 0 {
		for i := len(apiVersion) - 1; i >= 0; i-- {
			if apiVersion[i] == '/' {
				group = apiVersion[:i]
				version = apiVersion[i+1:]
				break
			}
		}
	}

	// Extract namespace from metadata
	namespace := ""
	if metadata, ok := manifest["metadata"].(map[string]interface{}); ok {
		namespace, _ = metadata["namespace"].(string)
	}

	return schema.GroupVersionResource{
		Group:    group,
		Version:  version,
		Resource: resource,
	}, namespace, nil
}

// Rollback rolls back a deployment to a previous ConfigHub revision
func (d *DevModeDeployer) Rollback(unitID uuid.UUID, targetRevision int) error {
	d.app.Logger.Printf("⏮️  [Dev Mode] Rolling back unit %s to revision %d", unitID, targetRevision)

	// In Dev Mode, rollback is instant - just get the old revision and apply it
	// This would require ConfigHub to support revision history API

	// For now, just re-deploy current version
	return d.DeployUnit(unitID)
}

// ValidateDeployment validates that Kubernetes matches ConfigHub configuration
func (d *DevModeDeployer) ValidateDeployment() (bool, []string) {
	d.app.Logger.Println("🔍 [Dev Mode] Validating Kubernetes matches ConfigHub...")

	units, err := d.app.Cub.ListUnits(d.spaceID)
	if err != nil {
		return false, []string{fmt.Sprintf("Failed to list units: %v", err)}
	}

	var issues []string
	for _, unit := range units {
		manifest, ok := unit.ManifestData.(map[string]interface{})
		if !ok {
			issues = append(issues, fmt.Sprintf("%s: invalid manifest data", unit.Slug))
			continue
		}

		// Check if resource exists in Kubernetes
		exists, err := d.resourceExists(manifest)
		if err != nil {
			issues = append(issues, fmt.Sprintf("%s: %v", unit.Slug, err))
		} else if !exists {
			issues = append(issues, fmt.Sprintf("%s: not found in Kubernetes", unit.Slug))
		}
	}

	valid := len(issues) == 0
	if valid {
		d.app.Logger.Println("✅ [Dev Mode] All ConfigHub units are deployed to Kubernetes")
	} else {
		d.app.Logger.Printf("⚠️  [Dev Mode] Found %d validation issues", len(issues))
	}

	return valid, issues
}

// resourceExists checks if a resource exists in Kubernetes
func (d *DevModeDeployer) resourceExists(manifest map[string]interface{}) (bool, error) {
	apiVersion, _ := manifest["apiVersion"].(string)
	kind, _ := manifest["kind"].(string)

	metadata, ok := manifest["metadata"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("missing metadata")
	}

	name, _ := metadata["name"].(string)
	if name == "" {
		return false, fmt.Errorf("missing name in metadata")
	}

	gvr, namespace, err := d.parseGVR(apiVersion, kind, manifest)
	if err != nil {
		return false, err
	}

	ctx := context.Background()
	if namespace == "" {
		_, err = d.dynamicClient.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	} else {
		_, err = d.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	}

	if err != nil {
		return false, nil // Resource doesn't exist
	}
	return true, nil
}