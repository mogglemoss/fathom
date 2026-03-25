package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/mogglemoss/fathom/noaa"
)

// significantPhaseNames are moon phases that get their full name displayed.
var significantPhaseNames = map[string]bool{
	"New Moon":      true,
	"First Quarter": true,
	"Full Moon":     true,
	"Last Quarter":  true,
}

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

	// How many rows fit in the available height (minus header + padding)
	visibleRows := height - 4
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

	// Top scroll indicator
	if offset > 0 {
		b.WriteString(S.HelpDesc.Render(fmt.Sprintf("  ▲ %d more above", offset)) + "\n")
	}

	end := offset + visibleRows
	if end > len(days) {
		end = len(days)
	}

	today := time.Now()
	todayStr := today.Format("2006-01-02")

	for i := offset; i < end; i++ {
		isToday := days[i].Date.Format("2006-01-02") == todayStr
		row := renderAlmanacRow(days[i], width, i == cursor, isToday)
		b.WriteString(row + "\n")
	}

	// Bottom scroll indicator
	if end < len(days) {
		b.WriteString(S.HelpDesc.Render(fmt.Sprintf("  ▼ %d more below", len(days)-end)))
	}

	return b.String()
}

func renderAlmanacRow(day noaa.DailyTide, width int, selected, isToday bool) string {
	// Date — bold for today
	dateStr := day.Date.Format("Mon Jan  2")
	var dateRendered string
	if isToday {
		dateRendered = lipgloss.NewStyle().Foreground(S.T.Accent).Bold(true).Render(dateStr)
	} else {
		dateRendered = S.AlmanacDate.Render(dateStr)
	}

	// Tide events
	var tides strings.Builder
	var maxHigh, minLow float64
	hasHigh, hasLow := false, false
	for _, p := range day.Predictions {
		if p.IsHigh {
			if !hasHigh || p.Level > maxHigh {
				maxHigh = p.Level
				hasHigh = true
			}
		} else {
			if !hasLow || p.Level < minLow {
				minLow = p.Level
				hasLow = true
			}
		}
	}
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

	// Tidal range
	rangeStr := ""
	if hasHigh && hasLow {
		rangeStr = "  " + S.Label.Render(fmt.Sprintf("%.1f ft", maxHigh-minLow))
	}

	// Moon: glyph always; name only for significant phases
	moonStr := ""
	if day.MoonName != "" {
		glyph := moonPhaseGlyph(day.MoonPhase)
		if significantPhaseNames[day.MoonName] {
			moonStr = "  " + S.AlmanacMoon.Render(glyph+" "+day.MoonName)
		} else {
			moonStr = "  " + S.AlmanacMoon.Render(glyph)
		}
	}

	row := "  " + dateRendered + "   " + tides.String() + rangeStr + moonStr

	if selected {
		rowWidth := lipgloss.Width(row)
		if rowWidth < width-2 {
			row += strings.Repeat(" ", width-2-rowWidth)
		}
		return S.AlmanacCursor.Render(row)
	}
	return row
}

// moonPhaseGlyph returns a Unicode moon emoji for the given phase fraction [0,1).
// Duplicated here to keep ui free of moon package dependency.
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
