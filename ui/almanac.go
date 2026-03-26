package ui

import (
	"fmt"
	"math"
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

// Fixed visual widths for the almanac grid columns.
// Every row must produce exactly these widths so columns stay aligned.
//
//	date:  "Mon Jan _2"     → always 10 chars  (Go _2 = space-padded day)
//	event: "  ▲  4:32p  5.8" → 16 chars each
//	       2(sep) + 1(▲) + 1(sp) + 6(time) + 1(sp) + 5(level) = 16
//	range: "   5.8ft ▓▓▓░░"  → 14 chars: 8 (number) + 6 (inline bar)
const (
	almDateW  = 10
	almEvtW   = 16 // includes 2-char leading separator
	almRangeW = 14 // 8 (number) + 6 (inline bar)
)

// almEvtHdr is the column header string for each event slot (16 chars).
// It aligns TIME over the time field and FT over the level field.
const almEvtHdr = "    TIME    FT  "

// RenderAlmanacView renders the 7–14 day tide forecast in a fixed-width grid.
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

	offset := AlmanacScrollOffset(cursor, visibleRows, len(days))

	end := offset + visibleRows
	if end > len(days) {
		end = len(days)
	}

	// Determine the maximum number of event columns needed in the visible window.
	maxEvents := 2
	for i := offset; i < end; i++ {
		if n := len(days[i].Predictions); n > maxEvents {
			maxEvents = n
		}
	}
	if maxEvents > 4 {
		maxEvents = 4
	}
	// Narrow-terminal fallback: drop to 2 events.
	minRowW := 2 + almDateW + 2*almEvtW + almRangeW + 6
	if width < minRowW {
		maxEvents = 2
	}

	// ── Section header (own line) ─────────────────────────────────────────
	b.WriteString(S.SectionHeader.Render("  TIDE FORECAST") + "\n")

	// ── Column headers (aligned with data) ───────────────────────────────
	b.WriteString("  " + strings.Repeat(" ", almDateW))
	for range make([]struct{}, maxEvents) {
		b.WriteString(S.Label.Render(almEvtHdr))
	}
	b.WriteString(S.Label.Render("  RANGE        ") + "\n")

	// ── Top scroll indicator ──────────────────────────────────────────────
	if offset > 0 {
		b.WriteString(S.HelpDesc.Render(fmt.Sprintf("  ▲ %d days above", offset)) + "\n")
	}

	// ── Compute maxRange across the visible window (for proportional bar) ──
	var maxRange float64
	for i := offset; i < end; i++ {
		var hi, lo float64
		hasH, hasL := false, false
		for _, p := range days[i].Predictions {
			if p.IsHigh && (!hasH || p.Level > hi) {
				hi = p.Level
				hasH = true
			}
			if !p.IsHigh && (!hasL || p.Level < lo) {
				lo = p.Level
				hasL = true
			}
		}
		if hasH && hasL && hi-lo > maxRange {
			maxRange = hi - lo
		}
	}

	// ── Data rows ─────────────────────────────────────────────────────────
	todayStr := time.Now().Format("2006-01-02")
	prevMoonName := ""
	for i := offset; i < end; i++ {
		isToday := days[i].Date.Format("2006-01-02") == todayStr
		b.WriteString(renderAlmanacRow(days[i], width, i == cursor, isToday, maxEvents, prevMoonName, i-offset, maxRange) + "\n")
		if days[i].MoonName != "" {
			prevMoonName = days[i].MoonName
		}
	}

	// ── Bottom scroll indicator ───────────────────────────────────────────
	if end < len(days) {
		b.WriteString(S.HelpDesc.Render(fmt.Sprintf("  ▼ %d more days", len(days)-end)))
	}

	return b.String()
}

// AlmanacScrollOffset computes the scroll offset from cursor position.
// Exported so the model can reuse it for mouse-click row mapping.
func AlmanacScrollOffset(cursor, visibleRows, totalDays int) int {
	offset := cursor - visibleRows + 3
	if offset < 0 {
		offset = 0
	}
	if offset > totalDays-visibleRows {
		offset = totalDays - visibleRows
		if offset < 0 {
			offset = 0
		}
	}
	return offset
}

