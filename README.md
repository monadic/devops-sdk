# DevOps SDK

A comprehensive Go SDK for building DevOps automation applications using ConfigHub as the configuration backend. This SDK provides reusable modules for cost analysis, waste detection, resource optimization, and deployment strategies.

## SDK Modules

- **`app.go`** - Base DevOps app framework with health checks and informers
- **`confighub.go`** - ConfigHub client with Sets, Filters, and BulkOps support
- **`claude.go`** - Claude AI integration for intelligent analysis
- **`kubernetes.go`** - Kubernetes utilities and informer setup
- **`cost.go`** - Cost analysis module for resource pricing
- **`waste.go`** - Waste detection module for over-provisioning
- **`optimizer.go`** - Optimization engine for resource rightsizing
- **`deployment.go`** - Core deployment strategies
- **`deployment_dev.go`** - Development mode deployment (direct to K8s)
- **`deployment_enterprise.go`** - Enterprise mode deployment (via Git)
- **`health.go`** - Health check endpoints for monitoring
- **`health_check.go`** - Comprehensive health checking system

## Overview

The DevOps SDK enables building persistent, event-driven DevOps applications that are superior to ephemeral workflow-based solutions. Key advantages:

- **Persistent Apps**: Run continuously with Kubernetes informers, not just when triggered
- **Event-Driven**: React immediately to changes, not on schedules
- **Stateful**: ConfigHub tracks all state and history
- **AI-Powered**: Integrated Claude AI for intelligent decisions
- **Bulk Operations**: Sets and Filters for cross-environment operations

## Core Modules

### 1. Cost Analysis Module (`cost.go`) - 744 lines
Analyzes resource costs across ConfigHub spaces and Kubernetes deployments.

**Features:**
- Multi-cloud pricing models (AWS, GCP, Azure)
- Kubernetes resource cost estimation
- ConfigHub unit cost analysis
- Hierarchical space analysis
- Cost breakdown by resource type
- Support for all Kubernetes resource units (Ki, Mi, Gi, Ti, Pi)

**Key Functions:**
- `NewCostAnalyzer()` - Create cost analyzer with ConfigHub integration
- `AnalyzeSpace()` - Analyze costs for a single space
- `AnalyzeHierarchy()` - Analyze full environment hierarchy
- `GenerateReport()` - Create detailed cost report
- `StoreAnalysisInConfigHub()` - Save analysis results
- `GetOptimizationRecommendations()` - Get cost-saving suggestions
- `ParseQuantity()` - Parse Kubernetes resource quantities

### 2. Waste Detection Module (`waste.go`) - 890 lines
Identifies resource waste by comparing allocated vs actual usage.

**Features:**
- CPU waste detection with safety checks
- Memory waste analysis
- Storage waste identification
- Idle replica detection
- Waste categorization and prioritization
- Negative waste ratio protection

**Key Functions:**
- `NewWasteAnalyzer()` - Create waste analyzer with thresholds
- `SetThresholds()` - Configure waste detection sensitivity
- `AnalyzeWaste()` - Perform comprehensive waste analysis
- `GenerateWasteReport()` - Create detailed waste report
- `IdentifyWaste()` - High-level waste identification helper

### 3. Optimization Engine (`optimizer.go`) - 1,308 lines
Generates optimized configurations based on waste analysis.

**Features:**
- Multi-container resource optimization
- Proportional resource distribution
- Safety margin calculations (20% buffer)
- Risk assessment (LOW/MEDIUM/HIGH)
- Requests/limits ratios (CPU 150%, Memory 120%)
- Deep manifest copying with type safety

**Key Functions:**
- `NewOptimizer()` - Create optimization engine
- `GenerateOptimizedConfiguration()` - Generate optimized manifests
- `ApplyOptimizations()` - Apply optimizations to ConfigHub
- `ValidateOptimizedConfiguration()` - Validate optimized configs
- `GenerateOptimizationReport()` - Create optimization report
- `StoreOptimizationInConfigHub()` - Save optimizations

### 4. Dev Mode Deployment (`deployment_dev.go`)
Direct ConfigHub → Kubernetes deployment for fast development cycles.

