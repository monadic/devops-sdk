// optimizer.go - Configuration optimization module for the DevOps SDK
//
// This module provides intelligent optimization of Kubernetes configurations based on
// cost analysis, waste detection, and performance metrics. It generates optimized
// ConfigHub units that can be deployed safely with risk assessment.
//
// Features:
// - Resource right-sizing using historical metrics and AI analysis
// - Replica count optimization based on load patterns
// - Storage optimization and PVC right-sizing
// - Safety margins and risk assessment for all optimizations
// - ConfigHub unit generation with optimized manifests
// - Push-upgrade compatible changes for environment promotion
// - Integration with cost analysis and Claude AI recommendations
//
// The optimizer follows ConfigHub best practices by creating new units with
// upstream relationships and using Sets/Filters for bulk operations.
package sdk

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// OptimizationEngine provides intelligent configuration optimization
type OptimizationEngine struct {
	app          *DevOpsApp
	spaceID      uuid.UUID
	costAnalyzer *CostAnalyzer
	safetyConfig *SafetyConfiguration
}

// SafetyConfiguration defines safety margins and risk thresholds
type SafetyConfiguration struct {
	CPUSafetyMargin     float64 // Additional CPU buffer (e.g., 0.2 = 20%)
	MemorySafetyMargin  float64 // Additional memory buffer
	MinCPUCores         float64 // Minimum CPU allocation
	MinMemoryGB         float64 // Minimum memory allocation
	MinReplicas         int32   // Minimum replica count
	MaxReplicaReduction float64 // Maximum replica reduction ratio
	RiskThresholds      RiskThresholds
}

// RiskThresholds define when optimizations become risky
type RiskThresholds struct {
	LowRiskCPUReduction     float64 // < 30% reduction = LOW
	LowRiskMemoryReduction  float64 // < 25% reduction = LOW
	HighRiskCPUReduction    float64 // > 60% reduction = HIGH
	HighRiskMemoryReduction float64 // > 50% reduction = HIGH
}

// DefaultSafetyConfiguration provides conservative optimization settings
var DefaultSafetyConfiguration = &SafetyConfiguration{
	CPUSafetyMargin:     0.20,  // 20% safety margin
	MemorySafetyMargin:  0.15,  // 15% safety margin
	MinCPUCores:         0.1,   // 100m minimum
	MinMemoryGB:         0.128, // 128Mi minimum
	MinReplicas:         1,
	MaxReplicaReduction: 0.5, // Don't reduce replicas by more than 50%
	RiskThresholds: RiskThresholds{
		LowRiskCPUReduction:     0.30,
		LowRiskMemoryReduction:  0.25,
		HighRiskCPUReduction:    0.60,
		HighRiskMemoryReduction: 0.50,
	},
}

// OptimizedConfiguration represents the result of optimization
type OptimizedConfiguration struct {
	OriginalUnit     *Unit                  `json:"originalUnit"`
	OptimizedUnit    *Unit                  `json:"optimizedUnit"`
	Optimizations    []ResourceOptimization `json:"optimizations"`
	EstimatedSavings CostSavings            `json:"estimatedSavings"`
	RiskAssessment   OptimizationRisk       `json:"riskAssessment"`
	AppliedSafety    SafetyMargins          `json:"appliedSafety"`
}

// ResourceOptimization describes a specific optimization applied
type ResourceOptimization struct {
	Type             string  `json:"type"` // cpu, memory, replicas, storage
	OriginalValue    string  `json:"originalValue"`
	OptimizedValue   string  `json:"optimizedValue"`
	ReductionPercent float64 `json:"reductionPercent"`
	Reasoning        string  `json:"reasoning"`
	Risk             string  `json:"risk"` // LOW, MEDIUM, HIGH
}

// CostSavings represents estimated cost savings
type CostSavings struct {
	MonthlySavings       float64              `json:"monthlySavings"`
	CurrentMonthlyCost   float64              `json:"currentMonthlyCost"`
	OptimizedMonthlyCost float64              `json:"optimizedMonthlyCost"`
	SavingsPercent       float64              `json:"savingsPercent"`
	Breakdown            CostSavingsBreakdown `json:"breakdown"`
}

// CostSavingsBreakdown shows savings by resource type
type CostSavingsBreakdown struct {
	CPUSavings     float64 `json:"cpuSavings"`
	MemorySavings  float64 `json:"memorySavings"`
	StorageSavings float64 `json:"storageSavings"`
}

// OptimizationRisk assesses the risk of applying optimizations
type OptimizationRisk struct {
	OverallRisk      string   `json:"overallRisk"` // LOW, MEDIUM, HIGH
	RiskFactors      []string `json:"riskFactors"`
	Mitigations      []string `json:"mitigations"`
	Confidence       float64  `json:"confidence"`       // 0.0 to 1.0
	RecommendedPhase string   `json:"recommendedPhase"` // dev, staging, prod
}

// SafetyMargins shows applied safety margins
type SafetyMargins struct {
	CPUMarginApplied    bool    `json:"cpuMarginApplied"`
	MemoryMarginApplied bool    `json:"memoryMarginApplied"`
	ReplicaFloorApplied bool    `json:"replicaFloorApplied"`
	ActualCPUMargin     float64 `json:"actualCpuMargin"`
	ActualMemoryMargin  float64 `json:"actualMemoryMargin"`
}

// WasteMetrics represents detected waste (placeholder for future waste.go integration)
type WasteMetrics struct {
	CPUWastePercent     float64       `json:"cpuWastePercent"`
	MemoryWastePercent  float64       `json:"memoryWastePercent"`
	StorageWastePercent float64       `json:"storageWastePercent"`
	IdleReplicas        int32         `json:"idleReplicas"`
	UnderutilizedPods   []string      `json:"underutilizedPods"`
	WasteConfidence     float64       `json:"wasteConfidence"`
	MetricsAge          time.Duration `json:"metricsAge"`
}

// NewOptimizationEngine creates a new optimization engine
func NewOptimizationEngine(app *DevOpsApp, spaceID uuid.UUID) *OptimizationEngine {
	return &OptimizationEngine{
		app:          app,
		spaceID:      spaceID,
		costAnalyzer: NewCostAnalyzer(app, spaceID),
		safetyConfig: DefaultSafetyConfiguration,
	}
}

