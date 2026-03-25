package moon

import (
	"testing"
	"time"
)

func TestPhase_KnownDates(t *testing.T) {
	cases := []struct {
		name    string
		date    time.Time
		wantMin float64
		wantMax float64
	}{
		// The epoch itself should be near 0 (new moon)
		{"epoch new moon", time.Date(2000, 1, 6, 18, 14, 0, 0, time.UTC), 0.0, 0.02},
		// Full moon is roughly 14.77 days after new moon = phase ~0.5
		{"full moon ~2000-01-21", time.Date(2000, 1, 21, 0, 0, 0, 0, time.UTC), 0.45, 0.55},
		// Phase is always in [0, 1)
		{"today", time.Now(), 0.0, 1.0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := Phase(c.date)
			if p < 0 || p >= 1.0 {
				t.Errorf("Phase(%v) = %f, want [0, 1)", c.date, p)
			}
			if p < c.wantMin || p > c.wantMax {
				t.Errorf("Phase(%v) = %f, want [%f, %f]", c.date, p, c.wantMin, c.wantMax)
			}
		})
	}
}

func TestPhaseName(t *testing.T) {
	cases := []struct {
		phase float64
		want  string
	}{
		{0.0, "New Moon"},
		{0.03, "New Moon"},
		{0.1, "Waxing Crescent"},
		{0.25, "First Quarter"},
		{0.4, "Waxing Gibbous"},
		{0.5, "Full Moon"},
		{0.6, "Waning Gibbous"},
		{0.75, "Last Quarter"},
		{0.85, "Waning Crescent"},
		{0.94, "New Moon"},
		{0.99, "New Moon"},
	}

	for _, c := range cases {
		got := PhaseName(c.phase)
		if got != c.want {
			t.Errorf("PhaseName(%f) = %q, want %q", c.phase, got, c.want)
		}
	}
}

func TestPhaseGlyph(t *testing.T) {
	glyphs := []string{"🌑", "🌒", "🌓", "🌔", "🌕", "🌖", "🌗", "🌘"}
	// Every phase should return one of the 8 glyphs
	for phase := 0.0; phase < 1.0; phase += 0.05 {
		g := PhaseGlyph(phase)
		found := false
		for _, want := range glyphs {
			if g == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("PhaseGlyph(%f) = %q, not in expected set", phase, g)
		}
	}
}

func TestPhaseMonotonicity(t *testing.T) {
	// Phase should increase (modulo 1) over time
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	prev := Phase(base)
	for i := 1; i <= 30; i++ {
		curr := Phase(base.AddDate(0, 0, i))
		// Allow for wrap-around at 0
		if curr < prev && prev-curr < 0.9 {
			t.Errorf("phase decreased non-monotonically: day %d prev=%f curr=%f", i, prev, curr)
		}
		prev = curr
	}
}
