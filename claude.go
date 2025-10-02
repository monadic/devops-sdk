package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ClaudeAPICall represents a single API interaction with Claude
type ClaudeAPICall struct {
	RequestID    string    `json:"request_id"`
	Timestamp    time.Time `json:"timestamp"`
	Prompt       string    `json:"prompt"`
	Response     string    `json:"response"`
	Duration     string    `json:"duration"`
	Success      bool      `json:"success"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// ClaudeClient provides a simple interface to Claude API with comprehensive logging
type ClaudeClient struct {
	apiKey         string
	client         *http.Client
	logger         *log.Logger
	enableDebugLog bool
	requestCounter int64
	history        []ClaudeAPICall
	historyMu      sync.RWMutex
	maxHistory     int
}

// NewClaudeClient creates a new Claude API client with logging
func NewClaudeClient(apiKey string) *ClaudeClient {
	// Create logger for Claude operations
	logger := log.New(os.Stdout, "[Claude] ", log.LstdFlags)

	// Enable debug logging if environment variable is set
	enableDebug := os.Getenv("CLAUDE_DEBUG_LOGGING") == "true" || os.Getenv("CLAUDE_DEBUG_LOG") == "true"

	if enableDebug {
		logger.Println("🔍 Debug logging enabled - all prompts and responses will be logged")
	}

	return &ClaudeClient{
		apiKey:         apiKey,
		client:         &http.Client{Timeout: 30 * time.Second},
		logger:         logger,
		enableDebugLog: enableDebug,
		requestCounter: 0,
		history:        make([]ClaudeAPICall, 0, 10),
		maxHistory:     10, // Keep last 10 API calls
	}
}

// GetRecentCalls returns the most recent API calls (for dashboard display)
func (c *ClaudeClient) GetRecentCalls() []ClaudeAPICall {
	c.historyMu.RLock()
	defer c.historyMu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]ClaudeAPICall, len(c.history))
	copy(result, c.history)
	return result
}

// addToHistory adds a call to the history buffer
func (c *ClaudeClient) addToHistory(call ClaudeAPICall) {
	c.historyMu.Lock()
	defer c.historyMu.Unlock()

	c.history = append(c.history, call)
	if len(c.history) > c.maxHistory {
		c.history = c.history[1:] // Remove oldest
	}
}

// Complete sends a prompt to Claude and returns the response
func (c *ClaudeClient) Complete(prompt string) (string, error) {
	c.requestCounter++
	startTime := time.Now()
	requestID := fmt.Sprintf("req-%d", c.requestCounter)

	// Log the request
	c.logRequest(requestID, prompt)

	request := map[string]interface{}{
		"model":      "claude-3-haiku-20240307",
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		c.logError(requestID, fmt.Errorf("marshal request: %w", err))
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		c.logError(requestID, fmt.Errorf("create request: %w", err))
		return "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	c.logger.Printf("%s → Sending API request", requestID)

	resp, err := c.client.Do(req)
	if err != nil {
		c.logError(requestID, fmt.Errorf("send request: %w", err))
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logError(requestID, fmt.Errorf("read response: %w", err))
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		apiErr := fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		c.logError(requestID, apiErr)
		return "", apiErr
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		c.logError(requestID, fmt.Errorf("unmarshal response: %w", err))
		c.addToHistory(ClaudeAPICall{
			RequestID:    requestID,
			Timestamp:    startTime,
			Prompt:       truncateString(prompt, 200),
			Response:     "",
			Duration:     time.Since(startTime).String(),
			Success:      false,
			ErrorMessage: fmt.Sprintf("unmarshal response: %v", err),
		})
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	if len(result.Content) == 0 {
		err := fmt.Errorf("empty response from Claude")
		c.logError(requestID, err)
		c.addToHistory(ClaudeAPICall{
			RequestID:    requestID,
			Timestamp:    startTime,
			Prompt:       truncateString(prompt, 200),
			Response:     "",
			Duration:     time.Since(startTime).String(),
			Success:      false,
			ErrorMessage: "empty response from Claude",
		})
		return "", err
	}

	response := result.Content[0].Text
	duration := time.Since(startTime)

	// Log the successful response
	c.logResponse(requestID, response, duration)

	// Add to history for dashboard display
	c.addToHistory(ClaudeAPICall{
		RequestID: requestID,
		Timestamp: startTime,
		Prompt:    truncateString(prompt, 200),
		Response:  truncateString(response, 500),
		Duration:  duration.String(),
		Success:   true,
	})

	return response, nil
}

// truncateString truncates a string to maxLen with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// AnalyzeJSON sends a prompt with JSON data to Claude for analysis
func (c *ClaudeClient) AnalyzeJSON(prompt string, data interface{}) (string, error) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal data: %w", err)
	}

	fullPrompt := fmt.Sprintf("%s\n\nData:\n```json\n%s\n```", prompt, string(jsonData))
	return c.Complete(fullPrompt)
}

// AnalyzeWithStructuredResponse sends a prompt and expects a JSON response
func (c *ClaudeClient) AnalyzeWithStructuredResponse(prompt string, data interface{}, result interface{}) error {
	response, err := c.AnalyzeJSON(prompt, data)
	if err != nil {
		return err
	}

	// Extract JSON from the response (Claude often wraps in markdown)
	jsonStart := bytes.Index([]byte(response), []byte("```json"))
	jsonEnd := bytes.LastIndex([]byte(response), []byte("```"))

	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		jsonStart += 7 // Skip "```json\n"
		response = response[jsonStart:jsonEnd]
	}

	if err := json.Unmarshal([]byte(response), result); err != nil {
		return fmt.Errorf("unmarshal structured response: %w", err)
	}

	return nil
}

// Logging helper methods

// logRequest logs the incoming prompt request
func (c *ClaudeClient) logRequest(requestID, prompt string) {
	promptPreview := c.truncateString(prompt, 200)
	c.logger.Printf("%s ◀ REQUEST: %s", requestID, promptPreview)

	if c.enableDebugLog {
		c.logger.Printf("%s ◀ FULL_PROMPT:\n%s", requestID, prompt)
	}
}

// logResponse logs the Claude response
func (c *ClaudeClient) logResponse(requestID, response string, duration time.Duration) {
	responsePreview := c.truncateString(response, 200)
	c.logger.Printf("%s ▶ RESPONSE (%v): %s", requestID, duration, responsePreview)

	if c.enableDebugLog {
		c.logger.Printf("%s ▶ FULL_RESPONSE:\n%s", requestID, response)
	}
}

// logError logs errors during Claude API calls
func (c *ClaudeClient) logError(requestID string, err error) {
	c.logger.Printf("%s ✗ ERROR: %v", requestID, err)
}

// truncateString truncates a string for preview logging
func (c *ClaudeClient) truncateString(s string, maxLen int) string {
	// Replace newlines with spaces for single-line preview
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// EnableDebugLogging enables full request/response logging
func (c *ClaudeClient) EnableDebugLogging() {
	c.enableDebugLog = true
	c.logger.Printf("Debug logging enabled - full prompts/responses will be logged")
}

// DisableDebugLogging disables full request/response logging
func (c *ClaudeClient) DisableDebugLogging() {
	c.enableDebugLog = false
	c.logger.Printf("Debug logging disabled - only previews will be logged")
}

// GetRequestStats returns basic request statistics
func (c *ClaudeClient) GetRequestStats() (int64, string) {
	return c.requestCounter, fmt.Sprintf("Total Claude API requests: %d", c.requestCounter)
}
