# AI Command Editor Widget for Fish
# Triggered by Shift+Cmd+K via Ghostty escape sequence

function _ai_cmd_edit
    set -l original_buffer (commandline)

    # Call the AI widget binary
    set -l result (command shell-ai-widget --buffer="$original_buffer" --shell=fish 2>/dev/null)
    set -l exit_code $status

    if test $exit_code -eq 0
        # Accepted - use the new buffer
        commandline -r -- $result
        commandline -f repaint
    else
        # Cancelled - restore original buffer
        commandline -r -- $original_buffer
        commandline -f repaint
    end
end

# Bind to ESC k (sent by Ghostty on Shift+Cmd+K)
bind \ek _ai_cmd_edit
