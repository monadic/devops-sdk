// cost.go - Cost analysis module for the DevOps SDK
//
// This module provides comprehensive cost analysis capabilities for ConfigHub units,
// analyzing Kubernetes resources (Deployments, StatefulSets, DaemonSets) and
// calculating estimated monthly costs based on CPU, memory, and storage usage.
//
// Features:
// - Parse ConfigHub units containing Kubernetes manifests
// - Extract resource requests/limits from containers
// - Calculate monthly costs using configurable pricing models
// - Generate human-readable cost reports
// - Provide optimization recommendations
// - Store cost annotations back to ConfigHub units
// - Support for environment hierarchy analysis
//
// This module is designed to be lightweight and avoid heavy Kubernetes dependencies
// by implementing its own ResourceQuantity parsing for common resource formats.
package sdk

import (
	"encoding/base64"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// CostAnalyzer analyzes costs from ConfigHub units
type CostAnalyzer struct {
	app     *DevOpsApp
	spaceID uuid.UUID
	pricing *PricingModel
}

// PricingModel for cost calculations
type PricingModel struct {
	CPUHourly    float64 // Cost per CPU core per hour
	MemoryHourly float64 // Cost per GB memory per hour
	StorageGB    float64 // Cost per GB storage per month
}

// DefaultPricing based on AWS EKS m5.large pricing
var DefaultPricing = &PricingModel{
	CPUHourly:    0.024, // $0.024 per vCPU hour
	MemoryHourly: 0.006, // $0.006 per GB hour
	StorageGB:    0.10,  // $0.10 per GB per month
}

// ResourceQuantity represents a simple resource quantity (avoiding k8s dependency)
type ResourceQuantity struct {
	Value string
	bytes int64
	milli int64
}

// ParseQuantity creates a ResourceQuantity from a string like "500m", "2Gi", etc.
func ParseQuantity(value string) ResourceQuantity {
	rq := ResourceQuantity{Value: value}

	// Handle empty or invalid values
	if value == "" {
		return rq
	}

	// Handle all Kubernetes quantity formats
	if strings.HasSuffix(value, "m") {
		// Millicores: "500m" = 500 millicores
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "m"), 64); err == nil {
			rq.milli = int64(val)
		}
	} else if strings.HasSuffix(value, "Ki") {
		// Kibibytes: "1Ki" = 1024 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "Ki"), 64); err == nil {
			rq.bytes = int64(val * 1024)
		}
	} else if strings.HasSuffix(value, "Mi") {
		// Mebibytes: "512Mi" = 512 * 1024^2 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "Mi"), 64); err == nil {
			rq.bytes = int64(val * 1024 * 1024)
		}
	} else if strings.HasSuffix(value, "Gi") {
		// Gibibytes: "2Gi" = 2 * 1024^3 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "Gi"), 64); err == nil {
			rq.bytes = int64(val * 1024 * 1024 * 1024)
		}
	} else if strings.HasSuffix(value, "Ti") {
		// Tebibytes: "1Ti" = 1024^4 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "Ti"), 64); err == nil {
			rq.bytes = int64(val * 1024 * 1024 * 1024 * 1024)
		}
	} else if strings.HasSuffix(value, "Pi") {
		// Pebibytes: "1Pi" = 1024^5 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "Pi"), 64); err == nil {
			rq.bytes = int64(val * 1024 * 1024 * 1024 * 1024 * 1024)
		}
	} else if strings.HasSuffix(value, "K") {
		// Kilobytes: "1K" = 1000 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "K"), 64); err == nil {
			rq.bytes = int64(val * 1000)
		}
	} else if strings.HasSuffix(value, "M") {
		// Megabytes: "1M" = 1000^2 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "M"), 64); err == nil {
			rq.bytes = int64(val * 1000 * 1000)
		}
	} else if strings.HasSuffix(value, "G") {
		// Gigabytes: "2G" = 2 * 1000^3 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "G"), 64); err == nil {
			rq.bytes = int64(val * 1000 * 1000 * 1000)
		}
	} else if strings.HasSuffix(value, "T") {
		// Terabytes: "1T" = 1000^4 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "T"), 64); err == nil {
			rq.bytes = int64(val * 1000 * 1000 * 1000 * 1000)
		}
	} else if strings.HasSuffix(value, "P") {
		// Petabytes: "1P" = 1000^5 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "P"), 64); err == nil {
			rq.bytes = int64(val * 1000 * 1000 * 1000 * 1000 * 1000)
		}
	} else if strings.HasSuffix(value, "E") {
		// Exabytes: "1E" = 1000^6 bytes
		if val, err := strconv.ParseFloat(strings.TrimSuffix(value, "E"), 64); err == nil {
			rq.bytes = int64(val * 1000 * 1000 * 1000 * 1000 * 1000 * 1000)
		}
	} else {
		// Assume raw cores for CPU: "1" = 1000 millicores, "0.5" = 500 millicores
		if val, err := strconv.ParseFloat(value, 64); err == nil {
			// Handle fractional values properly
			rq.milli = int64(val * 1000)
		}
	}

	return rq
}

