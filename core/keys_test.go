package core

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

// Ctrl+Shift+<letter> never reaches a widget as KeyCtrlX: tcell folds a
// Ctrl-modified rune into KeyCtrlA..KeyCtrlZ only when Ctrl is the sole
// modifier. The two Ctrl+Shift+O forms below are what Kitty (CSI 111;6u,
// base-layout rune) and xterm modifyOtherKeys (CSI 27;6;79~, shifted rune)
// actually deliver — both verified against a running goSSMS binary.
func TestNormalizeKey(t *testing.T) {
	tests := []struct {
		name    string
		in      *tcell.EventKey
		wantKey tcell.Key
		wantMod tcell.ModMask
	}{
		{"kitty ctrl+shift+o", tcell.NewEventKey(tcell.KeyRune, "o", tcell.ModCtrl|tcell.ModShift),
			tcell.KeyCtrlO, tcell.ModCtrl | tcell.ModShift},
		{"modifyOtherKeys ctrl+shift+O", tcell.NewEventKey(tcell.KeyRune, "O", tcell.ModCtrl|tcell.ModShift),
			tcell.KeyCtrlO, tcell.ModCtrl | tcell.ModShift},
		{"ctrl+shift+u", tcell.NewEventKey(tcell.KeyRune, "u", tcell.ModCtrl|tcell.ModShift),
			tcell.KeyCtrlU, tcell.ModCtrl | tcell.ModShift},
		{"advanced keys ctrl+a", tcell.NewEventKeyEx(tcell.KeyRune, "a", tcell.ModCtrl, true, 0, 1),
			tcell.KeyCtrlA, tcell.ModCtrl},
		{"already folded ctrl+o", tcell.NewEventKey(tcell.KeyCtrlO, "", tcell.ModCtrl),
			tcell.KeyCtrlO, tcell.ModCtrl},
		// AltGr arrives as Ctrl+Alt on many layouts and produces text.
		{"altgr letter stays a rune", tcell.NewEventKey(tcell.KeyRune, "o", tcell.ModCtrl|tcell.ModAlt),
			tcell.KeyRune, tcell.ModCtrl | tcell.ModAlt},
		{"ctrl+shift+digit stays a rune", tcell.NewEventKey(tcell.KeyRune, "1", tcell.ModCtrl|tcell.ModShift),
			tcell.KeyRune, tcell.ModCtrl | tcell.ModShift},
		{"ctrl+slash stays a rune", tcell.NewEventKey(tcell.KeyRune, "/", tcell.ModCtrl),
			tcell.KeyRune, tcell.ModCtrl},
		{"plain typing stays a rune", tcell.NewEventKey(tcell.KeyRune, "o", tcell.ModNone),
			tcell.KeyRune, tcell.ModNone},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeKey(tt.in)
			if got.Key() != tt.wantKey || got.Modifiers() != tt.wantMod {
				t.Fatalf("NormalizeKey = Key=%d Mod=%d, want Key=%d Mod=%d",
					got.Key(), got.Modifiers(), tt.wantKey, tt.wantMod)
			}
			if again := NormalizeKey(got); again != got {
				t.Fatal("NormalizeKey is not idempotent")
			}
		})
	}
}
