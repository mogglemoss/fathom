package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpItem struct {
	key  string
	desc string
}

var shortHelp = []helpItem{
	{"1/2/3", "tide · almanac · station"},
	{"j/k", "scroll"},
	{"r", "refresh"},
	{"?", "more"},
	{"q", "quit"},
}

var fullHelp = []helpItem{
	{"1", "tide view"},
	{"2", "almanac"},
	{"3", "station"},
	{"tab", "next view"},
	{"j/↓", "scroll down"},
	{"k/↑", "scroll up"},
	{"r", "refresh"},
	{"?", "toggle help"},
	{"q / ctrl+c", "quit"},
}

// RenderHelpBar renders the bottom help bar with graceful narrow-terminal fallback.
func RenderHelpBar(width int, showFull bool) string {
	items := shortHelp
	if showFull {
		items = fullHelp
	}

	sep := S.HelpSep.Render("  ·  ")
	var parts []string
	for _, item := range items {
		k := S.HelpKey.Render(item.key)
		d := S.HelpDesc.Render(" " + item.desc)
		parts = append(parts, k+d)
	}

	bar := strings.Join(parts, sep)
	if lipgloss.Width(bar) > width {
		bar = S.HelpKey.Render("?") + S.HelpDesc.Render(" help") +
			sep + S.HelpKey.Render("q") + S.HelpDesc.Render(" quit")
	}

	return lipgloss.NewStyle().
		Width(width).
		Foreground(S.T.TextSecondary).
		Padding(0, 1).
		Render(bar)
}
