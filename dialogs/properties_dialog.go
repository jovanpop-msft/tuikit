package dialogs

import (
	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// ---------------------------------------------------------------------------
// PropertiesDialog — generic key/value viewer
// ---------------------------------------------------------------------------

// Default dialog size, restored by every ShowProperties call so a caller that
// widened it with ShowPropertiesSized doesn't leave the next, unrelated showing
// oversized (the instance is reused across features).
const (
	propsDefaultW = 60
	propsDefaultH = 24

	// Minimum width of the Property column; ShowPropertiesSized widens it to
	// the longest key it was given, up to a third of the dialog.
	propsKeyW = 22
)

// PropertyRow is a single key/value pair, or — with Section set — a group
// heading whose Key spans the full width and whose Value is ignored.
type PropertyRow struct {
	Key     string
	Value   string
	Section bool
}

// PropertySection returns a group-heading row for the given caption.
func PropertySection(caption string) PropertyRow {
	return PropertyRow{Key: caption, Section: true}
}

// PropertiesDialog renders a scrollable table of PropertyRows.
// It embeds ModalDialog to inherit all focus-trap behaviour.
//
// One value row at a time carries a cursor (Up/Down, Home/End, a click), and
// the dialog is a read-only core.ClipboardHost over it: the host's Copy takes
// that row's Value, and a right-click on a row (core.ClipboardFocuser) moves
// the cursor there before the host's Copy menu opens. Section rows are never
// the cursor's, so there is always a value under it when any row has one.
type PropertiesDialog struct {
	ModalDialog
	rows   []PropertyRow
	scroll int
	keyW   int
	// cursor indexes rows; -1 when no row holds a value (empty, or sections
	// only).
	cursor int

	// OnClose, when set, runs after the user closes the dialog (Escape, Enter,
	// Close), for a host still filling it in the background.
	OnClose func()
}

// NewPropertiesDialog creates a PropertiesDialog.
func NewPropertiesDialog(s tcell.Screen) *PropertiesDialog {
	d := new(PropertiesDialog{keyW: propsKeyW})
	d.InitModal(s, Strings.PropertiesTitle, propsDefaultW, propsDefaultH)
	return d
}

// ShowProperties populates the dialog with the given title and rows at the
// default size, then shows it.
func (d *PropertiesDialog) ShowProperties(title string, rows []PropertyRow) {
	d.ShowPropertiesSized(title, rows, propsDefaultW, propsDefaultH)
}

// ShowPropertiesSized is ShowProperties with an explicit size, for content that
// doesn't fit the default 60x24 (the About box). recentre clamps w/h to the
// terminal, so asking for more than the screen has is safe.
func (d *PropertiesDialog) ShowPropertiesSized(title string, rows []PropertyRow, w, h int) {
	d.SetTitle(title)
	d.rows = rows
	d.scroll = 0
	d.cursor = d.nextValueRow(0, 1)
	d.keyW = propsKeyW
	for _, r := range rows {
		if r.Section {
			continue
		}
		if kw := core.DisplayWidth(r.Key); kw > d.keyW {
			d.keyW = kw
		}
	}
	d.keyW = min(d.keyW, max(propsKeyW, w/3))
	d.SetSize(w, h)
	d.ModalDialog.Show()
}

// ShowPropertiesFitted is ShowProperties widened (never narrowed) until the
// longest Value shows whole, for rows whose values are names a clip would make
// ambiguous: a cut "dbo.p (SQL_STORED_PROCEDURE, caller-dependent)" reads as an
// ordinary reference. recentre clamps the width to the terminal.
func (d *PropertiesDialog) ShowPropertiesFitted(title string, rows []PropertyRow) {
	keyW, valW := propsKeyW, 0
	for _, r := range rows {
		if !r.Section {
			keyW = max(keyW, core.DisplayWidth(r.Key))
			valW = max(valW, core.DisplayWidth(r.Value))
		}
	}
	// Border (2), the key column's leading space and " | " (4), one space after
	// the value: Draw puts the value at inner.X+keyW+4.
	d.ShowPropertiesSized(title, rows, max(propsDefaultW, keyW+valW+7), propsDefaultH)
}

// propsDataH is how many rows fit between the header and the button row. Five
// inner lines go to chrome (blank, column header, separator, button row, blank
// below), so a naive inner.H-3 draws the last two rows under the separator and
// buttons, where DrawSeparator/DrawButtons overwrite them and the scroll clamp
// never brings them into view.
func propsDataH(inner core.Rect) int { return max(0, inner.H-5) }

// Draw renders the properties dialog.
func (d *PropertiesDialog) Draw(s tcell.Screen) {
	if !d.visible {
		return
	}
	d.DrawBase(s)

	inner := d.InnerRect()
	keyW := d.keyW
	dataH := propsDataH(inner)

	// Header
	hdrStyle := theme.StyleGridHeader()
	core.FillRect(s, core.Rect{X: inner.X, Y: inner.Y + 1, W: inner.W, H: 1}, ' ', hdrStyle)
	core.DrawTextClipped(s, inner.X+1, inner.Y+1, keyW, hdrStyle, Strings.PropertyCol)
	core.PutRune(s, inner.X+keyW+2, inner.Y+1, '|', hdrStyle)
	core.DrawTextClipped(s, inner.X+keyW+4, inner.Y+1, inner.W-keyW-4, hdrStyle, Strings.ValueCol)

	keyStyle := theme.StyleDialogAccent().Bold(true)
	valStyle := theme.StyleDialog()
	secStyle := theme.StyleDialogAccent().Bold(true)
	ruleStyle := theme.StyleDialogLine()

	for row := range dataH {
		idx := d.scroll + row
		if idx >= len(d.rows) {
			break
		}
		pr := d.rows[idx]
		y := inner.Y + 2 + row
		if pr.Section {
			// A rule rather than a background band: GridHeader and DialogBg are the same
			// colour in several themes.
			core.FillRect(s, core.Rect{X: inner.X, Y: y, W: inner.W, H: 1}, '─', ruleStyle)
			core.DrawTextClipped(s, inner.X+1, y, inner.W-2, secStyle, " "+pr.Key+" ")
			continue
		}
		ks, vs := keyStyle, valStyle
		if idx == d.cursor {
			// The whole row, key included, so the cursor reads at a glance; the
			// value is what Copy takes.
			vs = theme.StyleSelected()
			ks = vs.Bold(true)
		}
		core.FillRect(s, core.Rect{X: inner.X, Y: y, W: inner.W, H: 1}, ' ', vs)
		core.DrawTextClipped(s, inner.X+1, y, keyW, ks, pr.Key)
		core.PutRune(s, inner.X+keyW+2, y, '|', vs)
		core.DrawTextClipped(s, inner.X+keyW+4, y, inner.W-keyW-4, vs, pr.Value)
	}
	if len(d.rows) > dataH {
		d.DrawContentScrollbar(s, inner.Y+2, dataH, len(d.rows), d.scroll)
	}

	d.DrawSeparator(s)
	d.DrawButtons(s, []string{Strings.Close}, 0)
}

// close hides the dialog and tells the host.
func (d *PropertiesDialog) close() {
	d.Hide()
	if d.OnClose != nil {
		d.OnClose()
	}
}

// HandleKey handles keyboard events.
func (d *PropertiesDialog) HandleKey(ev *tcell.EventKey) bool {
	if !d.visible {
		return false
	}
	inner := d.InnerRect()
	dataH := propsDataH(inner)
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyEnter:
		d.close()
	case tcell.KeyUp:
		d.moveCursor(d.nextValueRow(d.cursor-1, -1), -1, dataH)
	case tcell.KeyDown:
		d.moveCursor(d.nextValueRow(d.cursor+1, 1), 1, dataH)
	case tcell.KeyHome:
		d.moveCursor(d.nextValueRow(0, 1), -1, dataH)
	case tcell.KeyEnd:
		d.moveCursor(d.nextValueRow(len(d.rows)-1, -1), 1, dataH)
	}
	return true
}

