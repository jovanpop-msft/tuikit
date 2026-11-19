package editor

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

// Under tcell.OptAdvancedKeys, and for Ctrl+Shift chords, Ctrl+letter arrives
// as KeyRune+ModCtrl. The key table binds KeyCtrlA..KeyCtrlZ, so the editor
// folds the key itself (core.NormalizeKey) rather than rely on its host.
func TestEditorCtrlLetterInAdvancedKeyForm(t *testing.T) {
	adv := func(s string, m tcell.ModMask) *tcell.EventKey {
		return tcell.NewEventKeyEx(tcell.KeyRune, s, m, true, 0, 1)
	}

	e := newTestEditor("one\ntwo")
	if !e.HandleKey(adv("a", tcell.ModCtrl)) {
		t.Fatal("advanced Ctrl+A was not acted on")
	}
	if got := e.SelectedText(); got != "one\ntwo" {
		t.Fatalf("advanced Ctrl+A: selection = %q, want everything", got)
	}

	e = newTestEditor("one\ntwo")
	e.HandleKey(adv("d", tcell.ModCtrl)) // Duplicate Lines
	if e.Text() == "one\ntwo" {
		t.Fatal("advanced Ctrl+D did nothing")
	}
	e.HandleKey(adv("z", tcell.ModCtrl))
	if got := e.Text(); got != "one\ntwo" {
		t.Fatalf("advanced Ctrl+Z: Text() = %q, want the duplicate undone", got)
	}

	// Ctrl+Alt is AltGr on many layouts: never a Ctrl+letter shortcut.
	e = newTestEditor("one\ntwo")
	e.HandleKey(adv("a", tcell.ModCtrl|tcell.ModAlt))
	if e.HasSelection() {
		t.Fatal("Ctrl+Alt+A was folded into Ctrl+A")
	}
}
