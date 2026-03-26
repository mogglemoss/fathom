package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/mogglemoss/fathom/noaa"
)

// blockChars are the 8 bottom-aligned vertical block characters, indexed 0 (▁) to 7 (█).
var blockChars = []rune("▁▂▃▄▅▆▇█")

var loadingMessages = []string{
	"reaching into the water…",
	"consulting the buoy…",
	"asking the harbor master…",
	"reading the tide gauge…",
	"listening to the waves…",
	"checking the swell…",
	"calibrating the sensor…",
	"polling the ocean…",
}

func loadingMessage() string {
	idx := int(time.Now().Unix()/3) % len(loadingMessages)
	return "〰 " + loadingMessages[idx]
}

// RenderTideView renders the main tide dashboard.
//
// Parameters:
//   - obs: live water level observations (used for current level + direction)
//   - preds: hi/lo tide predictions (for next/prev tide cards)
//   - met: meteorological data (wind, temp, pressure)
//   - dayCurve: 6-minute interval predictions for viewDate (the chart data; may be stale)
//   - dayCurveStale: true when dayCurve is from a previous date (new fetch in-flight)
//   - viewDate: the day being displayed (zero = today)
//   - isToday: whether viewDate is today (precomputed by model)
//   - nowFrac: fraction of day elapsed [0,1] — used to mark "now" on chart
//   - width, height: terminal dimensions for this view
func RenderTideView(
	obs []noaa.WaterObs,
	preds []noaa.Prediction,
	met noaa.MetObs,
	dayCurve []noaa.WaterObs,
	dayCurveStale bool,
	viewDate time.Time,
	isToday bool,
	nowFrac float64,
	width int,
	height int,
) string {
	// Chart overhead = pre-chart lines + post-chart lines.
	//   Today (no met): 5 pre  +  3 post  = 8
	//   Today (met):    5 pre  +  5 post  = 10  (met strip adds \n\n + text + \n = 2 extra)
	//   Non-today:      4 pre  +  3 post  = 7
	hasMet := isToday && (met.WindSpeed > 0 || met.AirTemp != 0 || met.AirPressure != 0)
	overhead := 8
	if hasMet {
		overhead = 10
	} else if !isToday {
		overhead = 7
	}
	chartH := height - overhead
	if chartH < 4 {
		chartH = 4
	}
	if chartH > 28 {
		chartH = 28
	}

	var b strings.Builder
	b.WriteString("\n")

	if isToday {
		// ── Current level + direction + next tides (today only) ───────────
		if len(obs) == 0 {
			b.WriteString("  " + S.StatusMeta.Render(loadingMessage()) + "\n")
		} else {
			current := obs[len(obs)-1]
			dirLabel := ""
			if len(obs) >= 2 {
				prev := obs[len(obs)-2]
				switch {
				case current.Level > prev.Level:
					dirLabel = S.TideRising.Render("  ▲ rising")
				case current.Level < prev.Level:
					dirLabel = S.TideFalling.Render("  ▼ falling")
				default:
					dirLabel = S.StatusMeta.Render("  — steady")
				}
			}
			b.WriteString(S.TideLevel.Render(fmt.Sprintf("  %.2f ft", current.Level)) + dirLabel)

			// Next HIGH and LOW on the same line, right-padded
			nextHigh, nextLow := nextPredictions(preds)
			if nextHigh != nil || nextLow != nil {
				sep := S.Label.Render("    ")
				b.WriteString(sep + renderPredictionCompact(nextHigh, true) +
					S.Label.Render("  ·  ") + renderPredictionCompact(nextLow, false))
			}
			b.WriteString("\n")
		}
	} else {
		// ── Date navigation row (non-today) ───────────────────────────────
		b.WriteString(renderDateNav(viewDate, false, width) + "\n")

		// ── Day tide predictions (HIGH/LOW for that date) ──────────────────
		dayHigh, dayLow := predictionsForDate(preds, viewDate)
		if dayHigh != nil || dayLow != nil {
			b.WriteString("  ")
			if dayHigh != nil {
				b.WriteString(S.TideHigh.Render("HIGH") + "  " +
					S.Value.Render(fmt.Sprintf("%.1f ft", dayHigh.Level)) + "  " +
					S.Label.Render(fmtTideTime(dayHigh.Time)))
			}
			if dayHigh != nil && dayLow != nil {
				b.WriteString(S.Label.Render("     "))
			}
			if dayLow != nil {
				b.WriteString(S.TideLow.Render("LOW ") + "  " +
					S.Value.Render(fmt.Sprintf("%.1f ft", dayLow.Level)) + "  " +
					S.Label.Render(fmtTideTime(dayLow.Time)))
			}
			b.WriteString("\n")
		} else {
			b.WriteString("\n")
		}
	}

	// Blank separator before the chart section.
	// For non-today + stale, we skip this blank because the stale loading
	// indicator will occupy that same row (keeping overhead at 7).
	if isToday || !dayCurveStale || len(dayCurve) == 0 {
		b.WriteString("\n")
	}

	// ── Date navigation (today) ───────────────────────────────────────────
	if isToday {
		b.WriteString(renderDateNav(viewDate, true, width) + "\n")
	}

	if len(dayCurve) == 0 {
		// No data yet (initial load or station switch) — plain loading state.
		b.WriteString("\n  " + S.StatusMeta.Render(loadingMessage()) + "\n")
		return b.String()
	}

	// ── Row 4 (today) / Row 3 (non-today): now-marker or stale indicator ──
	// Today:     now-marker shows "now" position; stale → loading hint instead.
	// Non-today: stale → loading hint in the blank slot; otherwise nothing here.
	if isToday && !dayCurveStale {
		b.WriteString(renderNowMarker(nowFrac, dayCurve, width) + "\n")
	} else if dayCurveStale {
		b.WriteString("  " + S.StatusMeta.Render(loadingMessage()) + "\n")
	}

	// ── The chart ─────────────────────────────────────────────────────────
	b.WriteString(renderDayCurve(dayCurve, isToday && !dayCurveStale, nowFrac, width, chartH, dayCurveStale))

	// ── Time axis ─────────────────────────────────────────────────────────
	b.WriteString("\n")
	b.WriteString(renderTimeAxis(width))

	// ── Level stats ───────────────────────────────────────────────────────
	b.WriteString("\n")
	if len(dayCurve) > 0 {
		levels := make([]float64, len(dayCurve))
		for i, o := range dayCurve {
			levels[i] = o.Level
		}
		minV, maxV, _ := stats(levels)
		b.WriteString(fmt.Sprintf("  %s %s  %s %s",
			S.Label.Render("lo"),
			S.TideLow.Render(fmt.Sprintf("%.1f ft", minV)),
			S.Label.Render("hi"),
			S.TideHigh.Render(fmt.Sprintf("%.1f ft", maxV)),
		))
	}

	// ── Met strip (today only) ─────────────────────────────────────────────
	if isToday && (met.WindSpeed > 0 || met.AirTemp != 0 || met.AirPressure != 0) {
		b.WriteString("\n\n")
		b.WriteString(renderMetStrip(met, width))
	}

	b.WriteString("\n")
	return b.String()
}

