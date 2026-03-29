package model

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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
	animFrame     int
	showHelp      bool

	// Tide view date navigation
	viewDate        time.Time       // zero = today; explicit date = historical/future
	dayCurve        []noaa.WaterObs // 6-min predictions for viewDate
	dayCurveLoading bool
	dayCurveStale   bool // true when dayCurve is from a previous date (new fetch in-flight)

	// Almanac date navigation — zero means start from today
	almanacDate time.Time

	// Date input overlay state (tide view only)
	showDateInput bool
	dateInput     string
	dateInputErr  string

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
		m.fetchDayCurveCmd(),
		m.tidePollTickCmd(),
		m.predPollTickCmd(),
		m.animTickCmd(),
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
		// Refresh today's curve on each poll tick so the "now" marker advances.
		if m.viewDate.IsZero() {
			cmds = append(cmds, m.fetchDayCurveCmd())
		}

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
				// Refetch the day curve with the correct station timezone now that
				// we know it. The initial fetch used time.Local which may differ.
				if m.viewDate.IsZero() {
					// Timezone changed — old curve is for the wrong tz, clear entirely.
					m.dayCurve = nil
					m.dayCurveStale = false
					m.dayCurveLoading = true
					cmds = append(cmds, m.fetchDayCurveCmd())
				}
			}
		}

	case dayCurveLoadedMsg:
		m.dayCurveLoading = false
		m.dayCurveStale = false
		if msg.err != nil {
			if m.errMsg == "" {
				m.errMsg = "tide curve: " + msg.err.Error()
				cmds = append(cmds, errClearCmd())
			}
		} else if isSameCalendarDay(msg.date, m.currentViewDate(), m.loc) {
			m.dayCurve = msg.obs
		}

	case nearbyStationsLoadedMsg:
		m.nearbyLoading = false
		if msg.err == nil {
			m.nearbyStations = msg.stations
		}

	case refreshFlashClearMsg:
		m.refreshFlash = false

	case animTickMsg:
		m.animFrame++
		cmds = append(cmds, m.animTickCmd())

	case errClearMsg:
		m.errMsg = ""

	case tea.MouseMsg:
		if msg.Type == tea.MouseLeft {
			cmds = append(cmds, m.handleClick(msg.X, msg.Y)...)
		}

	case tea.KeyMsg:
		// Date input overlay intercepts keys while open.
		if m.showDateInput {
			cmds = append(cmds, m.updateDateInput(msg)...)
			return m, tea.Batch(cmds...)
		}
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

		case key.Matches(msg, m.keys.DateInput):
			m.showDateInput = true
			m.dateInput = ""
			m.dateInputErr = ""

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

		case key.Matches(msg, m.keys.PrevDay):
			if m.activeView == ViewTide {
				m.viewDate = m.currentViewDate().AddDate(0, 0, -1)
				m.dayCurveStale = true
				m.dayCurveLoading = true
				cmds = append(cmds, m.fetchDayCurveCmd())
			}

		case key.Matches(msg, m.keys.NextDay):
			if m.activeView == ViewTide {
				target := m.currentViewDate().AddDate(0, 0, 1)
				// Cap at one year ahead.
				limit := time.Now().AddDate(1, 0, 0)
				if target.Before(limit) {
					if isSameCalendarDay(target, time.Now(), m.loc) {
						m.viewDate = time.Time{} // back to "today" mode
					} else {
						m.viewDate = target
					}
					m.dayCurveStale = true
					m.dayCurveLoading = true
					cmds = append(cmds, m.fetchDayCurveCmd())
				}
			}

		case key.Matches(msg, m.keys.GoToday):
			if m.activeView == ViewTide && !m.viewDate.IsZero() {
				m.viewDate = time.Time{}
				m.dayCurveStale = true
				m.dayCurveLoading = true
				cmds = append(cmds, m.fetchDayCurveCmd())
			}
			if m.activeView == ViewAlmanac && !m.almanacDate.IsZero() {
				m.almanacDate = time.Time{}
				m.almanacCursor = 0
				cmds = append(cmds, m.fetchPredictionsCmd())
			}

		case key.Matches(msg, m.keys.Confirm):
			// Almanac Enter → drill into that day's tide chart.
			if m.activeView == ViewAlmanac && len(m.dailyTides) > 0 {
				day := m.dailyTides[m.almanacCursor]
				m.activeView = ViewTide
				if isSameCalendarDay(day.Date, time.Now(), m.loc) {
					m.viewDate = time.Time{}
				} else {
					m.viewDate = day.Date
				}
				m.dayCurveStale = true
				m.dayCurveLoading = true
				cmds = append(cmds, m.fetchDayCurveCmd())
			}

		case key.Matches(msg, m.keys.Refresh):
			cmds = append(cmds, m.fetchWaterLevelCmd(), m.fetchMetCmd(), m.fetchPredictionsCmd())

		case key.Matches(msg, m.keys.ToggleClock):
			m.cfg.Use24h = !m.cfg.Use24h
			ui.Use24h = m.cfg.Use24h
			_ = config.Save(m.cfg)
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

