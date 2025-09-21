package sdk

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"
)

// ClaudeClient provides a simple interface to Claude API
type ClaudeClient struct {
    apiKey string
    client *http.Client
}

// NewClaudeClient creates a new Claude API client
func NewClaudeClient(apiKey string) *ClaudeClient {
    return &ClaudeClient{
        apiKey: apiKey,
        client: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

// Complete sends a prompt to Claude and returns the response
func (c *ClaudeClient) Complete(prompt string) (string, error) {
    request := map[string]interface{}{
        "model": "claude-3-haiku-20240307",
        "max_tokens": 4096,
        "messages": []map[string]string{
            {"role": "user", "content": prompt},
        },
    }

    jsonData, err := json.Marshal(request)
    if err != nil {
        return "", fmt.Errorf("marshal request: %w", err)
    }

    req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
    if err != nil {
        return "", fmt.Errorf("create request: %w", err)
    }

    req.Header.Set("x-api-key", c.apiKey)
    req.Header.Set("anthropic-version", "2023-06-01")
    req.Header.Set("Content-Type", "application/json")

    resp, err := c.client.Do(req)
    if err != nil {
        return "", fmt.Errorf("send request: %w", err)
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return "", fmt.Errorf("read response: %w", err)
    }

    if resp.StatusCode != http.StatusOK {
        return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
    }

    var result struct {
        Content []struct {
            Text string `json:"text"`
        } `json:"content"`
    }

    if err := json.Unmarshal(body, &result); err != nil {
        return "", fmt.Errorf("unmarshal response: %w", err)
    }

    if len(result.Content) == 0 {
        return "", fmt.Errorf("empty response from Claude")
    }

    return result.Content[0].Text, nil
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