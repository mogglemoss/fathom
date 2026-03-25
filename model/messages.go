package model

import (
	"time"

	"github.com/mogglemoss/fathom/noaa"
)

// ── Tick messages ─────────────────────────────────────────────────────────────

// tidePollTickMsg fires every 60 seconds to trigger water level + met refresh.
type tidePollTickMsg time.Time

// predPollTickMsg fires every 6 hours to trigger predictions refresh.
type predPollTickMsg time.Time

// ── Data loaded messages ──────────────────────────────────────────────────────

type waterLevelLoadedMsg struct {
	obs []noaa.WaterObs
	err error
}

type predictionsLoadedMsg struct {
	preds []noaa.Prediction
	err   error
}

type metLoadedMsg struct {
	met noaa.MetObs
	err error
}

type stationLoadedMsg struct {
	meta noaa.StationMeta
	err  error
}

type nearbyStationsLoadedMsg struct {
	stations []noaa.NearbyStation
	err      error
}

// ── UI flash / clear messages ─────────────────────────────────────────────────

type refreshFlashClearMsg struct{}
type errClearMsg          struct{}
