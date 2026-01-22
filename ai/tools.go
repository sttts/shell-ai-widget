package ai

// ToolDefinition describes a tool that the AI can call
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ToolCall represents a request from the AI to execute a tool
type ToolCall struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolResult represents the result of executing a tool
type ToolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
}

// AvailableTools returns the list of tools available to the AI
func AvailableTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "web_search",
			Description: "Search the web for documentation, solutions, error explanations, or how to use commands. Use this when you need current information or don't know the answer.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "command_help",
			Description: "Run 'command --help' to get usage information for a shell command. Use this to learn about command flags and options.",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command": map[string]interface{}{
						"type":        "string",
						"description": "The command name (e.g., 'git', 'rsync', 'curl')",
					},
				},
				"required": []string{"command"},
			},
		},
	}
}
