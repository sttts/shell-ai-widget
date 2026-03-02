package ai

import (
	"context"
	"strings"

	"github.com/sttts/shell-ai-widget/config"
)

// extractJSON extracts a JSON object from text that may contain other content
func extractJSON(content string) string {
	content = strings.TrimSpace(content)

	// Try to find JSON in markdown code block first
	if idx := strings.Index(content, "```json"); idx != -1 {
		start := idx + 7
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}
	if idx := strings.Index(content, "```"); idx != -1 {
		start := idx + 3
		// Skip language identifier if present
		if nlIdx := strings.Index(content[start:], "\n"); nlIdx != -1 {
			start += nlIdx + 1
		}
		if end := strings.Index(content[start:], "```"); end != -1 {
			return strings.TrimSpace(content[start : start+end])
		}
	}

	// Try to find JSON object by looking for { and }
	if start := strings.Index(content, "{"); start != -1 {
		// Find matching closing brace
		depth := 0
		for i := start; i < len(content); i++ {
			if content[i] == '{' {
				depth++
			} else if content[i] == '}' {
				depth--
				if depth == 0 {
					return content[start : i+1]
				}
			}
		}
	}

	return content
}

// Message represents a chat message
type Message struct {
	Role       string      `json:"role"`
	Content    string      `json:"content"`
	ToolCalls  []ToolCall  `json:"tool_calls,omitempty"`  // AI's tool requests (for assistant messages)
	ToolResult *ToolResult `json:"tool_result,omitempty"` // Tool execution result (for tool messages)
}

// Response represents the AI's response
type Response struct {
	Command   string     `json:"command"`
	Reply     string     `json:"reply"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // Requested tool calls
	Done      bool       `json:"done"`                 // True when no more tools needed
}

// ToolsConfig holds tool-related configuration
type ToolsConfig struct {
	EnableWebSearch   bool
	EnableCommandHelp bool
}

// Client is the interface for AI providers
type Client interface {
	// Chat sends a message and returns the AI's response
	Chat(ctx context.Context, messages []Message, buffer, terminalContext, cwd string, toolsCfg ToolsConfig) (*Response, error)
}

// NewClient creates a new AI client based on config
func NewClient(cfg *config.Config) (Client, error) {
	switch cfg.AI.Provider {
	case "openai":
		return NewOpenAIClient(cfg.OpenAI.APIKey, cfg.OpenAI.Model)
	case "anthropic":
		return NewAnthropicClient(cfg.Anthropic.APIKey, cfg.Anthropic.Model)
	default:
		return NewOpenAIClient(cfg.OpenAI.APIKey, cfg.OpenAI.Model)
	}
}
