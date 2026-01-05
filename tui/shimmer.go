package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// ShimmerMsg is sent on each shimmer tick
type ShimmerMsg time.Time

// Shimmer is a model that renders text with a pulsing highlight effect
type Shimmer struct {
	Text     string
	Position int
	Speed    time.Duration
}

// NewShimmer creates a new shimmer model
func NewShimmer() Shimmer {
	return Shimmer{
		Speed: 50 * time.Millisecond,
	}
}

// Tick returns a command that ticks the shimmer animation
func (s Shimmer) Tick() tea.Cmd {
	return tea.Tick(s.Speed, func(t time.Time) tea.Msg {
		return ShimmerMsg(t)
	})
}

// Update advances the shimmer position
func (s Shimmer) Update(msg tea.Msg) (Shimmer, tea.Cmd) {
	switch msg.(type) {
	case ShimmerMsg:
		textLen := len([]rune(s.Text))
		if textLen > 0 {
			s.Position = (s.Position + 1) % (textLen + 6) // +6 for trail effect
		}
		return s, s.Tick()
	}
	return s, nil
}

// SetText sets the text to shimmer
func (s *Shimmer) SetText(text string) {
	s.Text = text
	s.Position = 0
}

// View renders the text with shimmer effect
func (s Shimmer) View() string {
	if s.Text == "" {
		return ""
	}

	runes := []rune(s.Text)
	var result strings.Builder

	for i, r := range runes {
		// Calculate distance from shimmer position
		dist := s.Position - i
		if dist < 0 {
			dist = -dist
		}

		// Apply brightness based on distance from shimmer center
		switch {
		case dist == 0:
			// Brightest - white
			result.WriteString("\033[97m")
		case dist == 1:
			// Very bright
			result.WriteString("\033[37m")
		case dist == 2:
			// Slightly bright
			result.WriteString("\033[38;5;250m")
		default:
			// Normal - dim grey
			result.WriteString("\033[38;5;243m")
		}
		result.WriteRune(r)
	}
	result.WriteString("\033[0m")
	return result.String()
}
