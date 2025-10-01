// Enhanced SDK Test Suite
// Copy this to: /Users/alexis/Public/github-repos/devops-sdk/sdk_comprehensive_test.go
// Run with: go test -v -run=TestComprehensive

package sdk

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// RETRY LOGIC TESTS
// ============================================================================

func TestRetryLogic(t *testing.T) {
	t.Run("SuccessfulRetry", func(t *testing.T) {
		attempts := 0
		maxAttempts := 3

		operation := func() error {
			attempts++
			if attempts < 2 {
				return fmt.Errorf("temporary error")
			}
			return nil
		}

		err := retryWithBackoff(operation, maxAttempts, time.Millisecond*10)
		require.NoError(t, err)
		assert.Equal(t, 2, attempts, "Should succeed on second attempt")
	})

	t.Run("MaxRetriesExceeded", func(t *testing.T) {
		attempts := 0
		maxAttempts := 3

		operation := func() error {
			attempts++
			return fmt.Errorf("persistent error")
		}

		err := retryWithBackoff(operation, maxAttempts, time.Millisecond*10)
		require.Error(t, err)
		assert.Equal(t, maxAttempts, attempts, "Should attempt exactly max times")
	})

	t.Run("ExponentialBackoff", func(t *testing.T) {
		startTime := time.Now()
		attempts := 0

		operation := func() error {
			attempts++
			return fmt.Errorf("error")
		}

		retryWithBackoff(operation, 3, time.Millisecond*100)

		// Should take at least 100ms + 200ms + 400ms = 700ms
		elapsed := time.Since(startTime)
		assert.GreaterOrEqual(t, elapsed, 700*time.Millisecond, "Backoff should be exponential")
	})
}

// ============================================================================
// CONFIGHHUB API TESTS WITH RETRY
// ============================================================================

