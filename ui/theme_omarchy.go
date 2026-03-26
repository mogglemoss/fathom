package ui

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// LoadOmarchyTheme attempts to read the active Omarchy theme from
// ~/.config/omarchy/current/theme/colors.toml and maps it to a fathom Theme.
// Returns (theme, true) on success, (zero, false) if Omarchy isn't present.
func LoadOmarchyTheme() (Theme, bool) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Theme{}, false
	}

	themeDir := filepath.Join(home, ".config", "omarchy", "current", "theme")
	data, err := os.ReadFile(filepath.Join(themeDir, "colors.toml"))
	if err != nil {
		return Theme{}, false
	}

	c := parseOmarchyColors(data)
	if c["foreground"] == "" || c["color1"] == "" {
		return Theme{}, false
	}

	// Light mode is signalled by an empty light.mode file in the theme dir.
	_, isLight := os.Stat(filepath.Join(themeDir, "light.mode"))
	light := isLight == nil

	adaptive := func(hex string) lipgloss.AdaptiveColor {
		return lipgloss.AdaptiveColor{Light: hex, Dark: hex}
	}

	selectedBg := c["selection_background"]
	if selectedBg == "" {
		if light {
			selectedBg = c["color4"]
		} else {
			selectedBg = c["color4"]
		}
	}

	border := c["color8"]
	if border == "" {
		border = c["color7"]
	}

	accent := c["accent"]
	if accent == "" {
		accent = c["color6"] // cyan/teal as fallback
	}

	return Theme{
		Rising:        lipgloss.Color(c["color2"]),  // green
		Falling:       lipgloss.Color(c["color4"]),  // blue
		HighTide:      lipgloss.Color(c["color6"]),  // cyan
		LowTide:       lipgloss.Color(c["color12"]), // bright blue
		Accent:        lipgloss.Color(accent),
		AccentSubtle:  adaptive(c["color5"]), // magenta/purple
		Selected:      adaptive(selectedBg),
		Border:        adaptive(border),
		TextPrimary:   adaptive(c["foreground"]),
		TextSecondary: adaptive(c["color8"]),
		SparkFuture:   lipgloss.Color(c["color4"]), // blue family, dim for future predictions
		TableAlt:      adaptive(c["color0"]),        // darkest terminal color
		Good:          lipgloss.Color(c["color2"]),
		Warn:          lipgloss.Color(c["color3"]),
		Bad:           lipgloss.Color(c["color1"]),
	}, true
}

func parseOmarchyColors(data []byte) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if key != "" && val != "" {
			result[key] = val
		}
	}
	return result
}
