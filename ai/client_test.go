package ai

import (
	"testing"

	"github.com/sttts/shell-ai-widget/config"
)

func TestNewClientCodexCLI(t *testing.T) {
	cfg := &config.Config{
		AI:       config.AIConfig{Provider: "codex-cli"},
		CodexCLI: config.CodexCLIConfig{Path: "/bin/sh"},
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, ok := client.(*CodexCLIClient); !ok {
		t.Fatalf("client type = %T, want *CodexCLIClient", client)
	}
}

func TestNewClientUnknownProvider(t *testing.T) {
	cfg := &config.Config{
		AI: config.AIConfig{Provider: "unknown"},
	}
	_, err := NewClient(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got, want := err.Error(), "unsupported AI provider: unknown"; got != want {
		t.Fatalf("error = %q, want %q", got, want)
	}
}
