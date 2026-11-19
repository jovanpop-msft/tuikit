package dialogs

import (
	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// ---------------------------------------------------------------------------
// TextDialog — scrollable reference text (a help screen), hanging-indent wrap
// ---------------------------------------------------------------------------

// TextDialogSpec configures a TextDialog.
type TextDialogSpec struct {
	Title string

	// Lines is the text, one entry per line, drawn as written: spacing is kept
	// (two-column "key  description" layouts line up), and a line wider than
	// the dialog wraps with core.WrapHanging.
	Lines []string

	// MinHeight is the dialog's height floor: below it the text is a sliver
	// even when the terminal has room. It is clamped to the screen regardless.
	MinHeight int

	// IsHeading reports whether a line is a heading, drawn bold in the accent
	// colour; every row a heading wraps to is styled with it. Nil means a
	// non-blank line that starts with no space.
	IsHeading func(line string) bool
}

// TextDialog is a read-only modal showing reference text with a Close button,
// sized to its widest line or the screen, whichever is narrower, and wrapped
// to the width that leaves so no tail is ever cut. Up/Down, PgUp/PgDn, the
// wheel and a scrollbar drag scroll it; Escape and Enter close it.
type TextDialog struct {
	ModalDialog
	spec   TextDialogSpec
	rows   []string // spec.Lines wrapped to the current text width (applySize)
	heads  []bool   // per row: true when it came from a heading line
	scroll int
}

// NewTextDialog creates a TextDialog from spec.
func NewTextDialog(spec TextDialogSpec) *TextDialog {
	if spec.IsHeading == nil {
		spec.IsHeading = func(l string) bool { return l != "" && l[0] != ' ' }
	}
	d := &TextDialog{spec: spec}
	d.InitModal(spec.Title, 0, spec.MinHeight)
	d.applySize()
	return d
}

// Show re-fits the dialog to the terminal, then shows it.
func (d *TextDialog) Show() {
	d.applySize()
	d.ModalDialog.Show()
}

// Relayout re-fits the dialog to a resized terminal: ModalDialog.Relayout
// only recentres at the size last requested.
func (d *TextDialog) Relayout() { d.applySize() }

// Rows is the text as currently wrapped, one entry per drawn row.
func (d *TextDialog) Rows() []string { return d.rows }

// Scroll is the index of the first row in view.
func (d *TextDialog) Scroll() int { return d.scroll }

// DataH is the number of text rows in view.
func (d *TextDialog) DataH() int { return d.dataH() }

// applySize sizes the dialog to its widest line, or to the screen when that is
// narrower, and wraps the lines to the text width that leaves. In goSSMS a
// fixed 62-column help box with no horizontal scroll cut every line over 57
// columns; wrapping at the drawn width means no tail is ever lost, whatever the
// terminal. The height grows with the screen down to MinHeight.
func (d *TextDialog) applySize() {
	textW := 0
	for _, l := range d.spec.Lines {
		textW = max(textW, core.DisplayWidth(l))
	}
	w, sh := textW+4, 0 // border + one column of padding on each side
	if s := d.Screen(); s != nil {
		var sw int
		sw, sh = s.Size()
		w = min(w, max(sw-2, 0))
	}
	var src []int
	d.rows, src = core.WrapHanging(d.spec.Lines, w-4)
	// Heads go by source line, not by the row's own first column: a heading's
	// continuation that hung under a two-space gap starts with a space and
	// would lose its style.
	d.heads = make([]bool, len(d.rows))
	for i, j := range src {
		d.heads[i] = d.spec.IsHeading(d.spec.Lines[j])
	}
	h := len(d.rows) + 5 // borders, top padding row, Close button row and the row under it
	if sh > 0 {
		h = min(h, max(sh-2, d.spec.MinHeight))
	}
	d.SetSize(w, h)
	d.scroll = max(0, min(d.scroll, len(d.rows)-d.dataH()))
}

// dataH is the number of rows drawn: those between the top padding row and the
// Close button row (ButtonRowY). In goSSMS it was InnerRect().H-2, one too
// many, so the button row overdrew the last row and the final line could never
// be scrolled into view.
func (d *TextDialog) dataH() int { return d.ButtonRowY() - d.InnerRect().Y - 1 }

// maxScroll is the scroll that puts the last row on the last line in view.
func (d *TextDialog) maxScroll() int { return max(0, len(d.rows)-d.dataH()) }

// Draw renders the dialog.
func (d *TextDialog) Draw(s tcell.Screen) {
	if !d.visible {
		return
	}
	d.DrawBase(s)
	p := theme.Active()
	contentStyle := theme.On(p.DialogBg, p.DialogText)
	headStyle := theme.On(p.DialogBg, p.DialogAccent).Bold(true)

	inner := d.InnerRect()
	dataH := d.dataH()
	for row := range dataH {
		idx := d.scroll + row
		if idx >= len(d.rows) {
			break
		}
		st := contentStyle
		if d.heads[idx] {
			st = headStyle
		}
		core.FillRect(s, core.Rect{X: inner.X, Y: inner.Y + 1 + row, W: inner.W, H: 1}, ' ', contentStyle)
		core.DrawTextClipped(s, inner.X+1, inner.Y+1+row, inner.W-2, st, d.rows[idx])
	}
	if len(d.rows) > dataH {
		d.DrawContentScrollbar(s, inner.Y+1, dataH, len(d.rows), d.scroll)
	}
	d.DrawButtons(s, []string{Strings.Close}, 0)
}

// HandleKey processes keyboard events. Every key is consumed while visible.
func (d *TextDialog) HandleKey(ev *tcell.EventKey) bool {
	if !d.visible {
		return false
	}
	dataH := d.dataH()
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyEnter:
		d.Hide()
	case tcell.KeyUp:
		d.scroll = max(0, d.scroll-1)
	case tcell.KeyDown:
		d.scroll = min(d.maxScroll(), d.scroll+1)
	case tcell.KeyPgUp:
		d.scroll = max(0, d.scroll-dataH)
	case tcell.KeyPgDn:
		d.scroll = min(d.maxScroll(), d.scroll+dataH)
	}
	return true
}

// HandleMouse handles mouse events: Close, the scrollbar and the wheel.
func (d *TextDialog) HandleMouse(ev *tcell.EventMouse) bool {
	if !d.visible {
		return false
	}
	if d.ConsumeOutsideClick(ev) {
		return true
	}
	if d.ButtonClicked(ev, []string{Strings.Close}) == 0 {
		d.Hide()
		return true
	}
	if d.ScrollbarDrag(ev, d.Rect().Right()-1, d.InnerRect().Y+1, d.dataH(), len(d.rows), &d.scroll) {
		return true
	}
	switch ev.Buttons() {
	case tcell.WheelUp:
		d.scroll = max(0, d.scroll-1)
	case tcell.WheelDown:
		d.scroll = min(d.maxScroll(), d.scroll+1)
	}
	return true
}
