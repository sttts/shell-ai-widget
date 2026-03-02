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

type OpenRouterClient struct {
	apiKey string
	model  string
}

func NewOpenRouterClient(apiKey, model string) (*OpenRouterClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("OpenRouter API key is required")
	}
	return &OpenRouterClient{
		apiKey: apiKey,
		model:  model,
	}, nil
}

func (c *OpenRouterClient) Chat(ctx context.Context, messages []Message, buffer, terminalContext, cwd, shell string, toolsCfg ToolsConfig) (*Response, error) {
	openAIMessages := []openAIMessage{
		{Role: "system", Content: SystemPrompt(shell)},
		{Role: "user", Content: BuildContextMessage(buffer, terminalContext, cwd, shell)},
	}

	for _, msg := range messages {
		oaiMsg := openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}

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

		if msg.ToolResult != nil {
			oaiMsg.Role = "tool"
			oaiMsg.ToolCallID = msg.ToolResult.ToolCallID
			oaiMsg.Content = msg.ToolResult.Content
		}

		openAIMessages = append(openAIMessages, oaiMsg)
	}

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

	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("HTTP-Referer", "https://github.com/sttts/shell-ai-widget")
	req.Header.Set("X-Title", "shell-ai-widget")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var openRouterResp openAIResponse
	if err := json.Unmarshal(body, &openRouterResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if openRouterResp.Error != nil {
		return nil, fmt.Errorf("OpenRouter API error: %s", openRouterResp.Error.Message)
	}

	if len(openRouterResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenRouter")
	}

	choice := openRouterResp.Choices[0]

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

	content := strings.TrimSpace(choice.Message.Content)
	jsonContent := extractJSON(content)

	var response Response
	if err := json.Unmarshal([]byte(jsonContent), &response); err != nil {
		return &Response{
			Command: buffer,
			Reply:   content,
			Done:    true,
		}, nil
	}

	response.Done = true
	return &response, nil
}
