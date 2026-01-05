package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type AnthropicClient struct {
	apiKey string
	model  string
}

func NewAnthropicClient(apiKey, model string) (*AnthropicClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("Anthropic API key is required")
	}
	return &AnthropicClient{
		apiKey: apiKey,
		model:  model,
	}, nil
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *AnthropicClient) Chat(ctx context.Context, messages []Message, buffer, terminalContext, cwd string) (*Response, error) {
	// Build the messages array (Anthropic uses a different format)
	anthropicMessages := []anthropicMessage{
		{Role: "user", Content: BuildContextMessage(buffer, terminalContext, cwd)},
	}

	// Add conversation history
	for _, msg := range messages {
		anthropicMessages = append(anthropicMessages, anthropicMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: 1024,
		System:    SystemPrompt,
		Messages:  anthropicMessages,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if anthropicResp.Error != nil {
		return nil, fmt.Errorf("Anthropic API error: %s", anthropicResp.Error.Message)
	}

	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("no response from Anthropic")
	}

	// Get the text content
	var content string
	for _, c := range anthropicResp.Content {
		if c.Type == "text" {
			content = c.Text
			break
		}
	}

	content = strings.TrimSpace(content)

	// Remove markdown code blocks if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var response Response
	if err := json.Unmarshal([]byte(content), &response); err != nil {
		// If JSON parsing fails, treat the whole response as a reply
		return &Response{
			Command: buffer,
			Reply:   content,
		}, nil
	}

	return &response, nil
}
