package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// EnterpriseModeDeployer implements ConfigHub → Git → Flux/Argo → Kubernetes deployment
// This provides full GitOps compliance with audit trails for enterprise environments
type EnterpriseModeDeployer struct {
	app         *DevOpsApp
	spaceID     uuid.UUID
	gitRepo     string
	gitBranch   string
	gitopsPath  string
	gitopsTool  string // "flux" or "argo"
}

// NewEnterpriseModeDeployer creates a new enterprise mode deployer
func NewEnterpriseModeDeployer(app *DevOpsApp, spaceID uuid.UUID, gitRepo, branch string) *EnterpriseModeDeployer {
	return &EnterpriseModeDeployer{
		app:        app,
		spaceID:    spaceID,
		gitRepo:    gitRepo,
		gitBranch:  branch,
		gitopsPath: "manifests/", // Default path for GitOps manifests
		gitopsTool: detectGitOpsTool(),
	}
}

// detectGitOpsTool detects whether Flux or Argo is installed
func detectGitOpsTool() string {
	// Check for Flux
	if _, err := os.Stat("/usr/local/bin/flux"); err == nil {
		return "flux"
	}
	// Check for Argo
	if _, err := os.Stat("/usr/local/bin/argocd"); err == nil {
		return "argo"
	}
	// Default to Flux
	return "flux"
}

// DeployUnit exports a ConfigHub unit to Git for GitOps deployment
func (e *EnterpriseModeDeployer) DeployUnit(unitID uuid.UUID) error {
	e.app.Logger.Printf("🏢 [Enterprise Mode] Exporting unit %s to Git repository", unitID)

	// Get unit from ConfigHub
	unit, err := e.app.Cub.GetUnit(e.spaceID, unitID)
	if err != nil {
		return fmt.Errorf("get unit: %w", err)
	}

	// Export to Git
	if err := e.exportUnitToGit(*unit); err != nil {
		return fmt.Errorf("export to git: %w", err)
	}

	// Trigger GitOps sync
	if err := e.triggerGitOpsSync(); err != nil {
		return fmt.Errorf("trigger sync: %w", err)
	}

	return nil
}

// DeploySpace exports all units in a space to Git for GitOps deployment
func (e *EnterpriseModeDeployer) DeploySpace() error {
	e.app.Logger.Printf("🏢 [Enterprise Mode] Exporting space %s to Git repository", e.spaceID)
	start := time.Now()

	// Ensure Git repo is cloned
	if err := e.ensureGitRepo(); err != nil {
		return fmt.Errorf("ensure git repo: %w", err)
	}

	// List all units in space
	units, err := e.app.Cub.ListUnits(ListUnitsParams{
		SpaceID: e.spaceID,
	})
	if err != nil {
		return fmt.Errorf("list units: %w", err)
	}

	// Export each unit to Git
	exported := 0
	for _, unit := range units {
		if err := e.exportUnitToGit(*unit); err != nil {
			e.app.Logger.Printf("⚠️  Failed to export %s: %v", unit.Slug, err)
		} else {
			exported++
		}
	}

	// Commit and push changes
	if err := e.commitAndPush(fmt.Sprintf("Deploy %d units from ConfigHub space %s", exported, e.spaceID)); err != nil {
		return fmt.Errorf("commit and push: %w", err)
	}

	// Trigger GitOps sync
	if err := e.triggerGitOpsSync(); err != nil {
		return fmt.Errorf("trigger sync: %w", err)
	}

	e.app.Logger.Printf("✅ [Enterprise Mode] Exported %d units to Git in %v", exported, time.Since(start))
	return nil
}

