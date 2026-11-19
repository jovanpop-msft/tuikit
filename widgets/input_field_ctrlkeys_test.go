package widgets

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

// Under tcell.OptAdvancedKeys, and for Ctrl+Shift chords, Ctrl+letter arrives
// as KeyRune+ModCtrl. InputField folds it itself (core.NormalizeKey), so a host
// that doesn't still gets Ctrl+A/Ctrl+U.
func TestInputFieldCtrlLetterInAdvancedKeyForm(t *testing.T) {
	adv := func(s string, m tcell.ModMask) *tcell.EventKey {
		return tcell.NewEventKeyEx(tcell.KeyRune, s, m, true, 0, 1)
	}

	f := newTestInputField("hello")
	if !f.HandleKey(adv("a", tcell.ModCtrl)) || f.SelectedText() != "hello" {
		t.Fatalf("advanced Ctrl+A: selection = %q, want %q", f.SelectedText(), "hello")
	}

	f = newTestInputField("hello")
	f.HandleKey(adv("u", tcell.ModCtrl))
	if got := f.Value(); got != "" {
		t.Fatalf("advanced Ctrl+U: value = %q, want it cleared", got)
	}

	// Ctrl+Alt is AltGr on many layouts: never a Ctrl+letter shortcut.
	f = newTestInputField("hello")
	f.HandleKey(adv("a", tcell.ModCtrl|tcell.ModAlt))
	if f.HasSelection() {
		t.Fatal("Ctrl+Alt+A was folded into Ctrl+A")
	}
}
