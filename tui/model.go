package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sttts/zsh-ai-widget/ai"
	"github.com/sttts/zsh-ai-widget/config"
)

// ChatMessage represents a message in the chat history
type ChatMessage struct {
	Role    string // "user" or "assistant"
	Content string
}

// Model is the bubbletea model for the TUI
type Model struct {
	Buffer          string        // Current command buffer
	OriginalBuffer  string        // Original buffer to restore on cancel
	TerminalContext string        // Recent terminal output
	Cwd             string        // Current working directory
	ChatHistory     []ChatMessage // Chat history
	Input           string        // Current input text
	Loading         bool          // Whether we're waiting for AI response
	Error           string        // Error message to display
	Accepted        bool          // Whether the user accepted the result
	Config          *config.Config
	aiClient        ai.Client
	Shimmer         Shimmer
	inputBuffer     string        // Buffer for detecting escape sequences
	HeightTracker   *int          // Pointer to track current height (shared with main)
}

// NewModel creates a new TUI model
func NewModel(buffer, terminalContext, cwd string, cfg *config.Config, heightTracker *int) Model {
	return Model{
		Buffer:          buffer,
		OriginalBuffer:  buffer,
		TerminalContext: terminalContext,
		Cwd:             cwd,
		ChatHistory:     []ChatMessage{},
		Input:           "",
		Loading:         false,
		Error:           "",
		Accepted:        false,
		Config:          cfg,
		Shimmer:         NewShimmer(),
		HeightTracker:   heightTracker,
	}
}

// aiResponseMsg is sent when the AI responds
type aiResponseMsg struct {
	response *ai.Response
	err      error
}

func (m Model) Init() tea.Cmd {
	// Initialize the AI client
	return func() tea.Msg {
		client, err := ai.NewClient(m.Config)
		if err != nil {
			return aiResponseMsg{err: err}
		}
		// Store the client - we'll handle this in Update
		return clientInitMsg{client: client}
	}
}

type clientInitMsg struct {
	client ai.Client
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case clientInitMsg:
		m.aiClient = msg.client
		return m, nil

	case ShimmerMsg:
		if m.Loading {
			var cmd tea.Cmd
			m.Shimmer, cmd = m.Shimmer.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Accept):
			// Shift+Cmd+K (appears as alt+k / ESC k) - accept and quit
			m.Accepted = true
			return m, tea.Quit

		case msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEscape:
			// Cancel - restore original buffer
			m.Buffer = m.OriginalBuffer
			m.Accepted = false
			return m, tea.Quit

		case msg.Type == tea.KeyEnter:
			if m.Loading {
				return m, nil
			}

			input := strings.TrimSpace(m.Input)
			if input == "" {
				// Empty input - accept current buffer
				m.Accepted = true
				return m, tea.Quit
			}

			// Send message to AI
			m.ChatHistory = append(m.ChatHistory, ChatMessage{
				Role:    "user",
				Content: input,
			})
			m.Input = ""
			m.Loading = true
			m.Error = ""

			// Set shimmer text and start animation
			m.Shimmer.SetText("> " + input)

			return m, tea.Batch(m.Shimmer.Tick(), m.sendToAI())

		case msg.Type == tea.KeyBackspace:
			if len(m.Input) > 0 {
				m.Input = m.Input[:len(m.Input)-1]
			}

		case msg.Type == tea.KeyRunes:
			m.Input += string(msg.Runes)

		case msg.Type == tea.KeySpace:
			m.Input += " "
		}

	case aiResponseMsg:
		m.Loading = false
		if msg.err != nil {
			m.Error = msg.err.Error()
			return m, nil
		}

		// Update buffer with AI's suggested command (only if non-empty)
		if msg.response.Command != "" {
			m.Buffer = msg.response.Command
		}

		// Add AI response to chat history
		m.ChatHistory = append(m.ChatHistory, ChatMessage{
			Role:    "assistant",
			Content: msg.response.Reply,
		})

		return m, nil
	}

	return m, nil
}

func (m Model) sendToAI() tea.Cmd {
	return func() tea.Msg {
		if m.aiClient == nil {
			return aiResponseMsg{err: nil}
		}

		// Convert chat history to AI messages
		messages := make([]ai.Message, 0, len(m.ChatHistory))
		for _, msg := range m.ChatHistory {
			messages = append(messages, ai.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		response, err := m.aiClient.Chat(messages, m.Buffer, m.TerminalContext, m.Cwd)
		return aiResponseMsg{response: response, err: err}
	}
}
