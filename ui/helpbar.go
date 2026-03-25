package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpItem struct {
	key  string
	desc string
}

// RenderHelpBar renders the context-sensitive bottom help bar.
// activeView: 0=Tide, 1=Almanac, 2=Station
// overlay: "", "picker", or "dateinput"
func RenderHelpBar(width int, showFull bool, activeView int, overlay string) string {
	items := contextHelp(showFull, activeView, overlay)

	sepRendered := S.HelpSep.Render("  ·  ")
	var b strings.Builder
	first := true
	for _, item := range items {
		if item.key == "·" && item.desc == "" {
			b.WriteString(sepRendered)
			first = true
			continue
		}
		if !first {
			b.WriteString(sepRendered)
		}
		first = false
		b.WriteString(S.HelpKey.Render(item.key))
		b.WriteString(S.HelpDesc.Render(" " + item.desc))
	}
	bar := b.String()
	if lipgloss.Width(bar) > width {
		// Narrow terminal fallback: show only the most essential actions.
		bar = narrowFallback(overlay, activeView)
	}

	return lipgloss.NewStyle().
		Width(width).
		Foreground(S.T.TextSecondary).
		Padding(0, 1).
		Render(bar)
}

func contextHelp(full bool, view int, overlay string) []helpItem {
	sep := helpItem{"·", ""}

	// Overlays get their own minimal help, ignoring view.
	switch overlay {
	case "picker":
		return []helpItem{
			{"j/k", "navigate"},
			sep,
			{"enter", "select"},
			sep,
			{"esc", "cancel"},
		}
	case "dateinput":
		return []helpItem{
			{"enter", "jump to date"},
			sep,
			{"esc", "cancel"},
		}
	}

	// View-specific help
	switch view {
	case 0: // Tide
		if full {
			return []helpItem{
				{"1/2/3", "views"},
				{"tab", "cycle"},
				sep,
				{"←/→", "prev/next day"},
				{"d", "go to date"},
				{"t", "today"},
				sep,
				{"s", "station"},
				{"r", "refresh"},
				{"?", "less"},
				{"q", "quit"},
			}
		}
		return []helpItem{
			{"←/→", "day"},
			{"d", "date"},
			{"t", "today"},
			sep,
			{"s", "station"},
			{"r", "refresh"},
			{"?", "more"},
			{"q", "quit"},
		}

	case 1: // Almanac
		if full {
			return []helpItem{
				{"1/2/3", "views"},
				{"tab", "cycle"},
				sep,
				{"↑/↓", "scroll"},
				{"enter", "drill into day"},
				{"d", "go to date"},
				sep,
				{"s", "station"},
				{"r", "refresh"},
				{"?", "less"},
				{"q", "quit"},
			}
		}
		return []helpItem{
			{"↑/↓", "scroll"},
			{"enter", "drill"},
			{"d", "date"},
			sep,
			{"s", "station"},
			{"r", "refresh"},
			{"?", "more"},
			{"q", "quit"},
		}

	case 2: // Station
		if full {
			return []helpItem{
				{"1/2/3", "views"},
				{"tab", "cycle"},
				sep,
				{"s", "search stations"},
				{"r", "refresh"},
				{"?", "less"},
				{"q", "quit"},
			}
		}
		return []helpItem{
			{"s", "station search"},
			{"r", "refresh"},
			{"?", "more"},
			{"q", "quit"},
		}
	}

	return []helpItem{{"q", "quit"}}
}

func narrowFallback(overlay string, _ int) string {
	sep := S.HelpSep.Render("  ·  ")
	if overlay != "" {
		return S.HelpKey.Render("enter") + S.HelpDesc.Render(" select") +
			sep + S.HelpKey.Render("esc") + S.HelpDesc.Render(" cancel")
	}
	return S.HelpKey.Render("?") + S.HelpDesc.Render(" help") +
		sep + S.HelpKey.Render("q") + S.HelpDesc.Render(" quit")
}
