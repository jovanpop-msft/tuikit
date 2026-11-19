package dialogs

import (
	"strconv"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

// newTestList builds a shown ListDialog over *rows (read live) on an 80x30
// screen, recording each OnButton call into *pressed as "b:sel".
func newTestList(t *testing.T, rows *[]string, activate bool, pressed *[]string) (*ListDialog, *tuikittest.Screen) {
	t.Helper()
	scr := tuikittest.NewScreen(80, 30)
	d := NewListDialog(ListDialogSpec{
		Title: "Pick", W: 56, H: 16,
		Buttons: []string{"Go", "Close"},
		Empty:   "(nothing)",
		Count:   func() int { return len(*rows) },
		Item:    func(i int) string { return (*rows)[i] },
		OnButton: func(b, sel int) bool {
			*pressed = append(*pressed, strconv.Itoa(b)+":"+strconv.Itoa(sel))
			return sel >= 0
		},
		ActivateOnClick: activate,
	})
	d.SetScreen(scr)
	d.Show()
	return d, scr
}

func numbered(n int) []string {
	rows := make([]string, n)
	for i := range rows {
		rows[i] = "Row " + strconv.Itoa(i)
	}
	return rows
}

// The last row of a full list scrolls onto a row above the button row, not
// onto the button row itself, where DrawButtons overdrew it.
func TestListDialogLastRowAboveButtons(t *testing.T) {
	rows := numbered(20)
	var pressed []string
	d, scr := newTestList(t, &rows, false, &pressed)
	d.HandleKey(key(tcell.KeyEnd))
	d.Draw(scr)
	if y := scr.RowOf("Row 19"); y < 0 || y >= d.ButtonRowY() {
		t.Errorf("last row drawn on row %d, want above the button row %d", y, d.ButtonRowY())
	}
	if d.Selected() != 19 {
		t.Errorf("End selected %d, want 19", d.Selected())
	}
}

// Count is read on every Draw: a row added while open is drawn without a
// re-Show, and a shrunk list pulls the selection back in range.
func TestListDialogReadsCountLive(t *testing.T) {
	rows := numbered(2)
	var pressed []string
	d, scr := newTestList(t, &rows, false, &pressed)
	d.Draw(scr)
	if scr.RowOf("(nothing)") >= 0 {
		t.Error("placeholder drawn over a non-empty list")
	}
	rows = append(rows, "Late arrival")
	d.Draw(scr)
	if scr.RowOf("Late arrival") < 0 {
		t.Error("a row added while open was not drawn")
	}
	d.HandleKey(key(tcell.KeyEnd))
	rows = rows[:1]
	if got := d.Selected(); got != 0 {
		t.Errorf("after the list shrank to one row, Selected() = %d, want 0", got)
	}
	rows = nil
	d.Draw(scr)
	if scr.RowOf("(nothing)") < 0 {
		t.Error("empty list did not draw its placeholder")
	}
	if got := d.Selected(); got != -1 {
		t.Errorf("empty list Selected() = %d, want -1", got)
	}
	d.HandleKey(key(tcell.KeyEnter))
	if len(pressed) != 1 || pressed[0] != "0:-1" {
		t.Errorf("Enter on an empty list pressed %v, want [0:-1]", pressed)
	}
	if !d.Visible() {
		t.Error("OnButton answered false, yet the dialog closed")
	}
}

func TestListDialogPagesAndWheelScrolls(t *testing.T) {
	rows := numbered(40)
	var pressed []string
	d, _ := newTestList(t, &rows, false, &pressed)
	dataH := d.dataH()

	d.HandleKey(key(tcell.KeyPgDn))
	if d.Selected() != dataH || d.Scroll() == 0 {
		t.Errorf("PgDn: sel %d scroll %d, want sel %d and a scroll", d.Selected(), d.Scroll(), dataH)
	}
	d.HandleKey(key(tcell.KeyHome))
	if d.Selected() != 0 || d.Scroll() != 0 {
		t.Errorf("Home: sel %d scroll %d, want 0 0", d.Selected(), d.Scroll())
	}

	inner := d.InnerRect()
	wheel := func(b tcell.ButtonMask) {
		d.HandleMouse(tcell.NewEventMouse(inner.X+2, inner.Y+2, b, tcell.ModNone))
	}
	wheel(tcell.WheelDown)
	if d.Scroll() != 1 {
		t.Errorf("WheelDown: scroll %d, want 1", d.Scroll())
	}
	if d.Selected() != 0 {
		t.Errorf("WheelDown moved the selection to %d", d.Selected())
	}
	for range 100 {
		wheel(tcell.WheelDown)
	}
	if want := len(rows) - dataH; d.Scroll() != want {
		t.Errorf("wheel past the end: scroll %d, want %d", d.Scroll(), want)
	}
	wheel(tcell.WheelUp)
	if want := len(rows) - dataH - 1; d.Scroll() != want {
		t.Errorf("WheelUp: scroll %d, want %d", d.Scroll(), want)
	}
}

func TestListDialogRowClick(t *testing.T) {
	click := func(d *ListDialog, row int) {
		inner := d.InnerRect()
		d.HandleMouse(tcell.NewEventMouse(inner.X+2, inner.Y+1+row, tcell.Button1, tcell.ModNone))
		d.HandleMouse(tcell.NewEventMouse(inner.X+2, inner.Y+1+row, tcell.ButtonNone, tcell.ModNone))
	}

	rows := numbered(5)
	var pressed []string
	d, _ := newTestList(t, &rows, false, &pressed)
	click(d, 2)
	if d.Selected() != 2 || len(pressed) != 0 || !d.Visible() {
		t.Errorf("select-only click: sel %d, pressed %v, visible %v; want 2, none, open",
			d.Selected(), pressed, d.Visible())
	}

	d, _ = newTestList(t, &rows, true, &pressed)
	click(d, 3)
	if len(pressed) != 1 || pressed[0] != "0:3" || d.Visible() {
		t.Errorf("activating click: pressed %v, visible %v; want [0:3], closed", pressed, d.Visible())
	}
}

// A drag that began elsewhere in the box and reaches a row selects it but
// does not activate it.
func TestListDialogDragOntoRowDoesNotActivate(t *testing.T) {
	rows := numbered(5)
	var pressed []string
	d, _ := newTestList(t, &rows, true, &pressed)
	inner := d.InnerRect()
	d.HandleMouse(tcell.NewEventMouse(inner.X+2, d.ButtonRowY()-1, tcell.Button1, tcell.ModNone))
	d.HandleMouse(tcell.NewEventMouse(inner.X+2, inner.Y+2, tcell.Button1, tcell.ModNone))
	if len(pressed) != 0 || !d.Visible() {
		t.Errorf("drag onto a row pressed %v, visible %v; want none, open", pressed, d.Visible())
	}
	if d.Selected() != 1 {
		t.Errorf("drag onto row 1 selected %d", d.Selected())
	}
}

// Clicking the scrollbar scrolls; it never selects or activates the row
// under it.
func TestListDialogScrollbarClickScrolls(t *testing.T) {
	rows := numbered(20)
	var pressed []string
	d, _ := newTestList(t, &rows, true, &pressed)
	inner := d.InnerRect()
	d.HandleMouse(tcell.NewEventMouse(d.Rect().Right()-1, inner.Y+d.dataH(), tcell.Button1, tcell.ModNone))
	if d.Scroll() == 0 {
		t.Error("clicking near the bottom of the track did not scroll")
	}
	if d.Selected() != 0 || len(pressed) != 0 || !d.Visible() {
		t.Errorf("scrollbar click: sel %d, pressed %v, visible %v", d.Selected(), pressed, d.Visible())
	}
}

// Tab and Backtab cycle the button row with wrap; Enter presses the focused
// button; a key the dialog has no use for reports unhandled.
func TestListDialogButtonRowKeys(t *testing.T) {
	rows := numbered(3)
	var pressed []string
	d, _ := newTestList(t, &rows, false, &pressed)
	if d.HandleKey(key(tcell.KeyF5)) {
		t.Error("F5 reported handled")
	}
	d.HandleKey(key(tcell.KeyTab))
	d.HandleKey(key(tcell.KeyEnter))
	if d.Visible() || len(pressed) != 0 {
		t.Errorf("Tab, Enter: visible %v, pressed %v; want Close pressed", d.Visible(), pressed)
	}
	d.Show()
	d.HandleKey(key(tcell.KeyBacktab))
	d.HandleKey(key(tcell.KeyBacktab))
	d.HandleKey(key(tcell.KeyDown))
	d.HandleKey(key(tcell.KeyEnter))
	if len(pressed) != 1 || pressed[0] != "0:1" || d.Visible() {
		t.Errorf("Backtab twice, Down, Enter: pressed %v, visible %v; want [0:1], closed", pressed, d.Visible())
	}
	d.Show()
	d.HandleKey(key(tcell.KeyEscape))
	if d.Visible() {
		t.Error("Escape did not close")
	}
}
