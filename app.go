package sdk

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"
)

// DevOpsApp provides base structure for DevOps applications
type DevOpsApp struct {
    Name          string
    Version       string
    Description   string
    RunInterval   time.Duration
    K8s          *K8sClients
    Claude       *ClaudeClient
    Cub          *ConfigHubClient
    Logger       *log.Logger
    stopChan     chan struct{}
    healthServer *HealthServer
}

// DevOpsAppConfig holds configuration for DevOps apps
type DevOpsAppConfig struct {
    Name          string
    Version       string
    Description   string
    RunInterval   time.Duration
    HealthPort    int
    ClaudeAPIKey  string
    CubToken      string
    CubBaseURL    string
}

// NewDevOpsApp creates a new DevOps application
func NewDevOpsApp(config DevOpsAppConfig) (*DevOpsApp, error) {
    // Set defaults
    if config.RunInterval == 0 {
        config.RunInterval = 5 * time.Minute
    }
    if config.HealthPort == 0 {
        config.HealthPort = 8080
    }

    // Initialize logger
    logger := log.New(os.Stdout, fmt.Sprintf("[%s] ", config.Name), log.LstdFlags)

    // Initialize Kubernetes clients
    k8s, err := NewK8sClients()
    if err != nil {
        return nil, fmt.Errorf("init k8s clients: %w", err)
    }

    // Initialize Claude client if API key provided
    var claude *ClaudeClient
    if config.ClaudeAPIKey != "" {
        claude = NewClaudeClient(config.ClaudeAPIKey)
    } else if key := os.Getenv("CLAUDE_API_KEY"); key != "" {
        claude = NewClaudeClient(key)
    }

    // Initialize ConfigHub client if token provided
    var cub *ConfigHubClient
    if config.CubToken != "" {
        cub = NewConfigHubClient(config.CubBaseURL, config.CubToken)
    } else if token := os.Getenv("CUB_TOKEN"); token != "" {
        baseURL := config.CubBaseURL
        if baseURL == "" {
            baseURL = os.Getenv("CUB_API_URL")
        }
        cub = NewConfigHubClient(baseURL, token)
    }

    app := &DevOpsApp{
        Name:        config.Name,
        Version:     config.Version,
        Description: config.Description,
        RunInterval: config.RunInterval,
        K8s:         k8s,
        Claude:      claude,
        Cub:         cub,
        Logger:      logger,
        stopChan:    make(chan struct{}),
    }

    // Start health server
    app.healthServer = NewHealthServer(config.HealthPort, app)
    go app.healthServer.Start()

    return app, nil
}

// Run starts the main application loop
func (app *DevOpsApp) Run(handler func() error) error {
    app.Logger.Printf("%s v%s started", app.Name, app.Version)
    app.Logger.Printf("Description: %s", app.Description)
    app.Logger.Printf("Run interval: %v", app.RunInterval)

    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    // Run initial execution
    if err := handler(); err != nil {
        app.Logger.Printf("Initial run error: %v", err)
    }

    // Start main loop
    ticker := time.NewTicker(app.RunInterval)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            app.Logger.Println("Running scheduled task...")
            if err := handler(); err != nil {
                app.Logger.Printf("Task error: %v", err)
                app.healthServer.SetHealthy(false, fmt.Sprintf("Task failed: %v", err))
            } else {
                app.healthServer.SetHealthy(true, "Running")
            }

        case <-sigChan:
            app.Logger.Println("Received shutdown signal")
            close(app.stopChan)
            return nil

        case <-app.stopChan:
            app.Logger.Println("Stopping application")
            return nil
        }
    }
}

// Stop gracefully stops the application
func (app *DevOpsApp) Stop() {
    close(app.stopChan)
}

// GetEnvOrDefault gets an environment variable with a default value
func GetEnvOrDefault(key, defaultValue string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return defaultValue
}

// GetEnvOrPanic gets an environment variable or panics
func GetEnvOrPanic(key string) string {
    value := os.Getenv(key)
    if value == "" {
        panic(fmt.Sprintf("Required environment variable %s not set", key))
    }
    return value
}

// GetEnvBool gets a boolean environment variable
func GetEnvBool(key string, defaultValue bool) bool {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }
    return value == "true" || value == "1" || value == "yes"
}

// GetEnvDuration gets a duration environment variable
func GetEnvDuration(key string, defaultValue time.Duration) time.Duration {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }

    duration, err := time.ParseDuration(value)
    if err != nil {
        return defaultValue
    }
    return duration
}

// GetEnvInt gets an integer environment variable
func GetEnvInt(key string, defaultValue int) int {
    value := os.Getenv(key)
    if value == "" {
        return defaultValue
    }

    var result int
    if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
        return defaultValue
    }
    return result
}

// RunWithRetry runs a function with exponential backoff retry
func RunWithRetry(ctx context.Context, maxRetries int, f func() error) error {
    var lastErr error
    backoff := time.Second

    for i := 0; i < maxRetries; i++ {
        if err := f(); err == nil {
            return nil
        } else {
            lastErr = err
        }

        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(backoff):
            backoff *= 2
            if backoff > time.Minute {
                backoff = time.Minute
            }
        }
    }

    return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}