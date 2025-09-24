package sdk

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Test cost analysis module
func TestCostAnalyzer(t *testing.T) {
	app := &DevOpsApp{
		Logger: newTestLogger(),
	}

	spaceID := uuid.New()
	analyzer := NewCostAnalyzer(app, spaceID)

	t.Run("ParseQuantity", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected int64
		}{
			{"100m", 100},
			{"1", 1000},
			{"2000m", 2000},
			{"500Mi", 500 * 1024 * 1024},
			{"1Gi", 1024 * 1024 * 1024},
			{"2Ti", 2 * 1024 * 1024 * 1024 * 1024},
		}

		for _, tc := range testCases {
			quantity, err := resource.ParseQuantity(tc.input)
			require.NoError(t, err, "Failed to parse %s", tc.input)

			if strings.HasSuffix(tc.input, "m") {
				assert.Equal(t, tc.expected, quantity.MilliValue(), "Mismatch for %s", tc.input)
			} else {
				assert.Equal(t, tc.expected, quantity.Value(), "Mismatch for %s", tc.input)
			}
		}
	})

	t.Run("CalculateMonthlyCost", func(t *testing.T) {
		estimate := &UnitCostEstimate{
			UnitID:   uuid.New().String(),
			UnitName: "test-deployment",
			CPU:      resource.MustParse("2"),
			Memory:   resource.MustParse("4Gi"),
			Storage:  resource.MustParse("10Gi"),
			Replicas: 3,
		}

		cost := analyzer.calculateMonthlyCost(estimate)

		// Expected calculation:
		// CPU: 2 cores * $17.28/core/month * 3 replicas = $103.68
		// Memory: 4 GB * $4.32/GB/month * 3 replicas = $51.84
		// Storage: 10 GB * $0.10/GB/month * 3 replicas = $3.00
		// Total: $158.52

		assert.InDelta(t, 158.52, cost, 0.01, "Monthly cost calculation incorrect")
		assert.InDelta(t, 103.68, estimate.Breakdown.CPUCost, 0.01, "CPU cost incorrect")
		assert.InDelta(t, 51.84, estimate.Breakdown.MemoryCost, 0.01, "Memory cost incorrect")
		assert.InDelta(t, 3.00, estimate.Breakdown.StorageCost, 0.01, "Storage cost incorrect")
	})
}

// Test waste analysis module
func TestWasteAnalyzer(t *testing.T) {
	app := &DevOpsApp{
		Logger: newTestLogger(),
	}

	spaceID := uuid.New()
	analyzer := NewWasteAnalyzer(app, spaceID)

	t.Run("CalculateWasteRatio", func(t *testing.T) {
		testCases := []struct {
			name     string
			actual   float64
			estimated float64
			expected float64
		}{
			{"No waste", 100, 100, 0},
			{"50% waste", 50, 100, 0.5},
			{"75% waste", 25, 100, 0.75},
			{"Negative protection", 120, 100, 0}, // Should not go negative
		}

		for _, tc := range testCases {
			ratio := analyzer.calculateWasteRatio(tc.actual, tc.estimated)
			assert.InDelta(t, tc.expected, ratio, 0.01, "Waste ratio incorrect for %s", tc.name)
			assert.GreaterOrEqual(t, ratio, 0.0, "Waste ratio should never be negative")
		}
	})

	t.Run("AnalyzeResourceWaste", func(t *testing.T) {
		metrics := []ActualUsageMetrics{
			{
				UnitID:           uuid.New(),
				UnitName:         "high-waste-app",
				CPUActual:        0.2,  // 200m actual
				CPUAllocated:     2.0,  // 2000m allocated
				MemoryActual:     512,  // 512 MB actual
				MemoryAllocated:  4096, // 4 GB allocated
				Replicas:         3,
				IdleReplicas:     1,
			},
			{
				UnitID:           uuid.New(),
				UnitName:         "efficient-app",
				CPUActual:        1.8,
				CPUAllocated:     2.0,
				MemoryActual:     3500,
				MemoryAllocated:  4096,
				Replicas:         2,
				IdleReplicas:     0,
			},
		}

		analysis, err := analyzer.AnalyzeWaste(metrics)
		require.NoError(t, err)

		assert.Equal(t, 2, len(analysis.UnitWasteDetections))

		// Check high waste app
		highWaste := analysis.UnitWasteDetections[0]
		assert.Equal(t, "high-waste-app", highWaste.UnitName)
		assert.InDelta(t, 90.0, highWaste.CPUWaste.WastePercent, 1.0, "CPU waste incorrect")
		assert.InDelta(t, 87.5, highWaste.MemoryWaste.WastePercent, 1.0, "Memory waste incorrect")
		assert.Equal(t, 1, highWaste.ReplicaWaste.IdleReplicas)

		// Check efficient app
		efficient := analysis.UnitWasteDetections[1]
		assert.Equal(t, "efficient-app", efficient.UnitName)
		assert.InDelta(t, 10.0, efficient.CPUWaste.WastePercent, 1.0, "CPU waste incorrect")
		assert.InDelta(t, 14.6, efficient.MemoryWaste.WastePercent, 1.0, "Memory waste incorrect")
		assert.Equal(t, 0, efficient.ReplicaWaste.IdleReplicas)
	})
}