**Features:**
- Direct manifest application
- No Git intermediary
- Watch and sync capabilities
- Instant rollback
- Validation tools

**Key Functions:**
- `NewDevModeDeployer()` - Create dev mode deployer
- `DeployUnit()` - Deploy single unit to Kubernetes
- `DeploySpace()` - Deploy entire space
- `DeployWithFilter()` - Deploy filtered units
- `WatchAndSync()` - Continuous sync from ConfigHub
- `Rollback()` - Rollback to previous revision
- `ValidateDeployment()` - Validate deployment status

### 5. Deployment Helper (`deployment.go`)
Core deployment strategies and environment management.

**Features:**
- Environment hierarchy creation (base → qa → staging → prod)
- Automatic space setup with unique prefixes
- Filter creation for bulk operations
- Base configuration loading
- Environment promotion patterns

**Key Functions:**
- `NewDeploymentHelper()` - Create deployment helper
- `SetupBaseSpace()` - Initialize base space with unique prefix
- `CreateStandardFilters()` - Create app/infra/all filters
- `LoadBaseConfigurations()` - Load configs from files
- `CreateEnvironmentHierarchy()` - Build full env hierarchy
- `CreateVariant()` - Create config variant
- `ApplyToEnvironment()` - Deploy to specific environment
- `PromoteEnvironment()` - Promote between environments
- `QuickDeploy()` - One-command deployment

### 6. Enterprise Mode Deployment (`deployment_enterprise.go`)
ConfigHub → Git → Flux/Argo → Kubernetes for production compliance.

**Features:**
- Git repository integration
- Flux and Argo CD support
- Full audit trail
- GitOps configuration generation
- Automated sync triggering

**Key Functions:**
- `NewEnterpriseModeDeployer()` - Create enterprise deployer
- `DeployUnit()` - Export unit to Git and trigger sync
- `DeploySpace()` - Export space to Git repository
- `CreateGitOpsConfig()` - Generate Flux/Argo configs
- `ValidateGitOpsDeployment()` - Validate GitOps deployment

## Base Components

### Claude AI Client (`claude.go`)
- Simple API for sending prompts to Claude
- JSON analysis with structured responses
- Automatic response parsing and error handling
- Comprehensive timestamped logging with request/response tracking

### ConfigHub Client (`confighub.go`)
- Full CRUD operations for units and spaces
- Type-safe API interactions with real ConfigHub APIs
- Token-based authentication
- High-level convenience helpers for common patterns
- Real space name resolution (no more mock UUIDs)

### Kubernetes Utilities (`kubernetes.go`)
- Multi-client initialization (standard, dynamic, metrics)
- Config detection (kubeconfig, in-cluster)
- Resource helper for nested field operations
- Namespace detection

### DevOps App Framework (`app.go`)
- Base structure for continuous DevOps applications
- Built-in health checks and metrics
- Signal handling and graceful shutdown
- Environment variable helpers
- Retry logic with exponential backoff
- Event-driven mode with `RunWithInformers()` for Kubernetes events

### Health Server (`health.go`)
- Health and readiness endpoints
- Metrics endpoint
- Status tracking

### Comprehensive Health Check (`health_check.go`)
- Complete system health validation
- Kubernetes cluster health monitoring
- ConfigHub connection verification
- Resource availability checks
- Compliance validation for ConfigHub-only commands
- Detailed error reporting and diagnostics

### Helm Integration (`helm.go`)
- Wrap ConfigHub's native Helm support
- Install and upgrade Helm charts as ConfigHub units
- List and manage Helm releases
- Detect drift in Helm chart versions
- Validate Helm values with ConfigHub functions
- Generate upgrade commands for corrections

### Package System (`package.go`)
- Export ConfigHub resources to distributable packages
- Load packages from local directories or remote URLs
- Clone entire environments with one command
- Backup and restore spaces with timestamps
- Publish packages to Git repositories
- Support for versioned packages

## Usage

### Creating a New DevOps App