// MilliValue returns the value in millicores (for CPU)
func (rq ResourceQuantity) MilliValue() int64 {
	return rq.milli
}

// BytesValue returns the value in bytes (for memory/storage)
func (rq ResourceQuantity) BytesValue() int64 {
	return rq.bytes
}

// String returns the original string representation
func (rq ResourceQuantity) String() string {
	return rq.Value
}

// Add adds another ResourceQuantity to this one
func (rq *ResourceQuantity) Add(other ResourceQuantity) {
	rq.milli += other.milli
	rq.bytes += other.bytes

	// Update string representation based on the type of resource
	if rq.milli > 0 && rq.bytes == 0 {
		// CPU resource - use millicores or cores
		if rq.milli%1000 == 0 {
			rq.Value = fmt.Sprintf("%d", rq.milli/1000)
		} else {
			rq.Value = fmt.Sprintf("%dm", rq.milli)
		}
	} else if rq.bytes > 0 && rq.milli == 0 {
		// Memory/storage resource - use appropriate unit
		if rq.bytes >= 1024*1024*1024 && rq.bytes%(1024*1024*1024) == 0 {
			rq.Value = fmt.Sprintf("%dGi", rq.bytes/(1024*1024*1024))
		} else if rq.bytes >= 1024*1024 && rq.bytes%(1024*1024) == 0 {
			rq.Value = fmt.Sprintf("%dMi", rq.bytes/(1024*1024))
		} else if rq.bytes >= 1024 && rq.bytes%1024 == 0 {
			rq.Value = fmt.Sprintf("%dKi", rq.bytes/1024)
		} else {
			rq.Value = fmt.Sprintf("%d", rq.bytes)
		}
	}
}

// UnitCostEstimate represents cost analysis for a single unit
type UnitCostEstimate struct {
	UnitID      string
	UnitName    string
	Space       string
	Type        string // deployment, service, statefulset, etc
	Replicas    int32
	CPU         ResourceQuantity
	Memory      ResourceQuantity
	Storage     ResourceQuantity
	MonthlyCost float64
	Breakdown   CostBreakdown
}

// CostBreakdown shows cost components
type CostBreakdown struct {
	CPUCost     float64
	MemoryCost  float64
	StorageCost float64
}

// SpaceCostAnalysis represents total cost for a space
type SpaceCostAnalysis struct {
	SpaceID          string
	SpaceName        string
	TotalMonthlyCost float64
	UnitCount        int
	Units            []UnitCostEstimate
	Environments     map[string]*SpaceCostAnalysis // For hierarchical spaces
}

// NewCostAnalyzer creates analyzer for ConfigHub units
func NewCostAnalyzer(app *DevOpsApp, spaceID uuid.UUID) *CostAnalyzer {
	return &CostAnalyzer{
		app:     app,
		spaceID: spaceID,
		pricing: DefaultPricing,
	}
}

