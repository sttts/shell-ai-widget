package ai

import (
	"context"

	"github.com/sttts/shell-ai-widget/config"
)

// Message represents a chat message
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Response represents the AI's response
type Response struct {
	Command string `json:"command"`
	Reply   string `json:"reply"`
}

// Client is the interface for AI providers
type Client interface {
	// Chat sends a message and returns the AI's response
	Chat(ctx context.Context, messages []Message, buffer, terminalContext, cwd string) (*Response, error)
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