// renderDateNav renders the date navigation row with ← prev  ·  date  ·  next →.
// isToday highlights the center with "◈ TODAY" badge.
// The ← zone is always on the left, → zone always on the right — both are mouse-clickable.
func renderDateNav(viewDate time.Time, isToday bool, width int) string {
	prev := viewDate.AddDate(0, 0, -1)
	next := viewDate.AddDate(0, 0, 1)

	leftPart := S.Label.Render("  ← " + prev.Format("Jan _2") + "  ")
	rightPart := S.Label.Render("  " + next.Format("Jan _2") + " →  ")

	var center string
	if isToday {
		center = S.SparkCursor.Render("◈ TODAY") +
			S.Label.Render("  ") +
			S.Value.Render(viewDate.Format("Mon Jan _2"))
	} else {
		center = S.SectionHeader.Render(viewDate.Format("Mon Jan _2, 2006"))
	}

	lw := lipgloss.Width(leftPart)
	rw := lipgloss.Width(rightPart)
	cw := lipgloss.Width(center)
	available := width - lw - rw
	if available < cw+2 {
		return leftPart + center + rightPart
	}
	leftPad := (available - cw) / 2
	rightPad := available - cw - leftPad
	return leftPart + strings.Repeat(" ", leftPad) + center + strings.Repeat(" ", rightPad) + rightPart
}

// renderNowMarker renders a row with ▾ at the current time column.
func renderNowMarker(nowFrac float64, dayCurve []noaa.WaterObs, width int) string {
	availWidth := width - 4
	if availWidth < 1 {
		availWidth = 1
	}
	n := len(dayCurve)
	if availWidth > n {
		availWidth = n
	}

	nowCol := int(nowFrac * float64(availWidth))
	if nowCol >= availWidth {
		nowCol = availWidth - 1
	}
	return "  " + strings.Repeat(" ", nowCol) + S.SparkCursor.Render("▾")
}