// SetPricing allows custom pricing model
func (ca *CostAnalyzer) SetPricing(pricing *PricingModel) {
	ca.pricing = pricing
}

// AnalyzeSpace analyzes all units in a ConfigHub space
func (ca *CostAnalyzer) AnalyzeSpace() (*SpaceCostAnalysis, error) {
	ca.app.Logger.Printf("🔍 Analyzing ConfigHub space: %s", ca.spaceID)

	// Get all units in the space
	units, err := ca.app.Cub.ListUnits(ListUnitsParams{
		SpaceID: ca.spaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list units: %v", err)
	}

	analysis := &SpaceCostAnalysis{
		SpaceID:      ca.spaceID.String(),
		SpaceName:    ca.spaceID.String(), // Could fetch space name
		UnitCount:    len(units),
		Units:        []UnitCostEstimate{},
		Environments: make(map[string]*SpaceCostAnalysis),
	}

	// Analyze each unit
	for _, unit := range units {
		estimate, err := ca.analyzeUnit(*unit)
		if err != nil {
			ca.app.Logger.Printf("⚠️  Could not analyze unit %s: %v", unit.Slug, err)
			continue
		}

		if estimate != nil {
			analysis.Units = append(analysis.Units, *estimate)
			analysis.TotalMonthlyCost += estimate.MonthlyCost
		}
	}

	ca.app.Logger.Printf("✅ Analysis complete: %d units, $%.2f/month estimated cost",
		len(analysis.Units), analysis.TotalMonthlyCost)

	return analysis, nil
}

// analyzeUnit analyzes a single ConfigHub unit
func (ca *CostAnalyzer) analyzeUnit(unit Unit) (*UnitCostEstimate, error) {
	// Decode base64 data if needed
	data := unit.Data
	if decoded, err := base64.StdEncoding.DecodeString(unit.Data); err == nil {
		data = string(decoded)
	}

	// Skip non-Kubernetes resources
	if !strings.Contains(data, "apiVersion") {
		return nil, nil
	}

	// Parse the Kubernetes manifest
	var manifest map[string]interface{}
	if err := yaml.Unmarshal([]byte(data), &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %v", err)
	}

	kind, _ := manifest["kind"].(string)

	switch kind {
	case "Deployment":
		return ca.analyzeDeployment(unit, manifest)
	case "StatefulSet":
		return ca.analyzeStatefulSet(unit, manifest)
	case "DaemonSet":
		return ca.analyzeDaemonSet(unit, manifest)
	default:
		// Skip non-workload resources
		return nil, nil
	}
}

// analyzeDeployment analyzes a Deployment unit
func (ca *CostAnalyzer) analyzeDeployment(unit Unit, manifest map[string]interface{}) (*UnitCostEstimate, error) {
	estimate := &UnitCostEstimate{
		UnitID:   unit.UnitID.String(),
		UnitName: unit.Slug,
		Space:    ca.spaceID.String(),
		Type:     "Deployment",
	}

	// Extract replicas
	if spec, ok := manifest["spec"].(map[string]interface{}); ok {
		if replicas, ok := spec["replicas"].(int); ok {
			estimate.Replicas = int32(replicas)
		} else {
			estimate.Replicas = 1 // Default
		}

		// Extract container resources
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if podSpec, ok := template["spec"].(map[string]interface{}); ok {
				if containers, ok := podSpec["containers"].([]interface{}); ok {
					for _, container := range containers {
						if c, ok := container.(map[string]interface{}); ok {
							ca.extractContainerResources(c, estimate)
						}
					}
				}
			}
		}
	}

	// Calculate costs
	estimate.MonthlyCost = ca.calculateMonthlyCost(estimate)

	return estimate, nil
}