// nextValueRow returns the first non-section row from i stepping by dir, or -1.
func (d *PropertiesDialog) nextValueRow(i, dir int) int {
	for ; i >= 0 && i < len(d.rows); i += dir {
		if !d.rows[i].Section {
			return i
		}
	}
	return -1
}

// moveCursor puts the cursor on row to (when there is one) and scrolls it into
// view. With no value row further in direction dir it still scrolls one row
// that way, so a section heading or trailing rows past the last value (above
// the first) can be brought on screen.
func (d *PropertiesDialog) moveCursor(to, dir, dataH int) {
	maxScroll := max(0, len(d.rows)-dataH)
	if to < 0 {
		d.scroll = min(maxScroll, max(0, d.scroll+dir))
		return
	}
	d.cursor = to
	// A section heading directly above the first value row scrolls in with it.
	top := to
	if top > 0 && d.nextValueRow(0, 1) == to {
		top = 0
	}
	if top < d.scroll {
		d.scroll = top
	}
	if to >= d.scroll+dataH {
		d.scroll = to - dataH + 1
	}
	d.scroll = min(maxScroll, max(0, d.scroll))
}

// rowAt returns the index of the value row drawn at screen cell (x, y), or -1.
func (d *PropertiesDialog) rowAt(x, y int) int {
	inner := d.InnerRect()
	row := y - (inner.Y + 2)
	if x < inner.X || x >= inner.X+inner.W || row < 0 || row >= propsDataH(inner) {
		return -1
	}
	idx := d.scroll + row
	if idx >= len(d.rows) || d.rows[idx].Section {
		return -1
	}
	return idx
}

