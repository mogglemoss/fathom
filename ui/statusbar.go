package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/mogglemoss/fathom/noaa"
)

// RenderStatusBar renders the one-line top bar.
// activeView: 0=Tide, 1=Almanac, 2=Station
// direction: "rising", "falling", "steady", or ""
func RenderStatusBar(
	station noaa.StationMeta,
	currentLevel float64,
	hasLevel bool,
	direction string,
	errMsg string,
	lastUpdated time.Time,
	width int,
	refreshFlash bool,
	activeView int,
) string {
	// ── Left: logo + view dots ─────────────────────────────────────────────
	logo := S.StatusLogo.Render("◈") + S.StatusLogo.Render(" fathom")

	viewLabels := []string{"TIDE", "ALMANAC", "STATION"}
	var viewDots string
	for i, label := range viewLabels {
		if i > 0 {
			viewDots += S.HelpSep.Render("  ")
		}
		if i == activeView {
			viewDots += lipgloss.NewStyle().Foreground(S.T.Accent).Bold(true).Render("● " + label)
		} else {
			viewDots += S.StatusMeta.Render("○ " + label)
		}
	}

	left := logo + S.HelpSep.Render("   ") + viewDots

	// ── Right: station + level + time ─────────────────────────────────────
	var right string
	switch {
	case errMsg != "":
		right = S.StatusBad.Render(errMsg)
	case !hasLevel:
		right = S.StatusMeta.Render("reaching into the water…")
	default:
		stationName := station.Name
		if stationName == "" {
			stationName = station.ID
		}
		stateStr := ""
		if station.State != "" {
			stateStr = ", " + station.State
		}

		// Direction arrow
		dirStr := ""
		switch direction {
		case "rising":
			dirStr = " " + S.TideRising.Render("▲")
		case "falling":
			dirStr = " " + S.TideFalling.Render("▼")
		}

		levelStr := S.TideLevel.Render(fmt.Sprintf("%.2f ft", currentLevel)) + dirStr
		timeStr := ""
		if !lastUpdated.IsZero() {
			timeStr = "  " + S.StatusMeta.Render(lastUpdated.Format("15:04"))
		}

		stationStr := lipgloss.NewStyle().Foreground(S.T.Accent).Bold(true).Render(stationName+stateStr) +
			S.StatusMeta.Render("  ·  ")

		right = stationStr + levelStr + timeStr
		if refreshFlash {
			right += lipgloss.NewStyle().Foreground(S.T.Good).Render(" ◦")
		}
	}

	// ── Compose with gap ──────────────────────────────────────────────────
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	gap := width - leftWidth - rightWidth - 2 // 2 for padding
	if gap < 1 {
		gap = 1
	}

	bar := left + fmt.Sprintf("%*s", gap, "") + right
	return S.StatusBar.Width(width).Render(bar)
}
