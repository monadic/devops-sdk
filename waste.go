// waste.go - Waste detection module for the DevOps SDK
//
// This module provides comprehensive waste detection by comparing estimated
// costs from ConfigHub units against actual usage metrics from OpenCost or
// other monitoring systems. It identifies over-provisioning, idle resources,
// and underutilized workloads to provide actionable cost optimization insights.
//
// Features:
// - Compare ConfigHub estimated costs vs actual usage metrics
// - Detect over-provisioning by comparing requests vs actual utilization
// - Identify idle resources with minimal usage
// - Calculate waste scoring and severity levels
// - Provide cost saving potential calculations
// - Support for configurable waste detection thresholds
// - Integration with existing CostAnalyzer infrastructure
//
// This module works in conjunction with cost.go to provide a complete
// cost optimization solution for Kubernetes workloads.
package sdk

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// WasteAnalyzer detects waste by comparing estimated vs actual costs
type WasteAnalyzer struct {
	app          *DevOpsApp
	spaceID      uuid.UUID
	thresholds   *WasteThresholds
	costAnalyzer *CostAnalyzer
}

// WasteThresholds defines when resources are considered wasteful
type WasteThresholds struct {
	// CPU utilization thresholds
	CPUIdleThreshold          float64 // Below this % utilization = idle (default: 5%)
	CPUUnderutilizedThreshold float64 // Below this % = underutilized (default: 30%)
	CPUOverprovisionedRatio   float64 // Requested/Used ratio above this = over-provisioned (default: 3.0)

	// Memory utilization thresholds
	MemoryIdleThreshold          float64 // Below this % utilization = idle (default: 10%)
	MemoryUnderutilizedThreshold float64 // Below this % = underutilized (default: 40%)
	MemoryOverprovisionedRatio   float64 // Requested/Used ratio above this = over-provisioned (default: 2.5)

	// Cost thresholds
	MinMonthlyCostForAnalysis float64 // Only analyze resources above this cost (default: $1.00)
	WasteScoreHighThreshold   float64 // Above this score = HIGH waste (default: 80.0)
	WasteScoreMediumThreshold float64 // Above this score = MEDIUM waste (default: 50.0)

	// Time-based thresholds
	IdleDurationDays          int // Days of idle usage to flag as waste (default: 7)
	UnderutilizedDurationDays int // Days of underutilization to flag (default: 14)
}

// DefaultWasteThresholds provides sensible defaults for waste detection
var DefaultWasteThresholds = &WasteThresholds{
	CPUIdleThreshold:             5.0,
	CPUUnderutilizedThreshold:    30.0,
	CPUOverprovisionedRatio:      3.0,
	MemoryIdleThreshold:          10.0,
	MemoryUnderutilizedThreshold: 40.0,
	MemoryOverprovisionedRatio:   2.5,
	MinMonthlyCostForAnalysis:    1.00,
	WasteScoreHighThreshold:      80.0,
	WasteScoreMediumThreshold:    50.0,
	IdleDurationDays:             7,
	UnderutilizedDurationDays:    14,
}

// ActualUsageMetrics represents real usage data from monitoring systems
type ActualUsageMetrics struct {
	UnitID         string
	UnitName       string
	Space          string
	TimeRangeStart time.Time
	TimeRangeEnd   time.Time

	// Resource utilization averages over time period
	CPUUtilizationPercent    float64 // Average CPU utilization %
	MemoryUtilizationPercent float64 // Average memory utilization %

	// Actual resource consumption
	CPUCoresUsed      float64 // Average cores actually used
	MemoryBytesUsed   int64   // Average memory bytes actually used
	NetworkBytesTotal int64   // Total network I/O
	StorageBytesUsed  int64   // Actual storage consumed

	// Cost data from monitoring systems (e.g., OpenCost)
	ActualMonthlyCost float64 // Actual cost based on usage

	// Replica and availability data
	AverageReplicas float64 // Average number of running replicas
	UptimePercent   float64 // Percentage of time pods were running

	// Peak usage for rightsizing recommendations
	CPUPeakPercent    float64 // Peak CPU utilization
	MemoryPeakPercent float64 // Peak memory utilization
}

