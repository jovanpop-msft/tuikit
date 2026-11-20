package charts

import (
	"strings"
	"testing"

	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

func readoutFixture() []ReadoutRow {
	return []ReadoutRow{
		{Label: "Data", Value: "12 MB", Color: colA},
		{Label: "Log", Value: "3 MB", Color: colB},
	}
}

func TestReadoutSizeFitsWidestLine(t *testing.T) {
	w, h := ReadoutSize("12:00:05", readoutFixture())
	// "Data" + "12 MB" + 3 gap = 12, wider than the 8-cell caption; frame 2,
	// padding 2. Rows 2, caption 1, frame 2.
	if w != 16 || h != 5 {
		t.Errorf("ReadoutSize = %dx%d, want 16x5", w, h)
	}
	if w, _ := ReadoutSize("a much longer caption", nil); w != 21+4 {
		t.Errorf("caption-only width = %d, want %d", w, 21+4)
	}
}

func TestPlaceReadoutRightBelowWhenItFits(t *testing.T) {
	view := core.Rect{X: 0, Y: 0, W: 80, H: 24}
	r := PlaceReadout(10, 4, 5, 5, view, -1)
	if r.X != 7 || r.Y != 6 {
		t.Errorf("placed at %d,%d, want 7,6 (right of and below the point)", r.X, r.Y)
	}
}

func TestPlaceReadoutFlipsAtTheEdges(t *testing.T) {
	view := core.Rect{X: 0, Y: 0, W: 40, H: 20}
	r := PlaceReadout(10, 4, 35, 18, view, -1)
	if r.X != 35-10-1 {
		t.Errorf("x = %d, want flipped left to %d", r.X, 35-10-1)
	}
	if r.Y != 18-4 {
		t.Errorf("y = %d, want flipped above to %d", r.Y, 18-4)
	}
}

func TestPlaceReadoutClampsIntoView(t *testing.T) {
	view := core.Rect{X: 2, Y: 1, W: 12, H: 5}
	r := PlaceReadout(20, 8, 3, 2, view, -1)
	if r.X != view.X || r.Y != view.Y {
		t.Errorf("placed at %d,%d, want clamped to the view's corner %d,%d", r.X, r.Y, view.X, view.Y)
	}
}

func TestPlaceReadoutAvoidsKeepOutRow(t *testing.T) {
	view := core.Rect{X: 0, Y: 0, W: 80, H: 24}
	r := PlaceReadout(10, 4, 5, 10, view, 12)
	if r.Y != 10-4 {
		t.Errorf("y = %d, want above the point (%d) to clear keep-out row 12", r.Y, 10-4)
	}
	// No room above: stay below even though it covers the row.
	if r := PlaceReadout(10, 4, 5, 2, view, 4); r.Y != 3 {
		t.Errorf("y = %d, want 3 (below; above does not fit)", r.Y)
	}
}

func TestDrawReadoutCaptionRowsAndColours(t *testing.T) {
	rows := readoutFixture()
	w, h := ReadoutSize("12:00:05", rows)
	c := core.NewBuffer(w, h)
	DrawReadout(c, c.Rect(), "12:00:05", rows)

	if !strings.Contains(c.Row(1), "12:00:05") {
		t.Errorf("row 1 = %q, want the caption", c.Row(1))
	}
	row := c.Row(2)
	if !strings.Contains(row, "Data") || !strings.HasSuffix(strings.TrimRight(row, "│ "), "12 MB") {
		t.Errorf("row 2 = %q, want the label and its value right-aligned", row)
	}
	_, style, _ := c.Get(2, 2) // the "D" of "Data"
	if style.GetForeground() != colA {
		t.Errorf("label colour = %v, want the series colour %v", style.GetForeground(), colA)
	}
	_, vstyle, _ := c.Get(w-3, 2) // the "B" of "12 MB"
	if vstyle.GetForeground() == colA {
		t.Error("value painted in the series colour, want the body style")
	}
	if _, bstyle, _ := c.Get(0, 0); bstyle != theme.StyleTooltipBorder() {
		t.Errorf("frame style = %v, want StyleTooltipBorder", bstyle)
	}
}

func TestDrawReadoutClipsRowsThatDoNotFit(t *testing.T) {
	rows := readoutFixture()
	w, _ := ReadoutSize("cap", rows)
	c := core.NewBuffer(w, 4) // room for the caption and one row
	DrawReadout(c, c.Rect(), "cap", rows)
	for y := range 4 {
		if strings.Contains(c.Row(y), "Log") {
			t.Errorf("row %d = %q, want the second row clipped", y, c.Row(y))
		}
	}
}

// axisFixture is a 30×1 canvas holding a time row of age labels, shown through
// a viewport at screen (2,5) scrolled 3 columns right.
func axisFixture() (*core.Buffer, *core.Viewport, core.Rect) {
	c := core.NewBuffer(30, 1)
	core.DrawText(c, 0, 0, theme.StyleChartAxis(), "-0:40     -0:30     -0:20    0")
	vp := &core.Viewport{X: 3, View: core.Rect{X: 2, Y: 5, W: 20, H: 1}, ContentW: 30, ContentH: 1}
	return c, vp, core.Rect{X: 0, Y: 0, W: 30, H: 1}
}

func blitAxis(c *core.Buffer, vp *core.Viewport) *core.Buffer {
	s := core.NewBuffer(24, 7)
	c.Blit(s, core.Rect{X: vp.X, Y: vp.Y, W: vp.View.W, H: vp.View.H}, vp.View)
	return s
}

func TestDrawAxisCalloutClearsOverlappedLabelsWhole(t *testing.T) {
	c, vp, row := axisFixture()
	s := blitAxis(c, vp)
	// Screen column 12 is canvas 13: inside "-0:30" (canvas 10..14).
	y := DrawAxisCallout(s, c, vp, row, 12, "1:34")
	if y != 5 {
		t.Fatalf("row = %d, want 5", y)
	}
	got := s.Row(5)[2:22]
	// The view starts at canvas 3 ("-0:40" shows as "40"). Callout centred at
	// screen 10..13 (canvas 11..14); the "-0:30" label it touches is cleared
	// whole, not left as a "-" stub.
	want := "40      1:34     -0:"
	if got != want {
		t.Errorf("row = %q, want %q", got, want)
	}
}

func TestDrawAxisCalloutStaysInsideTheView(t *testing.T) {
	c, vp, row := axisFixture()
	s := blitAxis(c, vp)
	if y := DrawAxisCallout(s, c, vp, row, 21, "12:00:05"); y != 5 {
		t.Fatalf("row = %d, want 5", y)
	}
	if got := s.Row(5)[14:22]; got != "12:00:05" {
		t.Errorf("right edge = %q, want the callout clamped flush to the view", got)
	}
}

func TestDrawAxisCalloutNoRoom(t *testing.T) {
	c, vp, row := axisFixture()
	s := blitAxis(c, vp)
	if y := DrawAxisCallout(s, c, vp, row, 10, strings.Repeat("9", 21)); y != -1 {
		t.Errorf("wider than the view: row = %d, want -1", y)
	}
	vp.Y = 1 // the time row scrolled out of view
	vp.ContentH = 2
	if y := DrawAxisCallout(s, c, vp, row, 10, "1:34"); y != -1 {
		t.Errorf("row off screen: got %d, want -1", y)
	}
	if y := DrawAxisCallout(s, c, vp, row, 10, ""); y != -1 {
		t.Errorf("empty text: got %d, want -1", y)
	}
}