func renderAlmanacRow(day noaa.DailyTide, width int, selected, isToday bool, numEvents int, prevMoonName string, rowIdx int, maxRange float64) string {
	// Sort predictions by time so columns are always in chronological order.
	preds := make([]noaa.Prediction, len(day.Predictions))
	copy(preds, day.Predictions)
	sort.Slice(preds, func(i, j int) bool { return preds[i].Time.Before(preds[j].Time) })

	// ── Date column (always almDateW = 10 chars) ──────────────────────────
	// "Mon Jan _2" uses _2 (space-padded day) → consistent 10 chars.
	dateStr := day.Date.Format("Mon Jan _2")
	var dateCol string
	if isToday {
		dateCol = lipgloss.NewStyle().Foreground(S.T.Accent).Bold(true).Render(dateStr)
	} else {
		dateCol = S.AlmanacDate.Render(dateStr)
	}

	// ── Event columns (each exactly almEvtW = 16 visual chars) ───────────
	var tideStr strings.Builder
	var maxHigh, minLow float64
	hasHigh, hasLow := false, false

	for i := 0; i < numEvents; i++ {
		// 2-char leading separator
		tideStr.WriteString("  ")
		if i < len(preds) {
			p := preds[i]
			if p.IsHigh {
				if !hasHigh || p.Level > maxHigh {
					maxHigh = p.Level
					hasHigh = true
				}
				tideStr.WriteString(
					S.AlmanacHigh.Render("▲") + " " +
						S.Value.Render(FmtTideTime(p.Time)) + " " +
						S.AlmanacHigh.Render(fmt.Sprintf("%5.1f", p.Level)),
				)
			} else {
				if !hasLow || p.Level < minLow {
					minLow = p.Level
					hasLow = true
				}
				tideStr.WriteString(
					S.AlmanacLow.Render("▼") + " " +
						S.Value.Render(FmtTideTime(p.Time)) + " " +
						S.AlmanacLow.Render(fmt.Sprintf("%5.1f", p.Level)),
				)
			}
		} else {
			// Empty slot — fill with spaces to preserve alignment.
			// almEvtW - 2 (already wrote sep) = 14 chars of content.
			tideStr.WriteString(strings.Repeat(" ", almEvtW-2))
		}
	}

	// ── Range column ──────────────────────────────────────────────────────
	var rangeStr string
	if hasHigh && hasLow {
		rng := maxHigh - minLow
		rangeStr = " " + S.Label.Render(fmt.Sprintf("%5.1fft", rng)) + S.Label.Render(rangeBar(rng, maxRange))
	} else {
		rangeStr = strings.Repeat(" ", almRangeW)
	}

	// ── Moon column ───────────────────────────────────────────────────────
	var moonStr string
	if day.MoonName != "" {
		glyph := moonPhaseGlyph(day.MoonPhase)
		// Show name only on first occurrence — suppress repeated consecutive labels.
		showName := significantPhaseNames[day.MoonName] && day.MoonName != prevMoonName
		if showName {
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
	if rowIdx%2 == 1 {
		rowWidth := lipgloss.Width(row)
		if rowWidth < width-2 {
			row += strings.Repeat(" ", width-2-rowWidth)
		}
		return S.AlmanacAlt.Render(row)
	}
	return row
}

// rangeBar returns a 6-char proportional bar (space + 5 block chars) showing
// how rng compares to maxRange. Each ▓ = 20% of maxRange; remainder is ░.
func rangeBar(rng, maxRange float64) string {
	if maxRange <= 0 {
		return " ·····"
	}
	filled := int(math.Round(rng / maxRange * 5))
	if filled > 5 {
		filled = 5
	}
	return " " + strings.Repeat("█", filled) + strings.Repeat("·", 5-filled)
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