// WasteDetection represents the results of waste analysis for a single unit
type WasteDetection struct {
	UnitID   string
	UnitName string
	Space    string
	Type     string // deployment, statefulset, etc.

	// Cost comparison
	EstimatedMonthlyCost float64 // From ConfigHub analysis
	ActualMonthlyCost    float64 // From actual usage
	WastedMonthlyCost    float64 // Difference between estimated and actual

	// Waste categorization
	WasteCategories []WasteCategory
	WasteScore      float64 // 0-100 score indicating severity
	WasteSeverity   string  // LOW, MEDIUM, HIGH

	// Resource-specific waste
	CPUWaste     ResourceWaste
	MemoryWaste  ResourceWaste
	StorageWaste ResourceWaste
	ReplicaWaste ReplicaWaste

	// Recommendations
	Recommendations  []WasteRecommendation
	PotentialSavings float64 // Monthly savings potential

	// Analysis metadata
	AnalyzedAt  time.Time
	DataQuality string // EXCELLENT, GOOD, FAIR, POOR
}

// WasteCategory represents different types of waste
type WasteCategory struct {
	Type        string  // idle, underutilized, over-provisioned, over-replicated
	Severity    string  // LOW, MEDIUM, HIGH
	Impact      float64 // Cost impact in dollars per month
	Description string
}

// ResourceWaste represents waste for a specific resource type
type ResourceWaste struct {
	Allocated          string  // Amount allocated (e.g., "2 cores", "4Gi")
	Used               string  // Amount actually used (e.g., "0.3 cores", "1.2Gi")
	UtilizationPercent float64 // Percentage utilization
	WastePercent       float64 // Percentage wasted
	WastedCost         float64 // Monthly cost of wasted resources
	Recommendation     string  // Suggested allocation
}

// ReplicaWaste represents waste in replica configuration
type ReplicaWaste struct {
	ConfiguredReplicas int32   // Number of replicas configured
	AverageReplicas    float64 // Average running replicas
	IdleReplicas       float64 // Average idle replicas
	WastedCost         float64 // Cost of idle replicas
	Recommendation     string  // Suggested replica count
}

// WasteRecommendation provides actionable waste reduction suggestions
type WasteRecommendation struct {
	Type             string  // resize, scale-down, consolidate, terminate
	Priority         string  // HIGH, MEDIUM, LOW
	Action           string  // Human-readable action description
	Implementation   string  // Technical implementation details
	PotentialSavings float64 // Monthly savings if implemented
	Risk             string  // LOW, MEDIUM, HIGH
	RiskDescription  string  // Description of implementation risks
	AutoApplyable    bool    // Whether this can be auto-applied
}

// SpaceWasteAnalysis represents waste analysis for an entire space
type SpaceWasteAnalysis struct {
	SpaceID    string
	SpaceName  string
	AnalyzedAt time.Time

	// Overall waste metrics
	TotalEstimatedCost float64
	TotalActualCost    float64
	TotalWastedCost    float64
	WastePercent       float64

	// Unit-level analysis
	UnitsAnalyzed       int
	UnitsWithWaste      int
	UnitWasteDetections []WasteDetection

	// Waste breakdown by category
	WasteBySeverity map[string]WasteSummary // HIGH, MEDIUM, LOW
	WasteByCategory map[string]WasteSummary // idle, underutilized, etc.
	WasteByResource map[string]WasteSummary // cpu, memory, storage

	// Top waste opportunities
	TopWasteUnits      []WasteDetection // Sorted by potential savings
	TopRecommendations []WasteRecommendation
}

// WasteSummary provides aggregated waste metrics
type WasteSummary struct {
	Count            int     // Number of instances
	TotalCost        float64 // Total cost impact
	AverageWaste     float64 // Average waste percentage
	PotentialSavings float64 // Total potential savings
}

// NewWasteAnalyzer creates a new waste analyzer
func NewWasteAnalyzer(app *DevOpsApp, spaceID uuid.UUID) *WasteAnalyzer {
	return &WasteAnalyzer{
		app:          app,
		spaceID:      spaceID,
		thresholds:   DefaultWasteThresholds,
		costAnalyzer: NewCostAnalyzer(app, spaceID),
	}
}

// SetThresholds allows customization of waste detection thresholds
func (wa *WasteAnalyzer) SetThresholds(thresholds *WasteThresholds) {
	wa.thresholds = thresholds
}

