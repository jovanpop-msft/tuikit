package core

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

func paneRows(b *Buffer) string {
	rows := b.Rows()
	for i, r := range rows {
		rows[i] = strings.TrimRight(r, " ")
	}
	return strings.Join(rows, "\n")
}

func TestLinePaneScrollClamps(t *testing.T) {
	var p LinePane
	if p.ScrollBy(-1, 10, 4) || p.Scroll != 0 {
		t.Fatalf("ScrollBy(-1) at the top moved to %d", p.Scroll)
	}
	if !p.ScrollBy(100, 10, 4) || p.Scroll != 6 {
		t.Fatalf("ScrollBy(100) -> %d, want 6 (10 lines in 4 rows)", p.Scroll)
	}
	if p.ScrollBy(1, 10, 4) {
		t.Error("ScrollBy past the end reported a move")
	}
	if !p.CanUp() || p.CanDown(10, 4) {
		t.Errorf("at the end: CanUp=%v CanDown=%v, want true,false", p.CanUp(), p.CanDown(10, 4))
	}
	// Lines that fit leave nothing to scroll.
	if !p.ScrollBy(0, 3, 4) || p.Scroll != 0 || p.MaxScroll(3, 4) != 0 {
		t.Errorf("3 lines in 4 rows: Scroll=%d Max=%d, want 0,0", p.Scroll, p.MaxScroll(3, 4))
	}
}

func TestLinePaneDrawInsetHintAndPlaceholder(t *testing.T) {
	b := NewBuffer(14, 3)
	p := LinePane{More: "▾ more"}
	lines := []string{"one", "two", "three", "four"}
	p.Draw(b, b.Rect(), tcell.StyleDefault, lines, nil)
	want := " one\n two\n three ▾ more"
	if got := paneRows(b); got != want {
		t.Errorf("top:\n%s\nwant:\n%s", got, want)
	}
	// At the bottom the hint goes.
	p.ScrollBy(1, len(lines), 3)
	p.Draw(b, b.Rect(), tcell.StyleDefault, lines, nil)
	if got, want := paneRows(b), " two\n three\n four"; got != want {
		t.Errorf("bottom:\n%s\nwant:\n%s", got, want)
	}
	p.DrawEmpty(b, b.Rect(), tcell.StyleDefault, "None")
	if got, want := paneRows(b), " None\n\n"; got != want {
		t.Errorf("empty:\n%q\nwant:\n%q", got, want)
	}
}

// A pane grown taller (or text rewrapped shorter) under an older Scroll shows
// its last line at the bottom, not blank rows below it.
func TestLinePaneDrawClampsStaleScroll(t *testing.T) {
	b := NewBuffer(8, 3)
	p := LinePane{Scroll: 5}
	p.Draw(b, b.Rect(), tcell.StyleDefault, []string{"a", "b", "c", "d"}, nil)
	if p.Scroll != 1 {
		t.Fatalf("Scroll = %d, want 1", p.Scroll)
	}
	if got, want := paneRows(b), " b\n c\n d"; got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestLinePaneDrawPerLineStyle(t *testing.T) {
	b := NewBuffer(8, 2)
	red := tcell.StyleDefault.Foreground(color.Red)
	var p LinePane
	p.Draw(b, b.Rect(), tcell.StyleDefault, []string{"a", "b"}, func(i int) tcell.Style {
		if i == 1 {
			return red
		}
		return tcell.StyleDefault
	})
	if b.StyleAt(1, 1) != red || b.StyleAt(1, 0) == red {
		t.Error("style(i) not applied to line i only")
	}
}
