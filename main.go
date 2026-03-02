package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sttts/shell-ai-widget/config"
	"github.com/sttts/shell-ai-widget/tui"
)

func main() {
	var buffer string
	var contextFile string
	var shell string

	flag.StringVar(&buffer, "buffer", "", "Current command line buffer")
	flag.StringVar(&contextFile, "context-file", "", "Path to file containing terminal scrollback")
	flag.StringVar(&shell, "shell", "", "Shell type (e.g. zsh, fish)")
	flag.Parse()
	shell = resolveShell(shell, os.Getenv("SHELL"))

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

	// Track height for dynamic growth (starts at 0, dynamic insert handles growth)
	currentHeight := 0

	// Create and run the TUI
	model := tui.NewModel(buffer, terminalContext, cwd, shell, cfg, &currentHeight)
	p := tea.NewProgram(model, tea.WithInput(ttyFile), tea.WithOutput(ttyFile))

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}

	m := finalModel.(tui.Model)

	// Clean up: move up by rendered height - 1, then clear to end of screen
	if currentHeight > 1 {
		fmt.Fprintf(ttyFile, "\033[%dA", currentHeight-1) // Move up
	}
	fmt.Fprint(ttyFile, "\033[J") // Clear from cursor to end of screen

	// Output the final buffer to stdout
	if m.Accepted {
		fmt.Print(m.Buffer)
		os.Exit(0)
	} else {
		// Cancelled - exit with error code so zsh widget restores original
		os.Exit(1)
	}
}

func resolveShell(cliShell, shellEnv string) string {
	trimmedCLI := strings.TrimSpace(cliShell)
	if trimmedCLI != "" {
		return trimmedCLI
	}

	trimmedEnv := strings.TrimSpace(shellEnv)
	if trimmedEnv != "" {
		envBase := strings.TrimSpace(filepath.Base(trimmedEnv))
		if envBase != "" && envBase != "." && envBase != string(filepath.Separator) {
			return envBase
		}
	}

	return "zsh"
}
