package clipboard

import "github.com/gdamore/tcell/v3"

// TerminalLikely reports whether an OSC 52 write (tcell's SetClipboard)
// probably reached the clipboard. OSC 52 has no acknowledgement, and VTE
// (xfce4-terminal, GNOME Terminal) ignores it, so a host's "Copied" can be
// false; this is what to hedge that message on.
//
// True when the terminal advertised OSC 52 in its DA1 reply (attribute 52), or
// is tmux, which forwards OSC 52 to the outer terminal (set-clipboard
// on/external) but answers DA1 itself without 52 (3.6: "?1;2;4c"), so
// HasClipboard alone would flag every copy over SSH-in-tmux. tmux is known by
// its XTVERSION reply ("tmux 3.6"). GNU screen answers no XTVERSION, and passes
// OSC 52 through only when wrapped in DCS, which tcell doesn't do — so the
// hedge is right there.
func TerminalLikely(s tcell.Screen) bool {
	if s.HasClipboard() {
		return true
	}
	name, _ := s.Terminal()
	return name == "tmux"
}