```go
package main

import (
    "log"
    sdk "github.com/monadic/devops-examples/devops-sdk"
)

func main() {
    config := sdk.DevOpsAppConfig{
        Name:        "my-devops-app",
        Version:     "1.0.0",
        Description: "My DevOps automation app",
        RunInterval: 5 * time.Minute,
        HealthPort:  8080,
    }

    app, err := sdk.NewDevOpsApp(config)
    if err != nil {
        log.Fatal(err)
    }

    // Run your main logic
    if err := app.Run(func() error {
        // Your reconciliation logic here
        return processResources(app)
    }); err != nil {
        log.Fatal(err)
    }
}

func processResources(app *sdk.DevOpsApp) error {
    // Use the pre-initialized clients
    ctx := context.Background()
    pods, err := app.K8s.Clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})

    // Get ConfigHub units using real API with space ID
    spaceID := uuid.MustParse("your-space-id")
    units, err := app.Cub.ListUnits(sdk.ListUnitsParams{
        SpaceID: spaceID,
        Where:   "Labels.monitor = 'true'",
    })

    // Analyze with Claude (now with comprehensive logging)
    response, err := app.Claude.Complete("Analyze this configuration and identify issues...")

    return nil
}
```

### Using Individual Components

#### Claude Client (with Logging)
```go
claude := sdk.NewClaudeClient(apiKey)

// Enable debug logging to see full prompts/responses
claude.EnableDebugLogging()
// Or set environment variable: CLAUDE_DEBUG_LOG=true

// Simple completion with automatic logging
response, err := claude.Complete("Analyze this Kubernetes configuration for issues")

// Analyze JSON data (logs request/response automatically)
analysis, err := claude.AnalyzeJSON(
    "Identify drift in these deployments and suggest fixes",
    driftData,
)

// Get structured response with logging
var result DriftAnalysis
err := claude.AnalyzeWithStructuredResponse(
    "Analyze configuration drift and return JSON with fixes",
    comparisonData,
    &result,
)

// Get request statistics
count, stats := claude.GetRequestStats()
fmt.Printf("Made %d Claude API calls\n", count)
```

#### ConfigHub Client (Real API)
```go
cub := sdk.NewConfigHubClient(baseURL, token)

// List units using real API with filters
units, err := cub.ListUnits(sdk.ListUnitsParams{
    SpaceID: spaceID,
    Where:   "Labels.tier = 'critical'",
})

// Create a unit with upstream relationship
unit, err := cub.CreateUnit(spaceID, sdk.CreateUnitRequest{
    Slug:           "my-deployment",
    DisplayName:    "My Application Deployment",
    Data:           yamlContent,
    UpstreamUnitID: &baseUnitID, // For inheritance
    Labels:         map[string]string{"tier": "critical"},
})

// Create a space with proper request
space, err := cub.CreateSpace(sdk.CreateSpaceRequest{
    Slug:        "new-space",
    DisplayName: "New Environment Space",
    Labels:      map[string]string{"environment": "dev"},
})

// Use Sets for bulk operations
set, err := cub.CreateSet(spaceID, sdk.CreateSetRequest{
    Slug:        "critical-services",
    DisplayName: "Critical Services Set",
})

// Apply changes using push-upgrade pattern
err = cub.BulkPatchUnits(sdk.BulkPatchParams{
    SpaceID: targetSpaceID,
    Where:   "SetID = '" + set.SetID.String() + "'",
    Patch:   patchData,
    Upgrade: true, // Push-upgrade to downstream
})
```

### High-Level Convenience Helpers
```go
// Get space by name (no more manual UUID lookups)
space, err := cub.GetSpaceBySlug("my-project-dev")

// Create space with unique prefix (like cub space new-prefix)
space, fullName, err := cub.CreateSpaceWithUniquePrefix("drift-detector",
    "Drift Detector App", map[string]string{"app": "drift-detector"})
// Result: space named "prefix-1234567890-drift-detector"

// Clone units with upstream relationships
units, err := cub.BulkCloneUnitsWithUpstream(
    sourceSpaceID, targetSpaceID,
    []string{"deployment", "service", "rbac"},
    map[string]string{"environment": "staging"},
)

// Apply units in dependency order
err = cub.ApplyUnitsInOrder(spaceID, []string{
    "namespace", "rbac", "service", "deployment",
})
```

