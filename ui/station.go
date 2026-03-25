package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/mogglemoss/fathom/noaa"
)

// RenderStationView renders station metadata using horizontal two-column layout.
func RenderStationView(
	meta noaa.StationMeta,
	lastUpdated time.Time,
	units string,
	width int,
) string {
	if meta.ID == "" {
		return "\n  " + S.StatusMeta.Render("〰 loading station data…") + "\n"
	}

	var b strings.Builder
	b.WriteString("\n")

	// ── Prominent station header ──────────────────────────────────────────
	stationTitle := meta.Name
	if meta.State != "" {
		stationTitle += ", " + meta.State
	}
	titleStyle := lipgloss.NewStyle().Foreground(S.T.Accent).Bold(true)
	b.WriteString("  " + titleStyle.Render(stationTitle))
	b.WriteString("  " + S.StatusMeta.Render("Station "+meta.ID))
	b.WriteString("\n\n")

	// ── Two-column section: station info | datums ─────────────────────────
	halfW := (width - 4) / 2
	if halfW < 20 {
		halfW = 20
	}

	leftLines := buildLocationLines(meta, lastUpdated, units)
	rightLines := buildDatumLines(meta, units)

	// Pad both sides to equal length
	for len(leftLines) < len(rightLines) {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < len(leftLines) {
		rightLines = append(rightLines, "")
	}

	// Column separator
	sep := S.HelpSep.Render("  │  ")

	for i, l := range leftLines {
		// Left column: pad to halfW so separator aligns
		lw := lipgloss.Width(l)
		pad := halfW - lw
		if pad < 0 {
			pad = 0
		}
		b.WriteString("  " + l + strings.Repeat(" ", pad) + sep + rightLines[i] + "\n")
	}

	b.WriteString("\n")

	// ── Bottom bar: data freshness ────────────────────────────────────────
	if !lastUpdated.IsZero() {
		b.WriteString("  " + S.Label.Render("updated ") +
			S.Value.Render(lastUpdated.Format("2006-01-02 15:04 MST")) +
			S.HelpSep.Render("  ·  ") +
			S.Label.Render("units ") +
			S.Value.Render(units))
		b.WriteString("\n")
	}

	// ── Station search hint ───────────────────────────────────────────────
	b.WriteString("\n  " + S.HelpDesc.Render("press ") + S.HelpKey.Render("s") +
		S.HelpDesc.Render(" to search for a different station") + "\n")

	return b.String()
}

func buildLocationLines(meta noaa.StationMeta, lastUpdated time.Time, units string) []string {
	var lines []string
	lines = append(lines, S.SectionHeader.Render("LOCATION"))

	if meta.Lat != 0 || meta.Lon != 0 {
		var coords string
		if meta.Lon > 0 {
			coords = fmt.Sprintf("%.4f° N  %.4f° E", meta.Lat, meta.Lon)
		} else {
			coords = fmt.Sprintf("%.4f° N  %.4f° W", meta.Lat, -meta.Lon)
		}
		lines = append(lines, metaCell("coords", coords))
	}
	if meta.State != "" {
		lines = append(lines, metaCell("state", meta.State))
	}
	if meta.TimeZone != "" {
		lines = append(lines, metaCell("timezone", meta.TimeZone))
	}
	lines = append(lines, metaCell("datum", "MLLW"))

	return lines
}

func buildDatumLines(meta noaa.StationMeta, units string) []string {
	if len(meta.Datums) == 0 {
		return nil
	}

	unitLabel := "ft"
	if units == "metric" {
		unitLabel = "m"
	}

	var lines []string
	lines = append(lines, S.SectionHeader.Render("DATUMS"))

	for _, d := range meta.Datums {
		lines = append(lines, fmt.Sprintf("%s  %s",
			S.Label.Render(fmt.Sprintf("%-6s", d.Name)),
			S.Value.Render(fmt.Sprintf("%.3f %s", d.Value, unitLabel)),
		))
	}
	return lines
}

func metaCell(label, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("%s  %s",
		S.Label.Render(fmt.Sprintf("%-8s", label)),
		S.Value.Render(value),
	)
}