// updateDateInput handles key events while the date input overlay is open.
func (m *Model) updateDateInput(msg tea.KeyMsg) []tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.showDateInput = false
		m.dateInput = ""
		m.dateInputErr = ""

	case key.Matches(msg, m.keys.Confirm):
		t, err := parseFuzzyDate(m.dateInput, m.loc)
		if err != nil {
			m.dateInputErr = err.Error()
			return nil
		}
		m.showDateInput = false
		m.dateInputErr = ""
		if isSameCalendarDay(t, time.Now(), m.loc) {
			m.viewDate = time.Time{}
			m.almanacDate = time.Time{}
		} else {
			m.viewDate = t
			m.almanacDate = t
		}
		m.almanacCursor = 0
		m.dayCurveStale = true
		m.dayCurveLoading = true
		return []tea.Cmd{m.fetchDayCurveCmd(), m.fetchPredictionsCmd()}

	default:
		switch msg.String() {
		case "backspace", "ctrl+h":
			if len(m.dateInput) > 0 {
				m.dateInput = m.dateInput[:len(m.dateInput)-1]
				m.dateInputErr = ""
			}
		default:
			// Use msg.String() so space (KeySpace, Runes==nil) and all
			// printable ASCII including '-', '/' are accepted.
			s := msg.String()
			if len(s) == 1 && s[0] >= 32 && s[0] < 127 {
				m.dateInput += s
				m.dateInputErr = ""
			}
		}
	}
	return nil
}

