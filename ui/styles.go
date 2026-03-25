package ui

import "github.com/charmbracelet/lipgloss"

// Theme holds all color values for a visual theme.
type Theme struct {
	Rising   lipgloss.Color // tide is rising
	Falling  lipgloss.Color // tide is falling
	HighTide lipgloss.Color // high tide markers
	LowTide  lipgloss.Color // low tide markers

	Accent        lipgloss.Color        // logo, section headers
	AccentSubtle  lipgloss.AdaptiveColor // help keys, labels
	Selected      lipgloss.AdaptiveColor // selected row background
	Border        lipgloss.AdaptiveColor // panel borders
	TextPrimary   lipgloss.AdaptiveColor
	TextSecondary lipgloss.AdaptiveColor

	Good lipgloss.Color // verified data, no errors
	Warn lipgloss.Color // preliminary data, warnings
	Bad  lipgloss.Color // errors, stale data
}

// Default is the built-in subaquatic retrofuture theme.
// Every color choice is intentional:
//   - Accent (#00E0C8): bioluminescent teal — the primary phosphor glow
//   - Rising (#00C985): deep-water green — incoming tide, life, motion
//   - Falling (#3A7FAA): ocean steel blue — outgoing tide, depth, stillness
//   - HighTide (#00DFFF): phosphor cyan — crisp technical readout color
//   - LowTide (#5B9EC9): muted steel — secondary tide reference
//   - Warn (#FFAA00): amber — instrument warning lamp, intentionally warm
//   - TextPrimary dark (#C4E8F0): pale aquamarine — easy on eyes, subaquatic
//   - Selected dark (#0D2535): deep ocean — where the cursor rests
var Default = Theme{
	Rising:   lipgloss.Color("#00C985"), // bioluminescent green
	Falling:  lipgloss.Color("#3A7FAA"), // ocean steel blue
	HighTide: lipgloss.Color("#00DFFF"), // phosphor cyan
	LowTide:  lipgloss.Color("#5B9EC9"), // muted steel blue

	Accent:        lipgloss.Color("#00E0C8"), // bioluminescent teal
	AccentSubtle:  lipgloss.AdaptiveColor{Light: "#007A8A", Dark: "#00A89A"},
	Selected:      lipgloss.AdaptiveColor{Light: "#D0F0F4", Dark: "#0D2535"},
	Border:        lipgloss.AdaptiveColor{Light: "#B0C8D0", Dark: "#1C3D50"},
	TextPrimary:   lipgloss.AdaptiveColor{Light: "#1A2830", Dark: "#C4E8F0"},
	TextSecondary: lipgloss.AdaptiveColor{Light: "#5A7480", Dark: "#4D7888"},

	Good: lipgloss.Color("#00C985"),
	Warn: lipgloss.Color("#FFAA00"), // amber warning lamp
	Bad:  lipgloss.Color("#FF4060"),
}

// Styles holds all pre-built lipgloss styles derived from the active theme.
type Styles struct {
	T Theme

	// Status bar
	StatusBar  lipgloss.Style
	StatusLogo lipgloss.Style
	StatusMeta lipgloss.Style
	StatusGood lipgloss.Style
	StatusBad  lipgloss.Style

	// Section headers (shared across views)
	SectionHeader lipgloss.Style
	Label         lipgloss.Style
	Value         lipgloss.Style

	// Tide view
	TideLevel   lipgloss.Style // large current water level number
	TideRising  lipgloss.Style // ▲ rising indicator
	TideFalling lipgloss.Style // ▼ falling indicator
	TideHigh    lipgloss.Style // next HIGH label and value
	TideLow     lipgloss.Style // next LOW label and value
	SparkHigh   lipgloss.Style // chart bars — past, above mean
	SparkLow    lipgloss.Style // chart bars — past, below mean
	SparkFuture lipgloss.Style // chart bars — future prediction (dim)
	SparkCursor lipgloss.Style // current time position (bright glow)

	// Almanac
	AlmanacDate   lipgloss.Style
	AlmanacHigh   lipgloss.Style
	AlmanacLow    lipgloss.Style
	AlmanacMoon   lipgloss.Style
	AlmanacCursor lipgloss.Style

	// Met strip
	MetLabel lipgloss.Style
	MetValue lipgloss.Style

	// Help bar
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style
	HelpSep  lipgloss.Style

	// Panel border
	PanelBorder lipgloss.Style
}

