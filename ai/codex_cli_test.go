package ai

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCodexCLIChatJSONResponse(t *testing.T) {
	path := writeExecScript(t, "#!/bin/sh\necho '{\"command\":\"ls -la\",\"reply\":\"updated\"}'\n")
	client, err := NewCodexCLIClient(path, []string{})
	if err != nil {
		t.Fatalf("NewCodexCLIClient: %v", err)
	}

	resp, err := client.Chat(context.Background(), nil, "ls", "", "/tmp", "fish", ToolsConfig{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Command != "ls -la" || resp.Reply != "updated" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}

func TestCodexCLIChatPlainTextFallback(t *testing.T) {
	path := writeExecScript(t, "#!/bin/sh\necho 'try: ls -la'\n")
	client, err := NewCodexCLIClient(path, []string{})
	if err != nil {
		t.Fatalf("NewCodexCLIClient: %v", err)
	}

	resp, err := client.Chat(context.Background(), nil, "pwd", "", "/tmp", "zsh", ToolsConfig{})
	if err != nil {
		t.Fatalf("Chat: %v", err)
	}
	if resp.Command != "pwd" {
		t.Fatalf("command = %q, want original buffer", resp.Command)
	}
	if resp.Reply != "try: ls -la" {
		t.Fatalf("reply = %q", resp.Reply)
	}
}

func TestCodexCLIChatFailureIncludesStderr(t *testing.T) {
	path := writeExecScript(t, "#!/bin/sh\necho 'boom error' 1>&2\nexit 42\n")
	client, err := NewCodexCLIClient(path, []string{})
	if err != nil {
		t.Fatalf("NewCodexCLIClient: %v", err)
	}

	_, err = client.Chat(context.Background(), nil, "pwd", "", "/tmp", "zsh", ToolsConfig{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "stderr: boom error") {
		t.Fatalf("error does not contain stderr: %v", err)
	}
}

func TestCodexCLIChatContextCancellation(t *testing.T) {
	path := writeExecScript(t, "#!/bin/sh\nsleep 5\n")
	client, err := NewCodexCLIClient(path, []string{})
	if err != nil {
		t.Fatalf("NewCodexCLIClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = client.Chat(ctx, nil, "pwd", "", "/tmp", "zsh", ToolsConfig{})
	if err == nil {
		t.Fatal("expected cancellation error, got nil")
	}
}

func TestParseResponseJSONCodexEventStreamAgentMessage(t *testing.T) {
	stream := strings.Join([]string{
		`{"type":"thread.started","thread_id":"x"}`,
		`{"type":"turn.started"}`,
		`{"type":"item.completed","item":{"id":"item_0","type":"reasoning","text":"thinking"}}`,
		`{"type":"item.completed","item":{"id":"item_1","type":"agent_message","text":"{\"command\":\"kubectl get pods -A -o custom-columns=NS:.metadata.namespace,NAME:.metadata.name --no-headers | awk '{print $1 \\\"/\\\" $2}'\",\"reply\":\"Lists all pods as ns/name.\"}"}}`,
		`{"type":"turn.completed","usage":{"input_tokens":1,"output_tokens":1}}`,
	}, "\n")

	resp, ok := parseResponseJSON(stream)
	if !ok {
		t.Fatal("expected parse success")
	}
	if resp.Command == "" {
		t.Fatalf("expected non-empty command, got %#v", resp)
	}
	if !strings.Contains(resp.Command, "kubectl get pods -A") {
		t.Fatalf("unexpected command: %q", resp.Command)
	}
}

func writeExecScript(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "cmd.sh")
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	return path
}