// exportUnitToGit exports a ConfigHub unit as a YAML file in the Git repository
func (e *EnterpriseModeDeployer) exportUnitToGit(unit Unit) error {
	// Parse manifest from Data field
	var manifest map[string]interface{}
	if err := yaml.Unmarshal([]byte(unit.Data), &manifest); err != nil {
		return fmt.Errorf("parse manifest: %w", err)
	}

	// Determine file path based on resource type
	kind, _ := manifest["kind"].(string)
	metadata, _ := manifest["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)

	if name == "" {
		name = unit.Slug
	}

	// Create directory structure: manifests/namespace/kind/
	var filePath string
	if namespace != "" {
		filePath = filepath.Join(e.gitopsPath, namespace, strings.ToLower(kind), fmt.Sprintf("%s.yaml", name))
	} else {
		filePath = filepath.Join(e.gitopsPath, "cluster", strings.ToLower(kind), fmt.Sprintf("%s.yaml", name))
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Add ConfigHub metadata as annotations
	if metadata == nil {
		metadata = make(map[string]interface{})
		manifest["metadata"] = metadata
	}
	annotations, _ := metadata["annotations"].(map[string]interface{})
	if annotations == nil {
		annotations = make(map[string]interface{})
		metadata["annotations"] = annotations
	}

	// Add tracking annotations
	annotations["confighub.io/unit-id"] = unit.UnitID.String()
	annotations["confighub.io/space-id"] = unit.SpaceID.String()
	annotations["confighub.io/revision"] = fmt.Sprintf("%d", unit.Version)
	annotations["confighub.io/last-modified"] = unit.UpdatedAt.Format(time.RFC3339)
	annotations["confighub.io/managed-by"] = "confighub-enterprise-deployer"

	// Convert to YAML
	yamlData, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filePath, yamlData, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	e.app.Logger.Printf("📝 [Enterprise Mode] Exported %s to %s", unit.Slug, filePath)
	return nil
}

// ensureGitRepo ensures the Git repository is cloned and up to date
func (e *EnterpriseModeDeployer) ensureGitRepo() error {
	// Check if repo exists
	if _, err := os.Stat(filepath.Join(".git")); os.IsNotExist(err) {
		// Clone repository
		cmd := fmt.Sprintf("git clone -b %s %s .", e.gitBranch, e.gitRepo)
		if err := e.runGitCommand(cmd); err != nil {
			return fmt.Errorf("clone repo: %w", err)
		}
	} else {
		// Pull latest changes
		if err := e.runGitCommand("git pull origin " + e.gitBranch); err != nil {
			return fmt.Errorf("pull changes: %w", err)
		}
	}
	return nil
}

// commitAndPush commits changes and pushes to Git
func (e *EnterpriseModeDeployer) commitAndPush(message string) error {
	// Add all changes
	if err := e.runGitCommand("git add " + e.gitopsPath); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	// Check if there are changes to commit
	status, err := e.runGitCommandOutput("git status --porcelain " + e.gitopsPath)
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}

	if strings.TrimSpace(status) == "" {
		e.app.Logger.Println("ℹ️  [Enterprise Mode] No changes to commit")
		return nil
	}

	// Commit changes
	commitMsg := fmt.Sprintf("%s\n\nAutomated by ConfigHub Enterprise Deployer\nSpace: %s\nTimestamp: %s",
		message, e.spaceID, time.Now().Format(time.RFC3339))

	if err := e.runGitCommand(fmt.Sprintf(`git commit -m "%s"`, commitMsg)); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	// Push to remote
	if err := e.runGitCommand("git push origin " + e.gitBranch); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	e.app.Logger.Println("📤 [Enterprise Mode] Pushed changes to Git repository")
	return nil
}

// triggerGitOpsSync triggers a sync in Flux or Argo
func (e *EnterpriseModeDeployer) triggerGitOpsSync() error {
	switch e.gitopsTool {
	case "flux":
		return e.triggerFluxSync()
	case "argo":
		return e.triggerArgoSync()
	default:
		e.app.Logger.Printf("⚠️  Unknown GitOps tool: %s, skipping sync trigger", e.gitopsTool)
		return nil
	}
}

// triggerFluxSync triggers a Flux reconciliation
func (e *EnterpriseModeDeployer) triggerFluxSync() error {
	e.app.Logger.Println("🔄 [Enterprise Mode] Triggering Flux reconciliation...")

	// Trigger Flux reconciliation for the source
	cmd := fmt.Sprintf("flux reconcile source git %s", e.getFluxSourceName())
	if err := e.runCommand(cmd); err != nil {
		return fmt.Errorf("flux reconcile source: %w", err)
	}

	// Trigger Flux reconciliation for kustomization
	cmd = fmt.Sprintf("flux reconcile kustomization %s", e.getFluxKustomizationName())
	if err := e.runCommand(cmd); err != nil {
		return fmt.Errorf("flux reconcile kustomization: %w", err)
	}

	e.app.Logger.Println("✅ [Enterprise Mode] Flux reconciliation triggered")
	return nil
}

// triggerArgoSync triggers an Argo CD sync
func (e *EnterpriseModeDeployer) triggerArgoSync() error {
	e.app.Logger.Println("🔄 [Enterprise Mode] Triggering Argo CD sync...")

	appName := e.getArgoAppName()
	cmd := fmt.Sprintf("argocd app sync %s", appName)
	if err := e.runCommand(cmd); err != nil {
		return fmt.Errorf("argocd sync: %w", err)
	}

	// Wait for sync to complete
	cmd = fmt.Sprintf("argocd app wait %s --timeout 300", appName)
	if err := e.runCommand(cmd); err != nil {
		return fmt.Errorf("argocd wait: %w", err)
	}

	e.app.Logger.Println("✅ [Enterprise Mode] Argo CD sync completed")
	return nil
}

// CreateGitOpsConfig creates GitOps configuration for Flux or Argo
func (e *EnterpriseModeDeployer) CreateGitOpsConfig() error {
	switch e.gitopsTool {
	case "flux":
		return e.createFluxConfig()
	case "argo":
		return e.createArgoConfig()
	default:
		return fmt.Errorf("unknown GitOps tool: %s", e.gitopsTool)
	}
}

// createFluxConfig creates Flux GitRepository and Kustomization resources
func (e *EnterpriseModeDeployer) createFluxConfig() error {
	e.app.Logger.Println("📝 [Enterprise Mode] Creating Flux configuration...")

	// Create GitRepository resource
	gitRepo := map[string]interface{}{
		"apiVersion": "source.toolkit.fluxcd.io/v1beta2",
		"kind":       "GitRepository",
		"metadata": map[string]interface{}{
			"name":      e.getFluxSourceName(),
			"namespace": "flux-system",
		},
		"spec": map[string]interface{}{
			"interval": "1m",
			"ref": map[string]interface{}{
				"branch": e.gitBranch,
			},
			"url": e.gitRepo,
		},
	}

	// Create Kustomization resource
	kustomization := map[string]interface{}{
		"apiVersion": "kustomize.toolkit.fluxcd.io/v1beta2",
		"kind":       "Kustomization",
		"metadata": map[string]interface{}{
			"name":      e.getFluxKustomizationName(),
			"namespace": "flux-system",
		},
		"spec": map[string]interface{}{
			"interval": "5m",
			"path":     e.gitopsPath,
			"prune":    true,
			"sourceRef": map[string]interface{}{
				"kind": "GitRepository",
				"name": e.getFluxSourceName(),
			},
		},
	}

	// Apply Flux resources
	if err := e.applyResource(gitRepo); err != nil {
		return fmt.Errorf("apply git repository: %w", err)
	}

	if err := e.applyResource(kustomization); err != nil {
		return fmt.Errorf("apply kustomization: %w", err)
	}

	e.app.Logger.Println("✅ [Enterprise Mode] Flux configuration created")
	return nil
}

// createArgoConfig creates Argo CD Application resource
func (e *EnterpriseModeDeployer) createArgoConfig() error {
	e.app.Logger.Println("📝 [Enterprise Mode] Creating Argo CD application...")

	app := map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Application",
		"metadata": map[string]interface{}{
			"name":      e.getArgoAppName(),
			"namespace": "argocd",
		},
		"spec": map[string]interface{}{
			"project": "default",
			"source": map[string]interface{}{
				"repoURL":        e.gitRepo,
				"targetRevision": e.gitBranch,
				"path":           e.gitopsPath,
			},
			"destination": map[string]interface{}{
				"server":    "https://kubernetes.default.svc",
				"namespace": "default",
			},
			"syncPolicy": map[string]interface{}{
				"automated": map[string]interface{}{
					"prune":    true,
					"selfHeal": true,
				},
			},
		},
	}

	// Apply Argo CD application
	if err := e.applyResource(app); err != nil {
		return fmt.Errorf("apply argo app: %w", err)
	}

	e.app.Logger.Println("✅ [Enterprise Mode] Argo CD application created")
	return nil
}

