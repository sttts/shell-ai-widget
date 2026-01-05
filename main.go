package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sttts/zsh-ai-widget/config"
	"github.com/sttts/zsh-ai-widget/tui"
)

func main() {
	var buffer string
	var contextFile string

	flag.StringVar(&buffer, "buffer", "", "Current command line buffer")
	flag.StringVar(&contextFile, "context-file", "", "Path to file containing terminal scrollback")
	flag.Parse()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Read terminal context from file
	var terminalContext string
	if contextFile != "" {
		data, err := os.ReadFile(contextFile)
		if err == nil {
			lines := strings.Split(string(data), "\n")
			// Take last N lines (configured context_lines, default 100)
			contextLines := cfg.UI.ContextLines
			if contextLines <= 0 {
				contextLines = 100
			}
			if len(lines) > contextLines {
				lines = lines[len(lines)-contextLines:]
			}
			terminalContext = strings.Join(lines, "\n")
		}
	}

	// Get current working directory
	cwd, _ := os.Getwd()

	// Open /dev/tty for TUI output (so stdout remains clean for the result)
	ttyFile, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening /dev/tty: %v\n", err)
		os.Exit(1)
	}
	defer ttyFile.Close()

	// Save cursor position (draw starting at current line)
	fmt.Fprint(ttyFile, "\033[s")      // Save cursor position

	// Track height for dynamic growth (starts at 0, dynamic insert handles growth)
	currentHeight := 0

	// Create and run the TUI
	model := tui.NewModel(buffer, terminalContext, cwd, cfg, &currentHeight)
	p := tea.NewProgram(model, tea.WithInput(ttyFile), tea.WithOutput(ttyFile))

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	m := finalModel.(tui.Model)

	// Delete widget lines and restore
	fmt.Fprint(ttyFile, "\033[u")      // Restore cursor position
	fmt.Fprintf(ttyFile, "\033[%dM", currentHeight) // Delete tracked lines

	// Output the final buffer to stdout
	if m.Accepted {
		fmt.Print(m.Buffer)
		os.Exit(0)
	} else {
		// Cancelled - exit with error code so zsh widget restores original
		os.Exit(1)
	}
}
