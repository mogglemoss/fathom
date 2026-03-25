package model

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mogglemoss/fathom/config"
	"github.com/mogglemoss/fathom/moon"
	"github.com/mogglemoss/fathom/noaa"
	"github.com/mogglemoss/fathom/ui"
)

// View identifies which view is active.
type View int

const (
	ViewTide    View = iota // live tide dashboard
	ViewAlmanac             // 7–14 day forecast
	ViewStation             // station metadata
)

const (
	tidePollInterval   = 60 * time.Second
	predPollInterval   = 6 * time.Hour
	errClearDelay      = 4 * time.Second
	predForecastDays   = 14
	nearbyStationCount = 8
)

// Model is the root BubbleTea model for fathom.
type Model struct {
	keys   KeyMap
	cfg    config.Config
	client *noaa.Client

	// Fetched data
	waterObs    []noaa.WaterObs
	predictions []noaa.Prediction
	met         noaa.MetObs
	station     noaa.StationMeta
	loc         *time.Location
	dailyTides  []noaa.DailyTide

	// UI state
	activeView    View
	almanacCursor int
	width         int
	height        int
	ready         bool
	errMsg        string
	lastUpdated   time.Time
	refreshFlash  bool
	showHelp      bool

	// Station picker state
	showPicker     bool
	pickerInput    string // typed station ID
	pickerCursor   int    // selected row in nearby list
	nearbyStations []noaa.NearbyStation
	nearbyLoading  bool
}

// New creates the initial model from the given config.
func New(cfg config.Config) Model {
	return Model{
		keys:   DefaultKeyMap(),
		cfg:    cfg,
		client: noaa.NewClient(cfg),
		loc:    time.Local,
	}
}

// Init kicks off the first round of data fetching and the polling loops.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchWaterLevelCmd(),
		m.fetchPredictionsCmd(),
		m.fetchMetCmd(),
		m.fetchStationCmd(),
		m.tidePollTickCmd(),
		m.predPollTickCmd(),
	)
}

// Update handles all incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tidePollTickMsg:
		cmds = append(cmds, m.tidePollTickCmd(), m.fetchWaterLevelCmd(), m.fetchMetCmd())

	case predPollTickMsg:
		cmds = append(cmds, m.predPollTickCmd(), m.fetchPredictionsCmd())

	case waterLevelLoadedMsg:
		if msg.err != nil {
			m.errMsg = msg.err.Error()
			cmds = append(cmds, errClearCmd())
		} else {
			m.waterObs = msg.obs
			m.lastUpdated = time.Now()
			m.refreshFlash = true
			cmds = append(cmds, refreshFlashClearCmd())
			m.errMsg = ""
		}

	case predictionsLoadedMsg:
		if msg.err != nil {
			if m.errMsg == "" {
				m.errMsg = "predictions: " + msg.err.Error()
				cmds = append(cmds, errClearCmd())
			}
		} else {
			m.predictions = msg.preds
			m.dailyTides = assembleDailyTides(msg.preds, m.loc)
		}

	case metLoadedMsg:
		if msg.err == nil {
			m.met = msg.met
		}

	case stationLoadedMsg:
		if msg.err == nil {
			m.station = msg.meta
			if loc, err := time.LoadLocation(msg.meta.TimeZone); err == nil {
				m.loc = loc
			}
		}

	case nearbyStationsLoadedMsg:
		m.nearbyLoading = false
		if msg.err == nil {
			m.nearbyStations = msg.stations
		}

	case refreshFlashClearMsg:
		m.refreshFlash = false

	case errClearMsg:
		m.errMsg = ""

	case tea.MouseMsg:
		if msg.Type == tea.MouseLeft {
			cmds = append(cmds, m.handleClick(msg.X, msg.Y)...)
		}

	case tea.KeyMsg:
		// Station picker intercepts keys while open.
		if m.showPicker {
			cmds = append(cmds, m.updatePicker(msg)...)
			return m, tea.Batch(cmds...)
		}

		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp

		case key.Matches(msg, m.keys.StationSearch):
			m.showPicker = true
			m.pickerInput = ""
			m.pickerCursor = 0
			if len(m.nearbyStations) == 0 {
				m.nearbyLoading = true
				cmds = append(cmds, m.fetchNearbyStationsCmd())
			}

		case key.Matches(msg, m.keys.NextView):
			m.activeView = View((int(m.activeView) + 1) % 3)

		case key.Matches(msg, m.keys.ViewTide):
			m.activeView = ViewTide

		case key.Matches(msg, m.keys.ViewAlmanac):
			m.activeView = ViewAlmanac

		case key.Matches(msg, m.keys.ViewStation):
			m.activeView = ViewStation

		case key.Matches(msg, m.keys.Up):
			if m.activeView == ViewAlmanac {
				if m.almanacCursor > 0 {
					m.almanacCursor--
				}
			}

		case key.Matches(msg, m.keys.Down):
			if m.activeView == ViewAlmanac {
				if m.almanacCursor < len(m.dailyTides)-1 {
					m.almanacCursor++
				}
			}

		case key.Matches(msg, m.keys.Refresh):
			cmds = append(cmds, m.fetchWaterLevelCmd(), m.fetchMetCmd(), m.fetchPredictionsCmd())
		}
	}

	return m, tea.Batch(cmds...)
}

