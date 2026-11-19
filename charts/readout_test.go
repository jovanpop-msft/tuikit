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