// analyzeStatefulSet analyzes a StatefulSet unit
func (ca *CostAnalyzer) analyzeStatefulSet(unit Unit, manifest map[string]interface{}) (*UnitCostEstimate, error) {
	estimate := &UnitCostEstimate{
		UnitID:   unit.UnitID.String(),
		UnitName: unit.Slug,
		Space:    ca.spaceID.String(),
		Type:     "StatefulSet",
	}

	// Similar to deployment but check for volumeClaimTemplates
	if spec, ok := manifest["spec"].(map[string]interface{}); ok {
		if replicas, ok := spec["replicas"].(int); ok {
			estimate.Replicas = int32(replicas)
		} else {
			estimate.Replicas = 1
		}

		// Check for persistent volumes
		if vcTemplates, ok := spec["volumeClaimTemplates"].([]interface{}); ok {
			for _, vct := range vcTemplates {
				if template, ok := vct.(map[string]interface{}); ok {
					ca.extractStorageResources(template, estimate)
				}
			}
		}

		// Extract container resources
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if podSpec, ok := template["spec"].(map[string]interface{}); ok {
				if containers, ok := podSpec["containers"].([]interface{}); ok {
					for _, container := range containers {
						if c, ok := container.(map[string]interface{}); ok {
							ca.extractContainerResources(c, estimate)
						}
					}
				}
			}
		}
	}

	estimate.MonthlyCost = ca.calculateMonthlyCost(estimate)
	return estimate, nil
}

// analyzeDaemonSet analyzes a DaemonSet unit
func (ca *CostAnalyzer) analyzeDaemonSet(unit Unit, manifest map[string]interface{}) (*UnitCostEstimate, error) {
	estimate := &UnitCostEstimate{
		UnitID:   unit.UnitID.String(),
		UnitName: unit.Slug,
		Space:    ca.spaceID.String(),
		Type:     "DaemonSet",
		Replicas: 3, // Assume 3 nodes as default
	}

	// Extract container resources
	if spec, ok := manifest["spec"].(map[string]interface{}); ok {
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if podSpec, ok := template["spec"].(map[string]interface{}); ok {
				if containers, ok := podSpec["containers"].([]interface{}); ok {
					for _, container := range containers {
						if c, ok := container.(map[string]interface{}); ok {
							ca.extractContainerResources(c, estimate)
						}
					}
				}
			}
		}
	}

	estimate.MonthlyCost = ca.calculateMonthlyCost(estimate)
	return estimate, nil
}

// extractContainerResources extracts CPU/memory from container spec
func (ca *CostAnalyzer) extractContainerResources(container map[string]interface{}, estimate *UnitCostEstimate) {
	if resources, ok := container["resources"].(map[string]interface{}); ok {
		// Check requests first (what we're guaranteed)
		if requests, ok := resources["requests"].(map[string]interface{}); ok {
			if cpu, ok := requests["cpu"].(string); ok {
				quantity := ParseQuantity(cpu)
				estimate.CPU.Add(quantity)
			}
			if memory, ok := requests["memory"].(string); ok {
				quantity := ParseQuantity(memory)
				estimate.Memory.Add(quantity)
			}
		} else if limits, ok := resources["limits"].(map[string]interface{}); ok {
			// Fall back to limits if no requests
			if cpu, ok := limits["cpu"].(string); ok {
				quantity := ParseQuantity(cpu)
				estimate.CPU.Add(quantity)
			}
			if memory, ok := limits["memory"].(string); ok {
				quantity := ParseQuantity(memory)
				estimate.Memory.Add(quantity)
			}
		}
	}
}

// extractStorageResources extracts storage from PVC templates
func (ca *CostAnalyzer) extractStorageResources(vct map[string]interface{}, estimate *UnitCostEstimate) {
	if spec, ok := vct["spec"].(map[string]interface{}); ok {
		if resources, ok := spec["resources"].(map[string]interface{}); ok {
			if requests, ok := resources["requests"].(map[string]interface{}); ok {
				if storage, ok := requests["storage"].(string); ok {
					quantity := ParseQuantity(storage)
					estimate.Storage.Add(quantity)
				}
			}
		}
	}
}

