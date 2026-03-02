package ai

import (
	"strings"
	"testing"
)

func TestSystemPromptIncludesShell(t *testing.T) {
	prompt := SystemPrompt("fish")
	if !strings.Contains(prompt, "fish prompt") {
		t.Fatalf("prompt does not include fish shell: %q", prompt)
	}
}

func TestBuildContextMessageIncludesShell(t *testing.T) {
	msg := BuildContextMessage("ls", "ok", "/tmp", "fish")
	if !strings.Contains(msg, "- Shell: fish") {
		t.Fatalf("context message does not include fish shell: %q", msg)
	}
}
