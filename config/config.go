package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	AI         AIConfig         `toml:"ai"`
	OpenAI     OpenAIConfig     `toml:"openai"`
	OpenRouter OpenRouterConfig `toml:"openrouter"`
	Anthropic  AnthropicConfig  `toml:"anthropic"`
	CodexCLI   CodexCLIConfig   `toml:"codex_cli"`
	UI         UIConfig         `toml:"ui"`
	Tools      ToolsConfig      `toml:"tools"`
}

type AIConfig struct {
	Provider string `toml:"provider"`
}

type OpenAIConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

type OpenRouterConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

type AnthropicConfig struct {
	APIKey string `toml:"api_key"`
	Model  string `toml:"model"`
}

type CodexCLIConfig struct {
	Path string   `toml:"path"`
	Args []string `toml:"args"`
}

type UIConfig struct {
	ContextLines int `toml:"context_lines"`
}

type ToolsConfig struct {
	EnableWebSearch   bool `toml:"enable_web_search"`
	EnableCommandHelp bool `toml:"enable_command_help"`
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
		OpenRouter: OpenRouterConfig{
			Model: "openai/gpt-4o-mini",
		},
		Anthropic: AnthropicConfig{
			Model: "claude-3-5-haiku-latest",
		},
		CodexCLI: CodexCLIConfig{
			Args: []string{},
		},
		UI: UIConfig{
			ContextLines: 100,
		},
		Tools: ToolsConfig{
			EnableWebSearch:   true,
			EnableCommandHelp: true,
		},
	}
}

var lookPath = exec.LookPath

// Load reads the config file and returns a Config struct
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to find config file
	configPath := getConfigPath()
	if configPath == "" {
		// No config file found, use defaults with env vars
		cfg.OpenAI.APIKey = os.Getenv("OPENAI_API_KEY")
		cfg.OpenRouter.APIKey = os.Getenv("OPENROUTER_API_KEY")
		cfg.Anthropic.APIKey = os.Getenv("ANTHROPIC_API_KEY")
		return cfg, validate(cfg)
	}

	// Parse config file
	if _, err := toml.DecodeFile(configPath, cfg); err != nil {
		return nil, err
	}

	// Override with env vars if config values are empty
	if cfg.OpenAI.APIKey == "" {
		cfg.OpenAI.APIKey = os.Getenv("OPENAI_API_KEY")
	}
	if cfg.OpenRouter.APIKey == "" {
		cfg.OpenRouter.APIKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if cfg.Anthropic.APIKey == "" {
		cfg.Anthropic.APIKey = os.Getenv("ANTHROPIC_API_KEY")
	}

	return cfg, validate(cfg)
}

func validate(cfg *Config) error {
	if cfg.AI.Provider == "codex-cli" {
		return validateCodexCLIConfig(&cfg.CodexCLI)
	}
	return nil
}

func validateCodexCLIConfig(cfg *CodexCLIConfig) error {
	if cfg.Path != "" {
		return nil
	}

	path, err := lookPath("codex")
	if err != nil {
		return fmt.Errorf("codex-cli provider selected but codex binary not found in PATH: %w", err)
	}
	cfg.Path = path
	return nil
}

func getConfigPath() string {
	// Check XDG config dir first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		path := filepath.Join(xdgConfig, "shell-ai-widget", "config.toml")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		return ""
	}

	// Fall back to ~/.config
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	path := filepath.Join(home, ".config", "shell-ai-widget", "config.toml")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}
