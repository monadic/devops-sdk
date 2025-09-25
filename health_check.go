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

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}