// parseFuzzyDate tries several common date formats and returns midnight of that
// day in loc. Formats without a year assume the current year.
func parseFuzzyDate(input string, loc *time.Location) (time.Time, error) {
	if loc == nil {
		loc = time.Local
	}
	// Normalize
	s := strings.TrimSpace(input)
	s = strings.ReplaceAll(s, ",", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	// Title-case so month names parse: "oct" → "Oct", "october" → "October"
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	s = strings.Join(words, " ")

	now := time.Now().In(loc)

	// Formats with explicit year
	for _, layout := range []string{
		"Jan 2 2006", "January 2 2006",
		"1/2/2006", "01/02/2006",
		"2006-01-02",
	} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc), nil
		}
	}

	// Formats without year — assume current year
	for _, layout := range []string{
		"Jan 2", "January 2", "1/2", "01/02",
	} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc), nil
		}
	}

	return time.Time{}, fmt.Errorf("try \"Oct 11\", \"Oct 11 2025\", or \"2025-10-11\"")
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

	// Reset date navigation to today for the new station (no stale data to show).
	m.viewDate = time.Time{}
	m.dayCurve = nil
	m.dayCurveStale = false
	m.dayCurveLoading = true

	return []tea.Cmd{
		m.fetchWaterLevelCmd(),
		m.fetchPredictionsCmd(),
		m.fetchMetCmd(),
		m.fetchStationCmd(),
		m.fetchDayCurveCmd(),
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
	overlay := ""
	if m.showPicker {
		overlay = "picker"
	} else if m.showDateInput {
		overlay = "dateinput"
	}
	helpBar := ui.RenderHelpBar(m.width, m.showHelp, int(m.activeView), overlay)
	bodyH := m.bodyHeight()

	ui.Use24h = m.cfg.Use24h

	var body string
	if m.showDateInput {
		body = ui.RenderDateInput(m.dateInput, m.dateInputErr, m.width, bodyH)
	} else if m.showPicker {
		body = ui.RenderStationPicker(m.pickerInput, m.nearbyStations, m.pickerCursor, m.nearbyLoading, m.cfg.StationID, m.width, bodyH)
	} else {
		switch m.activeView {
		case ViewTide:
			isToday := m.viewDate.IsZero()
			nowFrac := 0.0
			if isToday {
				loc := m.loc
				if loc == nil {
					loc = time.Local
				}
				now := time.Now().In(loc)
				nowFrac = float64(now.Hour()*60+now.Minute()) / (24 * 60)
			}
			body = ui.RenderTideView(
				m.waterObs, m.predictions, m.met,
				m.dayCurve, m.dayCurveStale, m.currentViewDate(), isToday, nowFrac,
				m.animFrame,
				m.width, bodyH,
			)
		case ViewAlmanac:
			body = ui.RenderAlmanacView(m.dailyTides, m.almanacCursor, m.width, bodyH)
		case ViewStation:
			body = ui.RenderStationView(m.station, m.lastUpdated, m.cfg.Units, m.width)
		}
	}

	// Pad body to exactly bodyH lines so the help bar is always pinned to the
	// bottom of the terminal regardless of which view or overlay is active.
	pinnedBody := lipgloss.NewStyle().Height(bodyH).Render(body)
	return statusBar + "\n" + pinnedBody + "\n" + helpBar
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

// currentViewDate returns the date being displayed in the tide view.
// When viewDate is zero (today mode), it returns today's midnight in m.loc.
func (m Model) currentViewDate() time.Time {
	if m.viewDate.IsZero() {
		loc := m.loc
		if loc == nil {
			loc = time.Local
		}
		now := time.Now().In(loc)
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	}
	return m.viewDate
}

// isSameCalendarDay returns true if a and b fall on the same calendar day in loc.
func isSameCalendarDay(a, b time.Time, loc *time.Location) bool {
	if loc == nil {
		loc = time.Local
	}
	ai := a.In(loc)
	bi := b.In(loc)
	return ai.Year() == bi.Year() && ai.Month() == bi.Month() && ai.Day() == bi.Day()
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

const animTickInterval = 600 * time.Millisecond

func (m Model) animTickCmd() tea.Cmd {
	return tea.Tick(animTickInterval, func(t time.Time) tea.Msg {
		return animTickMsg(t)
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
	begin := time.Now()
	if !m.almanacDate.IsZero() {
		begin = m.almanacDate
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
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

func (m Model) fetchDayCurveCmd() tea.Cmd {
	date := m.currentViewDate()
	loc := m.loc
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		obs, err := m.client.FetchDayCurve(ctx, date, loc)
		return dayCurveLoadedMsg{date: date, obs: obs, err: err}
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

	// ── Tide view: click the ← / → date navigation arrows ────────────────
	// For today:     nav row is body row 3 → y=4
	// For non-today: nav row is body row 1 → y=2
	// Left third of row = prev day, right third = next day.
	if m.activeView == ViewTide && !m.showPicker && !m.showDateInput {
		isToday := m.viewDate.IsZero()
		navY := 4
		if !isToday {
			navY = 2
		}
		if y == navY {
			third := m.width / 3
			if x < third {
				// ← prev day
				m.viewDate = m.currentViewDate().AddDate(0, 0, -1)
				m.dayCurveStale = true
				m.dayCurveLoading = true
				cmds = append(cmds, m.fetchDayCurveCmd())
			} else if x > 2*third {
				// → next day
				target := m.currentViewDate().AddDate(0, 0, 1)
				limit := time.Now().AddDate(1, 0, 0)
				if target.Before(limit) {
					if isSameCalendarDay(target, time.Now(), m.loc) {
						m.viewDate = time.Time{}
					} else {
						m.viewDate = target
					}
					m.dayCurveStale = true
					m.dayCurveLoading = true
					cmds = append(cmds, m.fetchDayCurveCmd())
				}
			}
			return cmds
		}
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
