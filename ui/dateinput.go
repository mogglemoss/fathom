package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// RenderDateInput renders the "go to date" overlay with a bordered frame.
// It replaces the body content when the user presses 'd'.
func RenderDateInput(input, errMsg string, width, height int) string {
	cursor := S.SparkCursor.Render("█")

	var errLine string
	if errMsg != "" {
		errLine = "\n" + S.TideFalling.Render("⚠  "+errMsg)
	}

	inner := S.SectionHeader.Render("▸ GO TO DATE") + "\n\n" +
		S.Label.Render("Type a date and press Enter") + "\n\n" +
		S.Value.Render("→ "+input) + cursor +
		"\n\n" +
		S.StatusMeta.Render("e.g.  Oct 11  ·  Oct 11 2025  ·  2025-10-11") +
		errLine +
		"\n\n" +
		S.HelpDesc.Render("press ") + S.HelpKey.Render("Enter") +
		S.HelpDesc.Render(" to jump  ·  ") + S.HelpKey.Render("Esc") +
		S.HelpDesc.Render(" to cancel")

	frameW := width - 8
	if frameW < 30 {
		frameW = 30
	}
	if frameW > 60 {
		frameW = 60
	}

	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(string(S.T.Accent))).
		Padding(1, 3).
		Width(frameW).
		Render(inner)

	// Center the frame horizontally.
	return lipgloss.NewStyle().
		Width(width).
		Align(lipgloss.Center).
		MarginTop(4).
		Render(frame)
}
