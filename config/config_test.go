package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCodexCLIConfigFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	cfgDir := filepath.Join(tmpDir, "shell-ai-widget")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	configText := `
[ai]
provider = "codex-cli"

[codex_cli]
path = "/usr/local/bin/codex"
args = ["exec", "--json"]
`
	if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"), []byte(configText), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.AI.Provider != "codex-cli" {
		t.Fatalf("provider = %q, want codex-cli", cfg.AI.Provider)
	}
	if cfg.CodexCLI.Path != "/usr/local/bin/codex" {
		t.Fatalf("codex path = %q", cfg.CodexCLI.Path)
	}
	if len(cfg.CodexCLI.Args) != 2 || cfg.CodexCLI.Args[0] != "exec" || cfg.CodexCLI.Args[1] != "--json" {
		t.Fatalf("unexpected codex args: %#v", cfg.CodexCLI.Args)
	}
}

func TestValidateCodexCLIConfigAutoDetectsPath(t *testing.T) {
	origLookPath := lookPath
	t.Cleanup(func() { lookPath = origLookPath })

	lookPath = func(file string) (string, error) {
		if file != "codex" {
			t.Fatalf("lookPath file = %q, want codex", file)
		}
		return "/opt/bin/codex", nil
	}

	cfg := &Config{
		AI: AIConfig{Provider: "codex-cli"},
	}
	if err := validate(cfg); err != nil {
		t.Fatalf("validate: %v", err)
	}
	if cfg.CodexCLI.Path != "/opt/bin/codex" {
		t.Fatalf("path = %q, want /opt/bin/codex", cfg.CodexCLI.Path)
	}
}

func TestValidateCodexCLIConfigFailsWhenMissing(t *testing.T) {
	origLookPath := lookPath
	t.Cleanup(func() { lookPath = origLookPath })

	lookPath = func(file string) (string, error) {
		return "", errors.New("not found")
	}

	cfg := &Config{
		AI: AIConfig{Provider: "codex-cli"},
	}
	err := validate(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoadOpenRouterAPIKeyFromEnv(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("OPENROUTER_API_KEY", "or-key")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.OpenRouter.APIKey != "or-key" {
		t.Fatalf("openrouter api key = %q, want or-key", cfg.OpenRouter.APIKey)
	}
}