// AnalyzeWaste performs comprehensive waste analysis by comparing estimates vs actuals
func (wa *WasteAnalyzer) AnalyzeWaste(actualUsageData []ActualUsageMetrics) (*SpaceWasteAnalysis, error) {
	wa.app.Logger.Printf("🔍 Analyzing waste in ConfigHub space: %s", wa.spaceID)

	// Get cost estimates from ConfigHub
	costAnalysis, err := wa.costAnalyzer.AnalyzeSpace()
	if err != nil {
		return nil, fmt.Errorf("failed to analyze costs: %v", err)
	}

	// Create usage lookup map
	usageMap := make(map[string]ActualUsageMetrics)
	for _, usage := range actualUsageData {
		usageMap[usage.UnitID] = usage
	}

	analysis := &SpaceWasteAnalysis{
		SpaceID:             wa.spaceID.String(),
		SpaceName:           costAnalysis.SpaceName,
		AnalyzedAt:          time.Now(),
		TotalEstimatedCost:  costAnalysis.TotalMonthlyCost,
		UnitWasteDetections: []WasteDetection{},
		WasteBySeverity:     make(map[string]WasteSummary),
		WasteByCategory:     make(map[string]WasteSummary),
		WasteByResource:     make(map[string]WasteSummary),
	}

	// Analyze waste for each unit
	for _, costEstimate := range costAnalysis.Units {
		usage, hasUsageData := usageMap[costEstimate.UnitID]

		wasteDetection := wa.analyzeUnitWaste(costEstimate, usage, hasUsageData)
		if wasteDetection != nil {
			analysis.UnitWasteDetections = append(analysis.UnitWasteDetections, *wasteDetection)

			// Update aggregates
			analysis.TotalActualCost += wasteDetection.ActualMonthlyCost
			analysis.TotalWastedCost += wasteDetection.WastedMonthlyCost

			if wasteDetection.WasteScore > 0 {
				analysis.UnitsWithWaste++
			}
		}
	}

	analysis.UnitsAnalyzed = len(analysis.UnitWasteDetections)
	if analysis.TotalEstimatedCost > 0 {
		analysis.WastePercent = (analysis.TotalWastedCost / analysis.TotalEstimatedCost) * 100
	}

	// Generate aggregated summaries
	wa.generateWasteSummaries(analysis)

	wa.app.Logger.Printf("✅ Waste analysis complete: %.1f%% waste detected, $%.2f potential savings",
		analysis.WastePercent, analysis.TotalWastedCost)

	return analysis, nil
}

// analyzeUnitWaste analyzes waste for a single unit
func (wa *WasteAnalyzer) analyzeUnitWaste(estimate UnitCostEstimate, usage ActualUsageMetrics, hasUsageData bool) *WasteDetection {
	// Skip units below minimum cost threshold
	if estimate.MonthlyCost < wa.thresholds.MinMonthlyCostForAnalysis {
		return nil
	}

	detection := &WasteDetection{
		UnitID:               estimate.UnitID,
		UnitName:             estimate.UnitName,
		Space:                estimate.Space,
		Type:                 estimate.Type,
		EstimatedMonthlyCost: estimate.MonthlyCost,
		ActualMonthlyCost:    estimate.MonthlyCost, // Default to estimate
		WasteCategories:      []WasteCategory{},
		Recommendations:      []WasteRecommendation{},
		AnalyzedAt:           time.Now(),
		DataQuality:          "POOR", // Default
	}

	if hasUsageData {
		detection.ActualMonthlyCost = usage.ActualMonthlyCost
		detection.DataQuality = wa.assessDataQuality(usage)

		// Analyze CPU waste
		detection.CPUWaste = wa.analyzeCPUWaste(estimate, usage)

		// Analyze memory waste
		detection.MemoryWaste = wa.analyzeMemoryWaste(estimate, usage)

		// Analyze replica waste
		detection.ReplicaWaste = wa.analyzeReplicaWaste(estimate, usage)

		// Categorize waste
		detection.WasteCategories = wa.categorizeWaste(detection, usage)

		// Generate recommendations
		detection.Recommendations = wa.generateWasteRecommendations(detection, estimate, usage)
	} else {
		// No usage data - use heuristic analysis
		wa.app.Logger.Printf("⚠️  No usage data for %s, using heuristic analysis", estimate.UnitName)
		detection = wa.analyzeWithoutUsageData(estimate)
	}

	// Calculate overall waste score and severity
	detection.WastedMonthlyCost = detection.EstimatedMonthlyCost - detection.ActualMonthlyCost
	detection.WasteScore = wa.calculateWasteScore(detection)
	detection.WasteSeverity = wa.determineWasteSeverity(detection.WasteScore)
	detection.PotentialSavings = wa.calculatePotentialSavings(detection)

	return detection
}