// renderTimeAxis renders the midnight / lunch / midnight row below the chart.
func renderTimeAxis(width int) string {
	availWidth := width - 4
	if availWidth < 20 {
		return "  " + S.Label.Render("midnight ── lunch ── midnight")
	}

	const (
		left   = "midnight"
		middle = "lunch"
		right  = "midnight"
	)

	midPos := availWidth/2 - len(middle)/2
	rightPos := availWidth - len(right)
	if midPos < len(left)+1 {
		midPos = len(left) + 1
	}
	if rightPos < midPos+len(middle)+1 {
		rightPos = midPos + len(middle) + 1
	}

	axis := left
	for len(axis) < midPos {
		axis += " "
	}
	axis += middle
	for len(axis) < rightPos {
		axis += " "
	}
	axis += right

	return "  " + S.Label.Render(axis)
}

// renderDayCurve renders the full-day (midnight-to-midnight) tide chart.
//
// The chart uses blockChars (▁▂▃▄▅▆▇█) for 8× vertical precision.
// For today, columns are colored by temporal position:
//   - past: SparkHigh (above mean) or SparkLow (below mean) — vivid
//   - current: SparkCursor — bright glow
//   - future: SparkFuture — dim, like peering into dark water
//
// For other days, all columns use SparkHigh/SparkLow based on mean.
func renderDayCurve(dayCurve []noaa.WaterObs, isToday bool, nowFrac float64, width, chartHeight int, stale bool) string {
	if len(dayCurve) == 0 {
		return S.Label.Render("  awaiting tide curve…")
	}

	levels := make([]float64, len(dayCurve))
	for i, o := range dayCurve {
		levels[i] = o.Level
	}
	minVal, maxVal, meanVal := stats(levels)

	availWidth := width - 4
	if availWidth < 1 {
		availWidth = 1
	}

	// Scale data columns to terminal width.
	n := len(levels)
	if availWidth > n {
		availWidth = n
	}

	// Sample uniformly if more data points than columns.
	sampled := make([]float64, availWidth)
	for i := range sampled {
		srcIdx := i * n / availWidth
		sampled[i] = levels[srcIdx]
	}

	// Compute nowCol: which column represents "now".
	nowCol := int(nowFrac * float64(availWidth))
	if nowCol >= availWidth {
		nowCol = availWidth - 1
	}

	// Compute scaled heights [0, chartHeight] with floor so min is never invisible.
	const floorFrac = 0.5 / 8
	sh := make([]float64, availWidth)
	for i, v := range sampled {
		if maxVal > minVal {
			sh[i] = (v-minVal)/(maxVal-minVal)*float64(chartHeight)*(1-floorFrac) + floorFrac
		} else {
			sh[i] = float64(chartHeight) / 2
		}
	}

	var sb strings.Builder

	for row := 0; row < chartHeight; row++ {
		if row > 0 {
			sb.WriteString("\n  ")
		} else {
			sb.WriteString("  ")
		}

		rowBot := float64(chartHeight - row - 1)
		rowTop := float64(chartHeight - row)

		for i, s := range sh {
			above := sampled[i] >= meanVal

			// Determine style by temporal zone.
			// Stale data (wrong date, fetch in-flight) always uses dim SparkFuture.
			var style lipgloss.Style
			if stale {
				style = S.SparkFuture
			} else if isToday {
				switch {
				case i == nowCol:
					style = S.SparkCursor
				case i < nowCol:
					if above {
						style = S.SparkHigh
					} else {
						style = S.SparkLow
					}
				default:
					style = S.SparkFuture
				}
			} else {
				if above {
					style = S.SparkHigh
				} else {
					style = S.SparkLow
				}
			}

			switch {
			case s >= rowTop:
				sb.WriteString(style.Render("█"))
			case s > rowBot:
				sRender := s
				// Dither: on shallow slopes, alternate odd columns half a step up
				// to soften visible stairstepping on nearly-flat sections.
				if i > 0 && math.Abs(s-sh[i-1]) < 0.5 && i%2 == 1 {
					sRender = math.Min(rowTop-0.001, s+0.0625)
				}
				frac := sRender - rowBot
				idx := int(frac * 8)
				if idx < 0 {
					idx = 0
				}
				if idx > 7 {
					idx = 7
				}
				sb.WriteString(style.Render(string(blockChars[idx])))
			default:
				sb.WriteString(" ")
			}
		}
	}

	return sb.String()
}

