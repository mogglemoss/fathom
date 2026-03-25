package moon

import (
	"math"
	"time"
)

// knownNewMoon is a reference new moon epoch (January 6, 2000, 18:14 UTC).
var knownNewMoon = time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC)

// synodicPeriod is the average time between new moons.
const synodicPeriod = time.Duration(29.53059 * 24 * float64(time.Hour))

// Phase returns the moon phase as a value in [0, 1) where:
//
//	0.0  = New Moon
//	0.25 = First Quarter
//	0.5  = Full Moon
//	0.75 = Last Quarter
func Phase(t time.Time) float64 {
	elapsed := t.Sub(knownNewMoon)
	cycles := float64(elapsed) / float64(synodicPeriod)
	phase := math.Mod(cycles, 1.0)
	if phase < 0 {
		phase += 1.0
	}
	return phase
}

// PhaseName returns the common name for the given phase value.
func PhaseName(phase float64) string {
	switch {
	case phase < 0.0625 || phase >= 0.9375:
		return "New Moon"
	case phase < 0.1875:
		return "Waxing Crescent"
	case phase < 0.3125:
		return "First Quarter"
	case phase < 0.4375:
		return "Waxing Gibbous"
	case phase < 0.5625:
		return "Full Moon"
	case phase < 0.6875:
		return "Waning Gibbous"
	case phase < 0.8125:
		return "Last Quarter"
	default:
		return "Waning Crescent"
	}
}

// PhaseGlyph returns a Unicode moon emoji for the given phase.
func PhaseGlyph(phase float64) string {
	switch {
	case phase < 0.0625 || phase >= 0.9375:
		return "🌑"
	case phase < 0.1875:
		return "🌒"
	case phase < 0.3125:
		return "🌓"
	case phase < 0.4375:
		return "🌔"
	case phase < 0.5625:
		return "🌕"
	case phase < 0.6875:
		return "🌖"
	case phase < 0.8125:
		return "🌗"
	default:
		return "🌘"
	}
}
