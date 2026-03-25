package noaa

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
)

// NearestWaterLevelStation returns the ID of the nearest NOAA water-level
// station to the caller's IP-geolocated position. It fetches the full list of
// NOAA water-level stations and finds the closest one using haversine distance.
func NearestWaterLevelStation(ctx context.Context) (string, error) {
	lat, lon, err := ipGeolocate(ctx)
	if err != nil {
		return "", fmt.Errorf("geolocation: %w", err)
	}
	return nearestStation(ctx, lat, lon)
}

// ── IP geolocation ─────────────────────────────────────────────────────────

type ipGeoResp struct {
	Lat float64 `json:"latitude"`
	Lon float64 `json:"longitude"`
}

func ipGeolocate(ctx context.Context) (lat, lon float64, err error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://ipapi.co/json/", nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "fathom/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	var geo ipGeoResp
	if err := json.NewDecoder(resp.Body).Decode(&geo); err != nil {
		return 0, 0, err
	}
	if geo.Lat == 0 && geo.Lon == 0 {
		return 0, 0, fmt.Errorf("empty coordinates returned by geolocation API")
	}
	return geo.Lat, geo.Lon, nil
}

// ── Station proximity ─────────────────────────────────────────────────────

type stationsListResp struct {
	Stations []struct {
		ID  string  `json:"id"`
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	} `json:"stations"`
}

func nearestStation(ctx context.Context, lat, lon float64) (string, error) {
	const url = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations.json?type=waterlevels"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "fathom/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var list stationsListResp
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return "", err
	}
	if len(list.Stations) == 0 {
		return "", fmt.Errorf("no water-level stations returned")
	}

	best := ""
	bestDist := math.MaxFloat64
	for _, s := range list.Stations {
		if d := haversineKm(lat, lon, s.Lat, s.Lng); d < bestDist {
			bestDist = d
			best = s.ID
		}
	}
	return best, nil
}

// haversineKm returns the great-circle distance in km between two lat/lon points.
func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	return R * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
