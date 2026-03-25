package noaa

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// noaaTimeLayout is the timestamp format used in NOAA API responses.
const noaaTimeLayout = "2006/01/02 15:04"

// ── Water level ───────────────────────────────────────────────────────────────

type rawWaterResp struct {
	Error *rawError  `json:"error"`
	Data  []rawWater `json:"data"`
}

type rawWater struct {
	T string `json:"t"` // "2024/01/15 00:00"
	V string `json:"v"` // "3.156"
	Q string `json:"q"` // "p" or "v"
}

// ParseWaterLevels parses a NOAA water_level JSON response into WaterObs.
func ParseWaterLevels(body []byte, loc *time.Location) ([]WaterObs, error) {
	var raw rawWaterResp
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("water_level: %w", err)
	}
	if raw.Error != nil {
		return nil, fmt.Errorf("water_level API error: %s", raw.Error.Message)
	}
	if loc == nil {
		loc = time.UTC
	}
	obs := make([]WaterObs, 0, len(raw.Data))
	for _, d := range raw.Data {
		t, err := time.ParseInLocation(noaaTimeLayout, d.T, loc)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(d.V, 64)
		if err != nil {
			continue
		}
		obs = append(obs, WaterObs{Time: t, Level: v, QC: d.Q})
	}
	return obs, nil
}

// ── Predictions ───────────────────────────────────────────────────────────────

type rawPredResp struct {
	Error       *rawError   `json:"error"`
	Predictions []rawPred   `json:"predictions"`
	Data        []rawPred   `json:"data"` // some products use "data" instead
}

type rawPred struct {
	T    string `json:"t"`
	V    string `json:"v"`
	Type string `json:"type"` // "H", "HH", "L", "LL"
}

// ParsePredictions parses a NOAA predictions or high_low JSON response.
func ParsePredictions(body []byte, loc *time.Location) ([]Prediction, error) {
	var raw rawPredResp
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("predictions: %w", err)
	}
	if raw.Error != nil {
		return nil, fmt.Errorf("predictions API error: %s", raw.Error.Message)
	}
	if loc == nil {
		loc = time.UTC
	}
	src := raw.Predictions
	if len(src) == 0 {
		src = raw.Data
	}
	preds := make([]Prediction, 0, len(src))
	for _, d := range src {
		t, err := time.ParseInLocation(noaaTimeLayout, d.T, loc)
		if err != nil {
			continue
		}
		v, err := strconv.ParseFloat(d.V, 64)
		if err != nil {
			continue
		}
		isHigh := d.Type == "H" || d.Type == "HH"
		preds = append(preds, Prediction{Time: t, Level: v, IsHigh: isHigh})
	}
	return preds, nil
}

// ── Meteorology ───────────────────────────────────────────────────────────────

type rawMetResp struct {
	Error *rawError `json:"error"`
	Data  []rawMet  `json:"data"`
}

type rawMet struct {
	T  string `json:"t"`
	V  string `json:"v"`   // for scalar products (temperature, pressure)
	S  string `json:"s"`   // wind speed
	D  string `json:"d"`   // wind direction degrees
	G  string `json:"g"`   // wind gust
	Dr string `json:"dr"`  // direction name (unused, we compute our own)
}

// ParseWind parses a NOAA wind JSON response, returning the most recent obs.
func ParseWind(body []byte, loc *time.Location) (speed, dir, gust float64, t time.Time, err error) {
	var raw rawMetResp
	if err = json.Unmarshal(body, &raw); err != nil {
		return
	}
	if raw.Error != nil {
		err = fmt.Errorf("wind API error: %s", raw.Error.Message)
		return
	}
	if len(raw.Data) == 0 {
		err = fmt.Errorf("wind: no data")
		return
	}
	if loc == nil {
		loc = time.UTC
	}
	d := raw.Data[len(raw.Data)-1]
	t, _ = time.ParseInLocation(noaaTimeLayout, d.T, loc)
	speed, _ = strconv.ParseFloat(d.S, 64)
	dir, _ = strconv.ParseFloat(d.D, 64)
	gust, _ = strconv.ParseFloat(d.G, 64)
	return
}

// ParseScalar parses a NOAA scalar (temperature or pressure) response, returning the most recent value.
func ParseScalar(body []byte, loc *time.Location) (float64, time.Time, error) {
	var raw rawMetResp
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, time.Time{}, fmt.Errorf("scalar: %w", err)
	}
	if raw.Error != nil {
		return 0, time.Time{}, fmt.Errorf("scalar API error: %s", raw.Error.Message)
	}
	if len(raw.Data) == 0 {
		return 0, time.Time{}, fmt.Errorf("scalar: no data")
	}
	if loc == nil {
		loc = time.UTC
	}
	d := raw.Data[len(raw.Data)-1]
	t, _ := time.ParseInLocation(noaaTimeLayout, d.T, loc)
	v, err := strconv.ParseFloat(d.V, 64)
	return v, t, err
}

// ── Station metadata ──────────────────────────────────────────────────────────

type rawStationResp struct {
	Stations []rawStation `json:"stations"`
}

type rawStation struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	State    string  `json:"state"`
	TimeZone string  `json:"timezone"`
}

// ParseStation parses a NOAA mdapi station JSON response.
func ParseStation(body []byte) (StationMeta, error) {
	var raw rawStationResp
	if err := json.Unmarshal(body, &raw); err != nil {
		return StationMeta{}, fmt.Errorf("station: %w", err)
	}
	if len(raw.Stations) == 0 {
		return StationMeta{}, fmt.Errorf("station: no station data returned")
	}
	s := raw.Stations[0]
	return StationMeta{
		ID:       s.ID,
		Name:     s.Name,
		Lat:      s.Lat,
		Lon:      s.Lng,
		State:    s.State,
		TimeZone: s.TimeZone,
	}, nil
}

// ── Datums ────────────────────────────────────────────────────────────────────

type rawDatumsResp struct {
	Datums []rawDatum `json:"datums"`
}

type rawDatum struct {
	Name  string  `json:"n"`
	Value float64 `json:"v"`
}

// ParseDatums parses a NOAA datums JSON response.
func ParseDatums(body []byte) ([]DatumInfo, error) {
	var raw rawDatumsResp
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("datums: %w", err)
	}
	datums := make([]DatumInfo, 0, len(raw.Datums))
	for _, d := range raw.Datums {
		datums = append(datums, DatumInfo{Name: d.Name, Value: d.Value})
	}
	return datums, nil
}

// ── Error helper ──────────────────────────────────────────────────────────────

type rawError struct {
	Message string `json:"message"`
}
