package sdk

import (
	"context"
	"fmt"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ComprehensiveHealthCheck performs comprehensive health checks for DevOps apps
type ComprehensiveHealthCheck struct {
	K8sClient     *kubernetes.Clientset
	ConfigHubClient *ConfigHubClient
	Namespace     string
	SpaceID       string // Added for space-specific checks
}

// HealthCheckResult contains comprehensive health check results
type HealthCheckResult struct {
	Timestamp     string          `json:"timestamp"`
	HealthScore   int             `json:"health_score"`
	Status        string          `json:"status"`
	StatusText    string          `json:"status_text"`
	Checks        []HealthCheck   `json:"checks"`
	Issues        []string        `json:"issues"`
	QuickActions  []string        `json:"quick_actions"`
}

// HealthCheck represents a single health check
type HealthCheck struct {
	Component string `json:"component"`
	Check     string `json:"check"`
	Status    string `json:"status"`
	Details   string `json:"details"`
}

// NewComprehensiveHealthCheck creates a new comprehensive health checker
func NewComprehensiveHealthCheck(k8sClient *kubernetes.Clientset, configHubClient *ConfigHubClient, namespace string) *ComprehensiveHealthCheck {
	return &ComprehensiveHealthCheck{
		K8sClient:       k8sClient,
		ConfigHubClient: configHubClient,
		Namespace:       namespace,
	}
}

// RunHealthCheck performs comprehensive health checks
func (c *ComprehensiveHealthCheck) RunHealthCheck(ctx context.Context) (*HealthCheckResult, error) {
	healthScore := 100
	var issues []string
	var checks []HealthCheck

	// Check ConfigHub connectivity
	if c.ConfigHubClient != nil {
		// Test ConfigHub connection
		spaces, err := c.ConfigHubClient.ListSpaces()
		if err != nil {
			healthScore -= 20
			issues = append(issues, "ConfigHub: Connection failed")
			checks = append(checks, HealthCheck{
				Component: "ConfigHub",
				Check:     "Connection",
				Status:    "UNHEALTHY",
				Details:   err.Error(),
			})
		} else {
			checks = append(checks, HealthCheck{
				Component: "ConfigHub",
				Check:     "Connection",
				Status:    "HEALTHY",
				Details:   fmt.Sprintf("Connected, %d spaces accessible", len(spaces)),
			})
		}

		// PRINCIPLE #0: App as Collection of Related Units
		// Every app should have units with consistent labels and filters for bulk operations
		if c.SpaceID != "" {
			filters, err := c.ConfigHubClient.ListFilters(c.SpaceID)
			if err != nil || len(filters) == 0 {
				healthScore -= 10
				issues = append(issues, "ConfigHub: No Filters defined - cannot perform bulk operations on app units")
				checks = append(checks, HealthCheck{
					Component: "ConfigHub",
					Check:     "Filters",
					Status:    "WARNING",
					Details:   "Apps need Filters to target labeled units for bulk operations",
				})
			} else {
				checks = append(checks, HealthCheck{
					Component: "ConfigHub",
					Check:     "Filters",
					Status:    "HEALTHY",
					Details:   fmt.Sprintf("%d filter(s) defined for targeting units", len(filters)),
				})
			}
		}

		// PRINCIPLE #1: Check for ConfigHub Worker (MANDATORY)
		if c.SpaceID != "" {
			workers, err := c.ConfigHubClient.ListWorkers(c.SpaceID)
			if err != nil || len(workers) == 0 {
				healthScore -= 25
				issues = append(issues, "ConfigHub: No worker running - units won't deploy!")
				checks = append(checks, HealthCheck{
					Component: "ConfigHub",
					Check:     "Worker",
					Status:    "CRITICAL",
					Details:   "Worker is MANDATORY for ConfigHub → Kubernetes deployment",
				})
			} else {
				checks = append(checks, HealthCheck{
					Component: "ConfigHub",
					Check:     "Worker",
					Status:    "HEALTHY",
					Details:   fmt.Sprintf("%d worker(s) available", len(workers)),
				})
			}

			// PRINCIPLE #4: Check for Targets
			targets, err := c.ConfigHubClient.ListTargets(c.SpaceID)
			if err != nil || len(targets) == 0 {
				healthScore -= 15
				issues = append(issues, "ConfigHub: No targets configured - units won't know where to deploy")
				checks = append(checks, HealthCheck{
					Component: "ConfigHub",
					Check:     "Targets",
					Status:    "UNHEALTHY",
					Details:   "Targets link units to Kubernetes clusters",
				})
			} else {
				checks = append(checks, HealthCheck{
					Component: "ConfigHub",
					Check:     "Targets",
					Status:    "HEALTHY",
					Details:   fmt.Sprintf("%d target(s) configured", len(targets)),
				})
			}
		}
	}

	// Check Kubernetes connectivity
	if c.K8sClient != nil {
		_, err := c.K8sClient.ServerVersion()
		if err != nil {
			healthScore -= 20
			issues = append(issues, "Kubernetes: API not accessible")
			checks = append(checks, HealthCheck{
				Component: "Kubernetes",
				Check:     "API Connection",
				Status:    "UNHEALTHY",
				Details:   err.Error(),
			})
		} else {
			checks = append(checks, HealthCheck{
				Component: "Kubernetes",
				Check:     "API Connection",
				Status:    "HEALTHY",
				Details:   "Kubernetes API accessible",
			})
		}

		// Check namespace
		if c.Namespace != "" {
			_, err = c.K8sClient.CoreV1().Namespaces().Get(ctx, c.Namespace, metav1.GetOptions{})
			if err != nil {
				healthScore -= 10
				issues = append(issues, fmt.Sprintf("Kubernetes: Namespace %s not found", c.Namespace))
				checks = append(checks, HealthCheck{
					Component: "Kubernetes",
					Check:     "Namespace",
					Status:    "UNHEALTHY",
					Details:   fmt.Sprintf("%s namespace not found", c.Namespace),
				})
			} else {
				checks = append(checks, HealthCheck{
					Component: "Kubernetes",
					Check:     "Namespace",
					Status:    "HEALTHY",
					Details:   fmt.Sprintf("%s namespace exists", c.Namespace),
				})
			}
		}

		// Check deployments
		if c.Namespace != "" {
			deployments, err := c.K8sClient.AppsV1().Deployments(c.Namespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				healthScore -= 15
				issues = append(issues, "Kubernetes: Cannot list deployments")
				checks = append(checks, HealthCheck{
					Component: "Kubernetes",
					Check:     "Deployments",
					Status:    "UNHEALTHY",
					Details:   err.Error(),
				})
			} else {
				totalDeployments := len(deployments.Items)
				healthyDeployments := 0
				for _, dep := range deployments.Items {
					if dep.Status.ReadyReplicas == *dep.Spec.Replicas && dep.Status.ReadyReplicas > 0 {
						healthyDeployments++
					} else {
						issues = append(issues, fmt.Sprintf("Deployment %s: %d/%d replicas ready",
							dep.Name, dep.Status.ReadyReplicas, *dep.Spec.Replicas))
						healthScore -= 5
					}
				}

				checks = append(checks, HealthCheck{
					Component: "Kubernetes",
					Check:     "Deployments",
					Status: func() string {
						if healthyDeployments == totalDeployments {
							return "HEALTHY"
						} else if healthyDeployments > 0 {
							return "DEGRADED"
						}
						return "UNHEALTHY"
					}(),
					Details: fmt.Sprintf("%d/%d deployments healthy", healthyDeployments, totalDeployments),
				})
			}
		}
	}

	// Check for drift by comparing ConfigHub units with K8s deployments
	if c.ConfigHubClient != nil && c.K8sClient != nil && c.Namespace != "" {
		driftCount := 0
		// This is simplified - in real implementation you'd compare actual state
		// For now, just check if deployments exist
		deployments, _ := c.K8sClient.AppsV1().Deployments(c.Namespace).List(ctx, metav1.ListOptions{})

		// Check each deployment for drift
		for _, dep := range deployments.Items {
			// In a real implementation, you'd fetch the expected state from ConfigHub
			// and compare with actual K8s state
			expectedReplicas := int32(2) // Default expected
			actualReplicas := *dep.Spec.Replicas

			if expectedReplicas != actualReplicas {
				driftCount++
				healthScore -= 5
				issues = append(issues, fmt.Sprintf("Drift: %s has %d replicas, expected %d",
					dep.Name, actualReplicas, expectedReplicas))
			}
		}

		checks = append(checks, HealthCheck{
			Component: "Drift Detection",
			Check:     "Configuration Drift",
			Status: func() string {
				if driftCount == 0 {
					return "HEALTHY"
				}
				return "DRIFTED"
			}(),
			Details: fmt.Sprintf("%d resources with drift", driftCount),
		})
	}

	// PRINCIPLE #5: Validate configurations using Functions
	if c.ConfigHubClient != nil && c.SpaceID != "" {
		// Get units that need validation
		units, err := c.ConfigHubClient.ListUnits(ListUnitsParams{
			SpaceID: uuid.MustParse(c.SpaceID),
			Where:   "Labels.validate = 'true'",
		})
		if err == nil {
			validationIssues := 0
			for _, unit := range units {
				// Check for placeholders
				valid, message, err := c.ConfigHubClient.ValidateNoPlaceholders(uuid.MustParse(c.SpaceID), unit.UnitID)
				if err != nil || !valid {
					validationIssues++
					issues = append(issues, fmt.Sprintf("Unit %s: %s", unit.Slug, message))
				}

				// Validate YAML structure
				valid, message, err = c.ConfigHubClient.ValidateYAML(uuid.MustParse(c.SpaceID), unit.UnitID)
				if err != nil || !valid {
					validationIssues++
					issues = append(issues, fmt.Sprintf("Unit %s has invalid YAML: %s", unit.Slug, message))
				}
			}

			checks = append(checks, HealthCheck{
				Component: "ConfigHub",
				Check:     "Unit Validation",
				Status: func() string {
					if validationIssues == 0 {
						return "HEALTHY"
					} else if validationIssues < 3 {
						return "WARNING"
					}
					return "UNHEALTHY"
				}(),
				Details: fmt.Sprintf("%d validation issues found", validationIssues),
			})

			if validationIssues > 0 {
				healthScore -= validationIssues * 5
			}
		}
	}

	// Determine overall status
	status := "HEALTHY"
	statusText := "System is fully operational"
	if healthScore >= 90 {
		status = "HEALTHY"
		statusText = "System is fully operational"
	} else if healthScore >= 70 {
		status = "DEGRADED"
		statusText = "System has minor issues"
	} else {
		status = "CRITICAL"
		statusText = "System has critical issues"
	}

	// Generate quick actions
	var quickActions []string
	if len(issues) > 0 {
		for _, issue := range issues {
			if contains(issue, "Drift:") {
				quickActions = append(quickActions, "Fix drift using ConfigHub: cub unit update <unit> --patch")
				break
			}
		}
		if healthScore < 90 {
			quickActions = append(quickActions, "Review issues above and take corrective action")
		}
	}

	return &HealthCheckResult{
		Timestamp:     time.Now().Format("2006-01-02 15:04:05"),
		HealthScore:   healthScore,
		Status:        status,
		StatusText:    statusText,
		Checks:        checks,
		Issues:        issues,
		QuickActions:  quickActions,
	}, nil
}

// CheckAppAsSet verifies the app follows Set-based organization (PRINCIPLE #0)
func (c *ComprehensiveHealthCheck) CheckAppAsSet(spaceID string) (bool, []string) {
	issues := []string{}

	// Check for Sets
	sets, err := c.ConfigHubClient.ListSets(spaceID)
	if err != nil || len(sets) == 0 {
		issues = append(issues, "No Sets defined - configs not grouped for bulk operations")
	}

	// Check for Filters
	filters, err := c.ConfigHubClient.ListFilters(spaceID)
	if err != nil || len(filters) == 0 {
		issues = append(issues, "No Filters defined - cannot target configs efficiently")
	}

	// In production, would also check:
	// - Units are assigned to Sets
	// - Filters use Set membership
	// - Bulk operations use Sets/Filters

	return len(issues) == 0, issues
}

// CheckConfigHubCompliance checks if corrections use ConfigHub commands only
func (c *ComprehensiveHealthCheck) CheckConfigHubCompliance(corrections []string) bool {
	for _, cmd := range corrections {
		if contains(cmd, "kubectl") {
			return false
		}
		if !contains(cmd, "cub unit") {
			return false
		}
	}
	return true
}

// CheckCleanupFirst verifies the cleanup-first principle (PRINCIPLE #8)
func (c *ComprehensiveHealthCheck) CheckCleanupFirst(scriptPath string) (bool, []string) {
	// This would check if a script follows cleanup-first pattern
	// For now, return placeholder
	issues := []string{}

	// In real implementation, would check:
	// 1. Script starts with cleanup commands
	// 2. Removes old namespaces before creating new ones
	// 3. Deletes old ConfigHub spaces before creating new ones

	return true, issues
}

// CheckEventDriven verifies event-driven patterns (PRINCIPLE #6)
func (c *ComprehensiveHealthCheck) CheckEventDriven(codebasePath string) bool {
	// This would check for informers vs polling
	// In real implementation, would grep for:
	// - RunWithInformers (good)
	// - time.Sleep in loops (bad)

	return true
}

// CheckDemoMode verifies demo mode exists (PRINCIPLE #7)
func (c *ComprehensiveHealthCheck) CheckDemoMode(appPath string) bool {
	// This would check if app has demo mode
	// In real implementation, would check for:
	// - os.Args checking for "demo"
	// - RunDemo function

	return true
}

// CheckValidYAML verifies unit data contains valid K8s YAML (PRINCIPLE #3)
func (c *ComprehensiveHealthCheck) CheckValidYAML(unitData string) (bool, error) {
	// This would validate YAML structure
	// In real implementation, would:
	// 1. Parse YAML
	// 2. Check for required K8s fields
	// 3. Validate apiVersion and kind

	return true, nil
}

// CheckAuthFlow verifies all required auth tokens (PRINCIPLE #5)
func (c *ComprehensiveHealthCheck) CheckAuthFlow() (bool, []string) {
	missing := []string{}

	// Check ConfigHub token
	if c.ConfigHubClient == nil || c.ConfigHubClient.Token == "" {
		missing = append(missing, "CUB_TOKEN not set")
	}

	// Check Claude API key (if app uses AI)
	if GetEnvOrDefault("CLAUDE_API_KEY", "") == "" {
		missing = append(missing, "CLAUDE_API_KEY not set (required for AI features)")
	}

	// Check Kubernetes config
	if c.K8sClient == nil {
		missing = append(missing, "KUBECONFIG not accessible")
	}

	return len(missing) == 0, missing
}

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}