package propsheet

import "github.com/radix29/tuikit/core"

// ---------------------------------------------------------------------------
// core.ClipboardTarget / core.ClipboardHost
// ---------------------------------------------------------------------------

func (p *PropertySheet) currentCopyText() (string, bool) {
	if p.zone != zoneForm {
		return "", false
	}
	f := p.PageForm(p.current)
	if f == nil {
		return "", false
	}
	return f.CopyText()
}

func (p *PropertySheet) focusedClipboardRow() (ClipboardRow, bool) {
	if p.zone != zoneForm {
		return nil, false
	}
	f := p.PageForm(p.current)
	if f == nil {
		return nil, false
	}
	cr, ok := f.FocusedRow().(ClipboardRow)
	return cr, ok
}

// HasSelection reports whether Ctrl+C has something to copy: a real selection
// in the focused field, or else any non-empty copyable value on the focused row
// (a static row, a checkbox's state, a grid's selected row/cell, a text field's
// whole value).
func (p *PropertySheet) HasSelection() bool {
	if cr, ok := p.focusedClipboardRow(); ok && cr.HasSelection() {
		return true
	}
	txt, ok := p.currentCopyText()
	return ok && txt != ""
}

// SelectedText returns what Ctrl+C would copy — see HasSelection.
func (p *PropertySheet) SelectedText() string {
	if cr, ok := p.focusedClipboardRow(); ok && cr.HasSelection() {
		return cr.SelectedText()
	}
	txt, _ := p.currentCopyText()
	return txt
}

// Cut removes and returns the focused field's real selection. Any other row
// kind, or a form locked for an Apply, has nothing to remove: Cut returns ""
// so the host leaves the clipboard alone and reports no cut.
func (p *PropertySheet) Cut() string {
	if cr, ok := p.focusedClipboardRow(); ok && cr.HasSelection() && !p.formLocked() {
		return cr.Cut()
	}
	return ""
}

// Paste inserts text into the focused field if editable and the form is not
// locked for an Apply (see HandleKey).
func (p *PropertySheet) Paste(text string) {
	if cr, ok := p.focusedClipboardRow(); ok && !p.formLocked() {
		cr.Paste(text)
	}
}

// SelectAll selects the focused field's entire contents, if editable.
func (p *PropertySheet) SelectAll() {
	if cr, ok := p.focusedClipboardRow(); ok {
		cr.SelectAll()
	}
}

// CanCopy, CanCut and CanPaste implement core.ClipboardTarget over the focused
// row, matching what HasSelection, Cut and Paste above would do: Copy takes the
// field's selection or else the row's whole value; Cut and Paste need an
// editable field and a form not locked for an Apply.
func (p *PropertySheet) CanCopy() bool {
	if cr, ok := p.focusedClipboardRow(); ok && cr.HasSelection() {
		return cr.CanCopy()
	}
	txt, ok := p.currentCopyText()
	return ok && txt != ""
}

func (p *PropertySheet) CanCut() bool {
	cr, ok := p.focusedClipboardRow()
	return ok && !p.formLocked() && cr.HasSelection() && cr.CanCut()
}

func (p *PropertySheet) CanPaste() bool {
	cr, ok := p.focusedClipboardRow()
	return ok && !p.formLocked() && cr.CanPaste()
}

// CanSelectAll is false on a row that is copied whole (a static or flat row, a
// checkbox): it has no selection for Select All to make.
func (p *PropertySheet) CanSelectAll() bool {
	cr, ok := p.focusedClipboardRow()
	return ok && cr.CanSelectAll()
}

// FocusClipboardTargetAt implements core.ClipboardFocuser: a right-click on a
// copyable row of the current page focuses it (Form.FocusCopyableAt), so the
// menu that follows acts on that row. Refused while applying, as every press
// on the form is.
func (p *PropertySheet) FocusClipboardTargetAt(x, y int) bool {
	f := p.PageForm(p.current)
	if f == nil || p.formLocked() || !f.FocusCopyableAt(x, y) {
		return false
	}
	p.setZone(zoneForm)
	return true
}

// FocusedClipboardTarget implements core.ClipboardHost. The sheet is its own
// target: every method above resolves through the focused row and answers
// harmlessly when focus is on the page list or button row. This makes every
// dialog embedding a PropertySheet (Properties and every New-<object> dialog) a
// clipboard host without each saying so.
func (p *PropertySheet) FocusedClipboardTarget() core.ClipboardTarget { return p }

// ClipboardTargetToken implements core.ClipboardTargetTokener: the focused row,
// as the identity behind the sheet-as-target above.
//
// Returning the sheet from FocusedClipboardTarget costs the application's "is
// this still the target?" paste guard its resolution: the sheet is the answer
// whichever row has focus. The row itself can tell. nil is a real answer, for
// focus on the page list or button row.
func (p *PropertySheet) ClipboardTargetToken() any {
	cr, ok := p.focusedClipboardRow()
	if !ok {
		return nil
	}
	return cr
}
