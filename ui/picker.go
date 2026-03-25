package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/mogglemoss/fathom/noaa"
)

// RenderStationPicker renders the station-search overlay panel.
// It is shown instead of the normal body when the user presses 's'.
//
// input       – current text the user has typed
// nearby      – list of nearby stations (may be empty while loading)
// cursor      – index of highlighted row in nearby list (-1 = none)
// loading     – true while nearby stations are being fetched
// currentID   – the station that is currently active (marked with ●)
func RenderStationPicker(
	input string,
	nearby []noaa.NearbyStation,
	cursor int,
	loading bool,
	currentID string,
	width, height int,
) string {
	var b strings.Builder
	b.WriteString("\n")

	// ── Title ─────────────────────────────────────────────────────────────
	b.WriteString("  " + S.SectionHeader.Render("STATION SEARCH") + "\n\n")

	// ── Text input ────────────────────────────────────────────────────────
	prompt := S.Label.Render("station ID  ") + S.AccentStyle().Render("> ")
	var inputDisplay string
	if input == "" {
		inputDisplay = S.StatusMeta.Render("type an ID  –or–  select from list below")
	} else {
		inputDisplay = S.TideLevel.Render(input) + S.SparkCursor.Render("█")
	}
	b.WriteString("  " + prompt + inputDisplay + "\n\n")

	// ── Nearby stations ───────────────────────────────────────────────────
	if loading {
		b.WriteString("  " + S.StatusMeta.Render("〰 finding nearby stations…") + "\n")
	} else if len(nearby) > 0 {
		b.WriteString("  " + S.Label.Render("NEARBY STATIONS") + "\n\n")

		// Compute column widths from data
		maxNameW := 20
		for _, s := range nearby {
			n := len(s.Name)
			if s.State != "" {
				n += 2 + len(s.State)
			}
			if n > maxNameW {
				maxNameW = n
			}
		}
		if maxNameW > 36 {
			maxNameW = 36
		}

		for i, s := range nearby {
			name := s.Name
			if s.State != "" {
				name += ", " + s.State
			}

			// Marker: ● for active station
			var marker string
			if s.ID == currentID {
				marker = S.TideRising.Render("●")
			} else {
				marker = S.StatusMeta.Render("○")
			}

			dist := fmt.Sprintf("%.0f km", s.DistKm)
			if s.DistKm < 1 {
				dist = "<1 km"
			}

			nameField := fmt.Sprintf("%-*s", maxNameW, name)
			line := marker + "  " + nameField + "  " +
				S.StatusMeta.Render(dist) +
				S.HelpSep.Render("  ") +
				S.Label.Render(s.ID)

			if i == cursor {
				// Selection highlight
				lineWidth := lipgloss.Width(line)
				pad := width - 6 - lineWidth
				if pad < 0 {
					pad = 0
				}
				line = "  " + S.AlmanacCursor.Render(" "+line+strings.Repeat(" ", pad)) + "\n"
			} else {
				line = "  " + line + "\n"
			}
			b.WriteString(line)
		}
	} else {
		b.WriteString("  " + S.StatusMeta.Render("no nearby station data available") + "\n")
	}

	b.WriteString("\n")

	// ── Key hints ─────────────────────────────────────────────────────────
	sep := S.HelpSep.Render("  ·  ")
	hints := S.HelpKey.Render("j/k") + S.HelpDesc.Render(" navigate") +
		sep + S.HelpKey.Render("enter") + S.HelpDesc.Render(" select") +
		sep + S.HelpKey.Render("esc") + S.HelpDesc.Render(" cancel")
	b.WriteString("  " + hints + "\n")

	return b.String()
}

// AccentStyle returns a one-off accent-colored style (avoids adding to Styles struct).
func (s Styles) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(s.T.Accent)
}
