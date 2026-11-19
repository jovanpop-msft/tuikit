package layout

import (
	"github.com/gdamore/tcell/v3"

	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// TitleBarRightRect is where DrawTitleBar puts right on a title bar at r: the
// right end of r's first row, one column in from the edge. Zero when right is
// empty or r is too narrow to leave the title a few columns beside it. A
// host that hit-tests the cell (a refresh button) computes it here, in
// SetBounds, so its hit-test and the drawing share one layout.
func TitleBarRightRect(r core.Rect, right string) core.Rect {
	w := core.DisplayWidth(right)
	if w == 0 || r.W < w+4 {
		return core.Rect{}
	}
	return core.Rect{X: r.X + r.W - w - 1, Y: r.Y, W: w, H: 1}
}

// DrawTitleBar paints a panel's one-row title bar on r's first row:
// theme.StyleTitleActive when active, theme.StyleTitleInactive otherwise,
// title clipped one column in from each edge and short of right, which is
// drawn right-aligned (TitleBarRightRect) in the same style. It returns the
// right cell's rect, zero when nothing was drawn there.
func DrawTitleBar(s tcell.Screen, r core.Rect, title string, active bool, right string) core.Rect {
	if r.W <= 0 || r.H <= 0 {
		return core.Rect{}
	}
	st := theme.StyleTitleInactive()
	if active {
		st = theme.StyleTitleActive()
	}
	core.FillRect(s, core.Rect{X: r.X, Y: r.Y, W: r.W, H: 1}, ' ', st)
	titleW := r.W - 2
	rr := TitleBarRightRect(r, right)
	if rr.W > 0 {
		titleW = rr.X - r.X - 2
		core.DrawText(s, rr.X, rr.Y, st, right)
	}
	core.DrawTextClipped(s, r.X+1, r.Y, titleW, st, title)
	return rr
}