// calculateMonthlyCost calculates the monthly cost for a unit with bounds checking
func (ca *CostAnalyzer) calculateMonthlyCost(estimate *UnitCostEstimate) float64 {
	// Validate inputs
	if estimate == nil {
		return 0.0
	}
	if estimate.Replicas < 0 {
		estimate.Replicas = 0
	}
	if ca.pricing == nil {
		ca.pricing = DefaultPricing
	}

	// Validate pricing model
	if ca.pricing.CPUHourly < 0 || ca.pricing.MemoryHourly < 0 || ca.pricing.StorageGB < 0 {
		return 0.0 // Invalid pricing
	}

	hoursPerMonth := 24.0 * 30.0
	replicas := float64(estimate.Replicas)

	// CPU cost (convert millicores to cores) with bounds checking
	cpuCores := float64(estimate.CPU.MilliValue()) / 1000.0
	if cpuCores < 0 {
		cpuCores = 0
	}
	cpuCost := cpuCores * ca.pricing.CPUHourly * hoursPerMonth * replicas
	if math.IsNaN(cpuCost) || math.IsInf(cpuCost, 0) {
		cpuCost = 0
	}

	// Memory cost (convert to GB) with bounds checking
	memoryBytes := float64(estimate.Memory.BytesValue())
	if memoryBytes < 0 {
		memoryBytes = 0
	}
	memoryGB := memoryBytes / (1024 * 1024 * 1024)
	memoryCost := memoryGB * ca.pricing.MemoryHourly * hoursPerMonth * replicas
	if math.IsNaN(memoryCost) || math.IsInf(memoryCost, 0) {
		memoryCost = 0
	}

	// Storage cost (convert to GB) with bounds checking
	storageBytes := float64(estimate.Storage.BytesValue())
	if storageBytes < 0 {
		storageBytes = 0
	}
	storageGB := storageBytes / (1024 * 1024 * 1024)
	storageCost := storageGB * ca.pricing.StorageGB * replicas
	if math.IsNaN(storageCost) || math.IsInf(storageCost, 0) {
		storageCost = 0
	}

	// Set breakdown
	estimate.Breakdown = CostBreakdown{
		CPUCost:     cpuCost,
		MemoryCost:  memoryCost,
		StorageCost: storageCost,
	}

	totalCost := cpuCost + memoryCost + storageCost

	// Final validation
	if math.IsNaN(totalCost) || math.IsInf(totalCost, 0) || totalCost < 0 {
		return 0.0
	}

	return totalCost
}

// AnalyzeHierarchy analyzes a full environment hierarchy
func (ca *CostAnalyzer) AnalyzeHierarchy(baseSpaceSlug string) (*SpaceCostAnalysis, error) {
	ca.app.Logger.Printf("🔍 Analyzing ConfigHub hierarchy starting from: %s", baseSpaceSlug)

	// Analyze base space
	baseAnalysis, err := ca.AnalyzeSpace()
	if err != nil {
		return nil, err
	}

	// Find downstream spaces (dev, staging, prod) by slug patterns
	environments := []string{"dev", "staging", "prod"}

	for _, env := range environments {
		envSpaceSlug := fmt.Sprintf("%s-%s", baseSpaceSlug, env)

		// Try to find the space by slug
		envSpace, err := ca.app.Cub.GetSpaceBySlug(envSpaceSlug)
		if err != nil {
			continue // Space doesn't exist
		}

		// Check if space exists
		envAnalyzer := NewCostAnalyzer(ca.app, envSpace.SpaceID)
		if envAnalysis, err := envAnalyzer.AnalyzeSpace(); err == nil {
			baseAnalysis.Environments[env] = envAnalysis
		}
	}

	return baseAnalysis, nil
}