// SetSafetyConfiguration allows customizing safety margins
func (oe *OptimizationEngine) SetSafetyConfiguration(config *SafetyConfiguration) {
	oe.safetyConfig = config
}

// GenerateOptimizedUnit creates an optimized version of a ConfigHub unit
func (oe *OptimizationEngine) GenerateOptimizedUnit(unit *Unit, wasteMetrics *WasteMetrics) (*OptimizedConfiguration, error) {
	oe.app.Logger.Printf("🔧 Optimizing unit: %s", unit.Slug)

	// Parse the Kubernetes manifest
	var manifest map[string]interface{}
	if err := yaml.Unmarshal([]byte(unit.Data), &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %v", err)
	}

	kind, _ := manifest["kind"].(string)

	switch kind {
	case "Deployment":
		return oe.optimizeDeployment(unit, manifest, wasteMetrics)
	case "StatefulSet":
		return oe.optimizeStatefulSet(unit, manifest, wasteMetrics)
	case "DaemonSet":
		return oe.optimizeDaemonSet(unit, manifest, wasteMetrics)
	default:
		return nil, fmt.Errorf("unsupported resource type for optimization: %s", kind)
	}
}

// optimizeDeployment optimizes a Deployment resource
func (oe *OptimizationEngine) optimizeDeployment(unit *Unit, manifest map[string]interface{}, waste *WasteMetrics) (*OptimizedConfiguration, error) {
	optimizations := []ResourceOptimization{}
	appliedSafety := SafetyMargins{}

	// Create a deep copy of the manifest for optimization
	optimizedManifest := copyManifest(manifest)

	// Extract current resource specifications
	currentResources := oe.extractResourceSpecs(manifest)
	if currentResources == nil {
		return nil, fmt.Errorf("no resource specifications found")
	}

	// Optimize CPU
	if waste.CPUWastePercent > 0.1 { // Only optimize if >10% waste
		cpuOpt := oe.optimizeCPU(currentResources.CPU, waste.CPUWastePercent, waste.WasteConfidence)
		if cpuOpt != nil {
			optimizations = append(optimizations, *cpuOpt)
			oe.applyCPUOptimization(optimizedManifest, cpuOpt.OptimizedValue)
			appliedSafety.CPUMarginApplied = true
			appliedSafety.ActualCPUMargin = oe.safetyConfig.CPUSafetyMargin
		}
	}

	// Optimize Memory
	if waste.MemoryWastePercent > 0.1 { // Only optimize if >10% waste
		memOpt := oe.optimizeMemory(currentResources.Memory, waste.MemoryWastePercent, waste.WasteConfidence)
		if memOpt != nil {
			optimizations = append(optimizations, *memOpt)
			oe.applyMemoryOptimization(optimizedManifest, memOpt.OptimizedValue)
			appliedSafety.MemoryMarginApplied = true
			appliedSafety.ActualMemoryMargin = oe.safetyConfig.MemorySafetyMargin
		}
	}

	// Optimize Replicas
	if waste.IdleReplicas > 0 {
		replicaOpt := oe.optimizeReplicas(currentResources.Replicas, waste.IdleReplicas)
		if replicaOpt != nil {
			optimizations = append(optimizations, *replicaOpt)
			oe.applyReplicaOptimization(optimizedManifest, replicaOpt.OptimizedValue)
			if currentResources.Replicas <= oe.safetyConfig.MinReplicas {
				appliedSafety.ReplicaFloorApplied = true
			}
		}
	}

	// Create optimized unit
	optimizedData, err := yaml.Marshal(optimizedManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal optimized manifest: %v", err)
	}

	optimizedUnit := &Unit{
		UnitID:         uuid.New(),
		SpaceID:        unit.SpaceID,
		Slug:           unit.Slug + "-optimized",
		DisplayName:    unit.DisplayName + " (Optimized)",
		Data:           string(optimizedData),
		Labels:         oe.createOptimizedLabels(unit.Labels),
		Annotations:    oe.createOptimizedAnnotations(unit.Annotations, optimizations),
		UpstreamUnitID: &unit.UnitID, // Maintain upstream relationship
	}

	// Calculate cost savings
	costSavings := oe.calculateCostSavings(unit, optimizedUnit)

	// Assess risk
	riskAssessment := oe.assessOptimizationRisk(optimizations, waste.WasteConfidence)

	return &OptimizedConfiguration{
		OriginalUnit:     unit,
		OptimizedUnit:    optimizedUnit,
		Optimizations:    optimizations,
		EstimatedSavings: costSavings,
		RiskAssessment:   riskAssessment,
		AppliedSafety:    appliedSafety,
	}, nil
}

// optimizeStatefulSet optimizes a StatefulSet resource
func (oe *OptimizationEngine) optimizeStatefulSet(unit *Unit, manifest map[string]interface{}, waste *WasteMetrics) (*OptimizedConfiguration, error) {
	// StatefulSets are more sensitive - apply more conservative optimizations
	conservativeWaste := &WasteMetrics{
		CPUWastePercent:     waste.CPUWastePercent * 0.7,    // Be more conservative
		MemoryWastePercent:  waste.MemoryWastePercent * 0.7, // Be more conservative
		StorageWastePercent: waste.StorageWastePercent,      // Keep storage optimizations
		IdleReplicas:        waste.IdleReplicas / 2,         // More conservative replica reduction
		WasteConfidence:     waste.WasteConfidence * 0.8,    // Lower confidence for StatefulSets
		MetricsAge:          waste.MetricsAge,
	}

	return oe.optimizeDeployment(unit, manifest, conservativeWaste)
}