### Helm Chart Management
```go
// Create Helm helper
helm := sdk.NewHelmHelper(cub, spaceID)

// Install a Helm chart
err := helm.InstallChart("my-nginx", "bitnami/nginx", sdk.HelmOptions{
    Namespace: "web",
    Version:   "15.5.2",
    Values: []string{
        "service.type=LoadBalancer",
        "replicaCount=3",
    },
})

// List Helm releases
releases, err := helm.ListHelmReleases()
for _, release := range releases {
    fmt.Printf("Release: %s, Chart: %s v%s\n",
        release.Name, release.Chart, release.Version)
}

// Check for newer versions
release, _ := helm.GetHelmRelease("my-nginx")
hasUpdate, newVersion, _ := helm.CompareChartVersions(release)
if hasUpdate {
    fmt.Printf("Update available: %s -> %s\n", release.Version, newVersion)

    // Upgrade the chart
    err = helm.UpgradeChart("my-nginx", "bitnami/nginx", sdk.HelmOptions{
        Version: newVersion,
        UpdateCRDs: true,
    })
}

// Generate correction command for drift
cmd := helm.GenerateUpgradeCommand(release, newVersion)
fmt.Printf("To fix drift, run: %s\n", cmd)
```

### Package Management (Experimental)
```go
// Note: Requires CONFIGHUB_EXPERIMENTAL=1 environment variable
pkg := sdk.NewPackageHelper(cub)

// Export app configuration to package
err := pkg.CreatePackage("./my-app-package", sdk.PackageOptions{
    SpaceID: spaceID,
    Where:   "Labels.app='drift-detector'",
})

// Load package from local directory
err = pkg.LoadPackage("./my-app-package", "staging")

// Load from GitHub
err = pkg.LoadPackageFromGitHub("monadic", "packages", "drift-detector", "prod")

// Clone entire environment
err = pkg.CloneEnvironment(prodSpaceID, "staging-clone")

// Backup space with timestamp
backupPath, err := pkg.BackupSpace(spaceID, "./backups")
fmt.Printf("Backup created at: %s\n", backupPath)

// Restore from backup
err = pkg.RestoreSpace(backupPath, "restored")

// Create versioned package
err = pkg.CreateVersionedPackage("./release", "v1.2.0", sdk.PackageOptions{
    SpaceID: spaceID,
})

// Publish package to Git
err = pkg.PublishPackage("./my-app-package",
    "https://github.com/myorg/packages.git",
    "Release v1.2.0")
```

#### Kubernetes Helpers
```go
// Get configured clients
k8s, err := sdk.NewK8sClients()

// Use different client types
pods, err := k8s.Clientset.CoreV1().Pods("").List(...)
metrics, err := k8s.MetricsClient.MetricsV1beta1().PodMetricses("").List(...)

// Resource helpers
helper := sdk.NewResourceHelper()
value := helper.GetResourceValue(resource, "spec.replicas")
helper.SetResourceValue(resource, "spec.replicas", 5)
```

## Environment Variables

The SDK automatically reads these environment variables:

- `CLAUDE_API_KEY`: Claude API key
- `CLAUDE_DEBUG_LOG`: Set to "true" to enable full prompt/response logging
- `CUB_TOKEN`: ConfigHub authentication token
- `CUB_API_URL`: ConfigHub API base URL
- `KUBECONFIG`: Path to kubeconfig file
- `NAMESPACE`: Default namespace for operations

### Claude Logging Levels

```bash
# Standard logging: request/response previews only
export CLAUDE_DEBUG_LOG=false

# Debug logging: full prompts and responses
export CLAUDE_DEBUG_LOG=true
```

Example log output:
```
[Claude] req-1 ◀ REQUEST: Analyze this Kubernetes configuration for drift...
[Claude] req-1 → Sending API request
[Claude] req-1 ▶ RESPONSE (2.1s): I found 3 configuration drift issues...
```

## Helper Functions

```go
// Environment variables with defaults
namespace := sdk.GetEnvOrDefault("NAMESPACE", "default")
required := sdk.GetEnvOrPanic("CLAUDE_API_KEY")
enabled := sdk.GetEnvBool("AUTO_APPLY", false)
interval := sdk.GetEnvDuration("CHECK_INTERVAL", 5*time.Minute)
port := sdk.GetEnvInt("PORT", 8080)

// Retry logic
err := sdk.RunWithRetry(ctx, 3, func() error {
    return apiCall()
})
```

