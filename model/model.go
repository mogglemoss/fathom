package model

import (
	"context"
	"sort"
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
	tidePollInterval = 60 * time.Second
	predPollInterval = 6 * time.Hour
	errClearDelay    = 4 * time.Second
	predForecastDays = 14
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
	loc         *time.Location // station local timezone
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
			// Don't overwrite a water-level error with a predictions error.
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
			// Load the station's local timezone for timestamp parsing.
			if loc, err := time.LoadLocation(msg.meta.TimeZone); err == nil {
				m.loc = loc
			}
		}

	case refreshFlashClearMsg:
		m.refreshFlash = false

	case errClearMsg:
		m.errMsg = ""

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, m.keys.Help):
			m.showHelp = !m.showHelp

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
				m.almanacCursor--
				if m.almanacCursor < 0 {
					m.almanacCursor = 0
				}
			}

		case key.Matches(msg, m.keys.Down):
			if m.activeView == ViewAlmanac {
				m.almanacCursor++
				if m.almanacCursor >= len(m.dailyTides) && len(m.dailyTides) > 0 {
					m.almanacCursor = len(m.dailyTides) - 1
				}
			}

		case key.Matches(msg, m.keys.Refresh):
			cmds = append(cmds, m.fetchWaterLevelCmd(), m.fetchMetCmd(), m.fetchPredictionsCmd())
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the full TUI.
func (m Model) View() string {
	if !m.ready {
		return "\n  〰 reaching into the water…\n"
	}

	currentLevel, hasLevel := m.currentLevel()
	statusBar := ui.RenderStatusBar(m.station, currentLevel, hasLevel, m.errMsg, m.lastUpdated, m.width, m.refreshFlash)
	helpBar := ui.RenderHelpBar(m.width, m.showHelp)
	bodyH := m.bodyHeight()

	var body string
	switch m.activeView {
	case ViewTide:
		body = ui.RenderTideView(m.waterObs, m.predictions, m.met, m.width, bodyH, !hasLevel)
	case ViewAlmanac:
		body = ui.RenderAlmanacView(m.dailyTides, m.almanacCursor, m.width, bodyH)
	case ViewStation:
		body = ui.RenderStationView(m.station, m.lastUpdated, m.cfg.Units, m.width)
	}

	// Pad body to fill available height so status+body+help fills the screen.
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

// assembleDailyTides groups predictions by calendar day and computes moon phases.
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
