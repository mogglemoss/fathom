package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mogglemoss/fathom/noaa"
)

const sparkChars = "▁▂▃▄▅▆▇█"

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
		return "\n  " + S.StatusMeta.Render("〰 reaching into the water…") + "\n"
	}

	var b strings.Builder

	// ── Current level ─────────────────────────────────────────────────────
	current := obs[len(obs)-1]
	b.WriteString("\n")

	// Rising or falling indicator
	direction := ""
	if len(obs) >= 2 {
		prev := obs[len(obs)-2]
		if current.Level > prev.Level {
			direction = S.TideRising.Render("  ▲ rising")
		} else if current.Level < prev.Level {
			direction = S.TideFalling.Render("  ▼ falling")
		} else {
			direction = S.StatusMeta.Render("  — steady")
		}
	}

	levelDisplay := fmt.Sprintf("  %.2f ft", current.Level)
	b.WriteString(S.TideLevel.Render(levelDisplay) + direction + "\n\n")

	// ── Next high / low predictions ───────────────────────────────────────
	nextHigh, nextLow := nextPredictions(preds)
	b.WriteString(renderPredictionRow(nextHigh, nextLow, width))
	b.WriteString("\n")

	// ── Sparkline ─────────────────────────────────────────────────────────
	b.WriteString(S.SectionHeader.Render("  24-HOUR WATER LEVEL") + "\n")
	b.WriteString(renderWaterSparkline(obs, width))
	b.WriteString("\n\n")

	// ── Met strip ─────────────────────────────────────────────────────────
	if met.WindSpeed > 0 || met.AirTemp != 0 || met.AirPressure != 0 {
		b.WriteString(renderMetStrip(met, width))
		b.WriteString("\n")
	}

	return b.String()
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
			return S.TideHigh.Render("HIGH  --")
		}
		return S.TideLow.Render("LOW   --")
	}

	label := "LOW "
	style := S.TideLow
	if isHigh {
		label = "HIGH"
		style = S.TideHigh
	}

	timeStr := p.Time.Format("3:04 PM")
	countdown := countdown(p.Time)
	levelStr := fmt.Sprintf("%.1f ft", p.Level)

	return style.Render(label) + "  " +
		S.Value.Render(levelStr) + "  " +
		S.StatusMeta.Render(timeStr) + "  " +
		S.Label.Render(countdown)
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

// renderWaterSparkline renders a single-row sparkline of the 24h water level.
func renderWaterSparkline(obs []noaa.WaterObs, width int) string {
	if len(obs) == 0 {
		return S.Label.Render("  awaiting tide data…")
	}

	levels := make([]float64, len(obs))
	for i, o := range obs {
		levels[i] = o.Level
	}

	minVal, maxVal, meanVal := stats(levels)

	// Scale to available width
	availWidth := width - 4
	if availWidth < 1 {
		availWidth = 1
	}
	if availWidth > len(levels) {
		availWidth = len(levels)
	}
	// Take the most recent availWidth observations
	levels = levels[len(levels)-availWidth:]
	obsSlice := obs[len(obs)-availWidth:]

	runes := []rune(sparkChars)
	var sb strings.Builder
	sb.WriteString("  ")

	for i, v := range levels {
		isLast := i == len(levels)-1

		idx := 0
		if maxVal > minVal {
			idx = int((v - minVal) / (maxVal - minVal) * float64(len(runes)-1))
		}
		if idx < 0 {
			idx = 0
		}
		if idx >= len(runes) {
			idx = len(runes) - 1
		}

		ch := string(runes[idx])
		if isLast {
			sb.WriteString(S.SparkCursor.Render(ch))
		} else if v >= meanVal {
			style := S.SparkHigh
			// Dim if data is preliminary
			if obsSlice[i].QC == "p" {
				style = S.SparkLow
			}
			sb.WriteString(style.Render(ch))
		} else {
			sb.WriteString(S.SparkLow.Render(ch))
		}
	}

	// Min/max labels
	sb.WriteString(fmt.Sprintf("  %s %s  %s %s",
		S.Label.Render("lo"),
		S.TideLow.Render(fmt.Sprintf("%.1f", minVal)),
		S.Label.Render("hi"),
		S.TideHigh.Render(fmt.Sprintf("%.1f", maxVal)),
	))

	return sb.String()
}

func renderMetStrip(met noaa.MetObs, width int) string {
	var parts []string

	if met.WindSpeed > 0 {
		arrow := windArrow(met.WindDir)
		windStr := fmt.Sprintf("%s %.0f kt", arrow, met.WindSpeed)
		if met.WindGust > met.WindSpeed+2 {
			windStr += fmt.Sprintf(" (gusts %.0f)", met.WindGust)
		}
		parts = append(parts, S.MetLabel.Render("wind ")+S.MetValue.Render(windStr))
	}

	if met.AirTemp != 0 {
		parts = append(parts, S.MetLabel.Render("air ")+S.MetValue.Render(fmt.Sprintf("%.0f°F", met.AirTemp)))
	}

	if met.AirPressure != 0 && width >= 60 {
		parts = append(parts, S.MetLabel.Render("mb ")+S.MetValue.Render(fmt.Sprintf("%.0f", met.AirPressure)))
	}

	if len(parts) == 0 {
		return ""
	}

	sep := S.HelpSep.Render("  ·  ")
	return "  " + strings.Join(parts, sep)
}

// windArrow returns an arrow pointing in the direction the wind is blowing toward.
// NOAA reports the direction the wind is coming FROM, so we add 180°.
func windArrow(fromDeg float64) string {
	// Convert "from" direction to "toward" direction
	toward := math.Mod(fromDeg+180, 360)
	// Quantize to 8 directions
	idx := int((toward+22.5)/45) % 8
	arrows := []string{"↑", "↗", "→", "↘", "↓", "↙", "←", "↖"}
	return arrows[idx]
}

func stats(v []float64) (min, max, mean float64) {
	if len(v) == 0 {
		return
	}
	min = v[0]
	max = v[0]
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
