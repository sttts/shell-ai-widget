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
	Tools     []anthropicToolDef `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string        `json:"role"`
	Content []interface{} `json:"content"`
}

type anthropicTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicToolUseContent struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

type anthropicToolResultContent struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

type anthropicToolDef struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type anthropicResponse struct {
	Content []struct {
		Type  string                 `json:"type"`
		Text  string                 `json:"text,omitempty"`
		ID    string                 `json:"id,omitempty"`
		Name  string                 `json:"name,omitempty"`
		Input map[string]interface{} `json:"input,omitempty"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Error      *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *AnthropicClient) Chat(ctx context.Context, messages []Message, buffer, terminalContext, cwd string, toolsCfg ToolsConfig) (*Response, error) {
	// Build the messages array (Anthropic uses a different format)
	anthropicMessages := []anthropicMessage{
		{
			Role: "user",
			Content: []interface{}{
				anthropicTextContent{
					Type: "text",
					Text: BuildContextMessage(buffer, terminalContext, cwd),
				},
			},
		},
	}

	// Add conversation history
	for _, msg := range messages {
		var content []interface{}

		// Handle tool results (these go in user messages)
		if msg.ToolResult != nil {
			content = append(content, anthropicToolResultContent{
				Type:      "tool_result",
				ToolUseID: msg.ToolResult.ToolCallID,
				Content:   msg.ToolResult.Content,
				IsError:   msg.ToolResult.IsError,
			})
			anthropicMessages = append(anthropicMessages, anthropicMessage{
				Role:    "user",
				Content: content,
			})
			continue
		}

		// Handle assistant messages with tool calls
		if msg.Role == "assistant" && len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				content = append(content, anthropicToolUseContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: tc.Arguments,
				})
			}
			anthropicMessages = append(anthropicMessages, anthropicMessage{
				Role:    "assistant",
				Content: content,
			})
			continue
		}

		// Regular text message
		if msg.Content != "" {
			content = append(content, anthropicTextContent{
				Type: "text",
				Text: msg.Content,
			})
			anthropicMessages = append(anthropicMessages, anthropicMessage{
				Role:    msg.Role,
				Content: content,
			})
		}
	}

	// Build tools array if any tools are enabled
	var tools []anthropicToolDef
	if toolsCfg.EnableWebSearch || toolsCfg.EnableCommandHelp {
		for _, td := range AvailableTools() {
			if td.Name == "web_search" && !toolsCfg.EnableWebSearch {
				continue
			}
			if td.Name == "command_help" && !toolsCfg.EnableCommandHelp {
				continue
			}
			tools = append(tools, anthropicToolDef{
				Name:        td.Name,
				Description: td.Description,
				InputSchema: td.Parameters,
			})
		}
	}

	reqBody := anthropicRequest{
		Model:     c.model,
		MaxTokens: 1024,
		System:    SystemPrompt,
		Messages:  anthropicMessages,
		Tools:     tools,
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

	// Check for tool use in response
	var toolCalls []ToolCall
	var textContent string
	for _, c := range anthropicResp.Content {
		if c.Type == "tool_use" {
			toolCalls = append(toolCalls, ToolCall{
				ID:        c.ID,
				Name:      c.Name,
				Arguments: c.Input,
			})
		} else if c.Type == "text" {
			textContent = c.Text
		}
	}

	// If there are tool calls, return them
	if len(toolCalls) > 0 {
		return &Response{
			ToolCalls: toolCalls,
			Done:      false,
		}, nil
	}

	// Parse the JSON response from the AI
	content := strings.TrimSpace(textContent)
	jsonContent := extractJSON(content)

	var response Response
	if err := json.Unmarshal([]byte(jsonContent), &response); err != nil {
		// If JSON parsing fails, treat the whole response as a reply
		return &Response{
			Command: buffer,
			Reply:   content,
			Done:    true,
		}, nil
	}

	response.Done = true
	return &response, nil
}
