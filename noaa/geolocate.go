package noaa

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
)

// NearbyStation is a NOAA water-level station with its distance from the user.
type NearbyStation struct {
	ID     string
	Name   string
	State  string
	DistKm float64
}

// NearestWaterLevelStation returns the ID of the single closest NOAA water-level
// station to the caller's IP-geolocated position.
func NearestWaterLevelStation(ctx context.Context) (string, error) {
	stations, err := NearestStations(ctx, 1)
	if err != nil {
		return "", err
	}
	if len(stations) == 0 {
		return "", fmt.Errorf("no stations found")
	}
	return stations[0].ID, nil
}

// NearestStations returns the n closest NOAA water-level stations to the
// caller's IP-geolocated position, sorted by ascending distance.
func NearestStations(ctx context.Context, n int) ([]NearbyStation, error) {
	lat, lon, err := ipGeolocate(ctx)
	if err != nil {
		return nil, fmt.Errorf("geolocation: %w", err)
	}
	return nearestNStations(ctx, lat, lon, n)
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

// ── Station list + proximity ─────────────────────────────────────────────

type stationsListResp struct {
	Stations []struct {
		ID    string  `json:"id"`
		Name  string  `json:"name"`
		State string  `json:"state"`
		Lat   float64 `json:"lat"`
		Lng   float64 `json:"lng"`
	} `json:"stations"`
}

func nearestNStations(ctx context.Context, lat, lon float64, n int) ([]NearbyStation, error) {
	const url = "https://api.tidesandcurrents.noaa.gov/mdapi/prod/webapi/stations.json?type=waterlevels"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fathom/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var list stationsListResp
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, err
	}
	if len(list.Stations) == 0 {
		return nil, fmt.Errorf("no water-level stations returned")
	}

	nearby := make([]NearbyStation, len(list.Stations))
	for i, s := range list.Stations {
		nearby[i] = NearbyStation{
			ID:     s.ID,
			Name:   s.Name,
			State:  s.State,
			DistKm: haversineKm(lat, lon, s.Lat, s.Lng),
		}
	}

	sort.Slice(nearby, func(i, j int) bool {
		return nearby[i].DistKm < nearby[j].DistKm
	})

	if n > 0 && n < len(nearby) {
		nearby = nearby[:n]
	}
	return nearby, nil
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