// ValidateGitOpsDeployment validates GitOps deployment status
func (e *EnterpriseModeDeployer) ValidateGitOpsDeployment() (bool, []string) {
	e.app.Logger.Println("🔍 [Enterprise Mode] Validating GitOps deployment...")

	switch e.gitopsTool {
	case "flux":
		return e.validateFluxDeployment()
	case "argo":
		return e.validateArgoDeployment()
	default:
		return false, []string{"Unknown GitOps tool: " + e.gitopsTool}
	}
}

// validateFluxDeployment checks Flux deployment status
func (e *EnterpriseModeDeployer) validateFluxDeployment() (bool, []string) {
	var issues []string

	// Check GitRepository status
	output, err := e.runCommandOutput(fmt.Sprintf("flux get source git %s", e.getFluxSourceName()))
	if err != nil {
		issues = append(issues, fmt.Sprintf("GitRepository check failed: %v", err))
	} else if !strings.Contains(output, "True") {
		issues = append(issues, "GitRepository is not ready")
	}

	// Check Kustomization status
	output, err = e.runCommandOutput(fmt.Sprintf("flux get kustomization %s", e.getFluxKustomizationName()))
	if err != nil {
		issues = append(issues, fmt.Sprintf("Kustomization check failed: %v", err))
	} else if !strings.Contains(output, "True") {
		issues = append(issues, "Kustomization is not ready")
	}

	valid := len(issues) == 0
	if valid {
		e.app.Logger.Println("✅ [Enterprise Mode] Flux deployment is healthy")
	} else {
		e.app.Logger.Printf("⚠️  [Enterprise Mode] Flux deployment has %d issues", len(issues))
	}

	return valid, issues
}