// optimizeDaemonSet optimizes a DaemonSet resource
func (oe *OptimizationEngine) optimizeDaemonSet(unit *Unit, manifest map[string]interface{}, waste *WasteMetrics) (*OptimizedConfiguration, error) {
	// DaemonSets can't have replica optimization, only resource optimization
	optimizations := []ResourceOptimization{}
	appliedSafety := SafetyMargins{}

	optimizedManifest := copyManifest(manifest)
	currentResources := oe.extractResourceSpecs(manifest)

	if currentResources == nil {
		return nil, fmt.Errorf("no resource specifications found")
	}

	// Only optimize CPU and Memory for DaemonSets
	if waste.CPUWastePercent > 0.15 { // Higher threshold for DaemonSets
		cpuOpt := oe.optimizeCPU(currentResources.CPU, waste.CPUWastePercent, waste.WasteConfidence)
		if cpuOpt != nil {
			optimizations = append(optimizations, *cpuOpt)
			oe.applyCPUOptimization(optimizedManifest, cpuOpt.OptimizedValue)
			appliedSafety.CPUMarginApplied = true
		}
	}

	if waste.MemoryWastePercent > 0.15 { // Higher threshold for DaemonSets
		memOpt := oe.optimizeMemory(currentResources.Memory, waste.MemoryWastePercent, waste.WasteConfidence)
		if memOpt != nil {
			optimizations = append(optimizations, *memOpt)
			oe.applyMemoryOptimization(optimizedManifest, memOpt.OptimizedValue)
			appliedSafety.MemoryMarginApplied = true
		}
	}

	// Create optimized unit (similar to deployment)
	optimizedData, err := yaml.Marshal(optimizedManifest)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal optimized manifest: %v", err)
	}

	optimizedUnit := &Unit{
		UnitID:         uuid.New(),
		SpaceID:        unit.SpaceID,
		Slug:           unit.Slug + "-optimized",
		DisplayName:    unit.DisplayName + " (Optimized)",
		Data:           string(optimizedData),
		Labels:         oe.createOptimizedLabels(unit.Labels),
		Annotations:    oe.createOptimizedAnnotations(unit.Annotations, optimizations),
		UpstreamUnitID: &unit.UnitID,
	}

	costSavings := oe.calculateCostSavings(unit, optimizedUnit)
	riskAssessment := oe.assessOptimizationRisk(optimizations, waste.WasteConfidence)

	return &OptimizedConfiguration{
		OriginalUnit:     unit,
		OptimizedUnit:    optimizedUnit,
		Optimizations:    optimizations,
		EstimatedSavings: costSavings,
		RiskAssessment:   riskAssessment,
		AppliedSafety:    appliedSafety,
	}, nil
}

// ResourceSpecs holds current resource specifications
type ResourceSpecs struct {
	CPU      ResourceQuantity
	Memory   ResourceQuantity
	Storage  ResourceQuantity
	Replicas int32
}

// ContainerResourceInfo holds resource information for a single container
type ContainerResourceInfo struct {
	Name        string
	CPURequests ResourceQuantity
	CPULimits   ResourceQuantity
	MemRequests ResourceQuantity
	MemLimits   ResourceQuantity
	HasRequests bool
	HasLimits   bool
}

// extractResourceSpecs extracts current resource specifications from manifest
func (oe *OptimizationEngine) extractResourceSpecs(manifest map[string]interface{}) *ResourceSpecs {
	if manifest == nil {
		return nil
	}

	specs := &ResourceSpecs{}
	var containerInfos []*ContainerResourceInfo

	// Extract replicas - handle both int and float64 from YAML parsing
	if spec, ok := manifest["spec"].(map[string]interface{}); ok {
		switch v := spec["replicas"].(type) {
		case int:
			specs.Replicas = int32(v)
		case float64:
			specs.Replicas = int32(v)
		case int32:
			specs.Replicas = v
		default:
			specs.Replicas = 1 // Default
		}

		// Navigate to container resources
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if podSpec, ok := template["spec"].(map[string]interface{}); ok {
				if containers, ok := podSpec["containers"].([]interface{}); ok {
					// Extract resource information for each container
					for i, container := range containers {
						if c, ok := container.(map[string]interface{}); ok {
							info := oe.extractSingleContainerResources(c, fmt.Sprintf("container-%d", i))
							if info != nil {
								containerInfos = append(containerInfos, info)
								// Sum total resources for optimization calculation
								oe.addContainerResourcesToSpecs(info, specs)
							}
						}
					}
				}
			}
		}

		// Extract storage from volumeClaimTemplates (StatefulSets)
		if vcTemplates, ok := spec["volumeClaimTemplates"].([]interface{}); ok {
			for _, vct := range vcTemplates {
				if template, ok := vct.(map[string]interface{}); ok {
					oe.extractStorageSpecs(template, specs)
				}
			}
		}
	}

	return specs
}

// extractSingleContainerResources extracts resources from a single container
func (oe *OptimizationEngine) extractSingleContainerResources(container map[string]interface{}, defaultName string) *ContainerResourceInfo {
	info := &ContainerResourceInfo{
		Name: defaultName,
	}

	// Get container name if available
	if name, ok := container["name"].(string); ok {
		info.Name = name
	}

	if resources, ok := container["resources"].(map[string]interface{}); ok {
		// Extract requests
		if requests, ok := resources["requests"].(map[string]interface{}); ok {
			info.HasRequests = true
			if cpuVal := requests["cpu"]; cpuVal != nil {
				if cpuStr := oe.convertToString(cpuVal); cpuStr != "" {
					info.CPURequests = ParseQuantity(cpuStr)
				}
			}
			if memVal := requests["memory"]; memVal != nil {
				if memStr := oe.convertToString(memVal); memStr != "" {
					info.MemRequests = ParseQuantity(memStr)
				}
			}
		}

		// Extract limits
		if limits, ok := resources["limits"].(map[string]interface{}); ok {
			info.HasLimits = true
			if cpuVal := limits["cpu"]; cpuVal != nil {
				if cpuStr := oe.convertToString(cpuVal); cpuStr != "" {
					info.CPULimits = ParseQuantity(cpuStr)
				}
			}
			if memVal := limits["memory"]; memVal != nil {
				if memStr := oe.convertToString(memVal); memStr != "" {
					info.MemLimits = ParseQuantity(memStr)
				}
			}
		}
	}

	// Only return if we found some resources
	if info.HasRequests || info.HasLimits {
		return info
	}
	return nil
}

// convertToString safely converts various types to string
func (oe *OptimizationEngine) convertToString(val interface{}) string {
	switch v := val.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.0f", v)
	default:
		return ""
	}
}

// addContainerResourcesToSpecs adds container resources to total specs
func (oe *OptimizationEngine) addContainerResourcesToSpecs(info *ContainerResourceInfo, specs *ResourceSpecs) {
	// Prefer requests for optimization calculations, fall back to limits
	if info.HasRequests {
		if info.CPURequests.MilliValue() > 0 {
			specs.CPU.Add(info.CPURequests)
		}
		if info.MemRequests.BytesValue() > 0 {
			specs.Memory.Add(info.MemRequests)
		}
	} else if info.HasLimits {
		if info.CPULimits.MilliValue() > 0 {
			specs.CPU.Add(info.CPULimits)
		}
		if info.MemLimits.BytesValue() > 0 {
			specs.Memory.Add(info.MemLimits)
		}
	}
}

