package noaa

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/mogglemoss/fathom/config"
)

const (
	dataBaseURL    = "https://api.tidesandcurrents.noaa.gov/api/prod/datagetter"
	stationBaseURL = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations"
	httpTimeout    = 15 * time.Second
)

// Client fetches NOAA tide and meteorological data for a configured station.
type Client struct {
	http      *http.Client
	stationID string
	units     string
	datum     string
}

// NewClient creates a Client from the given config.
func NewClient(cfg config.Config) *Client {
	return &Client{
		http:      &http.Client{Timeout: httpTimeout},
		stationID: cfg.StationID,
		units:     cfg.Units,
		datum:     cfg.Datum,
	}
}

// FetchWaterLevel fetches the last 24 hours of water level observations.
func (c *Client) FetchWaterLevel(ctx context.Context, loc *time.Location) ([]WaterObs, error) {
	params := c.baseParams("water_level")
	params.Set("range", "24")
	body, err := c.get(ctx, dataBaseURL, params)
	if err != nil {
		return nil, err
	}
	return ParseWaterLevels(body, loc)
}

// FetchPredictions fetches high/low tide predictions for the given date range.
func (c *Client) FetchPredictions(ctx context.Context, begin, end time.Time, loc *time.Location) ([]Prediction, error) {
	params := c.baseParams("predictions")
	params.Set("interval", "hilo")
	params.Set("begin_date", begin.Format("20060102"))
	params.Set("end_date", end.Format("20060102"))
	body, err := c.get(ctx, dataBaseURL, params)
	if err != nil {
		return nil, err
	}
	return ParsePredictions(body, loc)
}

// FetchMet fetches the latest wind, air temperature, and air pressure readings
// by making three concurrent requests and merging them into a single MetObs.
func (c *Client) FetchMet(ctx context.Context, loc *time.Location) (MetObs, error) {
	type result struct {
		product string
		body    []byte
		err     error
	}

	products := []string{"wind", "air_temperature", "air_pressure"}
	results := make([]result, len(products))
	var wg sync.WaitGroup

	for i, product := range products {
		wg.Add(1)
		go func(idx int, p string) {
			defer wg.Done()
			params := c.baseParams(p)
			params.Set("range", "1")
			body, err := c.get(ctx, dataBaseURL, params)
			results[idx] = result{product: p, body: body, err: err}
		}(i, product)
	}
	wg.Wait()

	var met MetObs

	// Wind
	if results[0].err == nil {
		speed, dir, gust, t, err := ParseWind(results[0].body, loc)
		if err == nil {
			met.WindSpeed = speed
			met.WindDir = dir
			met.WindGust = gust
			met.Time = t
		}
	}

	// Air temperature
	if results[1].err == nil {
		v, _, err := ParseScalar(results[1].body, loc)
		if err == nil {
			met.AirTemp = v
		}
	}

	// Air pressure
	if results[2].err == nil {
		v, _, err := ParseScalar(results[2].body, loc)
		if err == nil {
			met.AirPressure = v
		}
	}

	return met, nil
}

// FetchStation fetches station metadata from the NOAA mdapi.
func (c *Client) FetchStation(ctx context.Context) (StationMeta, error) {
	u := fmt.Sprintf("%s/%s.json", stationBaseURL, c.stationID)
	body, err := c.get(ctx, u, url.Values{})
	if err != nil {
		return StationMeta{}, err
	}
	meta, err := ParseStation(body)
	if err != nil {
		return StationMeta{}, err
	}

	// Best-effort datums fetch.
	datumsURL := fmt.Sprintf("%s/%s/datums.json", stationBaseURL, c.stationID)
	if db, derr := c.get(ctx, datumsURL, url.Values{"units": []string{c.units}}); derr == nil {
		if datums, derr := ParseDatums(db); derr == nil {
			meta.Datums = datums
		}
	}

	return meta, nil
}

// ── Internal helpers ──────────────────────────────────────────────────────────

func (c *Client) baseParams(product string) url.Values {
	p := url.Values{}
	p.Set("station", c.stationID)
	p.Set("product", product)
	p.Set("datum", c.datum)
	p.Set("units", c.units)
	p.Set("time_zone", "lst_ldt")
	p.Set("format", "json")
	p.Set("application", "fathom")
	return p
}

func (c *Client) get(ctx context.Context, base string, params url.Values) ([]byte, error) {
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	if len(params) > 0 {
		u.RawQuery = params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NOAA returned HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
