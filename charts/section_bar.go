package charts

import (
	"slices"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// SectionBarHeight is the rows DrawSectionBar's strip takes.
const SectionBarHeight = 1

// DrawSectionBar draws a dashboard section's title strip across r's first
// row, with any KPI readouts right-aligned along it. KPIs are placed from the
// right; one that would crowd the title is dropped, along with every KPI left
// of it, so the title always stays readable.
//
// Each KPI's LabelBg is overridden with the strip's colour: a label left on
// the panel background would punch a hole in the strip.
func DrawSectionBar(s tcell.Screen, r core.Rect, title string, kpis []KPI) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	style := theme.StyleChartSection()
	core.FillRect(s, core.Rect{X: r.X, Y: r.Y, W: r.W, H: SectionBarHeight}, ' ', style)
	core.DrawTextClipped(s, r.X+1, r.Y, r.W-2, style, title)

	right := r.Right()
	for _, k := range slices.Backward(kpis) {
		w := k.Width()
		if right-w <= r.X+core.DisplayWidth(title)+2 {
			return
		}
		k.LabelBg = theme.Active().ChartSectionBg
		k.Draw(s, core.Rect{X: right - w, Y: r.Y, W: w, H: 1})
		right -= w + 1
	}
}
