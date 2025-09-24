package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// HealthServer provides health and metrics endpoints
type HealthServer struct {
	port      int
	app       *DevOpsApp
	mu        sync.RWMutex
	healthy   bool
	lastCheck time.Time
	message   string
	metrics   map[string]interface{}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    string                 `json:"status"`
	App       string                 `json:"app"`
	Version   string                 `json:"version"`
	Healthy   bool                   `json:"healthy"`
	Message   string                 `json:"message,omitempty"`
	LastCheck string                 `json:"last_check"`
	Uptime    string                 `json:"uptime"`
	Metrics   map[string]interface{} `json:"metrics,omitempty"`
}

// NewHealthServer creates a new health server
func NewHealthServer(port int, app *DevOpsApp) *HealthServer {
	return &HealthServer{
		port:      port,
		app:       app,
		healthy:   true,
		lastCheck: time.Now(),
		message:   "Starting up",
		metrics:   make(map[string]interface{}),
	}
}

// Start starts the health server
func (h *HealthServer) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", h.healthHandler)
	mux.HandleFunc("/ready", h.readyHandler)
	mux.HandleFunc("/metrics", h.metricsHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", h.port),
		Handler: mux,
	}

	h.app.Logger.Printf("Health server started on port %d", h.port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		h.app.Logger.Printf("Health server error: %v", err)
	}
}

// SetHealthy updates the health status
func (h *HealthServer) SetHealthy(healthy bool, message string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.healthy = healthy
	h.message = message
	h.lastCheck = time.Now()
}

// UpdateMetric updates a metric value
func (h *HealthServer) UpdateMetric(key string, value interface{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.metrics[key] = value
}

// healthHandler handles health check requests
func (h *HealthServer) healthHandler(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	status := "healthy"
	if !h.healthy {
		status = "unhealthy"
	}

	response := HealthResponse{
		Status:    status,
		App:       h.app.Name,
		Version:   h.app.Version,
		Healthy:   h.healthy,
		Message:   h.message,
		LastCheck: h.lastCheck.Format(time.RFC3339),
		Uptime:    time.Since(h.lastCheck).String(),
		Metrics:   h.metrics,
	}

	w.Header().Set("Content-Type", "application/json")
	if !h.healthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// readyHandler handles readiness check requests
func (h *HealthServer) readyHandler(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if h.healthy {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Not Ready"))
	}
}

// metricsHandler handles metrics requests
func (h *HealthServer) metricsHandler(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.metrics)
}
