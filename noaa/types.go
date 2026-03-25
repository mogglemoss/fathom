package noaa

import "time"

// WaterObs is a single water level observation (typically 6-minute intervals).
type WaterObs struct {
	Time  time.Time
	Level float64 // feet (english) or meters (metric)
	QC    string  // "v" = verified, "p" = preliminary
}

// Prediction is a single predicted high or low tide.
type Prediction struct {
	Time   time.Time
	Level  float64
	IsHigh bool // true = high tide, false = low tide
}

// MetObs holds a merged meteorological snapshot.
type MetObs struct {
	Time        time.Time
	WindSpeed   float64 // knots (english) or m/s (metric)
	WindDir     float64 // degrees true (where wind is FROM)
	WindGust    float64
	AirTemp     float64 // °F (english) or °C (metric)
	AirPressure float64 // millibars
}

// StationMeta holds station metadata from the NOAA metadata API.
type StationMeta struct {
	ID       string
	Name     string
	Lat      float64
	Lon      float64
	State    string
	TimeZone string // IANA timezone name, e.g. "America/New_York"
	Datums   []DatumInfo
}

// DatumInfo is one tidal datum and its offset from the station's zero.
type DatumInfo struct {
	Name  string
	Value float64
}

// DailyTide groups tide predictions for a single calendar day, with moon phase.
type DailyTide struct {
	Date        time.Time    // midnight local time for the day
	Predictions []Prediction // high/low events for that day
	MoonPhase   float64      // 0.0–1.0 synodic fraction (0 = new moon)
	MoonName    string       // "New Moon", "Waxing Crescent", etc.
}