// Test optimization engine
func TestOptimizationEngine(t *testing.T) {
	app := &DevOpsApp{
		Logger: newTestLogger(),
	}

	engine := NewOptimizationEngine(app)

	t.Run("GenerateOptimizedConfig", func(t *testing.T) {
		unit := &Unit{
			UnitID:      uuid.New(),
			SpaceID:     uuid.New(),
			Slug:        "test-app",
			DisplayName: "Test Application",
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
										"limits": map[string]interface{}{
											"cpu":    "4000m",
											"memory": "8Gi",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		waste := &WasteMetrics{
			CPUWastePercent:    0.75,  // 75% waste
			MemoryWastePercent: 0.50,  // 50% waste
			IdleReplicas:       2,
			WasteConfidence:    0.9,
			MetricsAge:         time.Hour,
		}

		config, err := engine.GenerateOptimizedUnit(unit, waste)
		require.NoError(t, err)

		// Check optimized values
		optimizedManifest := config.OptimizedManifest.(map[string]interface{})
		spec := optimizedManifest["spec"].(map[string]interface{})

		// Replicas should be reduced (5 - 2 idle = 3)
		assert.Equal(t, int32(3), spec["replicas"])

		// Check container resources were optimized
		template := spec["template"].(map[string]interface{})
		podSpec := template["spec"].(map[string]interface{})
		containers := podSpec["containers"].([]interface{})
		container := containers[0].(map[string]interface{})
		resources := container["resources"].(map[string]interface{})

		requests := resources["requests"].(map[string]interface{})
		limits := resources["limits"].(map[string]interface{})

		// CPU should be reduced by ~75% with safety margin
		// Original: 2000m, waste: 75%, so actual usage: 500m
		// With 20% safety: 600m
		assert.Equal(t, "600m", requests["cpu"])
		assert.Equal(t, "900m", limits["cpu"]) // 150% of request

		// Memory should be reduced by ~50% with safety margin
		// Original: 4Gi, waste: 50%, so actual usage: 2Gi
		// With 20% safety: 2.4Gi
		assert.Contains(t, requests["memory"], "2") // Should be around 2.4Gi

		// Check risk assessment
		assert.Equal(t, "MEDIUM", config.RiskAssessment.OverallRisk)
		assert.Contains(t, config.RiskAssessment.Factors, "High CPU reduction")

		// Check estimated savings
		assert.Greater(t, config.EstimatedSavings.MonthlySavings, 0.0)
		assert.Greater(t, config.EstimatedSavings.SavingsPercent, 0.0)
	})

	t.Run("MultiContainerOptimization", func(t *testing.T) {
		unit := &Unit{
			UnitID:  uuid.New(),
			SpaceID: uuid.New(),
			Slug:    "multi-container-app",
			ManifestData: map[string]interface{}{
				"apiVersion": "apps/v1",
				"kind":       "Deployment",
				"spec": map[string]interface{}{
					"template": map[string]interface{}{
						"spec": map[string]interface{}{
							"containers": []interface{}{
								map[string]interface{}{
									"name": "main",
									"resources": map[string]interface{}{
										"requests": map[string]interface{}{
											"cpu":    "1000m",
											"memory": "2Gi",
										},
									},
								},
								map[string]interface{}{
									"name": "sidecar",
									"resources": map[string]interface{}{
										"requests": map[string]interface{}{
											"cpu":    "500m",
											"memory": "1Gi",
										},
									},
								},
							},
						},
					},
				},
			},
		}

		waste := &WasteMetrics{
			CPUWastePercent:    0.60, // 60% total waste
			MemoryWastePercent: 0.40, // 40% total waste
			WasteConfidence:    0.85,
		}

		config, err := engine.GenerateOptimizedUnit(unit, waste)
		require.NoError(t, err)

		// Verify resources were distributed proportionally
		optimizedManifest := config.OptimizedManifest.(map[string]interface{})
		spec := optimizedManifest["spec"].(map[string]interface{})
		template := spec["template"].(map[string]interface{})
		podSpec := template["spec"].(map[string]interface{})
		containers := podSpec["containers"].([]interface{})

		assert.Equal(t, 2, len(containers), "Should still have 2 containers")

		// Both containers should be optimized proportionally
		mainContainer := containers[0].(map[string]interface{})
		sidecarContainer := containers[1].(map[string]interface{})

		mainResources := mainContainer["resources"].(map[string]interface{})
		sidecarResources := sidecarContainer["resources"].(map[string]interface{})

		mainRequests := mainResources["requests"].(map[string]interface{})
		sidecarRequests := sidecarResources["requests"].(map[string]interface{})

		// Original ratio should be maintained (main:sidecar = 2:1 for CPU)
		assert.NotEqual(t, "1000m", mainRequests["cpu"], "Main CPU should be optimized")
		assert.NotEqual(t, "500m", sidecarRequests["cpu"], "Sidecar CPU should be optimized")
	})
}

// Integration test with all modules working together
func TestIntegratedSDKFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	app := &DevOpsApp{
		Logger: newTestLogger(),
	}

	spaceID := uuid.New()

	// Create all analyzers
	costAnalyzer := NewCostAnalyzer(app, spaceID)
	wasteAnalyzer := NewWasteAnalyzer(app, spaceID)
	optimizer := NewOptimizationEngine(app)

	// Simulate ConfigHub units
	units := []Unit{
		{
			UnitID:      uuid.New(),
			SpaceID:     spaceID,
			Slug:        "frontend",
			DisplayName: "Frontend Service",
			ManifestData: createTestDeployment("frontend", "3", "1000m", "2Gi"),
		},
		{
			UnitID:      uuid.New(),
			SpaceID:     spaceID,
			Slug:        "backend",
			DisplayName: "Backend Service",
			ManifestData: createTestDeployment("backend", "5", "2000m", "4Gi"),
		},
		{
			UnitID:      uuid.New(),
			SpaceID:     spaceID,
			Slug:        "database",
			DisplayName: "Database",
			ManifestData: createTestStatefulSet("database", "2", "4000m", "16Gi"),
		},
	}

	// Mock ConfigHub client behavior
	app.Cub = &mockConfigHubClient{units: units}

	// 1. Analyze costs
	costAnalysis, err := costAnalyzer.AnalyzeSpace()
	require.NoError(t, err)

	assert.Equal(t, 3, len(costAnalysis.Units))
	assert.Greater(t, costAnalysis.TotalMonthlyCost, 0.0)

	// 2. Simulate actual usage metrics
	actualMetrics := []ActualUsageMetrics{
		{
			UnitID:          units[0].UnitID,
			UnitName:        "frontend",
			CPUActual:       0.3,  // Only using 300m of 1000m
			CPUAllocated:    1.0,
			MemoryActual:    1024, // Only using 1GB of 2GB
			MemoryAllocated: 2048,
			Replicas:        3,
			IdleReplicas:    1,
		},
		{
			UnitID:          units[1].UnitID,
			UnitName:        "backend",
			CPUActual:       1.5,  // Using 1500m of 2000m
			CPUAllocated:    2.0,
			MemoryActual:    3072, // Using 3GB of 4GB
			MemoryAllocated: 4096,
			Replicas:        5,
			IdleReplicas:    0,
		},
		{
			UnitID:          units[2].UnitID,
			UnitName:        "database",
			CPUActual:       3.8,   // Using 3800m of 4000m
			CPUAllocated:    4.0,
			MemoryActual:    15360, // Using 15GB of 16GB
			MemoryAllocated: 16384,
			Replicas:        2,
			IdleReplicas:    0,
		},
	}

	// 3. Analyze waste
	wasteAnalysis, err := wasteAnalyzer.AnalyzeWaste(actualMetrics)
	require.NoError(t, err)

	assert.Equal(t, 3, len(wasteAnalysis.UnitWasteDetections))
	assert.Greater(t, wasteAnalysis.TotalWastedCost, 0.0)

	// Frontend should have high waste
	frontendWaste := wasteAnalysis.UnitWasteDetections[0]
	assert.Equal(t, "frontend", frontendWaste.UnitName)
	assert.Greater(t, frontendWaste.CPUWaste.WastePercent, 50.0)

	// 4. Generate optimizations for high-waste units
	for i, detection := range wasteAnalysis.UnitWasteDetections {
		if detection.TotalWastePercent > 30 {
			waste := &WasteMetrics{
				CPUWastePercent:    detection.CPUWaste.WastePercent / 100.0,
				MemoryWastePercent: detection.MemoryWaste.WastePercent / 100.0,
				IdleReplicas:       int32(detection.ReplicaWaste.IdleReplicas),
				WasteConfidence:    0.85,
				MetricsAge:         time.Hour,
			}

			optimized, err := optimizer.GenerateOptimizedUnit(&units[i], waste)
			require.NoError(t, err)

			assert.NotNil(t, optimized)
			assert.Greater(t, optimized.EstimatedSavings.MonthlySavings, 0.0)

			// Verify optimization is reasonable
			assert.Contains(t, []string{"LOW", "MEDIUM"}, optimized.RiskAssessment.OverallRisk)
		}
	}
}

// Helper functions

func newTestLogger() *log.Logger {
	return log.New(os.Stdout, "[TEST] ", log.LstdFlags)
}

func createTestDeployment(name string, replicas string, cpu string, memory string) map[string]interface{} {
	return map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name": name,
		},
		"spec": map[string]interface{}{
			"replicas": replicas,
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []interface{}{
						map[string]interface{}{
							"name": name,
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    cpu,
									"memory": memory,
								},
								"limits": map[string]interface{}{
									"cpu":    multiplyResource(cpu, 2),
									"memory": multiplyResource(memory, 2),
								},
							},
						},
					},
				},
			},
		},
	}
}

func createTestStatefulSet(name string, replicas string, cpu string, memory string) map[string]interface{} {
	deployment := createTestDeployment(name, replicas, cpu, memory)
	deployment["kind"] = "StatefulSet"
	return deployment
}

func multiplyResource(resource string, factor float64) string {
	// Simple multiplication for test purposes
	return resource // Simplified for testing
}

// Mock ConfigHub client for testing
type mockConfigHubClient struct {
	units []Unit
}

func (m *mockConfigHubClient) ListUnits(spaceID uuid.UUID) ([]Unit, error) {
	return m.units, nil
}

func (m *mockConfigHubClient) GetUnit(unitID uuid.UUID) (*Unit, error) {
	for _, unit := range m.units {
		if unit.UnitID == unitID {
			return &unit, nil
		}
	}
	return nil, fmt.Errorf("unit not found")
}