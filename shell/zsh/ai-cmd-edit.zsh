# AI Command Editor Widget for Zsh
# Triggered by Shift+Cmd+K via Ghostty escape sequence

_ai_cmd_edit_widget() {
    local original_buffer="$BUFFER"

    # Call the AI widget binary
    local result
    result="$(~/.bin/shell-ai-widget --buffer="$BUFFER" --shell=zsh 2>/dev/null)"
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
