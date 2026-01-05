package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	AI        AIConfig        `toml:"ai"`
	OpenAI    OpenAIConfig    `toml:"openai"`
	Anthropic AnthropicConfig `toml:"anthropic"`
	UI        UIConfig        `toml:"ui"`
}

type AIConfig struct {
	Provider string `toml:"provider"`
}

type OpenAIConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

type AnthropicConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

type UIConfig struct {
	ContextLines int `toml:"context_lines"`
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		AI: AIConfig{
			Provider: "openai",
		},
		OpenAI: OpenAIConfig{
			Model: "gpt-4o-mini",
		},
		Anthropic: AnthropicConfig{
			Model: "claude-3-5-haiku-latest",
		},
		UI: UIConfig{
			ContextLines: 100,
		},
	}
}

// Load reads the config file and returns a Config struct
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to find config file
	configPath := getConfigPath()
	if configPath == "" {
		// No config file found, use defaults with env vars
		cfg.OpenAI.APIKey = os.Getenv("OPENAI_API_KEY")
		cfg.Anthropic.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		return cfg, nil
	}

	// Parse config file
	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		return nil, err
	}

	// Override with env vars if config values are empty
	if cfg.OpenAI.APIKey == "" {
		cfg.OpenAI.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if cfg.Anthropic.APIKey == "" {
		cfg.Anthropic.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	return cfg, nil
}

func getConfigPath() string {
	// Check XDG config dir first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		path := filepath.Join(xdgConfig, "zsh-ai-widget", "config.toml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	path := filepath.Join(home, ".config", "zsh-ai-widget", "config.toml")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}
