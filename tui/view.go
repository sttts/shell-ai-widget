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
			line := "> " + msg.Content
			// Show spinner after the last user message if loading
			if m.Loading && i == len(m.ChatHistory)-1 {
				line += " " + m.Spinner.View()
			}
			lines = append(lines, line)
		} else {
			// Light green ⏺ for bot response: 38;5;119 = light green
			lines = append(lines, "\033[38;5;119m⏺\033[0m "+msg.Content)
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
			lines = append(lines, "> "+m.Input+"█")
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
