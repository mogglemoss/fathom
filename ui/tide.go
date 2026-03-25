package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/mogglemoss/fathom/noaa"
)

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

	direction := ""
	dirLabel := ""
	if len(obs) >= 2 {
		prev := obs[len(obs)-2]
		switch {
		case current.Level > prev.Level:
			direction = "rising"
			dirLabel = S.TideRising.Render("  ▲ rising")
		case current.Level < prev.Level:
			direction = "falling"
			dirLabel = S.TideFalling.Render("  ▼ falling")
		default:
			dirLabel = S.StatusMeta.Render("  — steady")
		}
	}
	_ = direction // consumed by callers via RenderStatusBar; shown inline here too

	levelDisplay := fmt.Sprintf("  %.2f ft", current.Level)
	b.WriteString(S.TideLevel.Render(levelDisplay) + dirLabel + "\n")

	// ── Previous tide ─────────────────────────────────────────────────────
	prevHigh, prevLow := prevPredictions(preds)
	if prevHigh != nil || prevLow != nil {
		b.WriteString("  ")
		if prevLow != nil && (prevHigh == nil || prevLow.Time.After(prevHigh.Time)) {
			b.WriteString(S.Label.Render("last ") + S.TideLow.Render("LOW") +
				S.StatusMeta.Render(fmt.Sprintf("  %.1f ft  %s", prevLow.Level, prevLow.Time.Format("3:04 PM"))))
		} else if prevHigh != nil {
			b.WriteString(S.Label.Render("last ") + S.TideHigh.Render("HIGH") +
				S.StatusMeta.Render(fmt.Sprintf("  %.1f ft  %s", prevHigh.Level, prevHigh.Time.Format("3:04 PM"))))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// ── Next high / low predictions ───────────────────────────────────────
	nextHigh, nextLow := nextPredictions(preds)
	b.WriteString(renderPredictionRow(nextHigh, nextLow, width))
	b.WriteString("\n")

	// ── Water level fill chart ─────────────────────────────────────────────
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

// TideDirection returns "rising", "falling", or "steady" for the current obs.
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
		return S.TideLow.Render("LOW") + "  " + S.Label.Render("--")
	}

	label := "LOW "
	style := S.TideLow
	if isHigh {
		label = "HIGH"
		style = S.TideHigh
	}

	timeStr := p.Time.Format("3:04 PM")
	levelStr := fmt.Sprintf("%.1f ft", p.Level)

	return style.Render(label) + "  " +
		S.Value.Render(levelStr) + "  " +
		S.StatusMeta.Render(timeStr) + "  " +
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

// renderWaterFill renders a tall filled area chart of the 24h water level.
// It uses full-block (█) and half-block (▄) characters for smooth fill.
func renderWaterFill(obs []noaa.WaterObs, width, height int) string {
	if len(obs) == 0 {
		return S.Label.Render("  awaiting tide data…")
	}

	// Chart height: use available vertical space, capped generously
	chartHeight := height - 12
	if chartHeight < 4 {
		chartHeight = 4
	}
	if chartHeight > 16 {
		chartHeight = 16
	}

	levels := make([]float64, len(obs))
	for i, o := range obs {
		levels[i] = o.Level
	}

	minVal, maxVal, meanVal := stats(levels)

	// Scale to available width (leave 2 for left indent + label space)
	availWidth := width - 4
	if availWidth < 1 {
		availWidth = 1
	}
	if availWidth > len(levels) {
		availWidth = len(levels)
	}
	levels = levels[len(levels)-availWidth:]
	obsSlice := obs[len(obs)-availWidth:]

	var sb strings.Builder

	// Render rows from top (row=0) to bottom (row=chartHeight-1).
	// Each row covers a 0.5-unit band of the scaled height [0, chartHeight].
	// We use half-block resolution: each row has 2 half-cells.
	for row := 0; row < chartHeight; row++ {
		sb.WriteString("  ")

		for i, v := range levels {
			isLast := i == len(levels)-1

			// scaledHeight in [0, chartHeight]
			var scaledH float64
			if maxVal > minVal {
				scaledH = (v - minVal) / (maxVal - minVal) * float64(chartHeight)
			}

			// Distance from top: heightFromBottom = chartHeight - row
			// This row spans [chartHeight - row - 1, chartHeight - row)
			rowTop := float64(chartHeight - row)
			rowBot := float64(chartHeight - row - 1)

			var ch string
			switch {
			case scaledH >= rowTop:
				// Fully filled
				ch = "█"
			case scaledH > rowBot+0.5:
				// Upper half filled → show full block (looks better than ▀)
				ch = "▄"
			case scaledH > rowBot:
				// Lower quarter — small nub, use ▄ too
				ch = "▄"
			default:
				ch = " "
			}

			if ch == " " {
				sb.WriteString(" ")
				continue
			}

			above := v >= meanVal
			if isLast {
				sb.WriteString(S.SparkCursor.Render(ch))
			} else if above {
				if obsSlice[i].QC == "p" {
					// Preliminary: dimmer color
					sb.WriteString(S.SparkLow.Render(ch))
				} else {
					sb.WriteString(S.SparkHigh.Render(ch))
				}
			} else {
				sb.WriteString(S.SparkLow.Render(ch))
			}
		}

		if row < chartHeight-1 {
			sb.WriteString("\n")
		}
	}

	// Min/max labels below the chart
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

// windArrow returns an arrow pointing in the direction the wind is blowing toward.
// NOAA reports the direction the wind is coming FROM, so we add 180°.
func windArrow(fromDeg float64) string {
	toward := math.Mod(fromDeg+180, 360)
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