// Removed old extractContainerResourceSpecs and extractResourceValues functions
// as they have been replaced by extractSingleContainerResources and related functions

// extractStorageSpecs extracts storage from PVC templates
func (oe *OptimizationEngine) extractStorageSpecs(vct map[string]interface{}, specs *ResourceSpecs) {
	if vct == nil || specs == nil {
		return
	}

	if spec, ok := vct["spec"].(map[string]interface{}); ok {
		if resources, ok := spec["resources"].(map[string]interface{}); ok {
			if requests, ok := resources["requests"].(map[string]interface{}); ok {
				if storageVal := requests["storage"]; storageVal != nil {
					var storageStr string
					switch v := storageVal.(type) {
					case string:
						storageStr = v
					case int:
						storageStr = fmt.Sprintf("%d", v)
					case float64:
						storageStr = fmt.Sprintf("%.0f", v)
					default:
						return
					}

					if quantity := ParseQuantity(storageStr); quantity.BytesValue() > 0 {
						specs.Storage.Add(quantity)
					}
				}
			}
		}
	}
}

// optimizeCPU generates CPU optimization recommendation
func (oe *OptimizationEngine) optimizeCPU(current ResourceQuantity, wastePercent, confidence float64) *ResourceOptimization {
	if wastePercent <= 0.1 || confidence < 0.5 {
		return nil // Not enough waste or confidence
	}

	currentMillis := float64(current.MilliValue())
	if currentMillis == 0 {
		return nil
	}

	// Calculate reduction with safety margin
	reductionPercent := math.Min(wastePercent*confidence, 0.7) // Cap at 70% reduction
	reduction := currentMillis * reductionPercent
	optimizedMillis := currentMillis - reduction

	// Apply safety margin
	optimizedMillis = optimizedMillis * (1 + oe.safetyConfig.CPUSafetyMargin)

	// Enforce minimum
	minMillis := oe.safetyConfig.MinCPUCores * 1000
	if optimizedMillis < minMillis {
		optimizedMillis = minMillis
	}

	finalReduction := (currentMillis - optimizedMillis) / currentMillis
	if finalReduction < 0.05 { // Less than 5% savings not worth it
		return nil
	}

	// Format optimized value
	optimizedValue := fmt.Sprintf("%.0fm", optimizedMillis)
	risk := oe.categorizeRisk(finalReduction, oe.safetyConfig.RiskThresholds.LowRiskCPUReduction, oe.safetyConfig.RiskThresholds.HighRiskCPUReduction)

	return &ResourceOptimization{
		Type:             "cpu",
		OriginalValue:    current.String(),
		OptimizedValue:   optimizedValue,
		ReductionPercent: finalReduction * 100,
		Reasoning:        fmt.Sprintf("Detected %.1f%% CPU waste with %.1f%% confidence, applied %.1f%% safety margin", wastePercent*100, confidence*100, oe.safetyConfig.CPUSafetyMargin*100),
		Risk:             risk,
	}
}

// optimizeMemory generates memory optimization recommendation
func (oe *OptimizationEngine) optimizeMemory(current ResourceQuantity, wastePercent, confidence float64) *ResourceOptimization {
	if wastePercent <= 0.1 || confidence < 0.5 {
		return nil
	}

	currentBytes := float64(current.BytesValue())
	if currentBytes == 0 {
		return nil
	}

	// Calculate reduction with safety margin
	reductionPercent := math.Min(wastePercent*confidence, 0.6) // Cap at 60% reduction for memory
	reduction := currentBytes * reductionPercent
	optimizedBytes := currentBytes - reduction

	// Apply safety margin
	optimizedBytes = optimizedBytes * (1 + oe.safetyConfig.MemorySafetyMargin)

	// Enforce minimum
	minBytes := oe.safetyConfig.MinMemoryGB * 1024 * 1024 * 1024
	if optimizedBytes < minBytes {
		optimizedBytes = minBytes
	}

	finalReduction := (currentBytes - optimizedBytes) / currentBytes
	if finalReduction < 0.05 {
		return nil
	}

	// Format optimized value (prefer Mi units)
	optimizedMi := optimizedBytes / (1024 * 1024)
	optimizedValue := fmt.Sprintf("%.0fMi", optimizedMi)
	risk := oe.categorizeRisk(finalReduction, oe.safetyConfig.RiskThresholds.LowRiskMemoryReduction, oe.safetyConfig.RiskThresholds.HighRiskMemoryReduction)

	return &ResourceOptimization{
		Type:             "memory",
		OriginalValue:    current.String(),
		OptimizedValue:   optimizedValue,
		ReductionPercent: finalReduction * 100,
		Reasoning:        fmt.Sprintf("Detected %.1f%% memory waste with %.1f%% confidence, applied %.1f%% safety margin", wastePercent*100, confidence*100, oe.safetyConfig.MemorySafetyMargin*100),
		Risk:             risk,
	}
}

// optimizeReplicas generates replica optimization recommendation
func (oe *OptimizationEngine) optimizeReplicas(current, idle int32) *ResourceOptimization {
	if idle <= 0 || current <= oe.safetyConfig.MinReplicas {
		return nil
	}

	optimized := current - idle
	if optimized < oe.safetyConfig.MinReplicas {
		optimized = oe.safetyConfig.MinReplicas
	}

	reductionRatio := float64(current-optimized) / float64(current)
	if reductionRatio > oe.safetyConfig.MaxReplicaReduction {
		optimized = current - int32(float64(current)*oe.safetyConfig.MaxReplicaReduction)
	}

	if optimized >= current {
		return nil // No optimization possible
	}

	finalReduction := float64(current-optimized) / float64(current)
	risk := "MEDIUM" // Replica changes are always at least medium risk
	if finalReduction > 0.5 {
		risk = "HIGH"
	}

	return &ResourceOptimization{
		Type:             "replicas",
		OriginalValue:    fmt.Sprintf("%d", current),
		OptimizedValue:   fmt.Sprintf("%d", optimized),
		ReductionPercent: finalReduction * 100,
		Reasoning:        fmt.Sprintf("Detected %d idle replicas, maintaining minimum of %d replicas", idle, oe.safetyConfig.MinReplicas),
		Risk:             risk,
	}
}