// analyzeCPUWaste analyzes CPU resource waste
func (wa *WasteAnalyzer) analyzeCPUWaste(estimate UnitCostEstimate, usage ActualUsageMetrics) ResourceWaste {
	allocatedCores := float64(estimate.CPU.MilliValue()) / 1000.0
	usedCores := usage.CPUCoresUsed
	utilizationPercent := usage.CPUUtilizationPercent

	var wastePercent float64
	if allocatedCores > 0 {
		wastePercent = ((allocatedCores - usedCores) / allocatedCores) * 100
	}

	// Calculate recommended allocation (110% of peak usage with minimum safety buffer)
	recommendedCores := math.Max(usage.CPUPeakPercent/100.0*allocatedCores*1.1, 0.1)

	return ResourceWaste{
		Allocated:          fmt.Sprintf("%.2f cores", allocatedCores),
		Used:               fmt.Sprintf("%.2f cores", usedCores),
		UtilizationPercent: utilizationPercent,
		WastePercent:       wastePercent,
		WastedCost:         estimate.Breakdown.CPUCost * (wastePercent / 100.0),
		Recommendation:     fmt.Sprintf("%.1f cores", recommendedCores),
	}
}

// analyzeMemoryWaste analyzes memory resource waste
func (wa *WasteAnalyzer) analyzeMemoryWaste(estimate UnitCostEstimate, usage ActualUsageMetrics) ResourceWaste {
	allocatedBytes := estimate.Memory.BytesValue()
	usedBytes := usage.MemoryBytesUsed
	utilizationPercent := usage.MemoryUtilizationPercent

	var wastePercent float64
	if allocatedBytes > 0 {
		wastePercent = (float64(allocatedBytes-usedBytes) / float64(allocatedBytes)) * 100
	}

	// Calculate recommended allocation (120% of peak usage with minimum safety buffer)
	recommendedGB := math.Max(float64(allocatedBytes)*(usage.MemoryPeakPercent/100.0)*1.2/(1024*1024*1024), 0.128)

	return ResourceWaste{
		Allocated:          fmt.Sprintf("%.2fGi", float64(allocatedBytes)/(1024*1024*1024)),
		Used:               fmt.Sprintf("%.2fGi", float64(usedBytes)/(1024*1024*1024)),
		UtilizationPercent: utilizationPercent,
		WastePercent:       wastePercent,
		WastedCost:         estimate.Breakdown.MemoryCost * (wastePercent / 100.0),
		Recommendation:     fmt.Sprintf("%.1fGi", recommendedGB),
	}
}

// analyzeReplicaWaste analyzes replica count waste
func (wa *WasteAnalyzer) analyzeReplicaWaste(estimate UnitCostEstimate, usage ActualUsageMetrics) ReplicaWaste {
	configured := estimate.Replicas
	average := usage.AverageReplicas
	idle := math.Max(float64(configured)-average, 0)

	// Calculate cost per replica
	costPerReplica := estimate.MonthlyCost / float64(configured)
	wastedCost := idle * costPerReplica

	// Recommend based on average usage + 1 for availability
	recommended := int(math.Ceil(average)) + 1
	if recommended < 2 {
		recommended = 2 // Minimum for availability
	}

	return ReplicaWaste{
		ConfiguredReplicas: configured,
		AverageReplicas:    average,
		IdleReplicas:       idle,
		WastedCost:         wastedCost,
		Recommendation:     fmt.Sprintf("%d replicas", recommended),
	}
}

// categorizeWaste categorizes the types of waste detected
func (wa *WasteAnalyzer) categorizeWaste(detection *WasteDetection, usage ActualUsageMetrics) []WasteCategory {
	var categories []WasteCategory

	// Check for idle resources
	if usage.CPUUtilizationPercent < wa.thresholds.CPUIdleThreshold &&
		usage.MemoryUtilizationPercent < wa.thresholds.MemoryIdleThreshold {
		categories = append(categories, WasteCategory{
			Type:        "idle",
			Severity:    "HIGH",
			Impact:      detection.EstimatedMonthlyCost * 0.8,
			Description: "Resource is largely idle with minimal CPU and memory usage",
		})
	}

	// Check for CPU over-provisioning
	if detection.CPUWaste.UtilizationPercent < wa.thresholds.CPUUnderutilizedThreshold {
		severity := "MEDIUM"
		if detection.CPUWaste.UtilizationPercent < wa.thresholds.CPUIdleThreshold {
			severity = "HIGH"
		}

		categories = append(categories, WasteCategory{
			Type:        "cpu-over-provisioned",
			Severity:    severity,
			Impact:      detection.CPUWaste.WastedCost,
			Description: fmt.Sprintf("CPU utilization is only %.1f%%, significantly over-provisioned", detection.CPUWaste.UtilizationPercent),
		})
	}

	// Check for memory over-provisioning
	if detection.MemoryWaste.UtilizationPercent < wa.thresholds.MemoryUnderutilizedThreshold {
		severity := "MEDIUM"
		if detection.MemoryWaste.UtilizationPercent < wa.thresholds.MemoryIdleThreshold {
			severity = "HIGH"
		}

		categories = append(categories, WasteCategory{
			Type:        "memory-over-provisioned",
			Severity:    severity,
			Impact:      detection.MemoryWaste.WastedCost,
			Description: fmt.Sprintf("Memory utilization is only %.1f%%, significantly over-provisioned", detection.MemoryWaste.UtilizationPercent),
		})
	}

	// Check for over-replication
	if detection.ReplicaWaste.IdleReplicas > 0.5 {
		categories = append(categories, WasteCategory{
			Type:        "over-replicated",
			Severity:    "MEDIUM",
			Impact:      detection.ReplicaWaste.WastedCost,
			Description: fmt.Sprintf("Average of %.1f idle replicas detected", detection.ReplicaWaste.IdleReplicas),
		})
	}

	return categories
}