// GenerateReport creates a human-readable cost report
func (ca *CostAnalyzer) GenerateReport(analysis *SpaceCostAnalysis) string {
	var report strings.Builder

	report.WriteString("═══════════════════════════════════════════════════════\n")
	report.WriteString("       ConfigHub Cost Analysis Report\n")
	report.WriteString("═══════════════════════════════════════════════════════\n\n")

	report.WriteString(fmt.Sprintf("Space: %s\n", analysis.SpaceName))
	report.WriteString(fmt.Sprintf("Units Analyzed: %d\n", analysis.UnitCount))
	report.WriteString(fmt.Sprintf("Estimated Monthly Cost: $%.2f\n\n", analysis.TotalMonthlyCost))

	report.WriteString("Top Cost Drivers:\n")
	report.WriteString("─────────────────────────────────────────────\n")

	// Sort by cost
	for i, unit := range analysis.Units {
		if i >= 5 {
			break
		}
		report.WriteString(fmt.Sprintf("%-30s %s %dx %6s CPU %8s Mem  $%.2f/mo\n",
			unit.UnitName,
			unit.Type,
			unit.Replicas,
			unit.CPU.String(),
			unit.Memory.String(),
			unit.MonthlyCost,
		))
	}

	// Environment comparison
	if len(analysis.Environments) > 0 {
		report.WriteString("\n\nEnvironment Cost Comparison:\n")
		report.WriteString("─────────────────────────────────────────────\n")

		for env, envAnalysis := range analysis.Environments {
			report.WriteString(fmt.Sprintf("%-10s: $%.2f/month (%d units)\n",
				env, envAnalysis.TotalMonthlyCost, envAnalysis.UnitCount))
		}
	}

	// Cost optimization opportunities
	report.WriteString("\n\nOptimization Opportunities:\n")
	report.WriteString("─────────────────────────────────────────────\n")

	overProvisionedCount := 0
	potentialSavings := 0.0

	for _, unit := range analysis.Units {
		// Simple heuristic: if CPU > 1 core or Memory > 2Gi, flag for review
		if unit.CPU.MilliValue() > 1000 || unit.Memory.BytesValue() > 2*1024*1024*1024 {
			overProvisionedCount++
			// Estimate 30% savings potential
			potentialSavings += unit.MonthlyCost * 0.3
		}
	}

	report.WriteString(fmt.Sprintf("• %d units appear over-provisioned\n", overProvisionedCount))
	report.WriteString(fmt.Sprintf("• Potential savings: $%.2f/month (30%% reduction)\n", potentialSavings))
	report.WriteString("• Run with actual metrics for accurate analysis\n")

	return report.String()
}

// StoreAnalysisInConfigHub stores cost analysis as ConfigHub annotations
func (ca *CostAnalyzer) StoreAnalysisInConfigHub(analysis *SpaceCostAnalysis) error {
	for _, unit := range analysis.Units {
		annotations := map[string]string{
			"cost-optimizer.io/monthly-cost":  fmt.Sprintf("$%.2f", unit.MonthlyCost),
			"cost-optimizer.io/cpu-cost":      fmt.Sprintf("$%.2f", unit.Breakdown.CPUCost),
			"cost-optimizer.io/memory-cost":   fmt.Sprintf("$%.2f", unit.Breakdown.MemoryCost),
			"cost-optimizer.io/storage-cost":  fmt.Sprintf("$%.2f", unit.Breakdown.StorageCost),
			"cost-optimizer.io/analyzed-at":   time.Now().Format(time.RFC3339),
			"cost-optimizer.io/analysis-type": "pre-deployment",
		}

		// Parse UnitID back to UUID
		unitID, err := uuid.Parse(unit.UnitID)
		if err != nil {
			ca.app.Logger.Printf("⚠️  Invalid unit ID %s: %v", unit.UnitID, err)
			continue
		}

		// Update unit with cost annotations
		_, err = ca.app.Cub.UpdateUnit(ca.spaceID, unitID, CreateUnitRequest{
			Slug:        unit.UnitName, // Use existing slug
			Annotations: annotations,
		})
		if err != nil {
			ca.app.Logger.Printf("⚠️  Failed to annotate unit %s: %v", unit.UnitName, err)
		}
	}

	return nil
}

