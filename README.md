# DevOps SDK

A reusable Go SDK for building DevOps applications that integrate with Kubernetes, ConfigHub, and Claude AI.

## Features

### Core Components

1. **Claude AI Client** (`claude.go`)
   - Simple API for sending prompts to Claude
   - JSON analysis with structured responses
   - Automatic response parsing and error handling

2. **ConfigHub Client** (`confighub.go`)
   - Full CRUD operations for units and spaces
   - Type-safe API interactions
   - Token-based authentication

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
    pods, err := app.K8s.Clientset.CoreV1().Pods("default").List(...)

    // Get ConfigHub units
    units, err := app.Cub.GetUnits("my-space")

    // Analyze with Claude
    response, err := app.Claude.Complete("Analyze this configuration...")

    return nil
}
```

### Using Individual Components

#### Claude Client
```go
claude := sdk.NewClaudeClient(apiKey)

// Simple completion
response, err := claude.Complete("What is the capital of France?")

// Analyze JSON data
analysis, err := claude.AnalyzeJSON(
    "Identify issues in this configuration",
    configData,
)

// Get structured response
var result DriftAnalysis
err := claude.AnalyzeWithStructuredResponse(
    "Analyze drift and return JSON",
    data,
    &result,
)
```

#### ConfigHub Client
```go
cub := sdk.NewCubClient(baseURL, token)

// Get units
units, err := cub.GetUnits("my-space")

// Update a unit
unit.Data["replicas"] = 3
err = cub.UpdateUnit("my-space", unit)

// Create a space
space := sdk.Space{
    Slug: "new-space",
    Name: "New Space",
    Parent: "parent-space",
}
err = cub.CreateSpace(space)
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
- `CUB_TOKEN`: ConfigHub authentication token
- `CUB_API_URL`: ConfigHub API base URL
- `KUBECONFIG`: Path to kubeconfig file
- `NAMESPACE`: Default namespace for operations

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

### Drift Detector (Simplified)
```go
func main() {
    app, _ := sdk.NewDevOpsApp(sdk.DevOpsAppConfig{
        Name: "drift-detector",
    })

    app.Run(func() error {
        // Get units from ConfigHub
        units, _ := app.Cub.GetUnits(space)

        // Get resources from Kubernetes
        deployments, _ := app.K8s.Clientset.AppsV1().Deployments("").List(...)

        // Compare and analyze with Claude
        analysis, _ := app.Claude.AnalyzeJSON("Find drift", comparison)

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