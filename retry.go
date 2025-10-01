// Retry Logic and Circuit Breaker Module
// Copy this to: /Users/alexis/Public/github-repos/devops-sdk/retry.go
// Provides resilient HTTP client operations with retries and circuit breaker

package sdk

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// RetryConfig configures retry behavior
type RetryConfig struct {
	MaxAttempts     int           // Maximum number of retry attempts
	InitialDelay    time.Duration // Initial delay between retries
	MaxDelay        time.Duration // Maximum delay between retries
	Multiplier      float64       // Backoff multiplier (exponential)
	RetryableErrors []string      // List of retryable error patterns
}

// DefaultRetryConfig provides sensible defaults
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:  3,
	InitialDelay: 100 * time.Millisecond,
	MaxDelay:     30 * time.Second,
	Multiplier:   2.0,
	RetryableErrors: []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"service unavailable",
	},
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	maxFailures  int
	resetTimeout time.Duration
	state        CircuitState
	failures     int
	lastFailTime time.Time
	mu           sync.RWMutex
	logger       *log.Logger
}

// CircuitState represents the circuit breaker state
type CircuitState int

const (
	StateClosed CircuitState = iota // Normal operation
	StateOpen                        // Circuit is open, rejecting requests
	StateHalfOpen                    // Testing if service recovered
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration, logger *log.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        StateClosed,
		logger:       logger,
	}
}

// Execute runs the operation through the circuit breaker
func (cb *CircuitBreaker) Execute(operation func() error) error {
	if !cb.canAttempt() {
		return fmt.Errorf("circuit breaker is OPEN")
	}

	err := operation()

	if err != nil {
		cb.recordFailure()
		return err
	}

	cb.recordSuccess()
	return nil
}

// canAttempt checks if we can attempt the operation
func (cb *CircuitBreaker) canAttempt() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if enough time has passed to try again
		if time.Since(cb.lastFailTime) > cb.resetTimeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = StateHalfOpen
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case StateHalfOpen:
		return true
	}

	return false
}

// recordFailure records a failed attempt
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.lastFailTime = time.Now()

	if cb.failures >= cb.maxFailures {
		cb.state = StateOpen
		if cb.logger != nil {
			cb.logger.Printf("⚠️  Circuit breaker OPENED after %d failures", cb.failures)
		}
	}
}

// recordSuccess records a successful attempt
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.state = StateClosed
		if cb.logger != nil {
			cb.logger.Printf("✓ Circuit breaker CLOSED - service recovered")
		}
	}

	cb.failures = 0
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// RetryableClient wraps operations with retry logic and circuit breaker
type RetryableClient struct {
	config         RetryConfig
	circuitBreaker *CircuitBreaker
	logger         *log.Logger
}

// NewRetryableClient creates a new client with retry capabilities
func NewRetryableClient(config RetryConfig, logger *log.Logger) *RetryableClient {
	return &RetryableClient{
		config:         config,
		circuitBreaker: NewCircuitBreaker(5, 60*time.Second, logger),
		logger:         logger,
	}
}

// ExecuteWithRetry executes an operation with retry logic and circuit breaker
func (rc *RetryableClient) ExecuteWithRetry(operationName string, operation func() error) error {
	return rc.circuitBreaker.Execute(func() error {
		return rc.retryWithBackoff(operationName, operation)
	})
}

