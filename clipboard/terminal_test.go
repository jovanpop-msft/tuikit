package clipboard

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

// stubScreen answers only what TerminalLikely asks.
type stubScreen struct {
	tcell.Screen
	da52     bool
	termName string
}

func (s stubScreen) HasClipboard() bool         { return s.da52 }
func (s stubScreen) Terminal() (string, string) { return s.termName, "" }

// VTE (xfce4-terminal, Terminator) answers DA1 with "?61;1;21;22;28c" and
// ignores an OSC 52 write; tmux 3.6 answers "?1;2;4c" yet forwards it.
func TestTerminalLikely(t *testing.T) {
	for _, tc := range []struct {
		name     string
		da52     bool
		termName string
		want     bool
	}{
		{"VTE", false, "", false},
		{"advertises 52", true, "", true},
		{"tmux", false, "tmux", true},
		{"screen", false, "screen", false},
	} {
		if got := TerminalLikely(stubScreen{da52: tc.da52, termName: tc.termName}); got != tc.want {
			t.Errorf("%s: TerminalLikely = %v, want %v", tc.name, got, tc.want)
		}
	}
}
