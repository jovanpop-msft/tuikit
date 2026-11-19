package core

import (
	"strings"

	"github.com/gdamore/tcell/v3"
)

// MouseEventName is EventKey.Name()'s counterpart for a mouse event, which
// tcell lacks: the modifiers the terminal delivered, then the button —
// "Ctrl+Shift+Button1", "WheelUp", "Release" (no button held). For a key or
// mouse diagnostic and debug logs, where "Shift+Button1" against a bare
// "Button1" is the whole diagnosis (a terminal can keep Shift+click for its
// own selection and never forward it).
//
// A wheel bit wins over a held button; of several held buttons the lowest
// numbered is named; Button4..8 are "Button".
func MouseEventName(ev *tcell.EventMouse) string {
	var parts []string
	for _, m := range []struct {
		mask tcell.ModMask
		name string
	}{
		{tcell.ModCtrl, "Ctrl"},
		{tcell.ModAlt, "Alt"},
		{tcell.ModShift, "Shift"},
		{tcell.ModMeta, "Meta"},
	} {
		if ev.Modifiers()&m.mask != 0 {
			parts = append(parts, m.name)
		}
	}
	button := "Release"
	switch b := ev.Buttons(); {
	case b&tcell.WheelUp != 0:
		button = "WheelUp"
	case b&tcell.WheelDown != 0:
		button = "WheelDown"
	case b&tcell.WheelLeft != 0:
		button = "WheelLeft"
	case b&tcell.WheelRight != 0:
		button = "WheelRight"
	case b&tcell.Button1 != 0:
		button = "Button1"
	case b&tcell.Button2 != 0:
		button = "Button2"
	case b&tcell.Button3 != 0:
		button = "Button3"
	case b != tcell.ButtonNone:
		button = "Button"
	}
	return strings.Join(append(parts, button), "+")
}
