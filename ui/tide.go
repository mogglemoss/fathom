package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	colorful "github.com/lucasb-eyer/go-colorful"

	"github.com/mogglemoss/fathom/noaa"
)

// blockChars are the 8 bottom-aligned vertical block characters, indexed 0 (▁) to 7 (█).
var blockChars = []rune("▁▂▃▄▅▆▇█")

// animFrames cycles for the "now" cursor star animation.
var animFrames = []string{"✦", "✦", "✧", "✧"}

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
//   - animFrame: animation frame counter for the cursor star
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
	animFrame int,
	width int,
	height int,
) string {
	// Chart overhead = pre-chart lines + post-chart lines.
	//   Today (no met): 5 pre  +  3 post  = 8
	//   Today (met):    5 pre  +  5 post  = 10  (met strip adds \n\n + text + \n = 2 extra)
	//   Non-today:      4 pre  +  3 post  = 7   (stale)
	//   Non-today non-stale: also 8 (gains chart-markers row)
	hasMet := isToday && (met.WindSpeed > 0 || met.AirTemp != 0 || met.AirPressure != 0)
	overhead := 8
	if hasMet {
		overhead = 10
	} else if !isToday && dayCurveStale {
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
					S.Label.Render(FmtTideTime(dayHigh.Time)))
			}
			if dayHigh != nil && dayLow != nil {
				b.WriteString(S.Label.Render("     "))
			}
			if dayLow != nil {
				b.WriteString(S.TideLow.Render("LOW ") + "  " +
					S.Value.Render(fmt.Sprintf("%.1f ft", dayLow.Level)) + "  " +
					S.Label.Render(FmtTideTime(dayLow.Time)))
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

	// Compute peak and trough column positions for the marker row.
	availWidthForMarkers := width - 4
	if availWidthForMarkers < 1 {
		availWidthForMarkers = 1
	}
	nForMarkers := len(dayCurve)
	if availWidthForMarkers > nForMarkers {
		availWidthForMarkers = nForMarkers
	}
	peakCol, troughCol := 0, 0
	if nForMarkers > 0 {
		for i := range dayCurve {
			if dayCurve[i].Level > dayCurve[peakCol].Level {
				peakCol = i
			}
			if dayCurve[i].Level < dayCurve[troughCol].Level {
				troughCol = i
			}
		}
		peakCol = peakCol * availWidthForMarkers / nForMarkers
		troughCol = troughCol * availWidthForMarkers / nForMarkers
	}

	// ── Row 4 (today) / Row 3 (non-today): now-marker or stale indicator ──
	// Today:     chart-markers row shows ▲ peak, ▼ trough, ▾ now; stale → loading hint instead.
	// Non-today: show peak/trough markers when not stale; stale → loading hint.
	if isToday && !dayCurveStale {
		b.WriteString(renderChartMarkers(nowFrac, true, peakCol, troughCol, availWidthForMarkers) + "\n")
	} else if !isToday && !dayCurveStale && len(dayCurve) > 0 {
		b.WriteString(renderChartMarkers(nowFrac, false, peakCol, troughCol, availWidthForMarkers) + "\n")
	} else if dayCurveStale {
		b.WriteString("  " + S.StatusMeta.Render(loadingMessage()) + "\n")
	}

	// ── The chart ─────────────────────────────────────────────────────────
	b.WriteString(renderDayCurve(dayCurve, isToday && !dayCurveStale, nowFrac, width, chartH, animFrame, dayCurveStale))

	// ── Time axis ─────────────────────────────────────────────────────────
	b.WriteString("\n")
	b.WriteString(renderTimeAxis(width, Use24h))

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

// renderChartMarkers renders the row above the chart with ▲ at peak, ▼ at trough,
// and ▾ at the current time (today only).
func renderChartMarkers(nowFrac float64, isToday bool, peakCol, troughCol, availWidth int) string {
	if availWidth <= 0 {
		return ""
	}

	row := make([]string, availWidth)

	if peakCol >= 0 && peakCol < availWidth {
		row[peakCol] = S.TideHigh.Render("▲")
	}
	if troughCol >= 0 && troughCol < availWidth {
		row[troughCol] = S.TideLow.Render("▼")
	}
	if isToday {
		nowCol := int(nowFrac * float64(availWidth))
		if nowCol >= availWidth {
			nowCol = availWidth - 1
		}
		if nowCol >= 0 {
			row[nowCol] = S.SparkCursor.Render("▾") // nowCol takes priority
		}
	}

	var sb strings.Builder
	sb.WriteString("  ")
	for _, c := range row {
		if c == "" {
			sb.WriteString(" ")
		} else {
			sb.WriteString(c)
		}
	}
	return sb.String()
}

// renderTimeAxis renders the time axis below the chart with 5 labeled positions.
func renderTimeAxis(width int, use24h bool) string {
	availWidth := width - 4
	if availWidth < 20 {
		if use24h {
			return "  " + S.Label.Render("00:00 ── 12:00 ── 24:00")
		}
		return "  " + S.Label.Render("12am ── 12pm ── 12am")
	}

	var labels [5]string
	if use24h {
		labels = [5]string{"00:00", "06:00", "12:00", "18:00", "24:00"}
	} else {
		labels = [5]string{"12am", "6am", "12pm", "6pm", "12am"}
	}

	// Target positions: 0%, 25%, 50%, 75%, 100% of availWidth.
	// Labels 1-3 are centered on their position; label 0 is left-aligned; label 4 is right-aligned.
	positions := [5]int{
		0,
		availWidth/4 - len(labels[1])/2,
		availWidth/2 - len(labels[2])/2,
		3*availWidth/4 - len(labels[3])/2,
		availWidth - len(labels[4]),
	}
	// Clamp: ensure no overlap and no out-of-bounds.
	for i := range positions {
		if positions[i] < 0 {
			positions[i] = 0
		}
		if i > 0 && positions[i] < positions[i-1]+len(labels[i-1])+1 {
			positions[i] = positions[i-1] + len(labels[i-1]) + 1
		}
		if positions[i]+len(labels[i]) > availWidth {
			positions[i] = availWidth - len(labels[i])
		}
	}

	buf := []byte(strings.Repeat(" ", availWidth))
	for i, label := range labels {
		pos := positions[i]
		for j := 0; j < len(label) && pos+j < availWidth; j++ {
			buf[pos+j] = label[j]
		}
	}
	return "  " + S.Label.Render(string(buf))
}

// renderDayCurve renders the full-day tide chart with height gradient, animated
// now-cursor, and optional MLLW zero-line.
func renderDayCurve(dayCurve []noaa.WaterObs, isToday bool, nowFrac float64, width, chartHeight, animFrame int, stale bool) string {
	if len(dayCurve) == 0 {
		return S.Label.Render("  awaiting tide curve…")
	}

	levels := make([]float64, len(dayCurve))
	for i, o := range dayCurve {
		levels[i] = o.Level
	}
	minVal, maxVal, _ := stats(levels)

	availWidth := width - 4
	if availWidth < 1 {
		availWidth = 1
	}
	n := len(levels)
	if availWidth > n {
		availWidth = n
	}

	// Catmull-Rom spline sampling: smoothly interpolate between source points
	// rather than snapping to the nearest index (removes staircase artifacts).
	sampled := make([]float64, availWidth)
	for i := range sampled {
		// Map column i to a fractional position in the source slice.
		srcFrac := float64(i) * float64(n-1) / math.Max(1, float64(availWidth-1))
		j := int(srcFrac)
		t := srcFrac - float64(j)
		// Clamp surrounding control-point indices.
		j0 := j - 1
		if j0 < 0 {
			j0 = 0
		}
		j1 := j
		j2 := j + 1
		if j2 >= n {
			j2 = n - 1
		}
		j3 := j + 2
		if j3 >= n {
			j3 = n - 1
		}
		sampled[i] = catmullRom(levels[j0], levels[j1], levels[j2], levels[j3], t)
	}

	nowCol := int(nowFrac * float64(availWidth))
	if nowCol >= availWidth {
		nowCol = availWidth - 1
	}

	const floorFrac = 0.5 / 8
	sh := make([]float64, availWidth)
	for i, v := range sampled {
		if maxVal > minVal {
			sh[i] = (v-minVal)/(maxVal-minVal)*float64(chartHeight)*(1-floorFrac) + floorFrac
		} else {
			sh[i] = float64(chartHeight) / 2
		}
	}

	// Gaussian blur (σ≈1, 5-point kernel) on the display heights to soften
	// any remaining quantisation edges without distorting the overall shape.
	if availWidth >= 3 {
		kernel := [5]float64{0.06, 0.24, 0.40, 0.24, 0.06}
		blurred := make([]float64, availWidth)
		for i := range sh {
			var sum, weight float64
			for k, w := range kernel {
				idx := i + k - 2
				if idx < 0 {
					idx = 0
				}
				if idx >= availWidth {
					idx = availWidth - 1
				}
				sum += w * sh[idx]
				weight += w
			}
			blurred[i] = sum / weight
		}
		copy(sh, blurred)
	}

	// Pre-compute gradient styles per row.
	// Row 0 = top (bright) → Row chartHeight-1 = bottom (dark).
	pastBotC, _ := colorful.Hex("#003459") // deep ocean navy
	pastTopC, _ := colorful.Hex("#00DFFF") // phosphor cyan
	futBotC, _ := colorful.Hex("#001828")  // midnight deep
	futTopC, _ := colorful.Hex("#005F80")  // ocean teal

	type rowStyle struct{ past, fut lipgloss.Style }
	rowStyles := make([]rowStyle, chartHeight)
	for r := 0; r < chartHeight; r++ {
		t := 1.0
		if chartHeight > 1 {
			t = float64(r) / float64(chartHeight-1)
		}
		pc := pastTopC.BlendHcl(pastBotC, t).Clamped()
		fc := futTopC.BlendHcl(futBotC, t).Clamped()
		rowStyles[r] = rowStyle{
			past: lipgloss.NewStyle().Foreground(lipgloss.Color(pc.Hex())),
			fut:  lipgloss.NewStyle().Foreground(lipgloss.Color(fc.Hex())),
		}
	}

	// Zero-line row: draw a dim ─ at the MLLW datum when it falls within the chart.
	zeroRow := -1
	if maxVal > minVal && minVal < 0 && maxVal > 0 {
		zeroFrac := -minVal / (maxVal - minVal)
		zr := chartHeight - 1 - int(zeroFrac*float64(chartHeight))
		if zr >= 0 && zr < chartHeight {
			zeroRow = zr
		}
	}

	// Animated star: which frame to show at the now-cursor tip.
	star := animFrames[animFrame%len(animFrames)]

	// Pre-compute the topmost row of the now-column bar.
	nowTopRow := -1
	if isToday && !stale && nowCol >= 0 {
		nowTopRow = chartHeight - 1 - int(sh[nowCol])
		if nowTopRow < 0 {
			nowTopRow = 0
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
			// Determine base style from gradient.
			var style lipgloss.Style
			if stale {
				style = rowStyles[row].fut
			} else if isToday {
				switch {
				case i == nowCol:
					style = S.SparkCursor
				case i < nowCol:
					style = rowStyles[row].past
				default:
					style = rowStyles[row].fut
				}
			} else {
				style = rowStyles[row].past
			}

			// Animated star at the top of the now-column.
			if isToday && !stale && i == nowCol && row == nowTopRow {
				sb.WriteString(S.SparkCursor.Render(star))
				continue
			}

			switch {
			case s >= rowTop:
				sb.WriteString(style.Render("█"))
			case s > rowBot:
				frac := s - rowBot
				// Round to nearest of 8 levels (0-7) rather than truncating,
				// so the transition between adjacent block chars is centred on
				// the actual height rather than always biased low.
				idx := int(math.Round(frac*8 - 0.5))
				if idx < 0 {
					idx = 0
				}
				if idx > 7 {
					idx = 7
				}
				sb.WriteString(style.Render(string(blockChars[idx])))
			default:
				if row == zeroRow {
					sb.WriteString(S.Label.Render("─"))
				} else {
					sb.WriteString(" ")
				}
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

// catmullRom interpolates a point at parameter t (0–1) between p1 and p2,
// using p0 and p3 as the outer control points (Catmull-Rom spline).
func catmullRom(p0, p1, p2, p3, t float64) float64 {
	t2 := t * t
	t3 := t2 * t
	return 0.5 * ((2*p1) +
		(-p0+p2)*t +
		(2*p0-5*p1+4*p2-p3)*t2 +
		(-p0+3*p1-3*p2+p3)*t3)
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