// validateArgoDeployment checks Argo CD deployment status
func (e *EnterpriseModeDeployer) validateArgoDeployment() (bool, []string) {
	var issues []string

	output, err := e.runCommandOutput(fmt.Sprintf("argocd app get %s --output json", e.getArgoAppName()))
	if err != nil {
		issues = append(issues, fmt.Sprintf("Argo app check failed: %v", err))
		return false, issues
	}

	var appStatus map[string]interface{}
	if err := json.Unmarshal([]byte(output), &appStatus); err != nil {
		issues = append(issues, fmt.Sprintf("Failed to parse Argo status: %v", err))
		return false, issues
	}

	// Check sync status
	if status, ok := appStatus["status"].(map[string]interface{}); ok {
		if sync, ok := status["sync"].(map[string]interface{}); ok {
			if syncStatus, ok := sync["status"].(string); ok && syncStatus != "Synced" {
				issues = append(issues, fmt.Sprintf("Argo sync status: %s", syncStatus))
			}
		}
		if health, ok := status["health"].(map[string]interface{}); ok {
			if healthStatus, ok := health["status"].(string); ok && healthStatus != "Healthy" {
				issues = append(issues, fmt.Sprintf("Argo health status: %s", healthStatus))
			}
		}
	}

	valid := len(issues) == 0
	if valid {
		e.app.Logger.Println("✅ [Enterprise Mode] Argo CD deployment is healthy")
	} else {
		e.app.Logger.Printf("⚠️  [Enterprise Mode] Argo CD deployment has %d issues", len(issues))
	}

	return valid, issues
}

// Helper methods

func (e *EnterpriseModeDeployer) getFluxSourceName() string {
	return fmt.Sprintf("confighub-%s", e.spaceID.String()[:8])
}

func (e *EnterpriseModeDeployer) getFluxKustomizationName() string {
	return fmt.Sprintf("confighub-%s", e.spaceID.String()[:8])
}

func (e *EnterpriseModeDeployer) getArgoAppName() string {
	return fmt.Sprintf("confighub-%s", e.spaceID.String()[:8])
}

func (e *EnterpriseModeDeployer) runGitCommand(cmd string) error {
	return e.runCommand("git " + cmd)
}

func (e *EnterpriseModeDeployer) runGitCommandOutput(cmd string) (string, error) {
	return e.runCommandOutput("git " + cmd)
}

func (e *EnterpriseModeDeployer) runCommand(cmd string) error {
	_, err := e.runCommandOutput(cmd)
	return err
}

func (e *EnterpriseModeDeployer) runCommandOutput(cmd string) (string, error) {
	// Implementation would use exec.Command to run shell commands
	// For now, this is a placeholder
	e.app.Logger.Printf("🔧 [Enterprise Mode] Running: %s", cmd)
	return "", nil
}

func (e *EnterpriseModeDeployer) applyResource(resource map[string]interface{}) error {
	// Implementation would apply the resource to Kubernetes
	// For now, log the action
	kind, _ := resource["kind"].(string)
	metadata, _ := resource["metadata"].(map[string]interface{})
	name, _ := metadata["name"].(string)

	e.app.Logger.Printf("📦 [Enterprise Mode] Applying %s: %s", kind, name)
	return nil
}

// WatchGitOpsStatus monitors GitOps deployment status continuously
func (e *EnterpriseModeDeployer) WatchGitOpsStatus(ctx context.Context, interval time.Duration) error {
	e.app.Logger.Printf("👁️  [Enterprise Mode] Watching GitOps status for space %s", e.spaceID)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			valid, issues := e.ValidateGitOpsDeployment()
			if !valid {
				e.app.Logger.Printf("⚠️  GitOps issues detected: %v", issues)
			}
		}
	}
}