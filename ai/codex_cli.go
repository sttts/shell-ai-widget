package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

type CodexCLIClient struct {
	path string
	args []string
}

func NewCodexCLIClient(path string, args []string) (*CodexCLIClient, error) {
	if path == "" {
		return nil, fmt.Errorf("codex CLI path is required")
	}
	if len(args) == 0 {
		args = []string{"exec", "--json"}
	}
	return &CodexCLIClient{
		path: path,
		args: args,
	}, nil
}

func (c *CodexCLIClient) Chat(ctx context.Context, messages []Message, buffer, terminalContext, cwd, shell string, _ ToolsConfig) (*Response, error) {
	prompt := c.buildPrompt(messages, buffer, terminalContext, cwd, shell)

	cmdArgs := append([]string{}, c.args...)
	cmdArgs = append(cmdArgs, prompt)

	cmd := exec.CommandContext(ctx, c.path, cmdArgs...)
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		var exitErr *exec.ExitError
		if ok := errors.As(err, &exitErr); ok {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		if stderr != "" {
			if len(stderr) > 400 {
				stderr = stderr[:400]
			}
			return nil, fmt.Errorf("codex CLI request failed: %w (stderr: %s)", err, stderr)
		}
		return nil, fmt.Errorf("codex CLI request failed: %w", err)
	}

	content := strings.TrimSpace(string(out))
	if content == "" {
		return &Response{
			Command: buffer,
			Reply:   "No response from Codex CLI.",
		}, nil
	}

	if resp, ok := parseResponseJSON(content); ok {
		if resp.Command == "" {
			resp.Command = buffer
		}
		return resp, nil
	}

	return &Response{
		Command: buffer,
		Reply:   content,
	}, nil
}

func (c *CodexCLIClient) buildPrompt(messages []Message, buffer, terminalContext, cwd, shell string) string {
	var b strings.Builder
	b.WriteString(SystemPrompt(shell))
	b.WriteString("\n\n")
	b.WriteString(BuildContextMessage(buffer, terminalContext, cwd, shell))
	for _, msg := range messages {
		b.WriteString("\n\n")
		b.WriteString(msg.Role)
		b.WriteString(": ")
		b.WriteString(msg.Content)
	}
	return b.String()
}

func parseResponseJSON(content string) (*Response, bool) {
	if resp, ok := parseCodexEventStream(content); ok {
		return resp, true
	}

	// Some codex CLI modes emit multiple JSON lines/events. Prefer the last
	// non-empty event-like object before falling back to a single extracted JSON blob.
	if strings.Contains(content, "\n") {
		lines := strings.Split(content, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line == "" || !strings.HasPrefix(line, "{") {
				continue
			}
			if resp, ok := parseSingleJSON(line); ok {
				return resp, true
			}
		}
	}

	return parseSingleJSON(extractJSON(content))
}

func parseCodexEventStream(content string) (*Response, bool) {
	type codexEvent struct {
		Type string `json:"type"`
		Item struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"item"`
	}

	lines := strings.Split(content, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}

		var ev codexEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		if ev.Type == "item.completed" && ev.Item.Type == "agent_message" && strings.TrimSpace(ev.Item.Text) != "" {
			if resp, ok := parseSingleJSON(ev.Item.Text); ok {
				return resp, true
			}
			return &Response{Reply: strings.TrimSpace(ev.Item.Text)}, true
		}
	}

	return nil, false
}

func parseSingleJSON(jsonContent string) (*Response, bool) {
	var response Response
	if err := json.Unmarshal([]byte(jsonContent), &response); err == nil {
		if strings.TrimSpace(response.Command) != "" || strings.TrimSpace(response.Reply) != "" {
			return &response, true
		}
	}

	var wrapped struct {
		Command string `json:"command"`
		Reply   string `json:"reply"`
		Output  string `json:"output"`
		Text    string `json:"text"`
		Content string `json:"content"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(jsonContent), &wrapped); err == nil {
		reply := wrapped.Reply
		if reply == "" {
			if wrapped.Output != "" {
				reply = wrapped.Output
			} else if wrapped.Content != "" {
				reply = wrapped.Content
			} else if wrapped.Message != "" {
				reply = wrapped.Message
			} else {
				reply = wrapped.Text
			}
		}
		if strings.TrimSpace(wrapped.Command) != "" || strings.TrimSpace(reply) != "" {
			return &Response{Command: wrapped.Command, Reply: reply}, true
		}
	}

	return nil, false
}