// retryWithBackoff implements exponential backoff retry logic
func (rc *RetryableClient) retryWithBackoff(operationName string, operation func() error) error {
	var lastErr error
	delay := rc.config.InitialDelay

	for attempt := 1; attempt <= rc.config.MaxAttempts; attempt++ {
		if rc.logger != nil && attempt > 1 {
			rc.logger.Printf("🔄 Retry %d/%d for %s (after %v delay)",
				attempt-1, rc.config.MaxAttempts-1, operationName, delay)
		}

		lastErr = operation()

		if lastErr == nil {
			if rc.logger != nil && attempt > 1 {
				rc.logger.Printf("✓ %s succeeded after %d attempts", operationName, attempt)
			}
			return nil
		}

		// Check if error is retryable
		if !rc.isRetryable(lastErr) {
			if rc.logger != nil {
				rc.logger.Printf("✗ %s failed with non-retryable error: %v", operationName, lastErr)
			}
			return lastErr
		}

		// Don't sleep after the last attempt
		if attempt < rc.config.MaxAttempts {
			if rc.logger != nil {
				rc.logger.Printf("⏳ Waiting %v before retry %d/%d",
					delay, attempt, rc.config.MaxAttempts-1)
			}
			time.Sleep(delay)

			// Calculate next delay with exponential backoff
			delay = time.Duration(float64(delay) * rc.config.Multiplier)
			if delay > rc.config.MaxDelay {
				delay = rc.config.MaxDelay
			}
		}
	}

	if rc.logger != nil {
		rc.logger.Printf("✗ %s failed after %d attempts: %v",
			operationName, rc.config.MaxAttempts, lastErr)
	}

	return fmt.Errorf("%s failed after %d attempts: %w",
		operationName, rc.config.MaxAttempts, lastErr)
}

// isRetryable checks if an error should trigger a retry
func (rc *RetryableClient) isRetryable(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	for _, pattern := range rc.config.RetryableErrors {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// ============================================================================
// INTEGRATION WITH CONFIGHHUB CLIENT
// ============================================================================

// Add these methods to ConfigHubClient:

/*
// Add to ConfigHubClient struct:
type ConfigHubClient struct {
	// ... existing fields ...
	retryClient *RetryableClient
}

// Modify NewConfigHubClient:
func NewConfigHubClient(apiURL, token string, logger *log.Logger) *ConfigHubClient {
	return &ConfigHubClient{
		APIBaseURL:  apiURL,
		Token:       token,
		HTTPClient:  &http.Client{Timeout: 30 * time.Second},
		Logger:      logger,
		retryClient: NewRetryableClient(DefaultRetryConfig, logger),
	}
}

// Example usage in CreateUnit:
func (c *ConfigHubClient) CreateUnit(spaceID uuid.UUID, req CreateUnitRequest) (*Unit, error) {
	var unit *Unit
	var err error

	err = c.retryClient.ExecuteWithRetry("CreateUnit", func() error {
		unit, err = c.createUnitOnce(spaceID, req)
		return err
	})

	return unit, err
}

// createUnitOnce is the actual implementation (no retry logic)
func (c *ConfigHubClient) createUnitOnce(spaceID uuid.UUID, req CreateUnitRequest) (*Unit, error) {
	// ... existing implementation ...
}
*/

// ============================================================================
// USAGE EXAMPLES
// ============================================================================

// Example 1: Simple retry
func ExampleRetryWithBackoff() {
	logger := log.New(os.Stdout, "[RETRY] ", log.LstdFlags)
	client := NewRetryableClient(DefaultRetryConfig, logger)

	err := client.ExecuteWithRetry("Database connection", func() error {
		// Your operation here
		return nil
	})

	if err != nil {
		log.Printf("Operation failed: %v", err)
	}
}

// Example 2: Circuit breaker
func ExampleCircuitBreaker() {
	logger := log.New(os.Stdout, "[CB] ", log.LstdFlags)
	cb := NewCircuitBreaker(3, 30*time.Second, logger)

	for i := 0; i < 10; i++ {
		err := cb.Execute(func() error {
			// Your operation here
			return fmt.Errorf("service unavailable")
		})

		if err != nil {
			log.Printf("Attempt %d failed: %v", i+1, err)
		}

		time.Sleep(time.Second)
	}
}

// Example 3: Custom retry configuration
func ExampleCustomRetryConfig() {
	customConfig := RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 200 * time.Millisecond,
		MaxDelay:     10 * time.Second,
		Multiplier:   1.5,
		RetryableErrors: []string{
			"network error",
			"timeout",
			"connection reset",
		},
	}

	logger := log.New(os.Stdout, "[CUSTOM] ", log.LstdFlags)
	client := NewRetryableClient(customConfig, logger)

	client.ExecuteWithRetry("Custom operation", func() error {
		// Your operation
		return nil
	})
}
