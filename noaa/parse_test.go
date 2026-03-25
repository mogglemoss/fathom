package noaa

import (
	"testing"
	"time"
)

func TestParseWaterLevels(t *testing.T) {
	body := []byte(`{
		"metadata": {"id":"8443970","name":"Boston","lat":"42.35","lon":"-71.05"},
		"data": [
			{"t":"2026-03-25 12:00","v":"0.379","q":"p"},
			{"t":"2026-03-25 12:06","v":"0.448","q":"p"},
			{"t":"2026-03-25 12:12","v":"0.603","q":"v"}
		]
	}`)

	obs, err := ParseWaterLevels(body, time.UTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(obs) != 3 {
		t.Fatalf("expected 3 observations, got %d", len(obs))
	}
	if obs[0].Level != 0.379 {
		t.Errorf("expected level 0.379, got %f", obs[0].Level)
	}
	if obs[0].QC != "p" {
		t.Errorf("expected QC 'p', got %q", obs[0].QC)
	}
	if obs[2].QC != "v" {
		t.Errorf("expected QC 'v', got %q", obs[2].QC)
	}
	if obs[0].Time.Hour() != 12 || obs[0].Time.Minute() != 0 {
		t.Errorf("unexpected time: %v", obs[0].Time)
	}
}

func TestParseWaterLevels_APIError(t *testing.T) {
	body := []byte(`{"error":{"message":"No data was found."}}`)
	_, err := ParseWaterLevels(body, time.UTC)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParsePredictions(t *testing.T) {
	body := []byte(`{
		"predictions": [
			{"t":"2026-03-25 04:35","v":"10.357","type":"H"},
			{"t":"2026-03-25 11:09","v":"0.005","type":"L"},
			{"t":"2026-03-25 17:23","v":"8.709","type":"HH"},
			{"t":"2026-03-25 23:24","v":"1.092","type":"LL"}
		]
	}`)

	preds, err := ParsePredictions(body, time.UTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(preds) != 4 {
		t.Fatalf("expected 4 predictions, got %d", len(preds))
	}
	if !preds[0].IsHigh {
		t.Error("expected first prediction to be high tide")
	}
	if preds[1].IsHigh {
		t.Error("expected second prediction to be low tide")
	}
	if !preds[2].IsHigh {
		t.Error("expected HH to be treated as high tide")
	}
	if preds[3].IsHigh {
		t.Error("expected LL to be treated as low tide")
	}
	if preds[0].Level != 10.357 {
		t.Errorf("expected level 10.357, got %f", preds[0].Level)
	}
}

func TestParseWind(t *testing.T) {
	body := []byte(`{
		"data": [
			{"t":"2026-03-25 12:00","s":"5.0","d":"90.0","g":"7.5","dr":"E"},
			{"t":"2026-03-25 12:06","s":"11.50","d":"170.0","g":"13.59","dr":"S"}
		]
	}`)

	speed, dir, gust, ts, err := ParseWind(body, time.UTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if speed != 11.50 {
		t.Errorf("expected speed 11.50, got %f", speed)
	}
	if dir != 170.0 {
		t.Errorf("expected dir 170.0, got %f", dir)
	}
	if gust != 13.59 {
		t.Errorf("expected gust 13.59, got %f", gust)
	}
	if ts.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestParseScalar(t *testing.T) {
	body := []byte(`{"data":[{"t":"2026-03-25 12:00","v":"42.3","f":"0,0"}]}`)
	v, ts, err := ParseScalar(body, time.UTC)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v != 42.3 {
		t.Errorf("expected 42.3, got %f", v)
	}
	if ts.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestParseStation(t *testing.T) {
	body := []byte(`{
		"count": 1,
		"stations": [{
			"id": "8443970",
			"name": "Boston",
			"lat": 42.35389,
			"lng": -71.05028,
			"state": "MA",
			"timezone": "EST",
			"timezonecorr": -5
		}]
	}`)

	meta, err := ParseStation(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.ID != "8443970" {
		t.Errorf("expected ID 8443970, got %q", meta.ID)
	}
	if meta.Name != "Boston" {
		t.Errorf("expected name Boston, got %q", meta.Name)
	}
	if meta.TimeZone != "America/New_York" {
		t.Errorf("expected America/New_York from EST, got %q", meta.TimeZone)
	}
	if meta.Lat != 42.35389 {
		t.Errorf("expected lat 42.35389, got %f", meta.Lat)
	}
}

func TestParseStation_NoData(t *testing.T) {
	body := []byte(`{"count":0,"stations":[]}`)
	_, err := ParseStation(body)
	if err == nil {
		t.Fatal("expected error for empty stations, got nil")
	}
}

func TestNoaaTimezoneToIANA(t *testing.T) {
	cases := []struct {
		abbr string
		corr int
		want string
	}{
		{"EST", -5, "America/New_York"},
		{"CST", -6, "America/Chicago"},
		{"PST", -8, "America/Los_Angeles"},
		{"HST", -10, "Pacific/Honolulu"},
		{"UNKNOWN", -7, "Etc/GMT+7"},
	}
	for _, c := range cases {
		got := noaaTimezoneToIANA(c.abbr, c.corr)
		if got != c.want {
			t.Errorf("noaaTimezoneToIANA(%q, %d) = %q, want %q", c.abbr, c.corr, got, c.want)
		}
	}
}

func TestParseDatums(t *testing.T) {
	// Real NOAA datums API uses "name" and "value" fields (not "n"/"v")
	body := []byte(`{
		"datums": [
			{"name":"MLLW","description":"Mean Lower-Low Water","value":0.0},
			{"name":"MHW","description":"Mean High Water","value":9.532},
			{"name":"MHHW","description":"Mean Higher-High Water","value":9.861}
		]
	}`)
	datums, err := ParseDatums(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(datums) != 3 {
		t.Fatalf("expected 3 datums, got %d", len(datums))
	}
	if datums[1].Name != "MHW" || datums[1].Value != 9.532 {
		t.Errorf("unexpected datum: %+v", datums[1])
	}
}
