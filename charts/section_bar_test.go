package charts

import (
	"strings"
	"testing"

	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// The strip is filled edge to edge in the section style, the title sits one
// column in, and KPIs anchor at the right end.
func TestDrawSectionBarFillsTitleAndRightAlignsKPIs(t *testing.T) {
	c := core.NewBuffer(40, 2)
	DrawSectionBar(c, core.Rect{X: 0, Y: 0, W: 40, H: 1}, "ACTIVITY", []KPI{{Label: "Conn", Value: "42"}})

	row := c.Rows()[0]
	if !strings.HasPrefix(row, " ACTIVITY") {
		t.Errorf("row = %q, want the title one column in", row)
	}
	if !strings.HasSuffix(row, "Conn 42 ") {
		t.Errorf("row = %q, want the KPI right-aligned", row)
	}
	want := theme.StyleChartSection()
	for _, x := range []int{0, 20} {
		if got := c.StyleAt(x, 0); got != want {
			t.Errorf("column %d style = %v, want the section style %v", x, got, want)
		}
	}
	if got := c.RuneAt(0, 1); got != 0 && got != ' ' {
		t.Errorf("the row under a one-row strip was drawn: %q", got)
	}
}

// A KPI's label is drawn on the strip's colour, not the panel's: a label
// left on its default would punch a hole in the strip.
func TestDrawSectionBarKPILabelSitsOnTheStrip(t *testing.T) {
	c := core.NewBuffer(40, 1)
	DrawSectionBar(c, c.Rect(), "T", []KPI{{Label: "Conn", Value: "42"}})

	x := strings.Index(c.Rows()[0], "Conn")
	bg := c.StyleAt(x, 0).GetBackground()
	if want := theme.Active().ChartSectionBg; bg != want {
		t.Errorf("KPI label background = %v, want the strip's %v", bg, want)
	}
}

// KPIs are placed from the right and the one that would crowd the title is
// dropped with everything left of it, so the title stays readable.
func TestDrawSectionBarDropsKPIsThatCrowdTheTitle(t *testing.T) {
	c := core.NewBuffer(30, 1)
	DrawSectionBar(c, c.Rect(), "A LONG SECTION", []KPI{
		{Label: "First", Value: "1"},
		{Label: "Last", Value: "2"},
	})

	row := c.Rows()[0]
	if !strings.Contains(row, "A LONG SECTION") || !strings.Contains(row, "Last") {
		t.Errorf("row = %q, want the title and the rightmost KPI", row)
	}
	if strings.Contains(row, "First") {
		t.Errorf("row = %q, the KPI that would crowd the title was drawn", row)
	}
}

func TestDrawSectionBarIgnoresAnEmptyRect(t *testing.T) {
	c := core.NewBuffer(10, 1)
	DrawSectionBar(c, core.Rect{X: 0, Y: 0, W: 0, H: 1}, "T", nil)
	if got := c.RuneAt(1, 0); got == 'T' {
		t.Error("a zero-width strip drew its title")
	}
}
