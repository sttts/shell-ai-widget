# Zsh AI Widget - Product Requirements Document

## Overview

Zsh AI Widget is an inline AI-powered command editor for zsh. Triggered by a keyboard shortcut, it opens a chat interface directly above your shell prompt where you can ask an AI to help write, modify, or explain shell commands.

## Problem Statement

Writing complex shell commands often requires:
- Remembering obscure flags and syntax
- Looking up documentation or Stack Overflow
- Trial and error with command variations

This context-switching breaks flow and slows down terminal work.

## Solution

An AI assistant that:
- Appears inline in your terminal (no app switching)
- Sees your current command and recent terminal output
- Understands what you're trying to do
- Suggests or modifies commands in place

## User Experience

### Triggering the Widget

Press **Shift+Cmd+K** (configurable) to open the widget. It appears as a dark grey input area above your current prompt:

```
> █ Enter = Accept, ESC = Cancel
❯ git log
```

### Asking for Help

Type a request and press Enter:

```
> show last 5 commits with dates and authors
❯ git log
```

The AI processes your request (shimmer animation shows progress):

```
> show last 5 commits with dates and authors
⏺ Here's the command with date and author formatting.
❯ git log -5 --format="%h %ad %an - %s" --date=short
```

### Iterating

Continue the conversation to refine:

```
> show last 5 commits with dates and authors
⏺ Here's the command with date and author formatting.
> also show the branch
⏺ Added branch decoration.
> █
❯ git log -5 --format="%h %ad %an - %s" --date=short --decorate
```

### Accepting or Cancelling

- **Enter** (empty input): Accept the command and close
- **ESC**: Cancel and restore original command
- **Ctrl+C**: Same as ESC

### During AI Processing

- **ESC** or **Ctrl+C**: Cancel the request, return to editing your question
- The widget won't accidentally close for 200ms after a response arrives

## Key Features

### Context Awareness

The AI receives:
- **Current command buffer**: What you've already typed
- **Terminal context**: Last ~100 lines of terminal output
- **Working directory**: Your current path

This lets it understand commands you've run, error messages, and file listings.

### Inline Display

The widget draws directly in your terminal:
- No alternate screen (preserves scrollback)
- Dynamic height: 1-5 lines as conversation grows
- Scrolls internally when exceeding 5 lines
- Cleans up completely on close

### Visual Design

| Element | Appearance |
|---------|------------|
| User input | Dark grey background, `>` prompt |
| AI response | Standard background, light green `⏺` marker |
| Command buffer | Green bold `❯` prompt |
| Loading | Shimmer animation across user text |

## Configuration

Configuration file: `~/.config/shell-ai-widget/config.toml`

```toml
[ai]
provider = "openai"  # or "anthropic"

[openai]
api_key = ""  # uses OPENAI_API_KEY env var if empty
model = "gpt-4o-mini"

[anthropic]
api_key = ""  # uses ANTHROPIC_API_KEY env var if empty
model = "claude-3-5-sonnet-latest"

[terminal]
context_lines = 100  # lines of scrollback to send
```

## Requirements

### Functional Requirements

1. **FR-1**: Widget appears inline above shell prompt on trigger
2. **FR-2**: User can type natural language requests
3. **FR-3**: AI responds with command suggestions and explanations
4. **FR-4**: Command buffer updates in real-time with suggestions
5. **FR-5**: Conversation history maintained within session
6. **FR-6**: ESC cancels and restores original command
7. **FR-7**: Enter on empty input accepts current command
8. **FR-8**: ESC during AI processing cancels the request
9. **FR-9**: Terminal content preserved (no alternate screen)

### Non-Functional Requirements

1. **NFR-1**: Response latency < 3s for typical requests
2. **NFR-2**: Widget opens in < 100ms
3. **NFR-3**: No terminal corruption on exit
4. **NFR-4**: Works with standard terminal emulators (Ghostty, iTerm2, etc.)

## Technical Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Terminal Emulator (Ghostty)                                 │
│  Shift+Cmd+K → triggers zsh widget                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ Zsh Widget (~/.zshrc.d/ai-cmd-edit.zsh)                     │
│  - Captures current $BUFFER                                 │
│  - Launches Go binary with buffer + context                 │
│  - Sets $BUFFER to returned command                         │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│ Go Binary (shell-ai-widget)                                   │
│  - TUI with bubbletea + lipgloss                            │
│  - Manages chat state and rendering                         │
│  - Calls OpenAI/Anthropic APIs                              │
│  - Outputs final command to stdout                          │
└─────────────────────────────────────────────────────────────┘
```

## AI Behavior

The AI is instructed to:
- Output valid shell commands only
- Be concise (one short sentence replies)
- Ask clarifying questions when requests are ambiguous
- Preserve the user's command style when possible
- Consider terminal context when relevant

Response format (internal JSON):
```json
{
  "command": "the shell command",
  "reply": "Brief explanation"
}
```

## Supported Platforms

- macOS (primary)
- Linux (untested but should work)
- Requires: zsh, Go 1.21+

## Future Considerations

- Command history integration
- Multiple command suggestions
- Syntax highlighting in buffer
- Custom system prompts
- Local LLM support
