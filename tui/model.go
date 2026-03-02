package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sttts/shell-ai-widget/ai"
	"github.com/sttts/shell-ai-widget/config"
)

// escCooldownMsg is sent when ESC cooldown expires
type escCooldownMsg struct{}

// ChatMessage represents a message in the chat history
type ChatMessage struct {
	Role       string         // "user" or "assistant"
	Content    string         // Text content
	ToolCalls  []ai.ToolCall  // Tool calls made by assistant
	ToolResult *ai.ToolResult // Tool result (for tool messages)
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
	inputBuffer     string             // Buffer for detecting escape sequences
	HeightTracker   *int               // Pointer to track current height (shared with main)
	EscCooldown     bool               // Whether ESC is in cooldown (can't close)
	PendingInput    string             // Input saved before sending to AI (for cancellation)
	cancelAI        context.CancelFunc // Function to cancel ongoing AI request
	ToolExecutor    *ai.ToolExecutor   // Tool executor
	ExecutingTool   *ai.ToolCall       // Currently executing tool (for display)
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
		ToolExecutor:    ai.NewToolExecutor(cfg.Tools.EnableWebSearch, cfg.Tools.EnableCommandHelp),
	}
}

// aiResponseMsg is sent when the AI responds
type aiResponseMsg struct {
	response  *ai.Response
	err       error
	cancelled bool
}

// toolResultMsg is sent when tool executions complete
type toolResultMsg struct {
	results   []ai.ToolResult // Results for all tool calls
	toolCalls []ai.ToolCall   // The original tool calls from the AI
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

	case escCooldownMsg:
		m.EscCooldown = false
		return m, nil

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Accept):
			// Shift+Cmd+K (appears as alt+k / ESC k) - accept and quit
			m.Accepted = true
			return m, tea.Quit

		case msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEscape:
			// During loading: cancel AI request, restore input for editing
			if m.Loading {
				// Cancel the HTTP request
				if m.cancelAI != nil {
					m.cancelAI()
					m.cancelAI = nil
				}
				m.Loading = false
				m.ExecutingTool = nil
				m.Input = m.PendingInput // Restore input for editing
				// Remove the last user message from chat history
				if len(m.ChatHistory) > 0 && m.ChatHistory[len(m.ChatHistory)-1].Role == "user" {
					m.ChatHistory = m.ChatHistory[:len(m.ChatHistory)-1]
				}
				return m, nil
			}
			// During cooldown: ignore ESC
			if m.EscCooldown {
				return m, nil
			}
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

			// Save input for potential cancellation
			m.PendingInput = input

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

			// Create cancellable context for AI request
			ctx, cancel := context.WithCancel(context.Background())
			m.cancelAI = cancel

			return m, tea.Batch(m.Shimmer.Tick(), m.sendToAI(ctx))

		case msg.Type == tea.KeyBackspace:
			if len(m.Input) > 0 {
				m.Input = m.Input[:len(m.Input)-1]
			}

		case msg.Type == tea.KeyRunes:
			m.Input += string(msg.Runes)

		case msg.Type == tea.KeySpace:
			m.Input += " "
		}

	case toolResultMsg:
		// Tool execution completed, add results to history and continue AI conversation
		m.ExecutingTool = nil

		// Add the assistant's tool calls to history
		m.ChatHistory = append(m.ChatHistory, ChatMessage{
			Role:      "assistant",
			ToolCalls: msg.toolCalls,
		})

		// Add tool result for each tool call
		for i := range msg.results {
			result := msg.results[i]
			m.ChatHistory = append(m.ChatHistory, ChatMessage{
				Role:       "tool",
				ToolResult: &result,
			})
		}

		// Continue the conversation with the tool results
		ctx, cancel := context.WithCancel(context.Background())
		m.cancelAI = cancel

		// Update shimmer to show we're thinking again
		m.Shimmer.SetText("Thinking...")

		return m, tea.Batch(m.Shimmer.Tick(), m.sendToAI(ctx))

	case aiResponseMsg:
		// Ignore cancelled requests
		if msg.cancelled {
			return m, nil
		}

		if msg.err != nil {
			m.Loading = false
			m.ExecutingTool = nil
			m.Error = msg.err.Error()
			m.EscCooldown = true
			return m, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
				return escCooldownMsg{}
			})
		}

		// Check if the AI wants to call tools
		if len(msg.response.ToolCalls) > 0 && !msg.response.Done {
			// Show first tool in shimmer for display
			toolCall := msg.response.ToolCalls[0]
			m.ExecutingTool = &toolCall

			// Update shimmer to show tool execution
			icon := ai.GetDisplayIcon(toolCall)
			text := ai.GetDisplayText(toolCall)
			m.Shimmer.SetText(icon + " " + text)

			// Execute all tools
			return m, tea.Batch(m.Shimmer.Tick(), m.executeTools(msg.response.ToolCalls))
		}

		// No more tools, we have a final response
		m.Loading = false
		m.ExecutingTool = nil
		m.EscCooldown = true // Start cooldown to prevent accidental ESC close

		// Update buffer with AI's suggested command (only if non-empty)
		if msg.response.Command != "" {
			m.Buffer = msg.response.Command
		}

		// Add AI response to chat history
		m.ChatHistory = append(m.ChatHistory, ChatMessage{
			Role:    "assistant",
			Content: msg.response.Reply,
		})

		// Start 200ms cooldown timer
		return m, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
			return escCooldownMsg{}
		})
	}

	return m, nil
}

func (m Model) sendToAI(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		if m.aiClient == nil {
			return aiResponseMsg{err: nil}
		}

		// Convert chat history to AI messages
		messages := make([]ai.Message, 0, len(m.ChatHistory))
		for _, msg := range m.ChatHistory {
			aiMsg := ai.Message{
				Role:       msg.Role,
				Content:    msg.Content,
				ToolCalls:  msg.ToolCalls,
				ToolResult: msg.ToolResult,
			}
			messages = append(messages, aiMsg)
		}

		toolsCfg := ai.ToolsConfig{
			EnableWebSearch:   m.Config.Tools.EnableWebSearch,
			EnableCommandHelp: m.Config.Tools.EnableCommandHelp,
		}

		response, err := m.aiClient.Chat(ctx, messages, m.Buffer, m.TerminalContext, m.Cwd, toolsCfg)
		// If context was cancelled, mark as cancelled so handler ignores it
		if ctx.Err() != nil {
			return aiResponseMsg{cancelled: true}
		}
		return aiResponseMsg{response: response, err: err}
	}
}

func (m Model) executeTools(calls []ai.ToolCall) tea.Cmd {
	return func() tea.Msg {
		results := make([]ai.ToolResult, len(calls))
		for i, call := range calls {
			results[i] = m.ToolExecutor.Execute(context.Background(), call)
		}
		return toolResultMsg{
			results:   results,
			toolCalls: calls,
		}
	}
}
