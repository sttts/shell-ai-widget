package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const widgetHeight = 5

var (
	// Dark grey background style for entire widget
	bgStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("236"))

	// User input style: dim grey with > prompt
	userPromptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Background(lipgloss.Color("236"))

	// AI response style: light grey, 2-space indent
	aiResponseStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("236"))

	// Input prompt style
	inputPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Background(lipgloss.Color("236"))

	// Error style
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Background(lipgloss.Color("236"))
)

func (m Model) View() string {
	// Get terminal width from /dev/tty
	width := 80
	if tty, err := os.Open("/dev/tty"); err == nil {
		if w, _, err := term.GetSize(int(tty.Fd())); err == nil && w > 0 {
			width = w
		}
		tty.Close()
	}

	// Collect all content lines
	var lines []string

	// Render chat history
	for i, msg := range m.ChatHistory {
		if msg.Role == "user" {
			// Show shimmer effect on last user message if loading (and no tool calls yet)
			if m.Loading && i == len(m.ChatHistory)-1 && m.ExecutingTool == nil {
				lines = append(lines, m.Shimmer.View())
			} else {
				lines = append(lines, "> "+msg.Content)
			}
		} else if msg.Role == "assistant" && msg.Content != "" {
			// Light green ⏺ for bot response: 38;5;119 = light green
			lines = append(lines, "\033[38;5;119m⏺\033[0m "+msg.Content)
		}
		// Skip tool calls and tool results in display (they're internal)
	}

	// Show shimmer for tool execution or continued thinking
	if m.Loading && (m.ExecutingTool != nil || len(m.ChatHistory) > 0) {
		// Check if we're past the initial user message phase
		lastIsUser := len(m.ChatHistory) > 0 && m.ChatHistory[len(m.ChatHistory)-1].Role == "user"
		if !lastIsUser {
			lines = append(lines, m.Shimmer.View())
		}
	}

	// Show error if any
	if m.Error != "" {
		lines = append(lines, "  Error: "+m.Error)
	}

	// Current input line (if not loading)
	if !m.Loading {
		if m.Input == "" {
			// Show hint in light grey when no input yet (marker for special handling)
			lines = append(lines, "hint:> █ Enter = Accept, ESC = Cancel")
		} else {
			// Wrap input text to terminal width
			lines = append(lines, wrapInput(m.Input, width)...)
		}
	}

	// Internal scrolling: if more than widgetHeight lines, show only the last ones
	if len(lines) > widgetHeight {
		lines = lines[len(lines)-widgetHeight:]
	}

	// Calculate new height (chat lines + 1 for buffer line)
	newHeight := len(lines) + 1 // +1 for the buffer line below
	if newHeight > widgetHeight+1 {
		newHeight = widgetHeight + 1
	}
	if newHeight < 2 {
		newHeight = 2 // At minimum: 1 input line + 1 buffer line
	}

	// Render each line with dark grey background at full width
	// Use ANSI escape: 48;5;236 = background color 236 (dark grey)
	bgOn := "\033[48;5;236m"
	bgOff := "\033[0m"

	var result strings.Builder

	// If height increased, insert new lines at the bottom
	if m.HeightTracker != nil {
		oldHeight := *m.HeightTracker
		if newHeight > oldHeight {
			linesToInsert := newHeight - oldHeight
			// Move to bottom of current content, insert lines there
			if oldHeight > 0 {
				result.WriteString(fmt.Sprintf("\033[%dB", oldHeight)) // Move down to bottom
			}
			for i := 0; i < linesToInsert; i++ {
				result.WriteString("\033[L") // Insert 1 line
			}
			if oldHeight > 0 {
				result.WriteString(fmt.Sprintf("\033[%dA", oldHeight)) // Move back up
			}
		}
		*m.HeightTracker = newHeight
	}

	for _, line := range lines {
		// Handle hint line specially
		if strings.HasPrefix(line, "hint:") {
			hintContent := strings.TrimPrefix(line, "hint:")
			// "> █" in normal color, rest in light grey, all on grey background
			visibleLen := lipgloss.Width(hintContent)
			padding := width - visibleLen
			if padding < 0 {
				padding = 0
			}
			result.WriteString(bgOn)
			result.WriteString("> █ ")
			result.WriteString("\033[38;5;242m") // Light grey for hint text
			result.WriteString("Enter = Accept, ESC = Cancel")
			result.WriteString(bgOff)
			result.WriteString(bgOn)
			result.WriteString(strings.Repeat(" ", padding))
			result.WriteString(bgOff)
			result.WriteString("\n")
			continue
		}
		// Handle input continuation lines (">>" marker -> display as " ")
		if strings.HasPrefix(line, ">>") {
			displayLine := "  " + strings.TrimPrefix(line, ">>")
			visibleLen := lipgloss.Width(displayLine)
			padding := width - visibleLen
			if padding < 0 {
				padding = 0
			}
			result.WriteString(bgOn)
			result.WriteString(displayLine)
			result.WriteString(strings.Repeat(" ", padding))
			result.WriteString(bgOff)
			result.WriteString("\n")
			continue
		}

		// Calculate visible length
		visibleLen := lipgloss.Width(line)
		padding := width - visibleLen
		if padding < 0 {
			padding = 0
		}

		// User lines (starting with ">") get grey background, AI lines get standard
		if strings.HasPrefix(line, ">") {
			result.WriteString(bgOn)
			result.WriteString(line)
			result.WriteString(strings.Repeat(" ", padding))
			result.WriteString(bgOff)
		} else {
			result.WriteString(line)
			result.WriteString(strings.Repeat(" ", padding))
		}
		result.WriteString("\n")
	}

	// Buffer line (6th line) with standard background, overwrites prompt
	// Green bold ❯ prompt: \033[1;32m = bold green
	result.WriteString("\033[1;32m❯\033[0m ")
	result.WriteString(m.Buffer)
	visibleLen := 2 + len(m.Buffer) // "❯ " + buffer
	padding := width - visibleLen
	if padding < 0 {
		padding = 0
	}
	result.WriteString(strings.Repeat(" ", padding)) // Clear rest of line

	return result.String()
}

