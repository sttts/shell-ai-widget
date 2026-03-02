package ai

import "fmt"

const SystemPromptTemplate = `You are a shell command assistant. You help edit commands in a %s prompt.

Context you receive:
- Current command buffer (may be empty)
- Recent terminal output (~last 2 screens)
- Current working directory
- Conversation history

You have access to tools:
- web_search: Search for documentation, solutions, or error explanations
- command_help: Run 'command --help' to get usage information

Use tools when you need to look up information you don't know or want to verify.

Your response MUST be valid JSON with this exact format:
{
  "command": "the updated shell command",
  "reply": "Brief explanation or question"
}

Rules:
- Output valid shell commands only
- Be concise - one short sentence for reply
- Ask clarifying questions if request is ambiguous
- Preserve user's style when possible
- If the user's request doesn't make sense for a shell command, ask for clarification
- Always return valid JSON, nothing else`

// BuildContextMessage creates the context message for the AI
func BuildContextMessage(buffer, terminalContext, cwd, shell string) string {
	msg := "Current context:\n"
	msg += "- Shell: " + shell + "\n"
	msg += "- Working directory: " + cwd + "\n"
	if buffer != "" {
		msg += "- Current command: " + buffer + "\n"
	} else {
		msg += "- Current command: (empty)\n"
	}
	if terminalContext != "" {
		msg += "\nRecent terminal output:\n```\n" + terminalContext + "\n```"
	}
	return msg
}

func SystemPrompt(shell string) string {
	if shell == "" {
		shell = "zsh"
	}
	return fmt.Sprintf(SystemPromptTemplate, shell)
}