## Comprehensive Health Checking

```go
// Create health checker
healthChecker := sdk.NewComprehensiveHealthCheck(
    k8s.Clientset,
    cub,
    "default",
)

// Run health check
ctx := context.Background()
result, err := healthChecker.RunHealthCheck(ctx)

// Check results
if result.Status == sdk.HealthStatusHealthy {
    fmt.Println("All systems operational")
} else {
    fmt.Printf("Issues detected: %v\n", result.Issues)
}

// Validate ConfigHub compliance
corrections := []string{
    "cub unit update backend --patch ...",
    "cub unit apply backend --space prod",
}
isCompliant := healthChecker.CheckConfigHubCompliance(corrections)
```

## Example Apps Using This SDK

The SDK is used by three production-ready DevOps apps in the [devops-examples](https://github.com/monadic/devops-examples) repository:

### 1. Drift Detector

Event-driven configuration drift detection with auto-correction.

**Key features:**
- Kubernetes informers for real-time drift detection
- ConfigHub Sets and Filters for targeting critical services
- Claude AI for root cause analysis
- Push-upgrade pattern for multi-environment fixes
- Web dashboard on :8080

**Deployment:**
```bash
cd devops-examples/drift-detector
bin/install-base    # Create ConfigHub structure
bin/setup-worker    # Install ConfigHub worker
bin/apply-base      # Deploy to Kubernetes
bin/test-workflow   # Validate everything works
```

See [QUICKSTART.md](https://github.com/monadic/devops-examples/blob/main/drift-detector/QUICKSTART.md) for full guide.

### 2. Cost Optimizer

AI-powered cost optimization with OpenCost integration.

**Key features:**
- Real-time cost analysis from OpenCost or AWS pricing
- Claude AI cost recommendations
- Bulk optimization across environments
- Interactive web dashboard on :8081
- Metrics server integration

**Deployment:**
```bash
cd devops-examples/cost-optimizer
bin/install-base    # Create ConfigHub structure
bin/setup-worker    # Install ConfigHub worker
bin/apply-base      # Deploy to Kubernetes
bin/test-workflow   # Validate everything works
```

See [QUICKSTART.md](https://github.com/monadic/devops-examples/blob/main/cost-optimizer/QUICKSTART.md) for full guide.

### 3. Cost Impact Monitor

Pre-deployment cost analysis with trigger-based hooks.

**Key features:**
- Monitors all ConfigHub spaces for cost changes
- Pre-apply warnings for high-cost deployments
- Post-apply verification and learning
- Cross-environment cost tracking
- Web dashboard on :8083

**Deployment:**
```bash
cd devops-examples/cost-impact-monitor
bin/install-base    # Create ConfigHub structure
bin/setup-worker    # Install ConfigHub worker
bin/apply-base      # Deploy to Kubernetes
bin/test-workflow   # Validate everything works
```

See [QUICKSTART.md](https://github.com/monadic/devops-examples/blob/main/cost-impact-monitor/QUICKSTART.md) for full guide.

### Common Workflow

All examples follow the same deployment pattern:

1. **Create ConfigHub structure** - `bin/install-base` creates spaces, filters, and units
2. **Set up worker** - `bin/setup-worker` installs ConfigHub worker for applies
3. **Deploy to Kubernetes** - `bin/apply-base` sets targets and applies units
4. **Validate** - `bin/test-workflow` checks everything is working

This demonstrates the **ConfigHub → Worker → Kubernetes** deployment workflow that replaces traditional `kubectl apply` with ConfigHub-managed deployments.

## Benefits

1. **Reduces Boilerplate**: No need to initialize clients in every app
2. **Standardized Patterns**: Consistent error handling and logging
3. **Built-in Operations**: Health checks, metrics, graceful shutdown
4. **Type Safety**: Strongly typed API clients
5. **Testable**: Easy to mock individual components
6. **Extensible**: Add new clients or utilities as needed