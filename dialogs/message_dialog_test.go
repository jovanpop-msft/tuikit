package dialogs

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/theme"
	"github.com/radix29/tuikit/tuikittest"
)

func newTestMessageDialog(pressed *[]int, escaped *bool) (*MessageDialog, *tuikittest.Screen) {
	scr := tuikittest.NewScreen(80, 24)
	spec := MessageDialogSpec{
		Title:    "Message",
		Lines:    []MessageLine{{Text: "first"}, {Text: "second"}},
		Buttons:  []string{"Copy", "Close"},
		Focus:    1,
		OnButton: func(i int) { *pressed = append(*pressed, i) },
		W:        50, H: 10,
	}
	if escaped != nil {
		spec.OnEscape = func() { *escaped = true }
	}
	d := withScreen(NewMessageDialog(spec), scr)
	d.Show()
	return d, scr
}

func TestMessageDialogKeysStepWrapAndFire(t *testing.T) {
	var pressed []int
	d, _ := newTestMessageDialog(&pressed, nil)
	if d.ButtonFocus() != 1 {
		t.Fatalf("focus on open = %d, want the spec's 1", d.ButtonFocus())
	}
	d.HandleKey(tuikittest.Key(tcell.KeyTab)) // wraps to 0
	d.HandleKey(tuikittest.Key(tcell.KeyEnter))
	d.HandleKey(tuikittest.Key(tcell.KeyLeft)) // wraps to 1
	d.HandleKey(tuikittest.Key(tcell.KeyEnter))
	if len(pressed) != 2 || pressed[0] != 0 || pressed[1] != 1 {
		t.Errorf("pressed %v, want [0 1]", pressed)
	}
	if !d.HandleKey(tuikittest.Rune('x')) {
		t.Error("a letter was not swallowed")
	}
}

func TestMessageDialogEscape(t *testing.T) {
	var pressed []int
	d, _ := newTestMessageDialog(&pressed, nil)
	d.HandleKey(tuikittest.Key(tcell.KeyEscape))
	if d.Visible() {
		t.Error("Escape with no OnEscape left the dialog open")
	}

	escaped := false
	d, _ = newTestMessageDialog(&pressed, &escaped)
	d.HandleKey(tuikittest.Key(tcell.KeyEscape))
	if !escaped || !d.Visible() {
		t.Errorf("OnEscape ran %v, visible %v: want the host's callback alone", escaped, d.Visible())
	}
}

func TestMessageDialogClickFiresButton(t *testing.T) {
	var pressed []int
	d, _ := newTestMessageDialog(&pressed, nil)
	x := d.ButtonRowStartX(d.Buttons()) + 2
	d.HandleMouse(tuikittest.Press(x, d.ButtonRowY()))
	if len(pressed) != 1 || pressed[0] != 0 || d.ButtonFocus() != 0 {
		t.Errorf("pressed %v focus %d, want [0] and focus 0", pressed, d.ButtonFocus())
	}
}

// SetButtons replaces the row, and a focus past a shorter one clamps.
func TestMessageDialogSetButtonsClamps(t *testing.T) {
	var pressed []int
	d, _ := newTestMessageDialog(&pressed, nil)
	d.SetButtons([]string{"Close"}, 5)
	if d.ButtonFocus() != 0 {
		t.Errorf("focus %d, want 0", d.ButtonFocus())
	}
	d.HandleKey(tuikittest.Key(tcell.KeyEnter))
	if len(pressed) != 1 || pressed[0] != 0 {
		t.Errorf("pressed %v, want [0]", pressed)
	}
}

// Lines draw in order, indented and styled, wrapped ones taking several rows,
// and none on or below the separator one above the button row.
func TestMessageDialogDrawsLines(t *testing.T) {
	var pressed []int
	d, scr := newTestMessageDialog(&pressed, nil)
	bold := theme.StyleDialog().Bold(true)
	d.SetLines([]MessageLine{
		{Text: "plain"},
		{Text: "indented", Indent: 2, Style: bold},
		{Text: "one two three four five six seven eight nine ten eleven twelve", Wrap: true},
		{Text: "LAST"},
	})
	d.SetSize(50, 12) // room for the five rows
	d.Draw(scr)
	in := d.InnerRect()
	x, y, ok := scr.Find("plain")
	if !ok || x != in.X+1 || y != in.Y+1 {
		t.Errorf("plain at (%d,%d,%v), want (%d,%d)", x, y, ok, in.X+1, in.Y+1)
	}
	if x, y, _ := scr.Find("indented"); x != in.X+3 || y != in.Y+2 {
		t.Errorf("indented at (%d,%d), want (%d,%d)", x, y, in.X+3, in.Y+2)
	}
	if got := scr.RowOf("twelve"); got <= in.Y+3 {
		t.Errorf("wrapped line ended on row %d, want past its first row %d", got, in.Y+3)
	}
	if got := scr.RowOf("LAST"); got < 0 || got >= d.ButtonRowY()-1 {
		t.Errorf("LAST on row %d, want above the separator %d", got, d.ButtonRowY()-1)
	}

	// A dialog too short for every line stops at the separator.
	d.SetSize(50, 7)
	scr2 := tuikittest.NewScreen(80, 24)
	d.Draw(scr2)
	if got := scr2.RowOf("LAST"); got >= 0 {
		t.Errorf("LAST drawn on row %d of a dialog with no room for it", got)
	}
}
