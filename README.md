# DevOps SDK

A reusable Go SDK for building DevOps applications that integrate with Kubernetes, ConfigHub, and Claude AI.

## Features

### Core Components

1. **Claude AI Client** (`claude.go`)
   - Simple API for sending prompts to Claude
   - JSON analysis with structured responses
   - Automatic response parsing and error handling
   - **Comprehensive timestamped logging** with request/response tracking

2. **ConfigHub Client** (`confighub.go`)
   - Full CRUD operations for units and spaces
   - Type-safe API interactions with real ConfigHub APIs
   - Token-based authentication
   - **High-level convenience helpers** for common patterns
   - **Real space name resolution** (no more mock UUIDs)

3. **Kubernetes Utilities** (`kubernetes.go`)
   - Multi-client initialization (standard, dynamic, metrics)
   - Config detection (kubeconfig, in-cluster)
   - Resource helper for nested field operations
   - Namespace detection

4. **DevOps App Framework** (`app.go`)
   - Base structure for continuous DevOps applications
   - Built-in health checks and metrics
   - Signal handling and graceful shutdown
   - Environment variable helpers
   - Retry logic with exponential backoff
   - **Event-driven mode** with `RunWithInformers()` for Kubernetes events

5. **Health Server** (`health.go`)
   - Health and readiness endpoints
   - Metrics endpoint
   - Status tracking

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

## Example Apps Using This SDK

### Drift Detector (Real Event-Driven Implementation)
```go
func main() {
    app, _ := sdk.NewDevOpsApp(sdk.DevOpsAppConfig{
        Name: "drift-detector",
    })

    // Event-driven approach using Kubernetes informers
    app.RunWithInformers(func() error {
        // Get units from ConfigHub using Sets and Filters
        criticalSet, _ := app.Cub.GetSet(spaceID, "critical-services")
        units, _ := app.Cub.ListUnits(sdk.ListUnitsParams{
            SpaceID: spaceID,
            Where:   fmt.Sprintf("'%s' IN SetIDs", criticalSet.SetID),
        })

        // Get live deployments (triggered by K8s events)
        deployments, _ := app.K8s.Clientset.AppsV1().Deployments("").List(ctx, metav1.ListOptions{})

        // Compare and analyze with Claude (with full logging)
        app.Claude.EnableDebugLogging()
        analysis, _ := app.Claude.AnalyzeJSON(
            "Compare expected vs actual Kubernetes state and identify drift",
            comparisonData,
        )

        // Apply fixes using push-upgrade pattern
        if analysis.HasDrift {
            app.Cub.BulkPatchUnits(sdk.BulkPatchParams{
                SpaceID: spaceID,
                Where:   "Labels.tier = 'critical'",
                Patch:   analysis.FixPatch,
                Upgrade: true,
            })
        }

        return nil
    })
}
```

### Cost Optimizer (Simplified)
```go
func main() {
    app, _ := sdk.NewDevOpsApp(sdk.DevOpsAppConfig{
        Name: "cost-optimizer",
    })

    app.Run(func() error {
        // Get resource metrics
        podMetrics, _ := app.K8s.MetricsClient.MetricsV1beta1().PodMetricses("").List(...)

        // Analyze costs with Claude
        recommendations, _ := app.Claude.AnalyzeJSON("Optimize costs", usage)

        // Create optimization space in ConfigHub
        app.Cub.CreateSpace(sdk.Space{
            Slug: fmt.Sprintf("cost-opt-%d", time.Now().Unix()),
        })

        return nil
    })
}
```

## Benefits

1. **Reduces Boilerplate**: No need to initialize clients in every app
2. **Standardized Patterns**: Consistent error handling and logging
3. **Built-in Operations**: Health checks, metrics, graceful shutdown
4. **Type Safety**: Strongly typed API clients
5. **Testable**: Easy to mock individual components
6. **Extensible**: Add new clients or utilities as needed