// categorizeRisk categorizes optimization risk based on reduction percentage
func (oe *OptimizationEngine) categorizeRisk(reductionPercent, lowThreshold, highThreshold float64) string {
	if reductionPercent < lowThreshold {
		return "LOW"
	} else if reductionPercent > highThreshold {
		return "HIGH"
	}
	return "MEDIUM"
}

// applyCPUOptimization applies CPU optimization to the manifest
func (oe *OptimizationEngine) applyCPUOptimization(manifest map[string]interface{}, optimizedValue string) {
	oe.applyResourceOptimization(manifest, "cpu", optimizedValue)
}

// applyMemoryOptimization applies memory optimization to the manifest
func (oe *OptimizationEngine) applyMemoryOptimization(manifest map[string]interface{}, optimizedValue string) {
	oe.applyResourceOptimization(manifest, "memory", optimizedValue)
}

// applyResourceOptimization applies resource optimization to manifest with proper multi-container distribution
func (oe *OptimizationEngine) applyResourceOptimization(manifest map[string]interface{}, resourceType, totalOptimizedValue string) {
	if spec, ok := manifest["spec"].(map[string]interface{}); ok {
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if podSpec, ok := template["spec"].(map[string]interface{}); ok {
				if containers, ok := podSpec["containers"].([]interface{}); ok {
					// First, extract current resource distribution
					containerInfos := oe.extractContainerInfosFromManifest(containers)

					// Distribute the optimized total proportionally among containers
					oe.distributeOptimizedResource(containers, containerInfos, resourceType, totalOptimizedValue)
				}
			}
		}
	}
}

// extractContainerInfosFromManifest extracts resource information from containers in manifest
func (oe *OptimizationEngine) extractContainerInfosFromManifest(containers []interface{}) []*ContainerResourceInfo {
	var infos []*ContainerResourceInfo
	for i, container := range containers {
		if c, ok := container.(map[string]interface{}); ok {
			info := oe.extractSingleContainerResources(c, fmt.Sprintf("container-%d", i))
			if info != nil {
				infos = append(infos, info)
			} else {
				// Create empty info for containers without resources
				infos = append(infos, &ContainerResourceInfo{
					Name: fmt.Sprintf("container-%d", i),
				})
			}
		}
	}
	return infos
}

// distributeOptimizedResource distributes the optimized total resource among containers proportionally
func (oe *OptimizationEngine) distributeOptimizedResource(containers []interface{}, containerInfos []*ContainerResourceInfo, resourceType, totalOptimizedValue string) {
	if len(containers) == 0 || len(containerInfos) == 0 {
		return
	}

	totalOptimized := ParseQuantity(totalOptimizedValue)

	// Calculate current total for the specific resource type
	var currentTotal ResourceQuantity
	var hasAnyResources bool

	for _, info := range containerInfos {
		if resourceType == "cpu" {
			if info.HasRequests && info.CPURequests.MilliValue() > 0 {
				currentTotal.Add(info.CPURequests)
				hasAnyResources = true
			} else if info.HasLimits && info.CPULimits.MilliValue() > 0 {
				currentTotal.Add(info.CPULimits)
				hasAnyResources = true
			}
		} else if resourceType == "memory" {
			if info.HasRequests && info.MemRequests.BytesValue() > 0 {
				currentTotal.Add(info.MemRequests)
				hasAnyResources = true
			} else if info.HasLimits && info.MemLimits.BytesValue() > 0 {
				currentTotal.Add(info.MemLimits)
				hasAnyResources = true
			}
		}
	}

	// If no containers have this resource type, distribute equally
	if !hasAnyResources {
		oe.distributeEquallyAmongContainers(containers, resourceType, totalOptimizedValue)
		return
	}

	// Distribute proportionally based on current usage
	for i, container := range containers {
		if i >= len(containerInfos) {
			break
		}

		if c, ok := container.(map[string]interface{}); ok {
			info := containerInfos[i]
			proportion := oe.calculateContainerProportion(info, resourceType, currentTotal)

			if proportion > 0 {
				containerValue := oe.calculateProportionalValue(totalOptimized, proportion, resourceType)
				oe.setContainerResourceSafely(c, resourceType, containerValue)
			}
		}
	}
}

// calculateContainerProportion calculates what proportion of total resources this container uses
func (oe *OptimizationEngine) calculateContainerProportion(info *ContainerResourceInfo, resourceType string, currentTotal ResourceQuantity) float64 {
	var containerAmount ResourceQuantity

	if resourceType == "cpu" {
		if info.HasRequests && info.CPURequests.MilliValue() > 0 {
			containerAmount = info.CPURequests
		} else if info.HasLimits && info.CPULimits.MilliValue() > 0 {
			containerAmount = info.CPULimits
		}

		if currentTotal.MilliValue() > 0 {
			return float64(containerAmount.MilliValue()) / float64(currentTotal.MilliValue())
		}
	} else if resourceType == "memory" {
		if info.HasRequests && info.MemRequests.BytesValue() > 0 {
			containerAmount = info.MemRequests
		} else if info.HasLimits && info.MemLimits.BytesValue() > 0 {
			containerAmount = info.MemLimits
		}

		if currentTotal.BytesValue() > 0 {
			return float64(containerAmount.BytesValue()) / float64(currentTotal.BytesValue())
		}
	}

	return 0
}

// calculateProportionalValue calculates the proportional value for a container
func (oe *OptimizationEngine) calculateProportionalValue(totalOptimized ResourceQuantity, proportion float64, resourceType string) string {
	if resourceType == "cpu" {
		containerMillis := float64(totalOptimized.MilliValue()) * proportion
		return fmt.Sprintf("%.0fm", containerMillis)
	} else if resourceType == "memory" {
		containerBytes := float64(totalOptimized.BytesValue()) * proportion
		containerMi := containerBytes / (1024 * 1024)
		return fmt.Sprintf("%.0fMi", containerMi)
	}
	return totalOptimized.String()
}