// wrapInput wraps input text to fit within terminal width.
// First line uses "> " prefix, continuation lines use ">>" marker (rendered as "  ").
// Cursor "█" is appended at the very end.
func wrapInput(input string, width int) []string {
	prefix := "> "
	contMarker := ">>"  // Marker for continuation (rendered as "  " with background)
	contDisplay := "  " // What continuation actually displays as (aligns with "> ")
	cursor := "█"

	// Available width for text (excluding prefix/continuation display)
	firstLineWidth := width - lipgloss.Width(prefix) - lipgloss.Width(cursor)
	contLineWidth := width - lipgloss.Width(contDisplay) - lipgloss.Width(cursor)

	if firstLineWidth < 10 {
		firstLineWidth = 10
	}
	if contLineWidth < 10 {
		contLineWidth = 10
	}

	// If input fits on one line, return as-is
	if lipgloss.Width(input) <= firstLineWidth {
		return []string{prefix + input + cursor}
	}

	var lines []string
	runes := []rune(input)

	// First line
	firstLine, runes := takeRunesForWidth(runes, firstLineWidth)
	lines = append(lines, prefix+firstLine)

	// Continuation lines (use marker, will be rendered as "  ")
	for len(runes) > 0 {
		// Skip leading spaces to avoid extra indentation
		for len(runes) > 0 && runes[0] == ' ' {
			runes = runes[1:]
		}
		if len(runes) == 0 {
			break
		}
		line, rest := takeRunesForWidth(runes, contLineWidth)
		lines = append(lines, contMarker+line)
		runes = rest
	}

	// Always add cursor to the last line
	lines[len(lines)-1] += cursor

	return lines
}

// takeRunesForWidth extracts runes from the slice that fit within maxWidth.
// Returns the extracted string and the remaining runes.
func takeRunesForWidth(runes []rune, maxWidth int) (string, []rune) {
	var result []rune
	currentWidth := 0

	for i, r := range runes {
		charWidth := lipgloss.Width(string(r))
		if currentWidth+charWidth > maxWidth {
			return string(result), runes[i:]
		}
		result = append(result, r)
		currentWidth += charWidth
	}

	return string(result), nil
}
