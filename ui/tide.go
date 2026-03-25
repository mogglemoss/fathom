package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

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
func RenderTideView(
	obs []noaa.WaterObs,
	preds []noaa.Prediction,
	met noaa.MetObs,
	width int,
	height int,
	loading bool,
) string {
	if loading || len(obs) == 0 {
		return "\n  " + S.StatusMeta.Render(loadingMessage()) + "\n"
	}

	var b strings.Builder

	// ── Current level ─────────────────────────────────────────────────────
	current := obs[len(obs)-1]
	b.WriteString("\n")

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

	b.WriteString(S.TideLevel.Render(fmt.Sprintf("  %.2f ft", current.Level)) + dirLabel + "\n")

	// ── Previous tide ─────────────────────────────────────────────────────
	prevHigh, prevLow := prevPredictions(preds)
	if prevHigh != nil || prevLow != nil {
		b.WriteString("  ")
		var prev *noaa.Prediction
		isHigh := true
		if prevLow != nil && (prevHigh == nil || prevLow.Time.After(prevHigh.Time)) {
			prev = prevLow
			isHigh = false
		} else {
			prev = prevHigh
		}
		if isHigh {
			b.WriteString(S.Label.Render("last ") + S.TideHigh.Render("HIGH") +
				S.StatusMeta.Render(fmt.Sprintf("  %.1f ft  %s", prev.Level, prev.Time.Format("3:04 PM"))))
		} else {
			b.WriteString(S.Label.Render("last ") + S.TideLow.Render("LOW") +
				S.StatusMeta.Render(fmt.Sprintf("  %.1f ft  %s", prev.Level, prev.Time.Format("3:04 PM"))))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// ── Next high / low predictions ───────────────────────────────────────
	nextHigh, nextLow := nextPredictions(preds)
	b.WriteString(renderPredictionRow(nextHigh, nextLow, width))
	b.WriteString("\n")

	// ── Water level chart ─────────────────────────────────────────────────
	b.WriteString(S.SectionHeader.Render("  24-HOUR WATER LEVEL") + "\n")
	b.WriteString(renderWaterFill(obs, width, height))
	b.WriteString("\n\n")

	// ── Met strip ─────────────────────────────────────────────────────────
	if met.WindSpeed > 0 || met.AirTemp != 0 || met.AirPressure != 0 {
		b.WriteString(renderMetStrip(met, width))
		b.WriteString("\n")
	}

	return b.String()
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

// prevPredictions returns the most recent past high and low tide predictions.
func prevPredictions(preds []noaa.Prediction) (high, low *noaa.Prediction) {
	now := time.Now()
	for i := len(preds) - 1; i >= 0; i-- {
		p := &preds[i]
		if p.Time.After(now) {
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

func renderPredictionRow(high, low *noaa.Prediction, width int) string {
	highStr := renderPrediction(high, true)
	lowStr := renderPrediction(low, false)
	sep := S.HelpSep.Render("    ")
	_ = width
	return "  " + highStr + sep + lowStr + "\n"
}

func renderPrediction(p *noaa.Prediction, isHigh bool) string {
	if p == nil {
		if isHigh {
			return S.TideHigh.Render("HIGH") + "  " + S.Label.Render("--")
		}
		return S.TideLow.Render("LOW ") + "  " + S.Label.Render("--")
	}

	label := "LOW "
	style := S.TideLow
	if isHigh {
		label = "HIGH"
		style = S.TideHigh
	}

	return style.Render(label) + "  " +
		S.Value.Render(fmt.Sprintf("%.1f ft", p.Level)) + "  " +
		S.StatusMeta.Render(p.Time.Format("3:04 PM")) + "  " +
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

// renderWaterFill renders a smooth area fill chart of the 24h water level.
//
// Each character column uses one of the 8 bottom-aligned block characters
// (▁▂▃▄▅▆▇█) for the topmost visible row, giving 8× vertical precision.
// Fully-filled rows use the solid block █. This produces a smooth wave
// silhouette without the jarring staircase of equal-height bars.
func renderWaterFill(obs []noaa.WaterObs, width, height int) string {
	if len(obs) == 0 {
		return S.Label.Render("  awaiting tide data…")
	}

	chartHeight := height - 12
	if chartHeight < 4 {
		chartHeight = 4
	}
	if chartHeight > 18 {
		chartHeight = 18
	}

	levels := make([]float64, len(obs))
	for i, o := range obs {
		levels[i] = o.Level
	}

	minVal, maxVal, meanVal := stats(levels)

	// Scale to available terminal width.
	availWidth := width - 4
	if availWidth < 1 {
		availWidth = 1
	}
	if availWidth > len(levels) {
		availWidth = len(levels)
	}
	levels = levels[len(levels)-availWidth:]
	obsSlice := obs[len(obs)-availWidth:]

	// Compute scaled height per column in char units [0, chartHeight].
	// We add a small floor so the minimum level is never invisible.
	sh := make([]float64, len(levels))
	const floorFrac = 0.5 / 8 // always show at least ▁ at minimum
	for i, v := range levels {
		if maxVal > minVal {
			sh[i] = (v-minVal)/(maxVal-minVal)*float64(chartHeight)*(1-floorFrac) + floorFrac
		} else {
			sh[i] = float64(chartHeight) / 2
		}
	}

	var sb strings.Builder

	// Render rows top to bottom.
	// Row 0 = top. For a fill chart, the wave starts at the bottom (chartHeight-1).
	for row := 0; row < chartHeight; row++ {
		if row > 0 {
			sb.WriteString("\n  ")
		} else {
			sb.WriteString("  ")
		}

		// This character row covers the range [rowBot, rowTop) in scaled char units
		// measured from the bottom.
		rowBot := float64(chartHeight - row - 1)
		rowTop := float64(chartHeight - row)

		for i, s := range sh {
			isLast := i == len(sh)-1
			above := levels[i] >= meanVal
			isPrelim := obsSlice[i].QC == "p"

			var rendered string
			switch {
			case s >= rowTop:
				// Fully filled — solid block
				ch := "█"
				if isLast {
					rendered = S.SparkCursor.Render(ch)
				} else if above && !isPrelim {
					rendered = S.SparkHigh.Render(ch)
				} else {
					rendered = S.SparkLow.Render(ch)
				}

			case s > rowBot:
				// Partial fill at the wave crest — choose the right block character.
				frac := s - rowBot // 0..1
				idx := int(frac * 8)
				if idx < 0 {
					idx = 0
				}
				if idx > 7 {
					idx = 7
				}
				ch := string(blockChars[idx])
				// Crest characters glow in accent/cursor color for visual pop.
				if isLast {
					rendered = S.SparkCursor.Render(ch)
				} else if above && !isPrelim {
					rendered = S.SparkHigh.Render(ch)
				} else {
					rendered = S.SparkLow.Render(ch)
				}

			default:
				rendered = " "
			}

			sb.WriteString(rendered)
		}
	}

	// Level labels below the chart.
	sb.WriteString(fmt.Sprintf("\n  %s %s  %s %s",
		S.Label.Render("lo"),
		S.TideLow.Render(fmt.Sprintf("%.1f ft", minVal)),
		S.Label.Render("hi"),
		S.TideHigh.Render(fmt.Sprintf("%.1f ft", maxVal)),
	))

	return sb.String()
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
