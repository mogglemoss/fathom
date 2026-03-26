package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds the persisted user configuration for fathom.
type Config struct {
	StationID string `json:"station_id"` // NOAA station ID, e.g. "8443970" (Boston)
	Units     string `json:"units"`      // "english" or "metric"
	Datum     string `json:"datum"`      // vertical reference datum, e.g. "MLLW"
	Theme     string `json:"theme"`      // "default", "catppuccin", "dracula", "nord"
	Use24h    bool   `json:"use_24h"`
}

// DefaultConfig returns sensible defaults (Boston, English units, MLLW datum).
func DefaultConfig() Config {
	return Config{
		StationID: "8443970",
		Units:     "english",
		Datum:     "MLLW",
		Theme:     "default",
		Use24h:    false,
	}
}

// Load reads ~/.config/fathom/config.json.
// Returns DefaultConfig() on any error — config is always best-effort.
func Load() (Config, error) {
	cfg := DefaultConfig()
	data, err := os.ReadFile(filepath.Join(configDir(), "config.json"))
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}
	// Fill missing fields with defaults.
	if cfg.StationID == "" {
		cfg.StationID = DefaultConfig().StationID
	}
	if cfg.Units == "" {
		cfg.Units = DefaultConfig().Units
	}
	if cfg.Datum == "" {
		cfg.Datum = DefaultConfig().Datum
	}
	return cfg, nil
}

// Save writes the config to ~/.config/fathom/config.json.
// Errors are silently ignored — config persistence is best-effort.
func Save(c Config) error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "config.json"), data, 0o600)
}

func configDir() string {
	base, err := os.UserConfigDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "fathom")
}
