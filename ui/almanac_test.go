package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/mogglemoss/fathom/noaa"
)

// TestRenderAlmanacStartsAtCursor verifies that when cursor=5, the rendered
// almanac shows days[5] as its first data row — not days[0].
func TestRenderAlmanacStartsAtCursor(t *testing.T) {
	loc := time.UTC
	days := make([]noaa.DailyTide, 10)
	for i := range days {
		days[i] = noaa.DailyTide{
			Date: time.Date(2026, 4, 1+i, 0, 0, 0, 0, loc),
		}
	}

	// Render with cursor at index 5 (Apr 6).
	out := RenderAlmanacView(days, 5, 120, 40)

	wantFirst := "Apr  6"
	wantAbsent := "Apr  1" // days before cursor should not appear

	if !strings.Contains(out, wantFirst) {
		t.Errorf("expected first visible day %q to appear in output\n%s", wantFirst, out)
	}
	if strings.Contains(out, wantAbsent) {
		t.Errorf("day %q should not appear when cursor=5, but it does\n%s", wantAbsent, out)
	}
}

func TestAlmanacScrollOffsetCursorAtTop(t *testing.T) {
	cases := []struct {
		cursor     int
		visible    int
		total      int
		wantOffset int
	}{
		// All 14 days fit on screen — offset should equal cursor so entered date is first row.
		{cursor: 0, visible: 35, total: 14, wantOffset: 0},
		{cursor: 5, visible: 35, total: 14, wantOffset: 5},
		{cursor: 13, visible: 35, total: 14, wantOffset: 13},
		// Cursor at last day — should not exceed totalDays-1.
		{cursor: 20, visible: 35, total: 14, wantOffset: 13},
		// Small visible window — cursor past end of visible, still returns cursor.
		{cursor: 3, visible: 5, total: 14, wantOffset: 3},
	}
	for _, c := range cases {
		got := AlmanacScrollOffset(c.cursor, c.visible, c.total)
		if got != c.wantOffset {
			t.Errorf("AlmanacScrollOffset(cursor=%d, visible=%d, total=%d) = %d, want %d",
				c.cursor, c.visible, c.total, got, c.wantOffset)
		}
	}
}
