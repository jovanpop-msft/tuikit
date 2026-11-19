package dialogs

import (
	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// ---------------------------------------------------------------------------
// ListDialog — pick a row from a scrolling list, then press a button
// ---------------------------------------------------------------------------

// ListDialogSpec configures a ListDialog.
type ListDialogSpec struct {
	Title string
	W, H  int

	// Buttons is the button row, in order. The first is the default (focused on
	// Show, and what a row click presses with ActivateOnClick); the last closes
	// the dialog without calling OnButton. At least two.
	Buttons []string

	// Empty is drawn on the first row when Count() is 0.
	Empty string

	// Count and Item are the rows. Both are read on every Draw and every event,
	// never snapshotted, so a source that changes while the dialog is open (a
	// task list whose progress updates) shows the change on the next repaint. A
	// dialog listing a snapshot takes it in its own Show before calling
	// ListDialog.Show.
	Count func() int
	Item  func(i int) string

	// OnButton runs button b (any but the last) on the selected row, sel; sel is
	// -1 when the list is empty. It reports whether the dialog should close.
	OnButton func(b, sel int) (close bool)

	// ActivateOnClick makes a click on a row press the first button on it (a
	// jump-to list); otherwise a click only selects (a list whose action is
	// destructive, so it must be asked for).
	ActivateOnClick bool
}

// ListDialog is a modal holding a scrolling list of one-line rows with a
// selection, an empty-list placeholder and a button row. Up/Down/PgUp/PgDn/
// Home/End move the selection and the wheel scrolls; the button row takes
// Left/Right, Tab/Backtab (wrapping, since the list never takes focus away
// from it), Enter and Escape.
type ListDialog struct {
	ModalDialog
	spec     ListDialogSpec
	sel      int
	scroll   int
	btnFocus int
}

// NewListDialog creates a ListDialog from spec.
func NewListDialog(spec ListDialogSpec) *ListDialog {
	d := new(ListDialog{spec: spec})
	d.InitModal(spec.Title, spec.W, spec.H)
	return d
}

// Show puts the selection, the scroll and the button focus back at the top and
// displays the dialog.
func (d *ListDialog) Show() {
	d.sel, d.scroll, d.btnFocus = 0, 0, 0
	d.ModalDialog.Show()
}

// Selected is the selected row, or -1 when the list is empty.
func (d *ListDialog) Selected() int {
	d.clamp()
	if d.count() == 0 {
		return -1
	}
	return d.sel
}

// Scroll is the index of the first row drawn.
func (d *ListDialog) Scroll() int {
	d.clamp()
	return d.scroll
}

func (d *ListDialog) count() int { return d.spec.Count() }

// dataH is the number of rows drawn: those between the top padding row and the
// button row (ButtonRowY). InnerRect().H-2 was one too many in both dialogs
// this replaced, so the button row overdrew the last row and the final entry of
// a full list could never be scrolled into view.
func (d *ListDialog) dataH() int { return max(0, d.ButtonRowY()-d.InnerRect().Y-1) }

// maxScroll is the scroll that puts the last row on the last visible line.
func (d *ListDialog) maxScroll() int { return max(0, d.count()-d.dataH()) }

// clamp pulls sel and scroll back into range: Count is live, so the list can
// have shrunk since they were set.
func (d *ListDialog) clamp() {
	n := d.count()
	d.sel = max(0, min(d.sel, n-1))
	d.scroll = max(0, min(d.scroll, d.maxScroll()))
}

// Draw renders the list and the button row.
func (d *ListDialog) Draw(s tcell.Screen) {
	if !d.visible {
		return
	}
	d.clamp()
	d.DrawBase(s)
	p := theme.Active()
	inner := d.InnerRect()
	dataH := d.dataH()
	n := d.count()

	if n == 0 {
		msgStyle := theme.On(p.DialogBg, p.DialogTextDim)
		core.DrawText(s, inner.X+1, inner.Y+1, msgStyle, d.spec.Empty)
	}
	for row := range dataH {
		idx := d.scroll + row
		if idx >= n {
			break
		}
		y := inner.Y + 1 + row
		st := theme.On(p.DialogBg, p.DialogText)
		if idx == d.sel {
			st = theme.StyleSelected()
		}
		core.FillRect(s, core.Rect{X: inner.X, Y: y, W: inner.W, H: 1}, ' ', st)
		core.DrawTextClipped(s, inner.X+1, y, inner.W-2, st, d.spec.Item(idx))
	}
	if n > dataH {
		d.DrawContentScrollbar(s, inner.Y+1, dataH, n, d.scroll)
	}
	d.DrawButtons(s, d.spec.Buttons, d.btnFocus)
}

// HandleKey processes keyboard events. A key the dialog has no use for reports
// unhandled.
func (d *ListDialog) HandleKey(ev *tcell.EventKey) bool {
	if !d.visible {
		return false
	}
	d.clamp()
	dataH := d.dataH()
	nb := len(d.spec.Buttons)
	switch ev.Key() {
	case tcell.KeyUp:
		d.moveTo(d.sel - 1)
	case tcell.KeyDown:
		d.moveTo(d.sel + 1)
	case tcell.KeyPgUp:
		d.moveTo(d.sel - max(1, dataH))
	case tcell.KeyPgDn:
		d.moveTo(d.sel + max(1, dataH))
	case tcell.KeyHome:
		d.moveTo(0)
	case tcell.KeyEnd:
		d.moveTo(d.count() - 1)
	case tcell.KeyLeft, tcell.KeyRight, tcell.KeyEnter, tcell.KeyEscape, tcell.KeyTab, tcell.KeyBacktab:
		ButtonRowKey(ev, &d.btnFocus, nb, nil, d.press, d.Hide, func(dir int) {
			if dir > 0 {
				d.btnFocus = NextFocus(d.btnFocus, nb)
			} else {
				d.btnFocus = PrevFocus(d.btnFocus, nb)
			}
		})
	default:
		return false
	}
	return true
}

// moveTo selects row i, clamped to the list, and scrolls it into view.
func (d *ListDialog) moveTo(i int) {
	d.sel = max(0, min(i, d.count()-1))
	d.scroll = ScrollToShow(d.sel, d.scroll, d.dataH())
}

// press runs the focused button.
func (d *ListDialog) press() { d.run(d.btnFocus) }

// run runs button b: the last closes, any other goes to OnButton.
func (d *ListDialog) run(b int) {
	if b == len(d.spec.Buttons)-1 {
		d.Hide()
		return
	}
	if d.spec.OnButton != nil && d.spec.OnButton(b, d.Selected()) {
		d.Hide()
	}
}

// HandleMouse processes mouse events.
func (d *ListDialog) HandleMouse(ev *tcell.EventMouse) bool {
	if !d.visible {
		return false
	}
	if d.ConsumeOutsideClick(ev) {
		return true
	}
	if i := d.ButtonClicked(ev, d.spec.Buttons); i >= 0 {
		d.btnFocus = i
		d.press()
		return true
	}
	d.clamp()
	inner := d.InnerRect()
	dataH := d.dataH()
	// Before the row hit-test: the bar sits on the border column, but a drag
	// that started on it continues wherever the pointer goes.
	if d.ScrollbarDrag(ev, d.Rect().Right()-1, inner.Y+1, dataH, d.count(), &d.scroll) {
		return true
	}
	switch ev.Buttons() {
	case tcell.Button1:
		mx, my := ev.Position()
		row := my - (inner.Y + 1)
		if mx < inner.X || mx >= inner.X+inner.W || row < 0 || row >= dataH {
			return true
		}
		idx := d.scroll + row
		if idx >= d.count() {
			return true
		}
		d.sel = idx
		// Only the press that began the gesture activates: a drag from
		// elsewhere that reaches a row only selects it.
		if d.spec.ActivateOnClick && d.fresh {
			d.fresh = false
			d.run(0)
		}
	case tcell.WheelUp:
		d.scroll = max(0, d.scroll-core.WheelRows)
	case tcell.WheelDown:
		d.scroll = min(d.maxScroll(), d.scroll+core.WheelRows)
	}
	return true
}