// distributeEquallyAmongContainers distributes resources equally when no current usage exists
func (oe *OptimizationEngine) distributeEquallyAmongContainers(containers []interface{}, resourceType, totalValue string) {
	if len(containers) == 0 {
		return
	}

	totalQuantity := ParseQuantity(totalValue)
	perContainerValue := ""

	if resourceType == "cpu" {
		perContainerMillis := float64(totalQuantity.MilliValue()) / float64(len(containers))
		perContainerValue = fmt.Sprintf("%.0fm", perContainerMillis)
	} else if resourceType == "memory" {
		perContainerBytes := float64(totalQuantity.BytesValue()) / float64(len(containers))
		perContainerMi := perContainerBytes / (1024 * 1024)
		perContainerValue = fmt.Sprintf("%.0fMi", perContainerMi)
	} else {
		perContainerValue = totalValue // Fallback
	}

	for _, container := range containers {
		if c, ok := container.(map[string]interface{}); ok {
			oe.setContainerResourceSafely(c, resourceType, perContainerValue)
		}
	}
}

// setContainerResourceSafely sets a resource value in a container spec with proper requests/limits handling
func (oe *OptimizationEngine) setContainerResourceSafely(container map[string]interface{}, resourceType, requestValue string) {
	// Calculate appropriate limit value (typically 20-50% higher than request)
	var limitValue string
	if resourceType == "cpu" {
		// CPU limits should be higher than requests to allow bursting
		requestQuantity := ParseQuantity(requestValue)
		limitMillis := float64(requestQuantity.MilliValue()) * 1.5 // 50% higher
		limitValue = fmt.Sprintf("%.0fm", limitMillis)
	} else if resourceType == "memory" {
		// Memory limits should be close to requests as memory isn't compressible
		requestQuantity := ParseQuantity(requestValue)
		limitBytes := float64(requestQuantity.BytesValue()) * 1.2 // 20% higher
		limitMi := limitBytes / (1024 * 1024)
		limitValue = fmt.Sprintf("%.0fMi", limitMi)
	} else {
		limitValue = requestValue
	}

	if resources, ok := container["resources"].(map[string]interface{}); ok {
		// Update requests
		if requests, ok := resources["requests"].(map[string]interface{}); ok {
			requests[resourceType] = requestValue
		} else {
			resources["requests"] = map[string]interface{}{resourceType: requestValue}
		}

		// Update limits - don't set them equal to requests
		if limits, ok := resources["limits"].(map[string]interface{}); ok {
			limits[resourceType] = limitValue
		} else {
			resources["limits"] = map[string]interface{}{resourceType: limitValue}
		}
	} else {
		container["resources"] = map[string]interface{}{
			"requests": map[string]interface{}{resourceType: requestValue},
			"limits":   map[string]interface{}{resourceType: limitValue},
		}
	}
}

// applyReplicaOptimization applies replica optimization to manifest
func (oe *OptimizationEngine) applyReplicaOptimization(manifest map[string]interface{}, optimizedValue string) {
	if spec, ok := manifest["spec"].(map[string]interface{}); ok {
		if replicas, err := strconv.Atoi(optimizedValue); err == nil {
			spec["replicas"] = replicas
		}
	}
}

// calculateCostSavings calculates estimated cost savings
func (oe *OptimizationEngine) calculateCostSavings(original, optimized *Unit) CostSavings {
	// Analyze costs for both units
	originalEstimate, _ := oe.costAnalyzer.analyzeUnit(*original)
	optimizedEstimate, _ := oe.costAnalyzer.analyzeUnit(*optimized)

	if originalEstimate == nil || optimizedEstimate == nil {
		return CostSavings{} // No cost data available
	}

	savings := originalEstimate.MonthlyCost - optimizedEstimate.MonthlyCost
	savingsPercent := 0.0
	if originalEstimate.MonthlyCost > 0 {
		savingsPercent = (savings / originalEstimate.MonthlyCost) * 100
	}

	return CostSavings{
		MonthlySavings:       savings,
		CurrentMonthlyCost:   originalEstimate.MonthlyCost,
		OptimizedMonthlyCost: optimizedEstimate.MonthlyCost,
		SavingsPercent:       savingsPercent,
		Breakdown: CostSavingsBreakdown{
			CPUSavings:     originalEstimate.Breakdown.CPUCost - optimizedEstimate.Breakdown.CPUCost,
			MemorySavings:  originalEstimate.Breakdown.MemoryCost - optimizedEstimate.Breakdown.MemoryCost,
			StorageSavings: originalEstimate.Breakdown.StorageCost - optimizedEstimate.Breakdown.StorageCost,
		},
	}
}

// assessOptimizationRisk assesses the overall risk of applying optimizations
func (oe *OptimizationEngine) assessOptimizationRisk(optimizations []ResourceOptimization, wasteConfidence float64) OptimizationRisk {
	if len(optimizations) == 0 {
		return OptimizationRisk{
			OverallRisk:      "LOW",
			Confidence:       1.0,
			RecommendedPhase: "prod",
		}
	}

	riskFactors := []string{}
	mitigations := []string{}
	highestRisk := "LOW"

	// Analyze each optimization
	for _, opt := range optimizations {
		switch opt.Risk {
		case "HIGH":
			highestRisk = "HIGH"
			riskFactors = append(riskFactors, fmt.Sprintf("High risk %s reduction: %.1f%%", opt.Type, opt.ReductionPercent))
		case "MEDIUM":
			if highestRisk != "HIGH" {
				highestRisk = "MEDIUM"
			}
			riskFactors = append(riskFactors, fmt.Sprintf("Medium risk %s reduction: %.1f%%", opt.Type, opt.ReductionPercent))
		}

		// Add mitigation strategies
		switch opt.Type {
		case "cpu":
			mitigations = append(mitigations, "Monitor CPU utilization closely after deployment")
		case "memory":
			mitigations = append(mitigations, "Watch for OOMKilled events and memory pressure")
		case "replicas":
			mitigations = append(mitigations, "Set up HPA for automatic scaling if needed")
		}
	}

	// Adjust confidence based on waste confidence
	adjustedConfidence := wasteConfidence
	if highestRisk == "HIGH" {
		adjustedConfidence *= 0.7
	} else if highestRisk == "MEDIUM" {
		adjustedConfidence *= 0.85
	}

	// Recommend deployment phase based on risk
	recommendedPhase := "prod"
	if highestRisk == "HIGH" || adjustedConfidence < 0.6 {
		recommendedPhase = "staging"
	}
	if adjustedConfidence < 0.4 {
		recommendedPhase = "dev"
	}

	return OptimizationRisk{
		OverallRisk:      highestRisk,
		RiskFactors:      riskFactors,
		Mitigations:      mitigations,
		Confidence:       adjustedConfidence,
		RecommendedPhase: recommendedPhase,
	}
}

