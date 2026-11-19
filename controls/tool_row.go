package controls

import (
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// ToolRowGap is the blank column between two ToolRow cells; a cell's label is
// drawn with one space of padding either side.
const ToolRowGap = 1

// ToolCell is one clickable cell of a ToolRow.
//
// Selected and Disabled are what Draw paints (Disabled unless the host passes its
// own predicate); the row never enforces them outside Run. A host that dims a cell
// still refuses the click through Run, since a dimmed control that acts on a click is the failure the
// context-gating rule exists to prevent. A host whose gate is a predicate asked
// per draw and per click (one that depends on a probe not yet run when the row was
// built) leaves them unset.
type ToolCell struct {
	Label    string
	Selected bool
	Disabled bool
	Action   func()

	// Reason is what to tell the user when they click the cell while it is
	// disabled — the rights they are missing, typically. Swallowing the click
	// silently is the thing the context-gating rule exists to prevent. Empty
	// for a cell whose greyed state speaks for itself (a Refresh already
	// running).
	Reason string

	// Rect is where Layout placed the cell; zero when it is hidden, which is
	// "neither drawn nor hit-tested". Truncating instead would put half a label
	// on the row and still accept clicks on it.
	Rect core.Rect
}

// ToolRow is a panel's one-row text toolbar: layout, hit-testing, the "More ▾"
// overflow, Run (the gate every path that acts on a cell goes through) and Draw.
// What a cell does and when it is dimmed stay with the host, because the hosts
// disagree on it (one dims per cell from its Disabled field, one dims the whole
// row while a read is in flight, one asks a predicate per cell); Draw takes both
// as arguments. Toolbar is the other kind: an application's icon strip.
type ToolRow struct {
	Cells []ToolCell

	// More is the stand-in cell for the hidden ones: zero Rect when everything
	// fitted or the row is too narrow even for it.
	More ToolCell

	// Hidden holds the indexes of the Cells Layout could not place — always a
	// suffix of the row, so the overflow menu holds the row's tail rather than
	// a scattered subset of it.
	Hidden []int

	// NoOverflow drops the cells that do not fit instead of collapsing them
	// behind More. Only for a row whose host sizes the labels to fit (Plan
	// Compare's two pickers): a dropped cell is unreachable by mouse.
	NoOverflow bool
}

// Layout places the cells left to right inside r, after prefix (a label the host
// draws, such as "Refresh rate:"), and returns the column past the last cell
// placed plus one gap — More included — which is where a host puts whatever shares
// the row (a filter field, a right-aligned status).
//
// Cells that do not fit are hidden right to left by the same policy Toolbar uses
// (hideToFit), with room for More taken out of the row first, or the cell that
// stands in for the overflow would overflow. More is placed even when it holds the
// whole row: a menu that is the entire toolbar still reaches every action. It also
// takes the prefix's place on a row too narrow for both, since a label naming
// controls that are not on the row is worse than no label. Only a row too narrow
// for More itself goes without (see PrefixShown).
func (tr *ToolRow) Layout(r core.Rect, prefix string) int {
	tr.More, tr.Hidden = ToolCell{}, nil
	for i := range tr.Cells {
		tr.Cells[i].Rect = core.Rect{}
	}
	if r.H != 1 {
		return r.X
	}
	x0 := r.X + 1
	if prefix != "" {
		x0 += core.DisplayWidth(prefix) + 1
	}
	moreW := core.DisplayWidth(Strings.More) + 2

	shown := make([]bool, len(tr.Cells))
	order := make([]int, len(tr.Cells))
	for i := range shown {
		shown[i], order[i] = true, len(tr.Cells)-1-i
	}
	need := func(shown []bool) int {
		x, placed, hidden := x0, false, false
		for i, c := range tr.Cells {
			if !shown[i] {
				hidden = true
				continue
			}
			x += core.DisplayWidth(c.Label) + 2 + ToolRowGap
			placed = true
		}
		if hidden && !tr.NoOverflow {
			x += moreW + ToolRowGap
			placed = true
		}
		if placed {
			x -= ToolRowGap
		}
		return x - r.X
	}
	hideToFit(shown, order, need, r.W)

	x := x0
	for i := range tr.Cells {
		if !shown[i] {
			tr.Hidden = append(tr.Hidden, i)
			continue
		}
		w := core.DisplayWidth(tr.Cells[i].Label) + 2
		tr.Cells[i].Rect = core.Rect{X: x, Y: r.Y, W: w, H: 1}
		x += w + ToolRowGap
	}
	if len(tr.Hidden) == 0 || tr.NoOverflow {
		return x
	}
	if x+moreW > r.Right() && prefix != "" {
		x = r.X + 1
	}
	if x+moreW <= r.Right() {
		tr.More = ToolCell{Label: Strings.More, Rect: core.Rect{X: x, Y: r.Y, W: moreW, H: 1}}
		x += moreW + ToolRowGap
	}
	return x
}

// PrefixShown reports whether the prefix Layout was given still has its place
// — More takes it on a row too narrow for both. r is the rect Layout was given.
func (tr *ToolRow) PrefixShown(r core.Rect) bool {
	return tr.More.Rect.IsZero() || tr.More.Rect.X > r.X+1
}

// CellAt returns the index of the cell under (mx, my), or -1. A hidden cell is
// never hit; neither is More (test More.Rect).
func (tr *ToolRow) CellAt(mx, my int) int {
	for i := range tr.Cells {
		if !tr.Cells[i].Rect.IsZero() && tr.Cells[i].Rect.Contains(mx, my) {
			return i
		}
	}
	return -1
}

// OverflowItems builds the menu behind More: one entry per hidden cell, each
// carrying the gate its cell would have — and its reason as the item's Note,
// which a MenuItem shows precisely while it is disabled, so a withheld action
// still explains itself the way the dimmed cell does. A nil disabled or reason
// reads the cell's Disabled or Reason field. A selected cell keeps a bullet,
// since the point of drawing a rate selector selected is to say which is in
// force.
func (tr *ToolRow) OverflowItems(disabled func(int) bool, reason func(int) string, run func(int)) []MenuItem {
	if disabled == nil {
		disabled = func(i int) bool { return tr.Cells[i].Disabled }
	}
	if reason == nil {
		reason = func(i int) string { return tr.Cells[i].Reason }
	}
	items := make([]MenuItem, 0, len(tr.Hidden))
	for _, i := range tr.Hidden {
		label := tr.Cells[i].Label
		if tr.Cells[i].Selected {
			label = ChoiceMark + label
		}
		// Note is fixed when the menu is built, while Enabled is asked on every draw.
		// Ask reason only for a cell withheld right now: a reason function may answer
		// for a state the cell isn't in (Query Store's returns the permission text
		// whenever the login can't force a plan, including for cells that don't need
		// it), and an enabled item never shows its note anyway. A menu is modal, so
		// nothing underneath it changes while it is open.
		note := ""
		if disabled(i) {
			note = reason(i)
		}
		items = append(items, MenuItem{
			Label:   label,
			Enabled: func() bool { return !disabled(i) },
			Note:    note,
			Action:  func() { run(i) },
		})
	}
	return items
}

// Run invokes cell i's action, or reports why it did not: the one gate a host
// gives clicks, key bindings and the More menu, so none of them can act on a
// cell another shows dimmed. disabled and reason are the pair the host passes
// OverflowItems, nil reading the cell's Disabled or Reason field as there. A
// withheld cell with a reason sends status "Label: reason" (runLabel); one without says
// nothing, its grey speaking for itself (a Refresh already running). An index
// outside the row, or a cell with no Action, does nothing.
func (tr *ToolRow) Run(i int, disabled func(int) bool, reason func(int) string, status func(string)) {
	if i < 0 || i >= len(tr.Cells) {
		return
	}
	if disabled == nil {
		disabled = func(i int) bool { return tr.Cells[i].Disabled }
	}
	if reason == nil {
		reason = func(i int) string { return tr.Cells[i].Reason }
	}
	if disabled(i) {
		if r := reason(i); r != "" && status != nil {
			status(runLabel(tr.Cells[i].Label) + ": " + r)
		}
		return
	}
	if a := tr.Cells[i].Action; a != nil {
		a()
	}
}

// Draw paints the row Layout placed in r: the bar in the menu-bar style, every
// shown cell in the tooltip scheme — dimmed when disabled, else in the selected
// look when Selected — then More, dimmed when moreDim. disabled nil reads each
// cell's Disabled field, as Run and OverflowItems do; pass the same predicate as
// to Run, so a cell that looks inert is. A host draws whatever else shares the
// row (a prefix, a status, a filter field) after Draw, into the columns Layout
// left free. A rect that is not one row high draws nothing.
//
// Disabled wins over Selected: a selected cell the host refuses is drawn dimmed,
// not as the one in force. Selected is painted at all because a row that marks a
// choice (a refresh rate, Fit) and never shows it hides state the More menu's
// bullet would reveal only once the cell has overflowed.
func (tr *ToolRow) Draw(s tcell.Screen, r core.Rect, disabled func(int) bool, moreDim bool) {
	if r.H != 1 {
		return
	}
	if disabled == nil {
		disabled = func(i int) bool { return tr.Cells[i].Disabled }
	}
	dim := theme.StyleTooltip().Foreground(theme.Active().TextDim)
	core.FillRect(s, r, ' ', theme.StyleMenuBar())
	for i, c := range tr.Cells {
		if c.Rect.IsZero() {
			continue
		}
		style := theme.StyleTooltip()
		switch {
		case disabled(i):
			style = dim
		case c.Selected:
			style = theme.StyleMenuSelected().Bold(true)
		}
		drawToolCell(s, c, style)
	}
	if !tr.More.Rect.IsZero() {
		style := theme.StyleTooltip()
		if moreDim {
			style = dim
		}
		drawToolCell(s, tr.More, style)
	}
}

// drawToolCell paints one cell: its rect filled, the label after one space of
// padding.
func drawToolCell(s tcell.Screen, c ToolCell, style tcell.Style) {
	core.FillRect(s, c.Rect, ' ', style)
	core.DrawText(s, c.Rect.X+1, c.Rect.Y, style, c.Label)
}

// runLabel is a cell's label as Run's refusal names it: without the trailing
// "..." or "▾" that promise a dialog or a menu, which read as noise before the
// colon ("Recycle...: Requires ...").
func runLabel(label string) string {
	label = strings.TrimSpace(strings.TrimSuffix(label, "▾"))
	for _, dots := range []string{"...", "…"} {
		label = strings.TrimSuffix(label, dots)
	}
	return label
}
