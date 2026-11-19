package core

import (
	"unicode"

	"github.com/gdamore/tcell/v3"
)

// NormalizeKey folds a Ctrl-modified letter back to KeyCtrlA..KeyCtrlZ,
// keeping other modifiers. Every tuikit widget that binds Ctrl+letter calls it
// at the top of its HandleKey; a host that matches Ctrl+letter keys itself
// should call it once on each key before dispatching.
//
// tcell folds only when Ctrl is the sole modifier, so Ctrl+Shift+<letter>
// arrives as KeyRune with ModCtrl|ModShift and matches no binding. Under
// tcell.OptAdvancedKeys(true) even a plain Ctrl+<letter> arrives that way.
// Kitty sends the base rune ('o'), xterm's modifyOtherKeys the shifted one
// ('O'), hence ToLower. Without this, Ctrl+Shift chords are dead on exactly
// the terminals that can encode them, and every Ctrl shortcut is dead under
// advanced key reporting, with no error to point at the cause.
//
// It is idempotent (a KeyCtrlA..KeyCtrlZ event comes back as is) and allocates
// only when it folds.
func NormalizeKey(ev *tcell.EventKey) *tcell.EventKey {
	// Ctrl+Shift only: Ctrl+Alt is AltGr on many layouts and must stay text.
	if ev.Key() != tcell.KeyRune || ev.Modifiers()&^tcell.ModShift != tcell.ModCtrl {
		return ev
	}
	str := []rune(ev.Str())
	if len(str) != 1 {
		return ev
	}
	r := unicode.ToLower(str[0])
	if r < 'a' || r > 'z' {
		return ev
	}
	return tcell.NewEventKey(tcell.KeyCtrlA+tcell.Key(r-'a'), "", ev.Modifiers())
}
