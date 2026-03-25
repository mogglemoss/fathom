package model

import (
	"testing"
	"time"

	"github.com/mogglemoss/fathom/noaa"
)

func TestAssembleDailyTides_GroupsByDay(t *testing.T) {
	loc, _ := time.LoadLocation("America/New_York")
	day1 := time.Date(2026, 3, 25, 4, 35, 0, 0, loc)
	day1b := time.Date(2026, 3, 25, 11, 9, 0, 0, loc)
	day2 := time.Date(2026, 3, 26, 5, 40, 0, 0, loc)

	preds := []noaa.Prediction{
		{Time: day1, Level: 10.36, IsHigh: true},
		{Time: day1b, Level: 0.01, IsHigh: false},
		{Time: day2, Level: 9.96, IsHigh: true},
	}

	daily := assembleDailyTides(preds, loc)

	if len(daily) != 2 {
		t.Fatalf("expected 2 days, got %d", len(daily))
	}
	if len(daily[0].Predictions) != 2 {
		t.Errorf("expected 2 predictions on day 1, got %d", len(daily[0].Predictions))
	}
	if len(daily[1].Predictions) != 1 {
		t.Errorf("expected 1 prediction on day 2, got %d", len(daily[1].Predictions))
	}
}

func TestAssembleDailyTides_SortedAscending(t *testing.T) {
	loc := time.UTC
	preds := []noaa.Prediction{
		{Time: time.Date(2026, 3, 27, 6, 0, 0, 0, loc), Level: 9.0, IsHigh: true},
		{Time: time.Date(2026, 3, 25, 4, 0, 0, 0, loc), Level: 10.0, IsHigh: true},
		{Time: time.Date(2026, 3, 26, 5, 0, 0, 0, loc), Level: 9.5, IsHigh: true},
	}

	daily := assembleDailyTides(preds, loc)

	if len(daily) != 3 {
		t.Fatalf("expected 3 days, got %d", len(daily))
	}
	if !daily[0].Date.Before(daily[1].Date) || !daily[1].Date.Before(daily[2].Date) {
		t.Error("daily tides not sorted in ascending date order")
	}
}

func TestAssembleDailyTides_MoonPhaseSet(t *testing.T) {
	loc := time.UTC
	preds := []noaa.Prediction{
		{Time: time.Date(2026, 3, 25, 4, 0, 0, 0, loc), Level: 10.0, IsHigh: true},
	}

	daily := assembleDailyTides(preds, loc)

	if len(daily) != 1 {
		t.Fatalf("expected 1 day, got %d", len(daily))
	}
	if daily[0].MoonPhase < 0 || daily[0].MoonPhase >= 1.0 {
		t.Errorf("moon phase out of range: %f", daily[0].MoonPhase)
	}
	if daily[0].MoonName == "" {
		t.Error("expected non-empty moon name")
	}
}

func TestAssembleDailyTides_Empty(t *testing.T) {
	daily := assembleDailyTides(nil, time.UTC)
	if len(daily) != 0 {
		t.Errorf("expected empty slice for nil input, got %d", len(daily))
	}
}

func TestModelCurrentLevel(t *testing.T) {
	m := Model{}

	_, hasLevel := m.currentLevel()
	if hasLevel {
		t.Error("empty model should have no level")
	}

	m.waterObs = []noaa.WaterObs{
		{Time: time.Now(), Level: 3.5},
		{Time: time.Now(), Level: 4.2},
	}

	level, hasLevel := m.currentLevel()
	if !hasLevel {
		t.Error("expected level to be available")
	}
	if level != 4.2 {
		t.Errorf("expected most recent level 4.2, got %f", level)
	}
}

func TestModelBodyHeight(t *testing.T) {
	m := Model{height: 40}
	if m.bodyHeight() != 38 {
		t.Errorf("expected body height 38, got %d", m.bodyHeight())
	}

	m.height = 1
	if m.bodyHeight() != 1 {
		t.Errorf("body height should be at least 1, got %d", m.bodyHeight())
	}
}
