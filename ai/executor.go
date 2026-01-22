package ai

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ToolExecutor executes tool calls
type ToolExecutor struct {
	EnableWebSearch   bool
	EnableCommandHelp bool
}

// NewToolExecutor creates a new tool executor with the given settings
func NewToolExecutor(enableWebSearch, enableCommandHelp bool) *ToolExecutor {
	return &ToolExecutor{
		EnableWebSearch:   enableWebSearch,
		EnableCommandHelp: enableCommandHelp,
	}
}

// blacklistedCommands are commands that should never be run with --help
var blacklistedCommands = map[string]bool{
	"rm":       true,
	"sudo":     true,
	"chmod":    true,
	"chown":    true,
	"dd":       true,
	"mkfs":     true,
	"su":       true,
	"kill":     true,
	"pkill":    true,
	"reboot":   true,
	"shutdown": true,
	"poweroff": true,
}

// validCommandName checks if a command name is safe to execute
// Only allows alphanumeric, dash, underscore, and dot
var validCommandName = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// Execute runs a tool call and returns the result
func (e *ToolExecutor) Execute(ctx context.Context, call ToolCall) ToolResult {
	switch call.Name {
	case "web_search":
		return e.executeWebSearch(ctx, call)
	case "command_help":
		return e.executeCommandHelp(ctx, call)
	default:
		return ToolResult{
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("Unknown tool: %s", call.Name),
			IsError:    true,
		}
	}
}

// executeWebSearch performs a web search
func (e *ToolExecutor) executeWebSearch(ctx context.Context, call ToolCall) ToolResult {
	if !e.EnableWebSearch {
		return ToolResult{
			ToolCallID: call.ID,
			Content:    "Web search is disabled",
			IsError:    true,
		}
	}

	query, ok := call.Arguments["query"].(string)
	if !ok || query == "" {
		return ToolResult{
			ToolCallID: call.ID,
			Content:    "Missing or invalid 'query' argument",
			IsError:    true,
		}
	}

	result, err := WebSearch(ctx, query)
	if err != nil {
		return ToolResult{
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("Search failed: %v", err),
			IsError:    true,
		}
	}

	return ToolResult{
		ToolCallID: call.ID,
		Content:    result,
		IsError:    false,
	}
}

// executeCommandHelp runs command --help
func (e *ToolExecutor) executeCommandHelp(ctx context.Context, call ToolCall) ToolResult {
	if !e.EnableCommandHelp {
		return ToolResult{
			ToolCallID: call.ID,
			Content:    "Command help is disabled",
			IsError:    true,
		}
	}

	command, ok := call.Arguments["command"].(string)
	if !ok || command == "" {
		return ToolResult{
			ToolCallID: call.ID,
			Content:    "Missing or invalid 'command' argument",
			IsError:    true,
		}
	}

	// Validate command name
	if !validCommandName.MatchString(command) {
		return ToolResult{
			ToolCallID: call.ID,
			Content:    "Invalid command name: contains disallowed characters",
			IsError:    true,
		}
	}

	// Check blacklist
	if blacklistedCommands[command] {
		return ToolResult{
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("Command '%s' is not allowed for security reasons", command),
			IsError:    true,
		}
	}

	// Create context with 10 second timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Run command --help
	cmd := exec.CommandContext(ctx, command, "--help")
	output, err := cmd.CombinedOutput()

	// Many commands return non-zero exit code for --help but still provide output
	result := string(output)

	// Truncate to 4KB
	if len(result) > 4096 {
		result = result[:4096] + "\n... (truncated)"
	}

	if result == "" && err != nil {
		return ToolResult{
			ToolCallID: call.ID,
			Content:    fmt.Sprintf("Failed to run '%s --help': %v", command, err),
			IsError:    true,
		}
	}

	return ToolResult{
		ToolCallID: call.ID,
		Content:    strings.TrimSpace(result),
		IsError:    false,
	}
}

// GetDisplayText returns a user-friendly display string for a tool call
func GetDisplayText(call ToolCall) string {
	switch call.Name {
	case "web_search":
		if query, ok := call.Arguments["query"].(string); ok {
			return fmt.Sprintf("Searching: %s", query)
		}
		return "Searching..."
	case "command_help":
		if cmd, ok := call.Arguments["command"].(string); ok {
			return fmt.Sprintf("Getting help: %s", cmd)
		}
		return "Getting help..."
	default:
		return fmt.Sprintf("Running: %s", call.Name)
	}
}

// GetDisplayIcon returns an icon for a tool call
func GetDisplayIcon(call ToolCall) string {
	switch call.Name {
	case "web_search":
		return "\U0001F310" // globe
	case "command_help":
		return "\U0001F4D6" // book
	default:
		return "\U0001F527" // wrench
	}
}
