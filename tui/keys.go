package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Send   key.Binding
	Accept key.Binding
	Cancel key.Binding
	Test   key.Binding
}

var keys = keyMap{
	Send: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "send message"),
	),
	Accept: key.NewBinding(
		key.WithKeys("alt+k"), // ESC k appears as alt+k in bubbletea
		key.WithHelp("cmd+k", "accept"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "cancel"),
	),
	Test: key.NewBinding(
		key.WithKeys("ctrl+r", "alt+enter"),
		key.WithHelp("ctrl+r", "run"),
	),
}
