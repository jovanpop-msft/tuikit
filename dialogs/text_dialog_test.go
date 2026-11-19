package dialogs

import (
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/tuikittest"
)

// testTextLines is a key reference long enough to scroll on a small screen,
// with entries wide enough to wrap on a narrow one.
func testTextLines() []string {
	lines := []string{"Keys", "────"}
	for i := range 40 {
		lines = append(lines, "  Ctrl+"+strconv.Itoa(i)+"      does thing number "+strconv.Itoa(i)+" to the selection, then reports it")
	}
	return append(lines, "", "Last heading", "  End         the very last line")
}

func newTestText(w, h int) (*TextDialog, *tuikittest.Screen) {
	scr := tuikittest.NewScreen(w, h)
	d := NewTextDialog(TextDialogSpec{Title: "Help", Lines: testTextLines(), MinHeight: 12})
	d.SetScreen(scr)
	d.Show()
	return d, scr
}

// At every terminal size the text is wrapped to the width it is drawn at, so
// no row is wider than the text area (DrawTextClipped would cut it), the rows
// carry every word in order, the dialog is on-screen, and the last row in view
// sits above the button row (else the button row overdraws it and the final
// line is never seen).
func TestTextDialogLosesNoTail(t *testing.T) {
	want := strings.Fields(strings.Join(testTextLines(), " "))
	for _, sz := range [][2]int{{30, 14}, {40, 20}, {62, 28}, {80, 24}, {200, 60}} {
		d, _ := newTestText(sz[0], sz[1])
		r := d.Rect()
		if r.X < 0 || r.Right() > sz[0] || r.Y < 0 || r.Y+r.H > sz[1] {
			t.Errorf("%dx%d: dialog %+v is off-screen", sz[0], sz[1], r)
		}
		if last := d.InnerRect().Y + d.DataH(); last >= d.ButtonRowY() {
			t.Errorf("%dx%d: last row %d overlaps the button row %d", sz[0], sz[1], last, d.ButtonRowY())
		}
		textW := d.InnerRect().W - 2
		for i, l := range d.Rows() {
			if w := core.DisplayWidth(l); w > textW {
				t.Errorf("%dx%d: row %d is %d columns, text area %d: %q", sz[0], sz[1], i, w, textW, l)
			}
		}
		if got := strings.Fields(strings.Join(d.Rows(), " ")); !slices.Equal(got, want) {
			t.Errorf("%dx%d: wrapped text lost or reordered words", sz[0], sz[1])
		}
	}
}

// A terminal wider than the widest line shows every line unwrapped.
func TestTextDialogWideTerminalNeedsNoWrap(t *testing.T) {
	d, _ := newTestText(200, 60)
	if !slices.Equal(d.Rows(), testTextLines()) {
		t.Errorf("at 200 columns: %d rows, want the %d lines as written", len(d.Rows()), len(testTextLines()))
	}
}

// PgDn to the end shows the last line; a row of a wrapped heading keeps the
// heading style, an entry's continuation does not take it.
func TestTextDialogScrollsToLastLine(t *testing.T) {
	d, scr := newTestText(80, 24)
	for range 20 {
		d.HandleKey(tcell.NewEventKey(tcell.KeyPgDn, "", tcell.ModNone))
	}
	if d.Scroll() != len(d.Rows())-d.DataH() {
		t.Fatalf("scroll %d after PgDn, want %d", d.Scroll(), len(d.Rows())-d.DataH())
	}
	d.Draw(scr)
	if scr.RowOf("the very last line") < 0 {
		t.Error("the last line is not drawn at the end of the scroll")
	}
	d.HandleKey(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone))
	if d.Scroll() != len(d.Rows())-d.DataH() {
		t.Error("Down scrolled past the end")
	}
	d.HandleKey(tcell.NewEventKey(tcell.KeyEscape, "", tcell.ModNone))
	if d.Visible() {
		t.Error("Escape did not close the dialog")
	}
}

func TestTextDialogHeadingStyleFollowsSourceLine(t *testing.T) {
	d := NewTextDialog(TextDialogSpec{Lines: []string{
		"Heading  with a gap that wraps here",
		"  Ctrl+X      one two three four five",
	}})
	d.SetScreen(tuikittest.NewScreen(34, 20)) // text width 34-2-4 = 28
	d.Show()
	if want := []bool{true, true, false, false}; !slices.Equal(d.heads, want) {
		t.Errorf("heads %v for rows %q, want %v", d.heads, d.Rows(), want)
	}
}

// A press near the bottom of the scrollbar track jumps the view forward; the
// wheel scrolls; Close closes.
func TestTextDialogMouse(t *testing.T) {
	d, _ := newTestText(80, 24)
	inner := d.InnerRect()
	if len(d.Rows()) <= d.DataH() {
		t.Fatalf("rows (%d) must exceed DataH (%d) for this test to mean anything", len(d.Rows()), d.DataH())
	}
	sbX := d.Rect().Right() - 1
	d.HandleMouse(tcell.NewEventMouse(sbX, inner.Y+d.DataH(), tcell.Button1, tcell.ModNone))
	d.HandleMouse(tcell.NewEventMouse(sbX, inner.Y+d.DataH(), tcell.ButtonNone, tcell.ModNone))
	if d.Scroll() == 0 {
		t.Error("a press near the bottom of the scrollbar track did not scroll")
	}
	before := d.Scroll()
	d.HandleMouse(tcell.NewEventMouse(inner.X+2, inner.Y+2, tcell.WheelUp, tcell.ModNone))
	if d.Scroll() != before-1 {
		t.Errorf("WheelUp: scroll %d, want %d", d.Scroll(), before-1)
	}
	x := d.ButtonRowStartX([]string{Strings.Close}) + 1
	d.HandleMouse(tcell.NewEventMouse(x, d.ButtonRowY(), tcell.Button1, tcell.ModNone))
	if d.Visible() {
		t.Error("clicking Close did not close the dialog")
	}
}
