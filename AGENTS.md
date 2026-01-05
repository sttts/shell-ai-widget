# Repository Guidelines

This document outlines standards for the **zsh-ai-widget** project, a Go-based TUI using Bubble Tea and lipgloss.

## Dependencies

- Use Bubble Tea v1: `github.com/charmbracelet/bubbletea`
- Use lipgloss for styling: `github.com/charmbracelet/lipgloss`
- Maintain consistent import paths throughout

## Code Standards

- Format with `gofmt` before committing (CI enforces this)
- Follow Go conventions: lower-case package names, CamelCase for exports
- Wrap errors using `fmt.Errorf("context: %w", err)` syntax
- Thread `context.Context` explicitly through call chains, especially for cancellable operations
- Keep functions focused and small

## Architecture

- `main.go` - Entry point, argument parsing, terminal setup
- `tui/` - Bubble Tea model, view, and key bindings
- `ai/` - AI provider interface and implementations (OpenAI, Anthropic)
- `config/` - TOML configuration parsing

## Testing

- Use standard `testing` package with table-driven tests
- Place tests in `*_test.go` files alongside the code
- Run tests with `go test ./...`

## Commits

- Use imperative mood in commit subjects ("Add feature" not "Added feature")
- Commit logically related changes together
- Include co-author trailer for AI-assisted commits:
  ```
  Co-Authored-By: Claude <noreply@anthropic.com>
  ```

## Security

- Never commit API keys or credentials
- Use environment variables (`OPENAI_API_KEY`, `ANTHROPIC_API_KEY`) or config file
- The config file at `~/.config/zsh-ai-widget/config.toml` should not be committed

## Building

```bash
go build -o zsh-ai-widget .
go test ./...
gofmt -w .
```
