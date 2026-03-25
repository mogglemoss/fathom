package ui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/mogglemoss/fathom/noaa"
)

// RenderStatusBar renders the one-line top bar.
func RenderStatusBar(
	station noaa.StationMeta,
	currentLevel float64,
	hasLevel bool,
	errMsg string,
	lastUpdated time.Time,
	width int,
	refreshFlash bool,
) string {
	logo := S.StatusLogo.Render("◈") + S.StatusLogo.Render(" fathom")

	var metaPart string
	switch {
	case errMsg != "":
		metaPart = S.StatusBad.Render(errMsg)
	case !hasLevel:
		metaPart = S.StatusMeta.Render("reaching into the water…")
	default:
		stationName := station.Name
		if stationName == "" {
			stationName = station.ID
		}
		levelStr := fmt.Sprintf("%.2f ft", currentLevel)
		timeStr := ""
		if !lastUpdated.IsZero() {
			timeStr = "  ·  " + lastUpdated.Format("15:04")
		}
		metaPart = S.StatusMeta.Render(stationName+" · ") +
			S.TideLevel.Render(levelStr) +
			S.StatusMeta.Render(timeStr)
		if refreshFlash {
			metaPart += lipgloss.NewStyle().Foreground(S.T.Good).Render(" ◦")
		}
	}

	logoWidth := lipgloss.Width(logo)
	metaWidth := lipgloss.Width(metaPart)
	gap := width - logoWidth - metaWidth - 2 // 2 for padding
	if gap < 1 {
		gap = 1
	}

	bar := logo + fmt.Sprintf("%*s", gap, "") + metaPart
	return S.StatusBar.Width(width).Render(bar)
}