// HandleMouse handles mouse events.
func (d *PropertiesDialog) HandleMouse(ev *tcell.EventMouse) bool {
	if !d.visible {
		return false
	}
	if d.ConsumeOutsideClick(ev) {
		return true
	}
	if d.ButtonClicked(ev, []string{Strings.Close}) == 0 {
		d.close()
		return true
	}
	inner := d.InnerRect()
	dataH := propsDataH(inner)
	if d.ScrollbarDrag(ev, d.Rect().Right()-1, inner.Y+2, dataH, len(d.rows), &d.scroll) {
		return true
	}
	switch ev.Buttons() {
	case tcell.Button1:
		if idx := d.rowAt(ev.Position()); idx >= 0 {
			d.cursor = idx
		}
	case tcell.WheelUp:
		d.scroll = max(0, d.scroll-core.WheelRows)
	case tcell.WheelDown:
		d.scroll = max(d.scroll, min(len(d.rows)-dataH, d.scroll+core.WheelRows))
	}
	return true
}

// FocusedClipboardTarget implements core.ClipboardHost: the dialog itself,
// copying the value under the cursor.
func (d *PropertiesDialog) FocusedClipboardTarget() core.ClipboardTarget { return d }

// ClipboardTargetToken implements core.ClipboardTargetTokener: the host hands
// back itself, so the cursor row is the identity of what it copies.
func (d *PropertiesDialog) ClipboardTargetToken() any { return d.cursor }

// FocusClipboardTargetAt implements core.ClipboardFocuser: a right-click on a
// value row moves the cursor there, so the host's Copy takes that value.
func (d *PropertiesDialog) FocusClipboardTargetAt(x, y int) bool {
	idx := d.rowAt(x, y)
	if !d.visible || idx < 0 {
		return false
	}
	d.cursor = idx
	return true
}

// HasSelection and the methods below make the dialog a read-only
// core.ClipboardTarget: the selection is the cursor row's whole Value.
func (d *PropertiesDialog) HasSelection() bool {
	return d.visible && d.cursor >= 0 && d.cursor < len(d.rows)
}

// SelectedText returns the cursor row's Value — the value alone, not
// "Key: Value", which is what someone copying a version or URL wants.
func (d *PropertiesDialog) SelectedText() string {
	if !d.HasSelection() {
		return ""
	}
	return d.rows[d.cursor].Value
}

// Cut returns "" and removes nothing: the dialog is read-only.
func (d *PropertiesDialog) Cut() string { return "" }

// Paste is a no-op: the dialog is read-only.
func (d *PropertiesDialog) Paste(string) {}

// SelectAll is a no-op: a row is copied whole and there is nothing to extend.
func (d *PropertiesDialog) SelectAll() {}

func (d *PropertiesDialog) CanCopy() bool      { return d.SelectedText() != "" }
func (d *PropertiesDialog) CanCut() bool       { return false }
func (d *PropertiesDialog) CanPaste() bool     { return false }
func (d *PropertiesDialog) CanSelectAll() bool { return false }
