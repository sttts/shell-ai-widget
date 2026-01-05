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
			lines = append(lines, "  "+msg.Content)
		}
	}

	// Show error if any
	if m.Error != "" {
		lines = append(lines, "  Error: "+m.Error)
	}

	// Current input line (if not loading)
	if !m.Loading {
		lines = append(lines, "> "+m.Input+"█")
	}

	// Internal scrolling: if more than widgetHeight lines, show only the last ones
	if len(lines) > widgetHeight {
		lines = lines[len(lines)-widgetHeight:]
	}

	// Calculate new height (1 to 5 lines)
	newHeight := len(lines)
	if newHeight > widgetHeight {
		newHeight = widgetHeight
	}
	if newHeight < 1 {
		newHeight = 1
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

	for i, line := range lines {
		// Calculate visible length
		visibleLen := lipgloss.Width(line)
		padding := width - visibleLen
		if padding < 0 {
			padding = 0
		}
		// Full line with background
		result.WriteString(bgOn)
		result.WriteString(line)
		result.WriteString(strings.Repeat(" ", padding))
		result.WriteString(bgOff)
		if i < len(lines)-1 {
			result.WriteString("\n")
		}
	}

	return result.String()
}