// generateWasteRecommendations generates actionable recommendations
func (wa *WasteAnalyzer) generateWasteRecommendations(detection *WasteDetection, estimate UnitCostEstimate, usage ActualUsageMetrics) []WasteRecommendation {
	var recommendations []WasteRecommendation

	// CPU rightsizing recommendation
	if detection.CPUWaste.WastePercent > 30 {
		recommendations = append(recommendations, WasteRecommendation{
			Type:             "resize",
			Priority:         wa.determinePriority(detection.CPUWaste.WastedCost),
			Action:           fmt.Sprintf("Reduce CPU allocation from %s to %s", detection.CPUWaste.Allocated, detection.CPUWaste.Recommendation),
			Implementation:   fmt.Sprintf("Update resources.requests.cpu to %s in deployment spec", detection.CPUWaste.Recommendation),
			PotentialSavings: detection.CPUWaste.WastedCost * 0.8, // Conservative estimate
			Risk:             "LOW",
			RiskDescription:  "CPU reduction based on actual usage patterns with 10% safety buffer",
			AutoApplyable:    true,
		})
	}

	// Memory rightsizing recommendation
	if detection.MemoryWaste.WastePercent > 30 {
		recommendations = append(recommendations, WasteRecommendation{
			Type:             "resize",
			Priority:         wa.determinePriority(detection.MemoryWaste.WastedCost),
			Action:           fmt.Sprintf("Reduce memory allocation from %s to %s", detection.MemoryWaste.Allocated, detection.MemoryWaste.Recommendation),
			Implementation:   fmt.Sprintf("Update resources.requests.memory to %s in deployment spec", detection.MemoryWaste.Recommendation),
			PotentialSavings: detection.MemoryWaste.WastedCost * 0.8,
			Risk:             "MEDIUM",
			RiskDescription:  "Memory reduction requires careful monitoring to avoid OOM kills",
			AutoApplyable:    false,
		})
	}

	// Replica scaling recommendation
	if detection.ReplicaWaste.IdleReplicas > 0.5 {
		recommendations = append(recommendations, WasteRecommendation{
			Type:             "scale-down",
			Priority:         wa.determinePriority(detection.ReplicaWaste.WastedCost),
			Action:           fmt.Sprintf("Reduce replica count from %d to %s", detection.ReplicaWaste.ConfiguredReplicas, detection.ReplicaWaste.Recommendation),
			Implementation:   fmt.Sprintf("Update spec.replicas in deployment to match %s", detection.ReplicaWaste.Recommendation),
			PotentialSavings: detection.ReplicaWaste.WastedCost * 0.9,
			Risk:             "HIGH",
			RiskDescription:  "Scaling down reduces availability and may impact performance during traffic spikes",
			AutoApplyable:    false,
		})
	}

	// Termination recommendation for completely idle resources
	if usage.CPUUtilizationPercent < 1.0 && usage.MemoryUtilizationPercent < 5.0 && usage.UptimePercent < 50.0 {
		recommendations = append(recommendations, WasteRecommendation{
			Type:             "terminate",
			Priority:         "HIGH",
			Action:           "Consider terminating this largely unused resource",
			Implementation:   "Review application requirements and consider removing deployment",
			PotentialSavings: detection.EstimatedMonthlyCost * 0.95,
			Risk:             "HIGH",
			RiskDescription:  "Termination may impact dependent services or future requirements",
			AutoApplyable:    false,
		})
	}

	return recommendations
}

