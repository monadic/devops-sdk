package sdk

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/client/clientset/versioned"
)

// K8sClients holds all Kubernetes client types
type K8sClients struct {
	Clientset     *kubernetes.Clientset
	DynamicClient dynamic.Interface
	MetricsClient *metricsv1beta1.Clientset
	Config        *rest.Config
}

// NewK8sClients creates all necessary Kubernetes clients
func NewK8sClients() (*K8sClients, error) {
	config, err := GetK8sConfig()
	if err != nil {
		return nil, fmt.Errorf("get k8s config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}

	metricsClient, err := metricsv1beta1.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create metrics client: %w", err)
	}

	return &K8sClients{
		Clientset:     clientset,
		DynamicClient: dynamicClient,
		MetricsClient: metricsClient,
		Config:        config,
	}, nil
}

// GetK8sConfig returns a Kubernetes config from kubeconfig or in-cluster
func GetK8sConfig() (*rest.Config, error) {
	// Try KUBECONFIG env var first
	kubeconfig := os.Getenv("KUBECONFIG")

	// If not set, try default location
	if kubeconfig == "" {
		home := os.Getenv("HOME")
		if home != "" {
			kubeconfig = filepath.Join(home, ".kube", "config")
		}
	}

	// Try to build from kubeconfig file
	if kubeconfig != "" {
		if _, err := os.Stat(kubeconfig); err == nil {
			config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err == nil {
				return config, nil
			}
		}
	}

	// Fall back to in-cluster config
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create config: %w", err)
	}

	return config, nil
}

// GetNamespace returns the namespace to operate in
func GetNamespace() string {
	// Check environment variable
	if ns := os.Getenv("NAMESPACE"); ns != "" {
		return ns
	}

	// Try to read from service account
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return string(data)
	}

	// Default
	return "default"
}

// ResourceHelper provides utility functions for resource operations
type ResourceHelper struct {
	Context context.Context
}

// NewResourceHelper creates a new resource helper
func NewResourceHelper() *ResourceHelper {
	return &ResourceHelper{
		Context: context.Background(),
	}
}

// GetResourceValue extracts a nested value from a resource map
func (r *ResourceHelper) GetResourceValue(data map[string]interface{}, path string) interface{} {
	keys := splitPath(path)
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			return current[key]
		}

		next, ok := current[key].(map[string]interface{})
		if !ok {
			return nil
		}
		current = next
	}

	return nil
}

// SetResourceValue sets a nested value in a resource map
func (r *ResourceHelper) SetResourceValue(data map[string]interface{}, path string, value interface{}) {
	keys := splitPath(path)
	current := data

	for i, key := range keys {
		if i == len(keys)-1 {
			current[key] = value
			return
		}

		next, ok := current[key].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			current[key] = next
		}
		current = next
	}
}

// CompareResourceValues compares two resource values
func (r *ResourceHelper) CompareResourceValues(expected, actual interface{}) bool {
	// Handle different types
	switch e := expected.(type) {
	case map[string]interface{}:
		a, ok := actual.(map[string]interface{})
		if !ok {
			return false
		}
		return r.compareMaps(e, a)
	case []interface{}:
		a, ok := actual.([]interface{})
		if !ok {
			return false
		}
		return r.compareSlices(e, a)
	default:
		return fmt.Sprintf("%v", expected) == fmt.Sprintf("%v", actual)
	}
}

func (r *ResourceHelper) compareMaps(expected, actual map[string]interface{}) bool {
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			return false
		}
		if !r.CompareResourceValues(expectedValue, actualValue) {
			return false
		}
	}
	return true
}

func (r *ResourceHelper) compareSlices(expected, actual []interface{}) bool {
	if len(expected) != len(actual) {
		return false
	}
	for i := range expected {
		if !r.CompareResourceValues(expected[i], actual[i]) {
			return false
		}
	}
	return true
}

func splitPath(path string) []string {
	var result []string
	current := ""

	for _, char := range path {
		if char == '.' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