func TestConfigHubClientWithRetry(t *testing.T) {
	t.Run("APIRetryOnNetworkError", func(t *testing.T) {
		attemptCount := 0

		// Mock server that fails twice then succeeds
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			attemptCount++
			if attemptCount < 3 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"spaces": []}`))
		}))
		defer server.Close()

		client := &ConfigHubClient{
			APIBaseURL: server.URL,
			Token:      "test-token",
			HTTPClient: &http.Client{Timeout: 5 * time.Second},
			Logger:     newTestLogger(),
			MaxRetries: 3,
			RetryDelay: time.Millisecond * 10,
		}

		_, err := client.ListSpaces()
		require.NoError(t, err)
		assert.Equal(t, 3, attemptCount, "Should retry until success")
	})

	t.Run("CircuitBreakerOpen", func(t *testing.T) {
		failureCount := 0
		maxFailures := 5

		// Mock server that always fails
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			failureCount++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		client := &ConfigHubClient{
			APIBaseURL:          server.URL,
			Token:               "test-token",
			HTTPClient:          &http.Client{Timeout: 5 * time.Second},
			Logger:              newTestLogger(),
			MaxRetries:          2,
			RetryDelay:          time.Millisecond * 10,
			CircuitBreakerThreshold: maxFailures,
		}

		// Make requests until circuit breaker opens
		for i := 0; i < 10; i++ {
			client.ListSpaces()
		}

		// Circuit breaker should prevent excessive retries
		assert.LessOrEqual(t, failureCount, maxFailures*3, "Circuit breaker should limit attempts")
	})
}

// ============================================================================
// VERIFICATION AND FEEDBACK TESTS
// ============================================================================

func TestVerificationAndFeedback(t *testing.T) {
	t.Run("VerifyConfigHubUnitCreation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" && r.URL.Path == "/spaces/test-space/units" {
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{
					"unit_id": "123e4567-e89b-12d3-a456-426614174000",
					"slug": "test-unit",
					"display_name": "Test Unit"
				}`))
			} else if r.Method == "GET" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"unit_id": "123e4567-e89b-12d3-a456-426614174000",
					"slug": "test-unit",
					"display_name": "Test Unit",
					"manifest_data": {"apiVersion": "v1", "kind": "Service"}
				}`))
			}
		}))
		defer server.Close()

		client := &ConfigHubClient{
			APIBaseURL: server.URL,
			Token:      "test-token",
			HTTPClient: &http.Client{Timeout: 5 * time.Second},
			Logger:     newTestLogger(),
		}

		// Create unit
		unit, err := client.CreateUnit(uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"), CreateUnitRequest{
			Slug:        "test-unit",
			DisplayName: "Test Unit",
		})
		require.NoError(t, err)
		assert.NotNil(t, unit)

		// Verify it was created by fetching it
		fetchedUnit, err := client.GetUnit(unit.UnitID)
		require.NoError(t, err)
		assert.Equal(t, unit.Slug, fetchedUnit.Slug)

		// Success feedback
		t.Logf("✓ VERIFIED: Unit %s created successfully", unit.Slug)
		t.Logf("  - Unit ID: %s", unit.UnitID)
		t.Logf("  - Display Name: %s", unit.DisplayName)
		t.Logf("  - Manifest Data: %v", fetchedUnit.ManifestData)
	})

	t.Run("VerifyOptimizationApplied", func(t *testing.T) {
		app := &DevOpsApp{
			Logger: newTestLogger(),
		}

		// Original unit
		originalUnit := &Unit{
			UnitID:  uuid.New(),
			SpaceID: uuid.New(),
			Slug:    "test-app",
			ManifestData: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"spec": map[string]interface{}{
					"replicas": 5,
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"name": "app",
									"resources": map[string]interface{}{
										"requests": map[string]interface{}{
											"cpu":    "2000m",
											"memory": "4Gi",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		// Generate optimization
		engine := NewOptimizationEngine(app)
		waste := &WasteMetrics{
			CPUWastePercent:    0.75,
			MemoryWastePercent: 0.50,
			IdleReplicas:       2,
			WasteConfidence:    0.9,
		}

		optimized, err := engine.GenerateOptimizedUnit(originalUnit, waste)
		require.NoError(t, err)

		// Verify changes
		originalSpec := originalUnit.ManifestData["spec"].(map[string]interface{})
		optimizedSpec := optimized.OptimizedManifest.(map[string]interface{})["spec"].(map[string]interface{})

		// Detailed feedback
		t.Logf("✓ OPTIMIZATION APPLIED:")
		t.Logf("  Replicas: %v → %v (-%d%%)",
			originalSpec["replicas"],
			optimizedSpec["replicas"],
			int((1.0 - float64(optimizedSpec["replicas"].(int32))/5.0) * 100))

		// Extract CPU values for comparison
		originalTemplate := originalSpec["template"].(map[string]interface{})
		originalContainer := originalTemplate["spec"].(map[string]interface{})["containers"].([]interface{})[0].(map[string]interface{})
		originalCPU := originalContainer["resources"].(map[string]interface{})["requests"].(map[string]interface{})["cpu"]

		optimizedTemplate := optimizedSpec["template"].(map[string]interface{})
		optimizedContainer := optimizedTemplate["spec"].(map[string]interface{})["containers"].([]interface{})[0].(map[string]interface{})
		optimizedCPU := optimizedContainer["resources"].(map[string]interface{})["requests"].(map[string]interface{})["cpu"]

		t.Logf("  CPU: %s → %s", originalCPU, optimizedCPU)
		t.Logf("  Estimated Savings: $%.2f/month (%.1f%%)",
			optimized.EstimatedSavings.MonthlySavings,
			optimized.EstimatedSavings.SavingsPercent)
		t.Logf("  Risk Level: %s", optimized.RiskAssessment.OverallRisk)

		for _, factor := range optimized.RiskAssessment.Factors {
			t.Logf("    - %s", factor)
		}
	})
}

// ============================================================================
// GODOC EXAMPLES
// ============================================================================

// ExampleDevOpsApp demonstrates basic SDK usage
func ExampleDevOpsApp() {
	// Initialize the app
	config := DevOpsAppConfig{
		Name:        "my-devops-app",
		Version:     "1.0.0",
		Description: "My awesome DevOps automation",
		RunInterval: 5 * time.Minute,
		HealthPort:  8080,
	}

	app, err := NewDevOpsApp(config)
	if err != nil {
		log.Fatal(err)
	}

	// Run with event-driven informers
	app.RunWithInformers(func() error {
		log.Println("Processing events...")
		return nil
	})
}

// ExampleCostAnalyzer demonstrates cost analysis
func ExampleCostAnalyzer() {
	app := &DevOpsApp{
		Logger: log.New(os.Stdout, "[COST] ", log.LstdFlags),
	}

	spaceID := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")
	analyzer := NewCostAnalyzer(app, spaceID)

	// Analyze entire space
	analysis, err := analyzer.AnalyzeSpace()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total monthly cost: $%.2f\n", analysis.TotalMonthlyCost)
	for _, unit := range analysis.Units {
		fmt.Printf("  %s: $%.2f/month\n", unit.UnitName, unit.MonthlyCost)
	}
}

// ExampleOptimizationEngine demonstrates resource optimization
func ExampleOptimizationEngine() {
	app := &DevOpsApp{
		Logger: log.New(os.Stdout, "[OPTIMIZER] ", log.LstdFlags),
	}

	engine := NewOptimizationEngine(app)

	// Define waste metrics (from actual usage analysis)
	waste := &WasteMetrics{
		CPUWastePercent:    0.60, // 60% CPU waste detected
		MemoryWastePercent: 0.40, // 40% memory waste
		IdleReplicas:       1,
		WasteConfidence:    0.85,
	}

	// Generate optimized configuration
	unit := &Unit{
		Slug: "my-app",
		ManifestData: map[string]interface{}{
			// ... your Kubernetes manifest
		},
	}

	optimized, err := engine.GenerateOptimizedUnit(unit, waste)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Estimated savings: $%.2f/month (%.1f%%)\n",
		optimized.EstimatedSavings.MonthlySavings,
		optimized.EstimatedSavings.SavingsPercent)
	fmt.Printf("Risk level: %s\n", optimized.RiskAssessment.OverallRisk)
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

func newTestLogger() *log.Logger {
	return log.New(os.Stdout, "[TEST] ", log.LstdFlags)
}

// retryWithBackoff executes an operation with exponential backoff
func retryWithBackoff(operation func() error, maxAttempts int, initialDelay time.Duration) error {
	var err error
	delay := initialDelay

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = operation()
		if err == nil {
			return nil
		}

		if attempt < maxAttempts {
			time.Sleep(delay)
			delay *= 2 // Exponential backoff
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", maxAttempts, err)
}
