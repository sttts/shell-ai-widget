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

type OpenAIClient struct {
	apiKey string
	model  string
}

func NewOpenAIClient(apiKey, model string) (*OpenAIClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	return &OpenAIClient{
		apiKey: apiKey,
		model:  model,
	}, nil
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	Tools       []openAITool    `json:"tools,omitempty"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content   string           `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (c *OpenAIClient) Chat(ctx context.Context, messages []Message, buffer, terminalContext, cwd, shell string, toolsCfg ToolsConfig) (*Response, error) {
	// Build the messages array
	openAIMessages := []openAIMessage{
		{Role: "system", Content: SystemPrompt(shell)},
		{Role: "user", Content: BuildContextMessage(buffer, terminalContext, cwd, shell)},
	}

	// Add conversation history
	for _, msg := range messages {
		oaiMsg := openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

		// Handle tool calls in assistant messages
		if len(msg.ToolCalls) > 0 {
			oaiMsg.ToolCalls = make([]openAIToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				args, _ := json.Marshal(tc.Arguments)
				oaiMsg.ToolCalls[i] = openAIToolCall{
					ID:   tc.ID,
					Type: "function",
				}
				oaiMsg.ToolCalls[i].Function.Name = tc.Name
				oaiMsg.ToolCalls[i].Function.Arguments = string(args)
			}
		}

		// Handle tool results
		if msg.ToolResult != nil {
			oaiMsg.Role = "tool"
			oaiMsg.ToolCallID = msg.ToolResult.ToolCallID
			oaiMsg.Content = msg.ToolResult.Content
		}

		openAIMessages = append(openAIMessages, oaiMsg)
	}

	// Build tools array if any tools are enabled
	var tools []openAITool
	if toolsCfg.EnableWebSearch || toolsCfg.EnableCommandHelp {
		for _, td := range AvailableTools() {
			if td.Name == "web_search" && !toolsCfg.EnableWebSearch {
				continue
			}
			if td.Name == "command_help" && !toolsCfg.EnableCommandHelp {
				continue
			}
			tools = append(tools, openAITool{
				Type: "function",
				Function: openAIFunction{
					Name:        td.Name,
					Description: td.Description,
					Parameters:  td.Parameters,
				},
			})
		}
	}

	reqBody := openAIRequest{
		Model:       c.model,
		Messages:    openAIMessages,
		Temperature: 0.3,
		Tools:       tools,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if openAIResp.Error != nil {
		return nil, fmt.Errorf("OpenAI API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	choice := openAIResp.Choices[0]

	// Check if the model wants to call tools
	if len(choice.Message.ToolCalls) > 0 {
		var toolCalls []ToolCall
		for _, tc := range choice.Message.ToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
				args = map[string]interface{}{}
			}
			toolCalls = append(toolCalls, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: args,
			})
		}
		return &Response{
			ToolCalls: toolCalls,
			Done:      false,
		}, nil
	}

	// Parse the JSON response from the AI
	content := strings.TrimSpace(choice.Message.Content)
	jsonContent := extractJSON(content)

	var response Response
	if err := json.Unmarshal([]byte(jsonContent), &response); err != nil {
		// If JSON parsing fails, treat the whole response as a reply
		return &Response{
			Command: buffer, // Keep the original buffer
			Reply:   content,
			Done:    true,
		}, nil
	}

	response.Done = true
	return &response, nil
}
