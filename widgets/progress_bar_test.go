package widgets

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

// progressBarText is DrawProgressBar's row at width w and pct, trailing blanks
// trimmed, on a screen wider than the bar so an overrun would show.
func progressBarText(w, pct int) string {
	s := tuikittest.NewScreen(w+10, 1)
	DrawProgressBar(s, 0, 0, w, pct, tcell.StyleDefault)
	return strings.TrimRight(s.Text(0, 0, w+10), " ")
}

func TestDrawProgressBar(t *testing.T) {
	for _, tc := range []struct {
		w, pct int
		want   string
	}{
		{15, 0, "░░░░░░░░░░ 0%"},
		{15, 40, "████░░░░░░ 40%"},
		{15, 100, "██████████ 100%"},
		// Over 100 clamps, so the label never overruns the w columns.
		{15, 250, "██████████ 100%"},
		// Indeterminate: an all-shaded bar, no percentage.
		{15, -1, "░░░░░░░░░░"},
		// Rounds the filled part down: 3 cells of 7 at 49%.
		{12, 49, "███░░░░ 49%"},
		// One-cell bar is the narrowest drawn.
		{6, 100, "█ 100%"},
		{5, 50, ""},
		{0, 50, ""},
	} {
		if got := progressBarText(tc.w, tc.pct); got != tc.want {
			t.Errorf("w=%d pct=%d: got %q, want %q", tc.w, tc.pct, got, tc.want)
		}
	}
}