// analyzeWithoutUsageData provides heuristic waste analysis when no metrics are available
func (wa *WasteAnalyzer) analyzeWithoutUsageData(estimate UnitCostEstimate) *WasteDetection {
	detection := &WasteDetection{
		UnitID:               estimate.UnitID,
		UnitName:             estimate.UnitName,
		Space:                estimate.Space,
		Type:                 estimate.Type,
		EstimatedMonthlyCost: estimate.MonthlyCost,
		ActualMonthlyCost:    estimate.MonthlyCost,
		WastedMonthlyCost:    0,
		WasteCategories:      []WasteCategory{},
		Recommendations:      []WasteRecommendation{},
		AnalyzedAt:           time.Now(),
		DataQuality:          "POOR",
	}

	// Apply heuristic rules based on resource allocation patterns
	cpuCores := float64(estimate.CPU.MilliValue()) / 1000.0
	memoryGi := float64(estimate.Memory.BytesValue()) / (1024 * 1024 * 1024)

	// Flag potentially over-provisioned resources based on common patterns
	if cpuCores > 2.0 {
		detection.WasteCategories = append(detection.WasteCategories, WasteCategory{
			Type:        "potentially-over-provisioned",
			Severity:    "MEDIUM",
			Impact:      estimate.Breakdown.CPUCost * 0.3,
			Description: "High CPU allocation may indicate over-provisioning",
		})
	}

	if memoryGi > 4.0 {
		detection.WasteCategories = append(detection.WasteCategories, WasteCategory{
			Type:        "potentially-over-provisioned",
			Severity:    "MEDIUM",
			Impact:      estimate.Breakdown.MemoryCost * 0.3,
			Description: "High memory allocation may indicate over-provisioning",
		})
	}

	// Conservative waste score without usage data
	detection.WasteScore = 25.0 // Low confidence score
	detection.WasteSeverity = "LOW"

	return detection
}

