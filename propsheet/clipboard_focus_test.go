package propsheet

import (
	"github.com/gdamore/tcell/v3"

	"testing"

	"github.com/radix29/tuikit/controls"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/tuikittest"
)

// TestRightClickFocusesTheCopyableRowUnderIt: FocusClipboardTargetAt is a
// right-click's click-to-focus. It must land on the row drawn there, leave a
// grid's own right-click menu alone, and keep the field's selection, which the
// menu it precedes acts on.
func TestRightClickFocusesTheCopyableRowUnderIt(t *testing.T) {
	name := Text("Name", "srv01", 20)
	desc := Text("Description", "a server", 20)
	g := controls.NewDataGrid()
	g.SetData([]string{"A"}, [][]string{{"1"}, {"2"}})
	grid := NewGridRow(g, 4)
	scr := tuikittest.NewScreen(100, 40)
	p := NewPropertySheet(scr, "Test Properties")
	p.SetSize(90, 28)
	p.SetPages([]string{"General"})
	p.Show()
	p.SetPageForm(0, p.pages[0].seq, NewForm(name, desc, grid))
	p.Draw(scr)

	desc.field.SelectAll()
	_, y, ok := scr.Find("Description")
	if !ok {
		t.Fatal("Description row not drawn")
	}
	if !p.FocusClipboardTargetAt(desc.field.Bounds().X, y) {
		t.Fatal("right-click on the Description row focused nothing")
	}
	if p.ClipboardTargetToken() != any(desc) {
		t.Fatalf("focused row = %v, want the Description row", p.ClipboardTargetToken())
	}
	if !desc.HasSelection() {
		t.Error("focusing the row dropped its selection")
	}
	if got := p.SelectedText(); got != "a server" {
		t.Errorf("SelectedText = %q, want the selection", got)
	}

	gr := grid.Grid.Bounds()
	if p.FocusClipboardTargetAt(gr.X+1, gr.Y+1) {
		t.Error("a right-click on a grid row was taken: the grid's own menu must get it")
	}
}

// TestSheetClipboardCanMethodsFollowTheFocusedRow: what the right-click menu
// offers comes from here, so a read-only page must answer no Cut or Paste and a
// Password row no Copy.
func TestSheetClipboardCanMethodsFollowTheFocusedRow(t *testing.T) {
	name := Text("Name", "srv01", 20)
	pw := Password("Password", 20)
	pw.field.SetValue("s3cret")
	f := NewForm(name, pw)
	p := newTestSheet("General")
	p.Show()
	p.SetPageForm(0, p.pages[0].seq, f)
	p.setZone(zoneForm)
	f.FocusFirst()

	var _ core.ClipboardTarget = p
	if !p.CanPaste() || !p.CanCopy() || p.CanCut() {
		t.Errorf("editable row: CanPaste=%v CanCopy=%v CanCut=%v, want true true false (no selection)", p.CanPaste(), p.CanCopy(), p.CanCut())
	}
	name.field.SelectAll()
	if !p.CanCut() {
		t.Error("CanCut false with a selection in an editable row")
	}

	f.FocusNext()
	if p.CanCopy() {
		t.Error("Password row: CanCopy true with nothing selected")
	}
	if got := p.SelectedText(); got != "" {
		t.Errorf("Password row: Ctrl+C would copy %q, want nothing", got)
	}
	pw.field.SelectAll()
	if p.CanCopy() || p.CanCut() {
		t.Error("Password row: CanCopy/CanCut true with a selection")
	}

	f.SetReadOnly(true)
	f.FocusFirst()
	if p.CanPaste() || p.CanCut() {
		t.Error("read-only form: CanPaste or CanCut true")
	}
	if !p.CanCopy() {
		t.Error("read-only form: CanCopy false on a row with a value")
	}
	if p.CanSelectAll() {
		t.Error("read-only form: CanSelectAll true on a flat row, which has nothing to select")
	}
}

// TestRightClickThroughAnOpenSelectListFocusesNothing: a SelectRow's open list
// is drawn over the rows below it, so a right-click on the list must not be
// taken for the text row underneath — the menu would open on a field the user
// cannot see, over a list that stays open.
func TestRightClickThroughAnOpenSelectListFocusesNothing(t *testing.T) {
	auth := Select("Auth", []string{"One", "Two", "Three", "Four"}, 0)
	name := Text("Name", "srv01", 20)
	scr := tuikittest.NewScreen(100, 40)
	p := NewPropertySheet(scr, "Test Properties")
	p.SetSize(90, 28)
	p.SetPages([]string{"General"})
	p.Show()
	f := NewForm(auth, name)
	p.SetPageForm(0, p.pages[0].seq, f)
	p.setZone(zoneForm)
	f.FocusFirst()
	p.Draw(scr)

	_, y, ok := scr.Find("Name")
	if !ok {
		t.Fatal("Name row not drawn")
	}
	x := name.field.Bounds().X
	f.HandleKey(tcell.NewEventKey(tcell.KeyF4, "", tcell.ModNone))
	if !f.OverlayActive() {
		t.Fatal("F4 did not open the Select row's list")
	}
	p.Draw(scr)
	if p.FocusClipboardTargetAt(x, y) {
		t.Error("a right-click on the open list focused the text row under it")
	}
	if f.FocusedRow() != Row(auth) || !f.OverlayActive() {
		t.Error("the right-click moved focus or closed the list")
	}
}