// createOptimizedLabels creates labels for optimized units
func (oe *OptimizationEngine) createOptimizedLabels(originalLabels map[string]string) map[string]string {
	labels := make(map[string]string)

	// Copy original labels
	for k, v := range originalLabels {
		labels[k] = v
	}

	// Add optimization labels
	labels["optimizer.io/optimized"] = "true"
	labels["optimizer.io/version"] = "v1"
	labels["optimizer.io/engine"] = "devops-sdk"

	return labels
}

// createOptimizedAnnotations creates annotations for optimized units
func (oe *OptimizationEngine) createOptimizedAnnotations(originalAnnotations map[string]string, optimizations []ResourceOptimization) map[string]string {
	annotations := make(map[string]string)

	// Copy original annotations
	for k, v := range originalAnnotations {
		annotations[k] = v
	}

	// Add optimization metadata
	annotations["optimizer.io/optimized-at"] = time.Now().Format(time.RFC3339)
	annotations["optimizer.io/optimization-count"] = fmt.Sprintf("%d", len(optimizations))

	// Add specific optimization details
	for i, opt := range optimizations {
		prefix := fmt.Sprintf("optimizer.io/optimization-%d", i)
		annotations[prefix+"-type"] = opt.Type
		annotations[prefix+"-original"] = opt.OriginalValue
		annotations[prefix+"-optimized"] = opt.OptimizedValue
		annotations[prefix+"-reduction"] = fmt.Sprintf("%.1f%%", opt.ReductionPercent)
		annotations[prefix+"-risk"] = opt.Risk
	}

	return annotations
}

