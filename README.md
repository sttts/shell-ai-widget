# shell-ai-widget

[![CI](https://github.com/sttts/shell-ai-widget/actions/workflows/ci.yml/badge.svg?branch=master)](https://github.com/sttts/shell-ai-widget/actions/workflows/ci.yml)

AI-powered inline command editor for zsh/fish triggered by Shift+Cmd+K, with terminal context awareness.

<img src="https://github.com/user-attachments/assets/29de76d5-fd2b-4c80-94b1-436754e87f97" width="75%">

## Features

- **Inline TUI**: Opens above your prompt without clearing the screen
- **Chat interface**: Iteratively refine commands through conversation
- **Context-aware**: Sees your current command buffer and working directory
- **Live preview**: Command updates in real-time as you chat
- **Shimmer animation**: Elegant loading indicator while waiting for AI
- **Clean restore**: Terminal returns to original state on close

## Requirements

- [Go](https://golang.org/) 1.21+ (for building)
- [Ghostty](https://ghostty.org/) terminal (for keybind integration)
- zsh or fish shell
- One AI backend: OpenAI API key, Anthropic API key, or local [Codex CLI](https://github.com/openai/codex) installation/authentication

## Installation

### 1. Build the binary

```bash
git clone https://github.com/sttts/shell-ai-widget.git
cd shell-ai-widget
go build -o ~/.bin/shell-ai-widget .
```

Make sure `~/.bin` is in your `$PATH`, or install to a different location.

### 2. Create the config file

Create `~/.config/shell-ai-widget/config.toml`:

```toml
[ai]
provider = "openai"  # or "anthropic" or "codex-cli"

[openai]
api_key = ""  # Leave empty to use OPENAI_API_KEY env var
model = "gpt-4o-mini"

[anthropic]
api_key = ""  # Leave empty to use ANTHROPIC_API_KEY env var
model = "claude-sonnet-4-20250514"

[codex_cli]
path = ""  # Leave empty to auto-detect "codex" in PATH
args = ["exec", "--json"]  # optional; default is ["exec","--json"]

[ui]
context_lines = 100
```

Set your API key (OpenAI/Anthropic only) either in the config or as an environment variable:

```bash
export OPENAI_API_KEY="sk-..."
# or
export ANTHROPIC_API_KEY="sk-ant-..."
```

### 3. Add the shell widget

#### Zsh

Copy or source the widget in your `.zshrc`:

```zsh
source /path/to/shell-ai-widget/shell/zsh/ai-cmd-edit.zsh
```

Or copy it to `~/.zshrc.d/ai-cmd-edit.zsh` if using a `.zshrc.d` directory.

#### Fish

Copy or source the widget in your fish config:

```fish
source /path/to/shell-ai-widget/shell/fish/ai-cmd-edit.fish
```

Or copy it to `~/.config/fish/conf.d/ai-cmd-edit.fish`.

### 4. Configure Ghostty keybind

Add to your Ghostty config (`~/.config/ghostty/config` or on macOS `~/Library/Application Support/com.mitchellh.ghostty/config`):

```
# AI Command Editor - Shift+Cmd+K to open/close
keybind = cmd+shift+k=text:\x1bk
```

This sends the escape sequence `ESC k` when you press Shift+Cmd+K, which triggers the shell widget.

### 5. Reload configuration

```bash
# Reload zsh config
source ~/.zshrc

# Restart Ghostty for keybind changes
```

## Usage

1. **Open**: Press `Shift+Cmd+K` (or your configured hotkey)
2. **Type**: Enter your request (e.g., "list files sorted by size")
3. **Chat**: Continue the conversation to refine the command
4. **Accept**: Press `Enter` on empty input, or `Shift+Cmd+K` again
5. **Cancel**: Press `ESC` or `Ctrl+C` to restore original command

### UI Elements

```
> your message here                    <- Your input (grey background)
⏺ AI response explaining the command  <- AI response (light green marker)
❯ ls -lahS                             <- Current command preview
```

### Keybindings

| Key | Action |
|-----|--------|
| `Enter` | Send message to AI, or accept if input empty |
| `Shift+Cmd+K` | Accept current command and close |
| `ESC` | Cancel and restore original command |
| `Ctrl+C` | Cancel and restore original command |

## Configuration

### AI Provider

```toml
[ai]
provider = "openai"  # "openai", "anthropic", or "codex-cli"
```

### OpenAI Settings

```toml
[openai]
api_key = ""  # or use OPENAI_API_KEY env var
model = "gpt-4o-mini"  # or "gpt-4o", "gpt-4-turbo", etc.
```

### Anthropic Settings

```toml
[anthropic]
api_key = ""  # or use ANTHROPIC_API_KEY env var
model = "claude-sonnet-4-20250514"  # or "claude-opus-4-20250514", etc.
```

### Codex CLI Settings

```toml
[ai]
provider = "codex-cli"

[codex_cli]
path = ""  # optional, auto-detected from PATH when empty
args = ["exec", "--json"]  # optional
```

With `provider = "codex-cli"`, no `OPENAI_API_KEY` is required.

### UI Settings

```toml
[ui]
context_lines = 100  # Lines of terminal context to send to AI
```

## Terminal Context (Optional)

For even better suggestions, you can pass recent terminal output to the AI. Ghostty supports `write_scrollback_file` for this:

```
# In Ghostty config (advanced setup)
keybind = cmd+shift+k=write_scrollback_file:/tmp/scrollback.txt,text:\x1bk
```

Then modify the zsh widget to pass the context:

```zsh
result="$(~/.bin/shell-ai-widget --buffer="$BUFFER" --context-file=/tmp/scrollback.txt --shell=zsh 2>/dev/null)"
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Ghostty                                                     │
│  Shift+Cmd+K → text:\x1bk (sends ESC k to terminal)        │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ Zsh Widget                                                  │
│  - Receives ESC k via bindkey                               │
│  - Passes $BUFFER to Go binary                              │
│  - Sets $BUFFER from stdout on success                      │
└─────────────────────────────────────────────────────────────┘
                              ↓
┌─────────────────────────────────────────────────────────────┐
│ Go Binary (shell-ai-widget)                                   │
│  - Renders TUI with bubbletea                               │
│  - Manages chat with AI provider                            │
│  - Outputs final command to stdout                          │
└─────────────────────────────────────────────────────────────┘
```

## Building from Source

```bash
git clone https://github.com/sttts/shell-ai-widget.git
cd shell-ai-widget
go build -o shell-ai-widget .
```

### Dependencies

- [bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [go-openai](https://github.com/sashabaranov/go-openai) - OpenAI client

## Disclaimer

This project was vibe coded with [Claude](https://claude.ai).

## License

Apache 2.0
