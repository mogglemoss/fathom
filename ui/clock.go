package ui

import (
	"fmt"
	"time"
)

// Use24h controls whether time displays use 24-hour clock format app-wide.
// Set by the model when the user toggles with the 'c' key.
var Use24h bool

// FmtTideTime formats a time per the current clock preference.
// Always returns 6 chars to maintain column alignment.
func FmtTideTime(t time.Time) string {
	if Use24h {
		return fmt.Sprintf("%6s", t.Format("15:04"))
	}
	// am/pm: " 4:32p" or "11:58a" — always 6 chars
	sfx := "a"
	if t.Hour() >= 12 {
		sfx = "p"
	}
	return fmt.Sprintf("%5s%s", t.Format("3:04"), sfx)
}