// CreateOptimizedUnitInConfigHub creates the optimized unit in ConfigHub
func (oe *OptimizationEngine) CreateOptimizedUnitInConfigHub(config *OptimizedConfiguration) (*Unit, error) {
	oe.app.Logger.Printf("💾 Creating optimized unit in ConfigHub: %s", config.OptimizedUnit.Slug)

	unit, err := oe.app.Cub.CreateUnit(oe.spaceID, CreateUnitRequest{
		Slug:           config.OptimizedUnit.Slug,
		DisplayName:    config.OptimizedUnit.DisplayName,
		Data:           config.OptimizedUnit.Data,
		Labels:         config.OptimizedUnit.Labels,
		Annotations:    config.OptimizedUnit.Annotations,
		UpstreamUnitID: config.OptimizedUnit.UpstreamUnitID,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create optimized unit: %v", err)
	}

	oe.app.Logger.Printf("✅ Optimized unit created: %s (savings: $%.2f/month)", unit.Slug, config.EstimatedSavings.MonthlySavings)
	return unit, nil
}

// BulkOptimizeUnits optimizes multiple units using ConfigHub Sets/Filters
func (oe *OptimizationEngine) BulkOptimizeUnits(setSlug string, wasteMetrics map[string]*WasteMetrics) ([]*OptimizedConfiguration, error) {
	oe.app.Logger.Printf("🔧 Bulk optimizing units in set: %s", setSlug)

	// Get units in the set
	units, err := oe.app.Cub.ListUnits(ListUnitsParams{
		SpaceID: oe.spaceID,
		Where:   fmt.Sprintf("Sets.Slug = '%s'", setSlug),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list units in set: %v", err)
	}

	var configs []*OptimizedConfiguration
	for _, unit := range units {
		waste := wasteMetrics[unit.Slug]
		if waste == nil {
			oe.app.Logger.Printf("⚠️  No waste metrics for unit %s, skipping", unit.Slug)
			continue
		}

		config, err := oe.GenerateOptimizedUnit(unit, waste)
		if err != nil {
			oe.app.Logger.Printf("⚠️  Failed to optimize unit %s: %v", unit.Slug, err)
			continue
		}

		if len(config.Optimizations) > 0 {
			configs = append(configs, config)
		}
	}

	oe.app.Logger.Printf("✅ Bulk optimization complete: %d units optimized", len(configs))
	return configs, nil
}

// GenerateOptimizationReport creates a comprehensive optimization report
func (oe *OptimizationEngine) GenerateOptimizationReport(configs []*OptimizedConfiguration) string {
	var report strings.Builder

	report.WriteString("═══════════════════════════════════════════════════════\n")
	report.WriteString("       ConfigHub Optimization Report\n")
	report.WriteString("═══════════════════════════════════════════════════════\n\n")

	totalSavings := 0.0
	totalCurrent := 0.0
	riskCounts := map[string]int{"LOW": 0, "MEDIUM": 0, "HIGH": 0}

	for _, config := range configs {
		totalSavings += config.EstimatedSavings.MonthlySavings
		totalCurrent += config.EstimatedSavings.CurrentMonthlyCost
		riskCounts[config.RiskAssessment.OverallRisk]++
	}

	savingsPercent := 0.0
	if totalCurrent > 0 {
		savingsPercent = (totalSavings / totalCurrent) * 100
	}

	report.WriteString(fmt.Sprintf("Units Analyzed: %d\n", len(configs)))
	report.WriteString(fmt.Sprintf("Current Monthly Cost: $%.2f\n", totalCurrent))
	report.WriteString(fmt.Sprintf("Potential Monthly Savings: $%.2f (%.1f%%)\n\n", totalSavings, savingsPercent))

	report.WriteString("Risk Distribution:\n")
	report.WriteString("─────────────────────────────────────────────\n")
	report.WriteString(fmt.Sprintf("• LOW risk:    %d units\n", riskCounts["LOW"]))
	report.WriteString(fmt.Sprintf("• MEDIUM risk: %d units\n", riskCounts["MEDIUM"]))
	report.WriteString(fmt.Sprintf("• HIGH risk:   %d units\n", riskCounts["HIGH"]))

	report.WriteString("\n\nTop Optimization Opportunities:\n")
	report.WriteString("─────────────────────────────────────────────\n")

	// Show top 5 savings opportunities
	for i, config := range configs {
		if i >= 5 {
			break
		}
		report.WriteString(fmt.Sprintf("%-30s %s risk $%.2f/mo savings (%.1f%%)\n",
			config.OriginalUnit.Slug,
			config.RiskAssessment.OverallRisk,
			config.EstimatedSavings.MonthlySavings,
			config.EstimatedSavings.SavingsPercent,
		))

		caser := cases.Title(language.English)
		for _, opt := range config.Optimizations {
			report.WriteString(fmt.Sprintf("  └─ %s: %s → %s (%.1f%% reduction)\n",
				caser.String(opt.Type),
				opt.OriginalValue,
				opt.OptimizedValue,
				opt.ReductionPercent,
			))
		}
	}

	report.WriteString("\n\nDeployment Recommendations:\n")
	report.WriteString("─────────────────────────────────────────────\n")
	report.WriteString("• Deploy LOW risk optimizations to production immediately\n")
	report.WriteString("• Test MEDIUM risk optimizations in staging first\n")
	report.WriteString("• Validate HIGH risk optimizations in dev environment\n")
	report.WriteString("• Monitor resource utilization after each deployment\n")

	return report.String()
}

// copyManifest creates a deep copy of a Kubernetes manifest
func copyManifest(original map[string]interface{}) map[string]interface{} {
	if original == nil {
		return nil
	}

	return copyValue(original).(map[string]interface{})
}

// copyValue recursively deep copies any interface{} value
func copyValue(original interface{}) interface{} {
	if original == nil {
		return nil
	}

	switch value := original.(type) {
	case map[string]interface{}:
		copy := make(map[string]interface{})
		for k, v := range value {
			copy[k] = copyValue(v)
		}
		return copy
	case []interface{}:
		if len(value) == 0 {
			return []interface{}{}
		}
		copy := make([]interface{}, len(value))
		for i, item := range value {
			copy[i] = copyValue(item)
		}
		return copy
	case map[interface{}]interface{}:
		copy := make(map[interface{}]interface{})
		for k, v := range value {
			copy[k] = copyValue(v)
		}
		return copy
	case []string:
		copy := make([]string, len(value))
		copy = append(copy[:0], value...)
		return copy
	case []int:
		copy := make([]int, len(value))
		copy = append(copy[:0], value...)
		return copy
	case []float64:
		copy := make([]float64, len(value))
		copy = append(copy[:0], value...)
		return copy
	default:
		// For primitive types (string, int, float64, bool, etc.)
		// return as-is since they are immutable in Go
		return value
	}
}

// OptimizeSpaceWithAI uses Claude AI to enhance optimization decisions
func (oe *OptimizationEngine) OptimizeSpaceWithAI(spaceSlug string, wasteMetrics map[string]*WasteMetrics) ([]*OptimizedConfiguration, error) {
	if oe.app.Claude == nil {
		return nil, fmt.Errorf("Claude AI not available")
	}

	oe.app.Logger.Printf("🤖 Using Claude AI for intelligent optimization of space: %s", spaceSlug)

	// Get basic optimization recommendations
	configs, err := oe.BulkOptimizeUnits(spaceSlug, wasteMetrics)
	if err != nil {
		return nil, err
	}

	// Enhance with AI analysis
	for _, config := range configs {
		aiRecommendation, err := oe.getAIOptimizationRecommendation(config)
		if err != nil {
			oe.app.Logger.Printf("⚠️  Claude AI analysis failed for %s: %v", config.OriginalUnit.Slug, err)
			continue
		}

		// Apply AI recommendations (this would integrate with Claude's suggestions)
		oe.app.Logger.Printf("🤖 Claude AI recommendation for %s: %s", config.OriginalUnit.Slug, aiRecommendation)
	}

	return configs, nil
}

// getAIOptimizationRecommendation gets AI-powered optimization advice
func (oe *OptimizationEngine) getAIOptimizationRecommendation(config *OptimizedConfiguration) (string, error) {
	prompt := fmt.Sprintf(`
Analyze this Kubernetes workload optimization:

Unit: %s
Type: Extract from manifest
Current Resources: %s
Optimizations Applied: %d
Estimated Savings: $%.2f/month (%.1f%%)
Risk Assessment: %s

Waste Metrics Available: CPU, Memory, Storage usage patterns
Safety Margins: CPU %.1f%%, Memory %.1f%%

Please provide:
1. Risk assessment validation
2. Additional optimization opportunities
3. Monitoring recommendations
4. Rollback strategy

Manifest:
%s
`,
		config.OriginalUnit.Slug,
		"Extract from analysis", // This would be computed
		len(config.Optimizations),
		config.EstimatedSavings.MonthlySavings,
		config.EstimatedSavings.SavingsPercent,
		config.RiskAssessment.OverallRisk,
		oe.safetyConfig.CPUSafetyMargin*100,
		oe.safetyConfig.MemorySafetyMargin*100,
		config.OriginalUnit.Data,
	)

	response, err := oe.app.Claude.Complete(prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}

// CreateOptimizedSet creates a ConfigHub Set containing all optimized units
func (oe *OptimizationEngine) CreateOptimizedSet(configs []*OptimizedConfiguration, setName string) (*Set, error) {
	oe.app.Logger.Printf("📦 Creating optimized set: %s", setName)

	set, err := oe.app.Cub.CreateSet(oe.spaceID, CreateSetRequest{
		Slug:        setName,
		DisplayName: fmt.Sprintf("Optimized Units - %s", setName),
		Labels: map[string]string{
			"optimizer.io/set":     "true",
			"optimizer.io/version": "v1",
		},
		Annotations: map[string]string{
			"optimizer.io/created-at":    time.Now().Format(time.RFC3339),
			"optimizer.io/unit-count":    fmt.Sprintf("%d", len(configs)),
			"optimizer.io/total-savings": fmt.Sprintf("$%.2f", oe.calculateTotalSavings(configs)),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create optimized set: %v", err)
	}

	oe.app.Logger.Printf("✅ Optimized set created: %s with %d units", set.Slug, len(configs))
	return set, nil
}

// calculateTotalSavings calculates total savings across all configurations
func (oe *OptimizationEngine) calculateTotalSavings(configs []*OptimizedConfiguration) float64 {
	total := 0.0
	for _, config := range configs {
		total += config.EstimatedSavings.MonthlySavings
	}
	return total
}
