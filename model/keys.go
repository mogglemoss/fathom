package model

import "github.com/charmbracelet/bubbles/key"

// KeyMap holds all key bindings for fathom.
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	ViewTide    key.Binding
	ViewAlmanac key.Binding
	ViewStation key.Binding
	NextView    key.Binding
	Refresh     key.Binding
	Help        key.Binding
	Quit        key.Binding
}

// DefaultKeyMap returns the default key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		ViewTide: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "tide view"),
		),
		ViewAlmanac: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "almanac"),
		),
		ViewStation: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "station"),
		),
		NextView: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next view"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}