// calculateWasteScore calculates a 0-100 waste score
func (wa *WasteAnalyzer) calculateWasteScore(detection *WasteDetection) float64 {
	if detection.EstimatedMonthlyCost == 0 {
		return 0
	}

	// Base score on cost waste percentage
	wasteRatio := detection.WastedMonthlyCost / detection.EstimatedMonthlyCost
	// Ensure non-negative waste ratio
	if wasteRatio < 0 {
		wasteRatio = 0
	}
	baseScore := wasteRatio * 100

	// Adjust based on waste categories
	severityMultiplier := 1.0
	for _, category := range detection.WasteCategories {
		switch category.Severity {
		case "HIGH":
			severityMultiplier *= 1.5
		case "MEDIUM":
			severityMultiplier *= 1.2
		}
	}

	// Adjust based on data quality
	qualityMultiplier := 1.0
	switch detection.DataQuality {
	case "EXCELLENT":
		qualityMultiplier = 1.0
	case "GOOD":
		qualityMultiplier = 0.9
	case "FAIR":
		qualityMultiplier = 0.7
	case "POOR":
		qualityMultiplier = 0.5
	}

	score := baseScore * severityMultiplier * qualityMultiplier

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// determineWasteSeverity determines severity level based on waste score
func (wa *WasteAnalyzer) determineWasteSeverity(wasteScore float64) string {
	if wasteScore >= wa.thresholds.WasteScoreHighThreshold {
		return "HIGH"
	} else if wasteScore >= wa.thresholds.WasteScoreMediumThreshold {
		return "MEDIUM"
	}
	return "LOW"
}

// calculatePotentialSavings calculates total potential monthly savings
func (wa *WasteAnalyzer) calculatePotentialSavings(detection *WasteDetection) float64 {
	var totalSavings float64

	for _, recommendation := range detection.Recommendations {
		totalSavings += recommendation.PotentialSavings
	}

	// Cap savings at 90% of estimated cost (keep some buffer)
	maxSavings := detection.EstimatedMonthlyCost * 0.9
	if totalSavings > maxSavings {
		totalSavings = maxSavings
	}

	return totalSavings
}

// assessDataQuality assesses the quality of usage data
func (wa *WasteAnalyzer) assessDataQuality(usage ActualUsageMetrics) string {
	dataAge := time.Since(usage.TimeRangeEnd)
	dataSpan := usage.TimeRangeEnd.Sub(usage.TimeRangeStart)

	// Assess based on data freshness and span
	if dataAge < 24*time.Hour && dataSpan >= 7*24*time.Hour {
		return "EXCELLENT"
	} else if dataAge < 3*24*time.Hour && dataSpan >= 3*24*time.Hour {
		return "GOOD"
	} else if dataAge < 7*24*time.Hour && dataSpan >= 24*time.Hour {
		return "FAIR"
	}

	return "POOR"
}

// determinePriority determines recommendation priority based on cost impact
func (wa *WasteAnalyzer) determinePriority(savings float64) string {
	if savings >= 50.0 {
		return "HIGH"
	} else if savings >= 20.0 {
		return "MEDIUM"
	}
	return "LOW"
}

// generateWasteSummaries generates aggregated waste summaries
func (wa *WasteAnalyzer) generateWasteSummaries(analysis *SpaceWasteAnalysis) {
	// Initialize maps
	analysis.WasteBySeverity = make(map[string]WasteSummary)
	analysis.WasteByCategory = make(map[string]WasteSummary)
	analysis.WasteByResource = make(map[string]WasteSummary)

	// Aggregate by severity
	severityCounts := make(map[string]int)
	severityCosts := make(map[string]float64)
	severitySavings := make(map[string]float64)

	// Aggregate by category
	categoryCounts := make(map[string]int)
	categoryCosts := make(map[string]float64)
	categorySavings := make(map[string]float64)

	// Aggregate by resource type
	resourceCounts := make(map[string]int)
	resourceCosts := make(map[string]float64)
	resourceSavings := make(map[string]float64)

	for _, detection := range analysis.UnitWasteDetections {
		severity := detection.WasteSeverity
		severityCounts[severity]++
		severityCosts[severity] += detection.WastedMonthlyCost
		severitySavings[severity] += detection.PotentialSavings

		// Process waste categories
		for _, category := range detection.WasteCategories {
			categoryCounts[category.Type]++
			categoryCosts[category.Type] += category.Impact
			// Find matching recommendations for savings
			for _, rec := range detection.Recommendations {
				if (category.Type == "cpu-over-provisioned" && rec.Type == "resize" && strings.Contains(rec.Action, "CPU")) ||
					(category.Type == "memory-over-provisioned" && rec.Type == "resize" && strings.Contains(rec.Action, "memory")) ||
					(category.Type == "over-replicated" && rec.Type == "scale-down") {
					categorySavings[category.Type] += rec.PotentialSavings
				}
			}
		}

		// Process resource-specific waste
		if detection.CPUWaste.WastedCost > 0 {
			resourceCounts["cpu"]++
			resourceCosts["cpu"] += detection.CPUWaste.WastedCost
			for _, rec := range detection.Recommendations {
				if rec.Type == "resize" && strings.Contains(rec.Action, "CPU") {
					resourceSavings["cpu"] += rec.PotentialSavings
					break
				}
			}
		}

		if detection.MemoryWaste.WastedCost > 0 {
			resourceCounts["memory"]++
			resourceCosts["memory"] += detection.MemoryWaste.WastedCost
			for _, rec := range detection.Recommendations {
				if rec.Type == "resize" && strings.Contains(rec.Action, "memory") {
					resourceSavings["memory"] += rec.PotentialSavings
					break
				}
			}
		}

		if detection.ReplicaWaste.WastedCost > 0 {
			resourceCounts["replicas"]++
			resourceCosts["replicas"] += detection.ReplicaWaste.WastedCost
			for _, rec := range detection.Recommendations {
				if rec.Type == "scale-down" {
					resourceSavings["replicas"] += rec.PotentialSavings
					break
				}
			}
		}
	}

	// Populate severity summaries
	for severity, count := range severityCounts {
		analysis.WasteBySeverity[severity] = WasteSummary{
			Count:            count,
			TotalCost:        severityCosts[severity],
			PotentialSavings: severitySavings[severity],
		}
	}

	// Populate category summaries
	for category, count := range categoryCounts {
		analysis.WasteByCategory[category] = WasteSummary{
			Count:            count,
			TotalCost:        categoryCosts[category],
			PotentialSavings: categorySavings[category],
		}
	}

	// Populate resource summaries
	for resource, count := range resourceCounts {
		analysis.WasteByResource[resource] = WasteSummary{
			Count:            count,
			TotalCost:        resourceCosts[resource],
			PotentialSavings: resourceSavings[resource],
		}
	}

	// Sort top waste units by potential savings
	sort.Slice(analysis.UnitWasteDetections, func(i, j int) bool {
		return analysis.UnitWasteDetections[i].PotentialSavings > analysis.UnitWasteDetections[j].PotentialSavings
	})

	// Take top 10 for summary
	topCount := 10
	if len(analysis.UnitWasteDetections) < topCount {
		topCount = len(analysis.UnitWasteDetections)
	}
	analysis.TopWasteUnits = analysis.UnitWasteDetections[:topCount]

	// Collect top recommendations
	allRecommendations := []WasteRecommendation{}
	for _, detection := range analysis.UnitWasteDetections {
		allRecommendations = append(allRecommendations, detection.Recommendations...)
	}

	// Sort recommendations by savings potential
	sort.Slice(allRecommendations, func(i, j int) bool {
		return allRecommendations[i].PotentialSavings > allRecommendations[j].PotentialSavings
	})

	// Take top 10 recommendations
	topRecommendationCount := 10
	if len(allRecommendations) < topRecommendationCount {
		topRecommendationCount = len(allRecommendations)
	}
	analysis.TopRecommendations = allRecommendations[:topRecommendationCount]
}

// GenerateWasteReport creates a human-readable waste analysis report
func (wa *WasteAnalyzer) GenerateWasteReport(analysis *SpaceWasteAnalysis) string {
	var report strings.Builder

	report.WriteString("═══════════════════════════════════════════════════════\n")
	report.WriteString("       ConfigHub Waste Analysis Report\n")
	report.WriteString("═══════════════════════════════════════════════════════\n\n")

	report.WriteString(fmt.Sprintf("Space: %s\n", analysis.SpaceName))
	report.WriteString(fmt.Sprintf("Analyzed At: %s\n", analysis.AnalyzedAt.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("Units Analyzed: %d\n", analysis.UnitsAnalyzed))
	report.WriteString(fmt.Sprintf("Units with Waste: %d\n\n", analysis.UnitsWithWaste))

	// Cost summary
	report.WriteString("Cost Summary:\n")
	report.WriteString("─────────────────────────────────────────────\n")
	report.WriteString(fmt.Sprintf("Estimated Monthly Cost: $%.2f\n", analysis.TotalEstimatedCost))
	report.WriteString(fmt.Sprintf("Actual Monthly Cost:    $%.2f\n", analysis.TotalActualCost))
	report.WriteString(fmt.Sprintf("Wasted Monthly Cost:    $%.2f (%.1f%%)\n\n",
		analysis.TotalWastedCost, analysis.WastePercent))

	// Waste by severity
	report.WriteString("Waste by Severity:\n")
	report.WriteString("─────────────────────────────────────────────\n")
	for severity, summary := range analysis.WasteBySeverity {
		report.WriteString(fmt.Sprintf("%-6s: %2d units, $%.2f wasted, $%.2f potential savings\n",
			severity, summary.Count, summary.TotalCost, summary.PotentialSavings))
	}

	// Top waste opportunities
	report.WriteString("\n\nTop Waste Opportunities:\n")
	report.WriteString("─────────────────────────────────────────────\n")
	for i, unit := range analysis.TopWasteUnits {
		if i >= 5 {
			break
		}
		report.WriteString(fmt.Sprintf("%-25s %8s  $%6.2f wasted  $%6.2f savings  [%s]\n",
			unit.UnitName, unit.WasteSeverity, unit.WastedMonthlyCost,
			unit.PotentialSavings, unit.Type))
	}

	// Top recommendations
	report.WriteString("\n\nTop Recommendations:\n")
	report.WriteString("─────────────────────────────────────────────\n")
	for i, rec := range analysis.TopRecommendations {
		if i >= 5 {
			break
		}
		report.WriteString(fmt.Sprintf("• [%s] %s ($%.2f savings)\n",
			rec.Priority, rec.Action, rec.PotentialSavings))
		report.WriteString(fmt.Sprintf("  Risk: %s - %s\n\n", rec.Risk, rec.RiskDescription))
	}

	return report.String()
}

// IdentifyWaste is the main entry point for waste detection
func IdentifyWaste(app *DevOpsApp, spaceSlug string, actualUsageData []ActualUsageMetrics) (*SpaceWasteAnalysis, error) {
	// Get space by slug
	space, err := app.Cub.GetSpaceBySlug(spaceSlug)
	if err != nil {
		return nil, fmt.Errorf("failed to find space %s: %v", spaceSlug, err)
	}

	// Create waste analyzer
	analyzer := NewWasteAnalyzer(app, space.SpaceID)

	// Analyze waste
	analysis, err := analyzer.AnalyzeWaste(actualUsageData)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze waste: %v", err)
	}

	return analysis, nil
}
