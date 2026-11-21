package editor

import (
	"strings"
	"testing"

	"github.com/radix29/tuikit/theme"
	"github.com/radix29/tuikit/tuikittest"
)

// lineMarks reads row y back as one character per column: 'h' for a cell on
// the current-line background, 's' for the selection style, '.' otherwise.
func lineMarks(s *tuikittest.Screen, y, w int) string {
	p := theme.Active()
	sel := theme.StyleSelected()
	var sb strings.Builder
	for x := range w {
		st := s.StyleAt(x, y)
		switch {
		case st == sel:
			sb.WriteByte('s')
		case st.GetBackground() == p.GridRowAlt:
			sb.WriteByte('h')
		default:
			sb.WriteByte('.')
		}
	}
	return sb.String()
}

// The cursor's line is shaded across the whole content width, padding
// included; other lines are not, and nothing is shaded while the option is off.
func TestCurrentLineHighlightShadesTheCursorLine(t *testing.T) {
	e := styleEditor("ab\ncd\nef", 6, 3)
	e.cursorRow = 1

	s := tuikittest.NewScreen(6, 3)
	e.Draw(s)
	if got := lineMarks(s, 1, 6); got != "......" {
		t.Errorf("off: row 1 = %q, want no shading", got)
	}

	e.SetCurrentLineHighlight(true)
	e.Draw(s)
	for y, want := range []string{"......", "hhhhhh", "......"} {
		if got := lineMarks(s, y, 6); got != want {
			t.Errorf("row %d = %q, want %q", y, got, want)
		}
	}
}

// A selection on the current line still draws in the selection style.
func TestCurrentLineHighlightUnderSelection(t *testing.T) {
	e := styleEditor("abcd", 6, 2)
	e.SetCurrentLineHighlight(true)
	e.SelectRange(0, 1, 0, 3)

	s := tuikittest.NewScreen(6, 2)
	e.Draw(s)
	if got, want := lineMarks(s, 0, 6), "hsshhh"; got != want {
		t.Errorf("row 0 = %q, want %q", got, want)
	}
}

// Wrapped, every visual row of the cursor's logical line is shaded.
func TestCurrentLineHighlightWrapped(t *testing.T) {
	e := styleEditor("aaaa bbbb cccc\nx", 6, 5)
	e.SetWrapMode(true)
	e.SetCurrentLineHighlight(true)

	s := tuikittest.NewScreen(6, 5)
	e.Draw(s)
	vls := e.buildVisualLines(e.wrapWidth())
	for y := range min(len(vls), 5) {
		want := "......"
		if vls[y].row == 0 {
			want = "hhhhhh"
		}
		if got := lineMarks(s, y, 6); got != want {
			t.Errorf("row %d (line %d) = %q, want %q", y, vls[y].row, got, want)
		}
	}
}
