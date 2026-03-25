package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mogglemoss/fathom/noaa"
)

// RenderAlmanacView renders the 7–14 day tide forecast.
func RenderAlmanacView(
	days []noaa.DailyTide,
	cursor int,
	width int,
	height int,
) string {
	if len(days) == 0 {
		return "\n  " + S.StatusMeta.Render("〰 loading forecast…") + "\n"
	}

	var b strings.Builder
	b.WriteString("\n")

	// Clamp cursor
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(days) {
		cursor = len(days) - 1
	}

	// How many rows fit in the available height (minus 2 for header + padding)
	visibleRows := height - 3
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Scroll offset: keep cursor visible
	offset := cursor - visibleRows + 3
	if offset < 0 {
		offset = 0
	}
	if offset > len(days)-visibleRows {
		offset = len(days) - visibleRows
		if offset < 0 {
			offset = 0
		}
	}

	b.WriteString(S.SectionHeader.Render("  TIDE FORECAST") + "\n\n")

	end := offset + visibleRows
	if end > len(days) {
		end = len(days)
	}

	for i := offset; i < end; i++ {
		row := renderAlmanacRow(days[i], width, i == cursor)
		b.WriteString(row + "\n")
	}

	// Scroll hints
	if offset > 0 {
		// Overwrite first rendered row hint — instead just add indicator at bottom
	}
	if end < len(days) {
		b.WriteString(S.HelpDesc.Render("  ▼ more  (" + fmt.Sprintf("%d", len(days)-end) + " days)"))
	}

	return b.String()
}

func renderAlmanacRow(day noaa.DailyTide, width int, selected bool) string {
	dateStr := day.Date.Format("Mon Jan  2")

	// Build tide events string
	var tides strings.Builder
	for _, p := range day.Predictions {
		if tides.Len() > 0 {
			tides.WriteString("  ")
		}
		timeStr := p.Time.Format("3:04")
		levelStr := fmt.Sprintf("%.1f", p.Level)
		if p.IsHigh {
			tides.WriteString(S.AlmanacHigh.Render("▲ " + timeStr + " " + levelStr))
		} else {
			tides.WriteString(S.AlmanacLow.Render("▼ " + timeStr + " " + levelStr))
		}
	}

	// Moon glyph and name
	moonGlyph := day.MoonName[:1] // safe default
	_ = moonGlyph
	moonStr := ""
	if day.MoonName != "" {
		glyph := moonPhaseGlyph(day.MoonPhase)
		moonStr = "  " + S.AlmanacMoon.Render(glyph+" "+day.MoonName)
	}

	row := "  " + dateStr + "   " + tides.String() + moonStr

	if selected {
		// Pad to full width so the selection background fills the row
		rowWidth := lipgloss.Width(row)
		if rowWidth < width-2 {
			row += strings.Repeat(" ", width-2-rowWidth)
		}
		return S.AlmanacCursor.Render(row)
	}
	return row
}

// moonPhaseGlyph returns a Unicode moon emoji. Duplicated here to avoid
// importing the moon package from ui (keep ui dependency-free of business logic).
func moonPhaseGlyph(phase float64) string {
	switch {
	case phase < 0.0625 || phase >= 0.9375:
		return "🌑"
	case phase < 0.1875:
		return "🌒"
	case phase < 0.3125:
		return "🌓"
	case phase < 0.4375:
		return "🌔"
	case phase < 0.5625:
		return "🌕"
	case phase < 0.6875:
		return "🌖"
	case phase < 0.8125:
		return "🌗"
	default:
		return "🌘"
	}
}
