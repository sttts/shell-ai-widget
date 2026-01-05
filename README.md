# zsh-ai-widget

An AI-powered command line editor for zsh. Press a hotkey to open an inline chat interface that helps you write, edit, and understand shell commands.

![Demo](demo.gif)

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
- zsh shell
- OpenAI API key (or Anthropic API key)

## Installation

### 1. Build the binary

```bash
git clone https://github.com/sttts/zsh-ai-widget.git
cd zsh-ai-widget
go build -o ~/.bin/zsh-ai-widget .
```

Make sure `~/.bin` is in your `$PATH`, or install to a different location.

### 2. Create the config file

Create `~/.config/zsh-ai-widget/config.toml`:

```toml
[ai]
provider = "openai"  # or "anthropic"

[openai]
api_key = ""  # Leave empty to use OPENAI_API_KEY env var
model = "gpt-4o-mini"

[anthropic]
api_key = ""  # Leave empty to use ANTHROPIC_API_KEY env var
model = "claude-sonnet-4-20250514"

[ui]
context_lines = 100
```

Set your API key either in the config or as an environment variable:

```bash
export OPENAI_API_KEY="sk-..."
# or
export ANTHROPIC_API_KEY="sk-ant-..."
```

### 3. Add the zsh widget

Create `~/.zshrc.d/ai-cmd-edit.zsh` (or add to your `.zshrc`):

```zsh
# AI Command Editor Widget
# Triggered by Shift+Cmd+K via Ghostty escape sequence

_ai_cmd_edit_widget() {
    local original_buffer="$BUFFER"

    # Call the AI widget binary
    local result
    result="$(~/.bin/zsh-ai-widget --buffer="$BUFFER" 2>/dev/null)"
    local exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        # Accepted - use the new buffer
        BUFFER="$result"
        CURSOR=${#BUFFER}
    else
        # Cancelled - restore original buffer
        BUFFER="$original_buffer"
        CURSOR=${#BUFFER}
    fi

    zle reset-prompt
}

# Register the widget
zle -N _ai_cmd_edit_widget

# Bind to ESC k (sent by Ghostty on Shift+Cmd+K)
bindkey '\ek' _ai_cmd_edit_widget
```

If using a `.zshrc.d` directory, make sure it's sourced in your `.zshrc`:

```zsh
for file in ~/.zshrc.d/*.zsh; do
    source "$file"
done
```

### 4. Configure Ghostty keybind

Add to your Ghostty config (`~/.config/ghostty/config` or on macOS `~/Library/Application Support/com.mitchellh.ghostty/config`):

```
# AI Command Editor - Shift+Cmd+K to open/close
keybind = cmd+shift+k=text:\x1bk
```

This sends the escape sequence `ESC k` when you press Shift+Cmd+K, which triggers the zsh widget.

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
provider = "openai"  # "openai" or "anthropic"
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
result="$(~/.bin/zsh-ai-widget --buffer="$BUFFER" --context-file=/tmp/scrollback.txt 2>/dev/null)"
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
│ Go Binary (zsh-ai-widget)                                   │
│  - Renders TUI with bubbletea                               │
│  - Manages chat with AI provider                            │
│  - Outputs final command to stdout                          │
└─────────────────────────────────────────────────────────────┘
```

## Building from Source

```bash
git clone https://github.com/sttts/zsh-ai-widget.git
cd zsh-ai-widget
go build -o zsh-ai-widget .
```

### Dependencies

- [bubbletea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Styling
- [go-openai](https://github.com/sashabaranov/go-openai) - OpenAI client

## License

MIT