// New builds a Styles from the given Theme.
func New(t Theme) Styles {
	s := Styles{T: t}

	// Status bar
	s.StatusBar = lipgloss.NewStyle().
		Background(t.Selected).
		Foreground(t.TextPrimary).
		Padding(0, 1)
	s.StatusLogo = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)
	s.StatusMeta = lipgloss.NewStyle().Foreground(t.TextSecondary)
	s.StatusGood = lipgloss.NewStyle().Foreground(t.Good)
	s.StatusBad = lipgloss.NewStyle().Foreground(t.Bad)

	// Shared — instrument panel style: bold, no underline (underline = generic web)
	s.SectionHeader = lipgloss.NewStyle().Foreground(t.Accent).Bold(true)
	s.Label = lipgloss.NewStyle().Foreground(t.TextSecondary)
	s.Value = lipgloss.NewStyle().Foreground(t.TextPrimary)

	// Tide view
	s.TideLevel = lipgloss.NewStyle().Foreground(t.HighTide).Bold(true)
	s.TideRising = lipgloss.NewStyle().Foreground(t.Rising).Bold(true)
	s.TideFalling = lipgloss.NewStyle().Foreground(t.Falling).Bold(true)
	s.TideHigh = lipgloss.NewStyle().Foreground(t.HighTide)
	s.TideLow = lipgloss.NewStyle().Foreground(t.LowTide)
	s.SparkHigh = lipgloss.NewStyle().Foreground(t.Rising)
	s.SparkLow = lipgloss.NewStyle().Foreground(t.Falling)
	// Future: dim secondary color — clearly muted vs vivid past, uncertain depth
	s.SparkFuture = lipgloss.NewStyle().Foreground(t.TextSecondary)
	s.SparkCursor = lipgloss.NewStyle().Foreground(t.Accent).Bold(true)

	// Almanac
	s.AlmanacDate = lipgloss.NewStyle().Foreground(t.TextPrimary)
	s.AlmanacHigh = lipgloss.NewStyle().Foreground(t.HighTide)
	s.AlmanacLow = lipgloss.NewStyle().Foreground(t.LowTide)
	s.AlmanacMoon = lipgloss.NewStyle().Foreground(t.Accent)
	s.AlmanacCursor = lipgloss.NewStyle().
		Background(t.Selected).
		Foreground(t.Accent).
		Bold(true)

	// Met strip
	s.MetLabel = lipgloss.NewStyle().Foreground(t.TextSecondary)
	s.MetValue = lipgloss.NewStyle().Foreground(t.TextPrimary)

	// Help bar
	s.HelpKey = lipgloss.NewStyle().Foreground(t.AccentSubtle).Bold(true)
	s.HelpDesc = lipgloss.NewStyle().Foreground(t.TextSecondary)
	s.HelpSep = lipgloss.NewStyle().Foreground(t.Border)

	// Panel border
	s.PanelBorder = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false).
		BorderForeground(t.Border)

	return s
}

// Presets is the map of named built-in themes selectable via --theme.
var Presets = map[string]Theme{
	"default": Default,
	"catppuccin": {
		Rising:        lipgloss.Color("#a6e3a1"),
		Falling:       lipgloss.Color("#89b4fa"),
		HighTide:      lipgloss.Color("#89dceb"),
		LowTide:       lipgloss.Color("#74c7ec"),
		Accent:        lipgloss.Color("#cba6f7"),
		AccentSubtle:  lipgloss.AdaptiveColor{Light: "#1e66f5", Dark: "#89b4fa"},
		Selected:      lipgloss.AdaptiveColor{Light: "#dce0e8", Dark: "#313244"},
		Border:        lipgloss.AdaptiveColor{Light: "#bcc0cc", Dark: "#45475a"},
		TextPrimary:   lipgloss.AdaptiveColor{Light: "#4c4f69", Dark: "#cdd6f4"},
		TextSecondary: lipgloss.AdaptiveColor{Light: "#6c6f85", Dark: "#9399b2"},
		Good:          lipgloss.Color("#a6e3a1"),
		Warn:          lipgloss.Color("#f9e2af"),
		Bad:           lipgloss.Color("#f38ba8"),
	},
	"dracula": {
		Rising:        lipgloss.Color("#50fa7b"),
		Falling:       lipgloss.Color("#8be9fd"),
		HighTide:      lipgloss.Color("#8be9fd"),
		LowTide:       lipgloss.Color("#6272a4"),
		Accent:        lipgloss.Color("#bd93f9"),
		AccentSubtle:  lipgloss.AdaptiveColor{Light: "#6272a4", Dark: "#8be9fd"},
		Selected:      lipgloss.AdaptiveColor{Light: "#f8f8f2", Dark: "#44475a"},
		Border:        lipgloss.AdaptiveColor{Light: "#6272a4", Dark: "#6272a4"},
		TextPrimary:   lipgloss.AdaptiveColor{Light: "#282a36", Dark: "#f8f8f2"},
		TextSecondary: lipgloss.AdaptiveColor{Light: "#6272a4", Dark: "#6272a4"},
		Good:          lipgloss.Color("#50fa7b"),
		Warn:          lipgloss.Color("#f1fa8c"),
		Bad:           lipgloss.Color("#ff5555"),
	},
	"nord": {
		Rising:        lipgloss.Color("#a3be8c"),
		Falling:       lipgloss.Color("#81a1c1"),
		HighTide:      lipgloss.Color("#88c0d0"),
		LowTide:       lipgloss.Color("#5e81ac"),
		Accent:        lipgloss.Color("#88c0d0"),
		AccentSubtle:  lipgloss.AdaptiveColor{Light: "#5e81ac", Dark: "#81a1c1"},
		Selected:      lipgloss.AdaptiveColor{Light: "#e5e9f0", Dark: "#3b4252"},
		Border:        lipgloss.AdaptiveColor{Light: "#d8dee9", Dark: "#4c566a"},
		TextPrimary:   lipgloss.AdaptiveColor{Light: "#2e3440", Dark: "#eceff4"},
		TextSecondary: lipgloss.AdaptiveColor{Light: "#4c566a", Dark: "#7b88a1"},
		Good:          lipgloss.Color("#a3be8c"),
		Warn:          lipgloss.Color("#ebcb8b"),
		Bad:           lipgloss.Color("#bf616a"),
	},
}

// S is the active styles singleton.
var S = initStyles()

func initStyles() Styles {
	if theme, ok := LoadOmarchyTheme(); ok {
		return New(theme)
	}
	return New(Default)
}

// SetTheme applies a named preset theme, overriding Omarchy detection.
// Unknown names are silently ignored.
func SetTheme(name string) {
	if t, ok := Presets[name]; ok {
		S = New(t)
	}
}
