package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/mogglemoss/fathom/noaa"
)

// significantPhaseNames are moon phases that display their full name.
var significantPhaseNames = map[string]bool{
	"New Moon":      true,
	"First Quarter": true,
	"Full Moon":     true,
	"Last Quarter":  true,
}

// Column widths (visual characters, excluding ANSI codes).
const (
	colDate  = 10 // "Mon Jan  2"
	colEvent = 14 // "▲ 07:14  10.4"  (arrow sp time 2sp level)
	colRange = 8  // "  10.1ft"
	colMoon  = 4  // "  🌕" (glyph always)
)

// RenderAlmanacView renders the 7–14 day tide forecast in a grid layout.
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

	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(days) {
		cursor = len(days) - 1
	}

	visibleRows := height - 5
	if visibleRows < 1 {
		visibleRows = 1
	}

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

	// Determine how many event columns we need (max tides in any visible day)
	end := offset + visibleRows
	if end > len(days) {
		end = len(days)
	}
	maxEvents := 2
	for i := offset; i < end; i++ {
		if n := len(days[i].Predictions); n > maxEvents {
			maxEvents = n
		}
	}
	if maxEvents > 4 {
		maxEvents = 4
	}

	// Narrow terminals: fewer columns
	rowWidth := 2 + colDate + maxEvents*colEvent + colRange + colMoon + 20
	if rowWidth > width {
		maxEvents = 2
	}

	// Header row
	header := renderAlmanacHeader(maxEvents)
	b.WriteString(header + "\n")

	// Top scroll indicator
	if offset > 0 {
		b.WriteString(S.HelpDesc.Render(fmt.Sprintf("  ▲ %d days above", offset)) + "\n")
	}

	todayStr := time.Now().Format("2006-01-02")

	for i := offset; i < end; i++ {
		isToday := days[i].Date.Format("2006-01-02") == todayStr
		row := renderAlmanacRow(days[i], width, i == cursor, isToday, maxEvents)
		b.WriteString(row + "\n")
	}

	if end < len(days) {
		b.WriteString(S.HelpDesc.Render(fmt.Sprintf("  ▼ %d more days", len(days)-end)))
	}

	return b.String()
}

func renderAlmanacHeader(numEvents int) string {
	var b strings.Builder
	b.WriteString(S.SectionHeader.Render("  TIDE FORECAST") + "  ")
	b.WriteString(S.Label.Render(strings.Repeat(" ", colDate)))
	for i := 0; i < numEvents; i++ {
		b.WriteString(S.Label.Render(fmt.Sprintf("  %-12s", "TIME   FT")))
	}
	return b.String()
}

func renderAlmanacRow(day noaa.DailyTide, width int, selected, isToday bool, numEvents int) string {
	// Sort predictions by time.
	preds := make([]noaa.Prediction, len(day.Predictions))
	copy(preds, day.Predictions)
	sort.Slice(preds, func(i, j int) bool { return preds[i].Time.Before(preds[j].Time) })

	// Date column
	dateStr := day.Date.Format("Mon Jan  2")
	var dateCol string
	if isToday {
		dateCol = lipgloss.NewStyle().Foreground(S.T.Accent).Bold(true).Render(dateStr)
	} else {
		dateCol = S.AlmanacDate.Render(dateStr)
	}

	// Tide event columns
	var tideStr strings.Builder
	var maxHigh, minLow float64
	hasHigh, hasLow := false, false

	for i := 0; i < numEvents; i++ {
		tideStr.WriteString("  ")
		if i < len(preds) {
			p := preds[i]
			if p.IsHigh {
				if !hasHigh || p.Level > maxHigh {
					maxHigh = p.Level
					hasHigh = true
				}
				arrow := S.AlmanacHigh.Render("▲")
				t := S.Value.Render(p.Time.Format("15:04"))
				v := S.AlmanacHigh.Render(fmt.Sprintf("%5.1f", p.Level))
				tideStr.WriteString(arrow + " " + t + " " + v)
			} else {
				if !hasLow || p.Level < minLow {
					minLow = p.Level
					hasLow = true
				}
				arrow := S.AlmanacLow.Render("▼")
				t := S.Value.Render(p.Time.Format("15:04"))
				v := S.AlmanacLow.Render(fmt.Sprintf("%5.1f", p.Level))
				tideStr.WriteString(arrow + " " + t + " " + v)
			}
		} else {
			// Empty slot — preserve alignment
			tideStr.WriteString(strings.Repeat(" ", colEvent-2))
		}
	}

	// Tidal range column
	var rangeStr string
	if hasHigh && hasLow {
		rangeStr = "  " + S.Label.Render(fmt.Sprintf("%4.1fft", maxHigh-minLow))
	} else {
		rangeStr = strings.Repeat(" ", colRange)
	}

	// Moon column — glyph always; name only for significant phases
	var moonStr string
	if day.MoonName != "" {
		glyph := moonPhaseGlyph(day.MoonPhase)
		if significantPhaseNames[day.MoonName] {
			moonStr = "  " + S.AlmanacMoon.Render(glyph+" "+day.MoonName)
		} else {
			moonStr = "  " + S.AlmanacMoon.Render(glyph)
		}
	}

	row := "  " + dateCol + tideStr.String() + rangeStr + moonStr

	if selected {
		rowWidth := lipgloss.Width(row)
		if rowWidth < width-2 {
			row += strings.Repeat(" ", width-2-rowWidth)
		}
		return S.AlmanacCursor.Render(row)
	}
	return row
}

// moonPhaseGlyph returns a Unicode moon emoji for the given synodic phase [0,1).
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
