package layout

import (
	"strings"
	"testing"

	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
	"github.com/radix29/tuikit/tuikittest"
)

func row(s *tuikittest.Screen, y int) string {
	return strings.Split(s.Snapshot(), "\n")[y]
}

func TestDrawTitleBarStyleFollowsActive(t *testing.T) {
	for _, active := range []bool{false, true} {
		s := tuikittest.NewScreen(20, 3)
		DrawTitleBar(s, core.Rect{X: 0, Y: 1, W: 20, H: 2}, "Title", active, "")
		want := theme.StyleTitleInactive()
		if active {
			want = theme.StyleTitleActive()
		}
		for _, x := range []int{0, 1, 19} {
			if got := s.StyleAt(x, 1); got != want {
				t.Errorf("active=%v: style at col %d = %v, want %v", active, x, got, want)
			}
		}
		if got := row(s, 1); !strings.HasPrefix(got, " Title") {
			t.Errorf("active=%v: row = %q, want the title one column in", active, got)
		}
		if got := strings.TrimSpace(row(s, 0) + row(s, 2)); got != "" {
			t.Errorf("active=%v: drew outside the first row: %q", active, got)
		}
	}
}

// The title is clipped short of the right cell, which sits one column in
// from the right edge, and the returned rect is where it was drawn.
func TestDrawTitleBarClipsTitleBeforeRightCell(t *testing.T) {
	s := tuikittest.NewScreen(12, 1)
	r := core.Rect{W: 12, H: 1}
	rr := DrawTitleBar(s, r, "ABCDEFGHIJKL", false, "[R]")
	if want := (core.Rect{X: 8, Y: 0, W: 3, H: 1}); rr != want {
		t.Fatalf("right rect = %+v, want %+v", rr, want)
	}
	if rr != TitleBarRightRect(r, "[R]") {
		t.Errorf("DrawTitleBar and TitleBarRightRect disagree")
	}
	if got, want := row(s, 0), " ABCDEF [R]"; got != want {
		t.Errorf("row = %q, want %q", got, want)
	}
}

func TestDrawTitleBarDropsRightCellWhenNarrow(t *testing.T) {
	s := tuikittest.NewScreen(6, 1)
	rr := DrawTitleBar(s, core.Rect{W: 6, H: 1}, "ABCDEFGH", true, "[R]")
	if rr.W != 0 {
		t.Fatalf("right rect = %+v, want zero at width 6", rr)
	}
	if got, want := row(s, 0), " ABCD"; got != want {
		t.Errorf("row = %q, want %q", got, want)
	}
}
