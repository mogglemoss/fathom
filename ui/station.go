package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/mogglemoss/fathom/noaa"
)

// RenderStationView renders station metadata and sensor inventory.
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

	// ── Station ───────────────────────────────────────────────────────────
	b.WriteString(S.SectionHeader.Render("  STATION") + "\n")
	b.WriteString(metaRow("name", meta.Name))
	b.WriteString(metaRow("id", meta.ID))
	if meta.Lat != 0 || meta.Lon != 0 {
		coords := fmt.Sprintf("%.4f° N, %.4f° W", meta.Lat, -meta.Lon)
		if meta.Lon > 0 {
			coords = fmt.Sprintf("%.4f° N, %.4f° E", meta.Lat, meta.Lon)
		}
		b.WriteString(metaRow("coordinates", coords))
	}
	if meta.State != "" {
		b.WriteString(metaRow("state", meta.State))
	}
	if meta.TimeZone != "" {
		b.WriteString(metaRow("timezone", meta.TimeZone))
	}
	b.WriteString("\n")

	// ── Datums ────────────────────────────────────────────────────────────
	if len(meta.Datums) > 0 {
		b.WriteString(S.SectionHeader.Render("  DATUMS") + "\n")
		unitLabel := "ft"
		if units == "metric" {
			unitLabel = "m"
		}
		for _, d := range meta.Datums {
			b.WriteString(fmt.Sprintf("  %s  %s\n",
				S.Label.Render(fmt.Sprintf("%-8s", d.Name)),
				S.Value.Render(fmt.Sprintf("%.3f %s", d.Value, unitLabel)),
			))
		}
		b.WriteString("\n")
	}

	// ── Last update ───────────────────────────────────────────────────────
	if !lastUpdated.IsZero() {
		b.WriteString(S.SectionHeader.Render("  DATA") + "\n")
		b.WriteString(metaRow("last update", lastUpdated.Format("2006-01-02 15:04:05 MST")))
		b.WriteString(metaRow("units", units))
	}

	_ = width
	return b.String()
}

func metaRow(label, value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf("  %s  %s\n",
		S.Label.Render(fmt.Sprintf("%-12s", label)),
		S.Value.Render(value),
	)
}

// windDirName returns a compass direction name for the given degrees.
func windDirName(deg float64) string {
	dirs := []string{"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE", "S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW"}
	idx := int((deg+11.25)/22.5) % 16
	return dirs[idx]
}

// WindDirName is exported for use from other ui files.
func WindDirName(deg float64) string {
	return windDirName(deg)
}

// Ensure unused import is used
var _ = strings.Builder{}