// TideDirection returns "rising", "falling", or "steady" for the current obs slice.
func TideDirection(obs []noaa.WaterObs) string {
	if len(obs) < 2 {
		return ""
	}
	current := obs[len(obs)-1]
	prev := obs[len(obs)-2]
	switch {
	case current.Level > prev.Level:
		return "rising"
	case current.Level < prev.Level:
		return "falling"
	default:
		return "steady"
	}
}

// nextPredictions returns the next upcoming high and low tide predictions.
func nextPredictions(preds []noaa.Prediction) (high, low *noaa.Prediction) {
	now := time.Now()
	for i := range preds {
		p := &preds[i]
		if p.Time.Before(now) {
			continue
		}
		if p.IsHigh && high == nil {
			high = p
		}
		if !p.IsHigh && low == nil {
			low = p
		}
		if high != nil && low != nil {
			break
		}
	}
	return
}

// predictionsForDate returns the first HIGH and LOW prediction for a given date.
func predictionsForDate(preds []noaa.Prediction, date time.Time) (high, low *noaa.Prediction) {
	// Match by calendar date (year/month/day)
	for i := range preds {
		p := &preds[i]
		pt := p.Time
		if pt.Year() == date.Year() && pt.Month() == date.Month() && pt.Day() == date.Day() {
			if p.IsHigh && high == nil {
				high = p
			}
			if !p.IsHigh && low == nil {
				low = p
			}
			if high != nil && low != nil {
				break
			}
		}
	}
	return
}

// renderPredictionCompact renders a single tide prediction in compact form.
func renderPredictionCompact(p *noaa.Prediction, isHigh bool) string {
	if p == nil {
		if isHigh {
			return S.TideHigh.Render("HIGH") + " " + S.Label.Render("--")
		}
		return S.TideLow.Render("LOW") + " " + S.Label.Render("--")
	}

	if isHigh {
		return S.TideHigh.Render("HIGH") + "  " +
			S.Value.Render(fmt.Sprintf("%.1f ft", p.Level)) + "  " +
			S.Label.Render(countdown(p.Time))
	}
	return S.TideLow.Render("LOW") + "  " +
		S.Value.Render(fmt.Sprintf("%.1f ft", p.Level)) + "  " +
		S.Label.Render(countdown(p.Time))
}

func countdown(t time.Time) string {
	d := time.Until(t).Round(time.Minute)
	if d < 0 {
		return ""
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("in %dh %dm", h, m)
	}
	return fmt.Sprintf("in %dm", m)
}

func renderMetStrip(met noaa.MetObs, width int) string {
	var parts []string

	if met.WindSpeed > 0 {
		arrow := windArrow(met.WindDir)
		windStr := fmt.Sprintf("%s %.0f kt", arrow, met.WindSpeed)
		if met.WindGust > met.WindSpeed+2 {
			windStr += fmt.Sprintf(" gusts %.0f", met.WindGust)
		}
		parts = append(parts, S.MetValue.Render(windStr))
	}

	if met.AirTemp != 0 {
		parts = append(parts, S.MetValue.Render(fmt.Sprintf("%.0f°F", met.AirTemp)))
	}

	if met.AirPressure != 0 && width >= 50 {
		parts = append(parts, S.MetValue.Render(fmt.Sprintf("%.0f mb", met.AirPressure)))
	}

	if len(parts) == 0 {
		return ""
	}

	sep := S.HelpSep.Render("  ·  ")
	return "  " + strings.Join(parts, sep)
}

// windArrow converts a "from" bearing to a "toward" bearing and returns an arrow.
func windArrow(fromDeg float64) string {
	toward := math.Mod(fromDeg+180, 360)
	idx := int((toward+22.5)/45) % 8
	return []string{"↑", "↗", "→", "↘", "↓", "↙", "←", "↖"}[idx]
}

func stats(v []float64) (min, max, mean float64) {
	if len(v) == 0 {
		return
	}
	min, max = v[0], v[0]
	sum := 0.0
	for _, x := range v {
		if x < min {
			min = x
		}
		if x > max {
			max = x
		}
		sum += x
	}
	mean = sum / float64(len(v))
	return
}