// updatePicker handles key events while the station picker is open.
// Returns any commands to batch.
func (m *Model) updatePicker(msg tea.KeyMsg) []tea.Cmd {
	var cmds []tea.Cmd

	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.showPicker = false
		m.pickerInput = ""

	case key.Matches(msg, m.keys.Confirm):
		cmds = append(cmds, m.applyPickerSelection()...)

	case key.Matches(msg, m.keys.Up):
		if m.pickerCursor > 0 {
			m.pickerCursor--
		}
		m.pickerInput = "" // clear typed input when navigating list

	case key.Matches(msg, m.keys.Down):
		if m.pickerCursor < len(m.nearbyStations)-1 {
			m.pickerCursor++
		}
		m.pickerInput = ""

	default:
		// Route printable characters to the text input.
		switch msg.String() {
		case "backspace", "ctrl+h":
			if len(m.pickerInput) > 0 {
				m.pickerInput = m.pickerInput[:len(m.pickerInput)-1]
				m.pickerCursor = -1
			}
		default:
			// Only accept printable ASCII (digits and letters for station IDs)
			if len(msg.Runes) > 0 {
				ch := string(msg.Runes)
				if strings.TrimSpace(ch) != "" {
					m.pickerInput += ch
					m.pickerCursor = -1 // typing deselects list
				}
			}
		}
	}

	return cmds
}

// applyPickerSelection switches to the selected or typed station and returns
// fetch commands for the new station's data.
func (m *Model) applyPickerSelection() []tea.Cmd {
	var newID string
	if m.pickerInput != "" {
		newID = strings.TrimSpace(m.pickerInput)
	} else if m.pickerCursor >= 0 && m.pickerCursor < len(m.nearbyStations) {
		newID = m.nearbyStations[m.pickerCursor].ID
	}

	m.showPicker = false
	m.pickerInput = ""

	if newID == "" || newID == m.cfg.StationID {
		return nil
	}

	m.cfg.StationID = newID
	m.client = noaa.NewClient(m.cfg)
	_ = config.Save(m.cfg)

	// Reset data for new station
	m.waterObs = nil
	m.predictions = nil
	m.met = noaa.MetObs{}
	m.station = noaa.StationMeta{}
	m.dailyTides = nil
	m.loc = time.Local
	m.lastUpdated = time.Time{}
	m.errMsg = ""

	return []tea.Cmd{
		m.fetchWaterLevelCmd(),
		m.fetchPredictionsCmd(),
		m.fetchMetCmd(),
		m.fetchStationCmd(),
	}
}

// View renders the full TUI.
func (m Model) View() string {
	if !m.ready {
		return "\n  〰 reaching into the water…\n"
	}

	currentLevel, hasLevel := m.currentLevel()
	direction := ui.TideDirection(m.waterObs)
	statusBar := ui.RenderStatusBar(m.station, currentLevel, hasLevel, direction, m.errMsg, m.lastUpdated, m.width, m.refreshFlash, int(m.activeView))
	helpBar := ui.RenderHelpBar(m.width, m.showHelp)
	bodyH := m.bodyHeight()

	var body string
	if m.showPicker {
		body = ui.RenderStationPicker(m.pickerInput, m.nearbyStations, m.pickerCursor, m.nearbyLoading, m.cfg.StationID, m.width, bodyH)
	} else {
		switch m.activeView {
		case ViewTide:
			body = ui.RenderTideView(m.waterObs, m.predictions, m.met, m.width, bodyH, !hasLevel)
		case ViewAlmanac:
			body = ui.RenderAlmanacView(m.dailyTides, m.almanacCursor, m.width, bodyH)
		case ViewStation:
			body = ui.RenderStationView(m.station, m.lastUpdated, m.cfg.Units, m.width)
		}
	}

	return statusBar + "\n" + body + helpBar
}

func (m Model) bodyHeight() int {
	h := m.height - 2
	if h < 1 {
		return 1
	}
	return h
}

func (m Model) currentLevel() (float64, bool) {
	if len(m.waterObs) == 0 {
		return 0, false
	}
	return m.waterObs[len(m.waterObs)-1].Level, true
}

// ── Data assembly ─────────────────────────────────────────────────────────────

