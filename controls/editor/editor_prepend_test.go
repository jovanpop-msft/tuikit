package editor

import (
	"strings"
	"testing"

	"github.com/radix29/tuikit/tuikittest"
)

// numbered returns n distinct short lines, each starting with prefix.
func numbered(prefix string, n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = prefix + strings.Repeat("x", i%3) + string(rune('a'+i%26))
	}
	return strings.Join(lines, "\n")
}

// TestPrependLinesKeepsSelection: the Status History bug. A selection made
// before a prepend still covers the same text after it, on a read-only editor,
// so a second Copy copies what the first did.
func TestPrependLinesKeepsSelection(t *testing.T) {
	e := NewEditor(nil)
	e.SetReadOnly(true)
	e.SetText("one\ntwo\nthree\nfour")
	e.SelectRange(1, 1, 2, 3)
	want := e.SelectedText()

	e.PrependLines("new1\nnew2", 0)

	if !e.HasSelection() {
		t.Fatal("selection dropped")
	}
	if got := e.SelectedText(); got != want {
		t.Fatalf("SelectedText = %q, want %q", got, want)
	}
	if r, c := e.CursorPos(); r != 4 || c != 3 {
		t.Fatalf("cursor = (%d,%d), want (4,3)", r, c)
	}
	if got, wantText := e.Text(), "new1\nnew2\none\ntwo\nthree\nfour"; got != wantText {
		t.Fatalf("Text = %q, want %q", got, wantText)
	}
}

// TestPrependLinesScroll: a view at the top stays there (the newest line is
// visible); a scrolled view moves down with its text.
func TestPrependLinesScroll(t *testing.T) {
	for _, wrap := range []bool{false, true} {
		e := NewEditor(nil)
		e.SetWrapMode(wrap)
		e.SetBounds(0, 0, 40, 5)
		e.SetText(numbered("l", 50))

		e.PrependLines("a\nb", 0)
		if r, _ := e.ScrollPos(); r != 0 {
			t.Errorf("wrap=%v: scroll at top moved to %d", wrap, r)
		}

		e.scrollRow = 10
		e.PrependLines("c\nd\ne", 0)
		if r, _ := e.ScrollPos(); r != 13 {
			t.Errorf("wrap=%v: scroll 10 -> %d, want 13", wrap, r)
		}
	}
}

// TestPrependLinesWrappedScrollCountsVisualRows: wrapped, a prepended line that
// wraps onto several rows moves the view by its visual rows, not by one.
func TestPrependLinesWrappedScrollCountsVisualRows(t *testing.T) {
	e := NewEditor(nil)
	e.SetGutterVisible(false)
	e.SetWrapMode(true)
	e.SetBounds(0, 0, 11, 5) // wrapWidth 10
	e.SetText(numbered("l", 30))
	e.scrollRow = 5
	top := e.doc.Line(5)

	e.PrependLines(strings.Repeat("word ", 6), 0) // 30 runes: three visual rows

	vls := e.buildVisualLines(e.wrapWidth())
	if got := e.doc.Line(vls[e.scrollRow].row); string(got) != string(top) {
		t.Fatalf("top line = %q, want %q (scrollRow %d)", got, top, e.scrollRow)
	}
}

// TestPrependLinesTrim: keepMax drops lines from the bottom, and a cursor,
// anchor or selection on them is clamped to the new last line.
func TestPrependLinesTrim(t *testing.T) {
	e := NewEditor(nil)
	e.SetText("a\nb\nc\nd")
	e.SelectRange(1, 0, 3, 1) // "b\nc\nd"

	e.PrependLines("x\ny", 4) // x y a b | c d dropped

	if got := e.Text(); got != "x\ny\na\nb" {
		t.Fatalf("Text = %q", got)
	}
	if r, c := e.CursorPos(); r != 3 || c != 1 {
		t.Fatalf("cursor = (%d,%d), want (3,1)", r, c)
	}
	if got := e.SelectedText(); got != "b" {
		t.Fatalf("SelectedText = %q, want %q", got, "b")
	}

	// A selection wholly inside the cut collapses to empty.
	e.SelectRange(2, 0, 3, 1)
	e.PrependLines("z\nw", 4) // z w x y | a b dropped
	if e.HasSelection() {
		t.Fatalf("selection survived its rows being trimmed: %q", e.SelectedText())
	}
	if r, c := e.CursorPos(); r != 3 || c != 1 {
		t.Fatalf("cursor = (%d,%d), want (3,1)", r, c)
	}

	// Trim alone (empty text) still applies.
	e.PrependLines("", 2)
	if got := e.Text(); got != "z\nw" {
		t.Fatalf("trim-only Text = %q", got)
	}
}

// TestPrependLinesOnEmptyDocument: the blank line SetText("") leaves is
// replaced, not kept as a trailing empty line.
func TestPrependLinesOnEmptyDocument(t *testing.T) {
	e := NewEditor(nil)
	e.PrependLines("first", 0)
	if got := e.Text(); got != "first" {
		t.Fatalf("Text = %q, want %q", got, "first")
	}
	if r, c := e.CursorPos(); r != 0 || c != 0 {
		t.Fatalf("cursor = (%d,%d), want (0,0)", r, c)
	}
	e.PrependLines("second", 0)
	if got := e.Text(); got != "second\nfirst" {
		t.Fatalf("Text = %q", got)
	}
}

// TestPrependLinesClearsUndo: the history's row numbers no longer match, so
// an Undo after a prepend must not splice old text in at the wrong place.
func TestPrependLinesClearsUndo(t *testing.T) {
	e := newTestEditor("abc")
	e.HandleKey(tuikittest.Rune('d'))
	if len(e.undoStack) == 0 {
		t.Fatal("setup: typing pushed no undo step")
	}
	e.PrependLines("top", 0)
	if len(e.undoStack) != 0 || len(e.redoStack) != 0 || e.stepOpen {
		t.Fatal("undo history kept across PrependLines")
	}
}

// TestPrependLinesBumpsVersion: highlight and wrap caches key on the version.
func TestPrependLinesBumpsVersion(t *testing.T) {
	e := newTestEditor("a")
	v := e.doc.Version()
	e.PrependLines("b", 0)
	if e.doc.Version() == v {
		t.Fatal("version unchanged")
	}
}
