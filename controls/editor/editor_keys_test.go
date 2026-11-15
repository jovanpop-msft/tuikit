package editor

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

// The tests here pin editorKeys' traits and each binding's selection outcome
// where no other test does: a wrong trait or outcome in the table still
// compiles and moves the cursor, so only behaviour catches it.

func TestEditorPgDnKeepsTheGoalColumn(t *testing.T) {
	e := newTestEditor("SELECT column_one\nx\nshort\nFROM some_long_table_name")
	e.SetBounds(0, 0, 60, 2) // PgDn moves two rows: 0 -> 2
	e.cursorRow, e.cursorCol, e.desiredCol = 0, 10, 10

	e.HandleKey(tuikittest.Key(tcell.KeyPgDn, tcell.ModNone))
	if e.cursorRow != 2 || e.cursorCol != len("short") {
		t.Fatalf("PgDn: cursor (%d,%d), want (2,5)", e.cursorRow, e.cursorCol)
	}
	e.HandleKey(tuikittest.Key(tcell.KeyDown, tcell.ModNone))
	if e.cursorCol != 10 {
		t.Fatalf("after PgDn then Down onto a long line: col %d, want 10 — PgDn re-anchored the goal column", e.cursorCol)
	}
}

// TestEditorRawDelDoesNotOpenCompletion: a terminal sending 0x7f decodes to a
// Backspace key that keeps Str "\x7f", which would read as a typed character without
// traitNotTyped and pop completion open on every Backspace.
func TestEditorRawDelDoesNotOpenCompletion(t *testing.T) {
	e := newTestEditor("Cust")
	e.SetCompletionProvider(testCompletionProvider("Customers"))
	e.cursorCol = 4

	ev := tcell.NewEventKey(tcell.KeyRune, "\x7f", tcell.ModNone)
	if (ev.Key() != tcell.KeyBackspace && ev.Key() != tcell.KeyBackspace2) || ev.Str() == "" {
		t.Fatalf("precondition: 0x7f decodes to key %v str %q", ev.Key(), ev.Str())
	}
	e.HandleKey(ev)
	if e.Text() != "Cus" {
		t.Fatalf("Backspace: text %q, want %q", e.Text(), "Cus")
	}
	if e.OverlayActive() {
		t.Fatal("Backspace opened the completion popup")
	}
}

func TestEditorShiftHomeExtendsTheSelection(t *testing.T) {
	e := newTestEditor("hello")
	e.cursorCol = 5
	e.HandleKey(tuikittest.Key(tcell.KeyHome, tcell.ModShift))
	if got := e.SelectedText(); got != "hello" {
		t.Fatalf("Shift+Home selected %q, want %q", got, "hello")
	}
}

// TestEditorKeysThatKeepTheSelection covers the bindings that leave the
// selection alone rather than dropping it as a plain key does.
func TestEditorKeysThatKeepTheSelection(t *testing.T) {
	for name, tc := range map[string]struct {
		ev    *tcell.EventKey
		setup func(*Editor)
	}{
		// Unbound while no comment prefix is set: the key is declined, so it must
		// not touch anything, the selection included.
		"Ctrl+/ without a prefix": {ev: tuikittest.Rune('/', tcell.ModCtrl)},
		// Neither a completion provider nor OnRightClick: nothing to fire.
		"Ctrl+Space with nothing installed": {ev: tcell.NewEventKey(tcell.KeyRune, " ", tcell.ModCtrl)},
		"Ctrl+Shift+Down (Move Line)":       {ev: tuikittest.Key(tcell.KeyDown, tcell.ModCtrl|tcell.ModShift)},
	} {
		t.Run(name, func(t *testing.T) {
			e := newTestEditor("one\ntwo\nthree")
			e.HandleKey(tuikittest.Key(tcell.KeyDown, tcell.ModShift))
			if !e.HasSelection() {
				t.Fatal("precondition: Shift+Down made no selection")
			}
			e.HandleKey(tc.ev)
			if !e.HasSelection() {
				t.Fatal("the key dropped the selection")
			}
		})
	}
}