func assembleDailyTides(preds []noaa.Prediction, loc *time.Location) []noaa.DailyTide {
	if loc == nil {
		loc = time.Local
	}

	daily := make(map[time.Time][]noaa.Prediction)
	for _, p := range preds {
		local := p.Time.In(loc)
		day := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
		daily[day] = append(daily[day], p)
	}

	days := make([]time.Time, 0, len(daily))
	for d := range daily {
		days = append(days, d)
	}
	sort.Slice(days, func(i, j int) bool { return days[i].Before(days[j]) })

	result := make([]noaa.DailyTide, 0, len(days))
	for _, d := range days {
		phase := moon.Phase(d)
		result = append(result, noaa.DailyTide{
			Date:        d,
			Predictions: daily[d],
			MoonPhase:   phase,
			MoonName:    moon.PhaseName(phase),
		})
	}
	return result
}

// ── Commands ──────────────────────────────────────────────────────────────────

func (m Model) tidePollTickCmd() tea.Cmd {
	return tea.Tick(tidePollInterval, func(t time.Time) tea.Msg {
		return tidePollTickMsg(t)
	})
}

func (m Model) predPollTickCmd() tea.Cmd {
	return tea.Tick(predPollInterval, func(t time.Time) tea.Msg {
		return predPollTickMsg(t)
	})
}

func (m Model) fetchWaterLevelCmd() tea.Cmd {
	loc := m.loc
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		obs, err := m.client.FetchWaterLevel(ctx, loc)
		return waterLevelLoadedMsg{obs: obs, err: err}
	}
}

func (m Model) fetchPredictionsCmd() tea.Cmd {
	loc := m.loc
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		begin := time.Now()
		end := begin.AddDate(0, 0, predForecastDays)
		preds, err := m.client.FetchPredictions(ctx, begin, end, loc)
		return predictionsLoadedMsg{preds: preds, err: err}
	}
}

func (m Model) fetchMetCmd() tea.Cmd {
	loc := m.loc
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		met, err := m.client.FetchMet(ctx, loc)
		return metLoadedMsg{met: met, err: err}
	}
}

func (m Model) fetchStationCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		meta, err := m.client.FetchStation(ctx)
		return stationLoadedMsg{meta: meta, err: err}
	}
}

func (m Model) fetchNearbyStationsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		stations, err := noaa.NearestStations(ctx, nearbyStationCount)
		return nearbyStationsLoadedMsg{stations: stations, err: err}
	}
}

// handleClick processes a left-button mouse click at terminal position (x, y).
func (m *Model) handleClick(x, y int) []tea.Cmd {
	var cmds []tea.Cmd

	// ── Status bar (row 0): click the view indicator dots ─────────────────
	// Layout: 1(pad) + 8(logo) + 3(sep) = col 12 where view dots start.
	// Each dot: "● VIEW" or "○ VIEW" with a 2-char gap between.
	if y == 0 {
		labels := []string{"TIDE", "ALMANAC", "STATION"}
		x0 := 1 + 8 + 3 // left-padding + logo + separator
		for i, label := range labels {
			w := 2 + len(label) // "● " + label
			if x >= x0 && x < x0+w {
				m.activeView = View(i)
				m.showPicker = false
				return cmds
			}
			x0 += w + 2 // "  " between dots
		}
	}

	body := y - 1 // body coordinate (0 = first line of body area)
	if body < 0 {
		return cmds
	}

	// ── Station picker: click on a nearby station row ──────────────────────
	// Picker layout (body rows):
	//   0: blank, 1: title, 2: blank, 3: input, 4: blank,
	//   5: "NEARBY STATIONS", 6: blank, 7+: items
	if m.showPicker {
		const pickerItemsStartRow = 7
		idx := body - pickerItemsStartRow
		if idx >= 0 && idx < len(m.nearbyStations) {
			m.pickerCursor = idx
			cmds = append(cmds, m.applyPickerSelection()...)
		}
		return cmds
	}

	// ── Almanac view: click on a day row to move the cursor ───────────────
	// Almanac body layout:
	//   0: blank, 1: section header, 2: column header,
	//   3 (or 4 if scroll indicator shown): first data row
	if m.activeView == ViewAlmanac {
		visibleRows := m.bodyHeight() - 5
		if visibleRows < 1 {
			visibleRows = 1
		}
		offset := ui.AlmanacScrollOffset(m.almanacCursor, visibleRows, len(m.dailyTides))
		topIndicator := 0
		if offset > 0 {
			topIndicator = 1
		}
		dataStart := 3 + topIndicator // body row where first data day appears
		dayRow := body - dataStart
		if dayRow >= 0 {
			dayIdx := dayRow + offset
			if dayIdx >= 0 && dayIdx < len(m.dailyTides) {
				m.almanacCursor = dayIdx
			}
		}
	}

	return cmds
}

func errClearCmd() tea.Cmd {
	return tea.Tick(errClearDelay, func(_ time.Time) tea.Msg {
		return errClearMsg{}
	})
}

func refreshFlashClearCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(_ time.Time) tea.Msg {
		return refreshFlashClearMsg{}
	})
}