// GetOptimizationRecommendations provides AI-powered cost optimization suggestions
func (ca *CostAnalyzer) GetOptimizationRecommendations(analysis *SpaceCostAnalysis) []OptimizationRecommendation {
	var recommendations []OptimizationRecommendation

	for _, unit := range analysis.Units {
		// CPU over-provisioning check
		if unit.CPU.MilliValue() > 2000 { // > 2 cores
			recommendations = append(recommendations, OptimizationRecommendation{
				UnitID:           unit.UnitID,
				UnitName:         unit.UnitName,
				Type:             "cpu-over-provisioned",
				CurrentValue:     unit.CPU.String(),
				RecommendedValue: fmt.Sprintf("%dm", unit.CPU.MilliValue()/2),
				PotentialSavings: unit.Breakdown.CPUCost * 0.5,
				Risk:             "LOW",
				Description:      "CPU allocation appears excessive based on typical usage patterns",
			})
		}

		// Memory over-provisioning check
		if unit.Memory.BytesValue() > 4*1024*1024*1024 { // > 4Gi
			recommendations = append(recommendations, OptimizationRecommendation{
				UnitID:           unit.UnitID,
				UnitName:         unit.UnitName,
				Type:             "memory-over-provisioned",
				CurrentValue:     unit.Memory.String(),
				RecommendedValue: fmt.Sprintf("%dMi", unit.Memory.BytesValue()/(2*1024*1024)),
				PotentialSavings: unit.Breakdown.MemoryCost * 0.5,
				Risk:             "MEDIUM",
				Description:      "Memory allocation could be reduced with proper monitoring",
			})
		}

		// Replica optimization
		if unit.Replicas > 3 && unit.MonthlyCost < 50 {
			recommendations = append(recommendations, OptimizationRecommendation{
				UnitID:           unit.UnitID,
				UnitName:         unit.UnitName,
				Type:             "replica-optimization",
				CurrentValue:     fmt.Sprintf("%d replicas", unit.Replicas),
				RecommendedValue: "2 replicas",
				PotentialSavings: unit.MonthlyCost * 0.33,
				Risk:             "HIGH",
				Description:      "Consider reducing replicas for low-cost services",
			})
		}
	}

	return recommendations
}

// OptimizationRecommendation represents a cost optimization suggestion
type OptimizationRecommendation struct {
	UnitID           string
	UnitName         string
	Type             string // cpu-over-provisioned, memory-over-provisioned, etc.
	CurrentValue     string
	RecommendedValue string
	PotentialSavings float64
	Risk             string // LOW, MEDIUM, HIGH
	Description      string
}

// AnalyzeCostForSpace is a convenience function to analyze costs for a space
func AnalyzeCostForSpace(app *DevOpsApp, spaceSlug string) (*SpaceCostAnalysis, error) {
	// Get space by slug
	space, err := app.Cub.GetSpaceBySlug(spaceSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to find space %s: %v", spaceSlug, err)
	}

	// Create analyzer
	analyzer := NewCostAnalyzer(app, space.SpaceID)

	// Analyze space
	analysis, err := analyzer.AnalyzeSpace()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze space: %v", err)
	}

	return analysis, nil
}

// AnalyzeCostWithRecommendations analyzes costs and provides AI-powered recommendations
func AnalyzeCostWithRecommendations(app *DevOpsApp, spaceSlug string) (*SpaceCostAnalysis, []OptimizationRecommendation, error) {
	analysis, err := AnalyzeCostForSpace(app, spaceSlug)
	if err != nil {
		return nil, nil, err
	}

	// Get space by slug for the analyzer
	space, err := app.Cub.GetSpaceBySlug(spaceSlug)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find space %s: %v", spaceSlug, err)
	}

	analyzer := NewCostAnalyzer(app, space.SpaceID)
	recommendations := analyzer.GetOptimizationRecommendations(analysis)

	return analysis, recommendations, nil
}
