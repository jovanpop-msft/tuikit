package editor

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

// replacePress is one press of a host's Replace button, as goSSMS wires it: replace
// the current match, or, when none is current, find one.
func replacePress(e *Editor) {
	if !e.ReplaceCurrent() {
		e.FindNext(1)
	}
}

// TestReplaceAllInSelectionAfterReplaceCurrent: a Replace on the scope's last row
// moves the text after it, so a scope captured once and never adjusted made the
// Replace All that followed filter against stale columns: a growing replacement
// pushed the last selected match out of scope, a shrinking one pulled text past
// the selection's end in.
func TestReplaceAllInSelectionAfterReplaceCurrent(t *testing.T) {
	for _, tc := range []struct {
		name, text, repl, want string
		n                      int
	}{
		{"growing", "foo foo foo;bar", "zzzzzz", "zzzzzz zzzzzz zzzzzz;bar", 2},
		{"shrinking", "foo foo foo;foo", "z", "z z z;foo", 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEditor(tc.text)
			e.SelectRange(0, 0, 0, 11)
			setSearch(t, e, SearchOptions{Query: "foo", Replace: tc.repl, InSelection: true})
			replacePress(e) // finds
			replacePress(e) // replaces
			if got := e.ReplaceAll(); got != tc.n {
				t.Errorf("ReplaceAll = %d, want %d", got, tc.n)
			}
			if got := e.Text(); got != tc.want {
				t.Errorf("text = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestInSelectionScopeFollowsLinesInsertedAbove: lines typed, pasted or undone
// above the scope move it with its text, so Replace All still replaces what was
// selected, not the row that now sits where the selection was.
func TestInSelectionScopeFollowsLinesInsertedAbove(t *testing.T) {
	e := newTestEditor("top\nfoo foo\nfoo")
	e.SetBounds(0, 0, 40, 10)
	e.SelectRange(1, 0, 1, 7)
	setSearch(t, e, SearchOptions{Query: "foo", Replace: "Z", InSelection: true})

	// Typed: click at the end of the first line, Enter, "foo".
	e.SelectRange(0, 3, 0, 3)
	e.HandleKey(tuikittest.Key(tcell.KeyEnter, tcell.ModNone))
	for _, r := range "foo" {
		e.HandleKey(tuikittest.Rune(r, tcell.ModNone))
	}
	e.Draw(tuikittest.NewScreen(40, 10))
	// Pasted, with no Draw in between: the scope catches up across both.
	e.SelectRange(0, 0, 0, 0)
	e.Paste("p1\np2\n")
	// And undone, which splices the lines back out.
	e.Undo()

	if got, want := e.ReplaceAll(), 2; got != want {
		t.Fatalf("ReplaceAll = %d, want %d (text %q)", got, want, e.Text())
	}
	if got, want := e.Text(), "top\nfoo\nZ Z\nfoo"; got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
}

// TestInSelectionScopeInvalidatedByEditOnBoundaryRow: an edit on the scope's
// first or last row may have moved its columns, which can't be known, so the
// scope empties and nothing is found or replaced, rather than text the user did
// not select.
func TestInSelectionScopeInvalidatedByEditOnBoundaryRow(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(e *Editor)
	}{
		{"typed before the first column", func(e *Editor) {
			e.SelectRange(0, 0, 0, 0)
			e.HandleKey(tuikittest.Rune('x', tcell.ModNone))
		}},
		{"replaced wholesale", func(e *Editor) { e.SetText("foo foo foo\nfoo") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := newTestEditor("foo foo foo\nfoo")
			e.SelectRange(0, 4, 0, 11)
			setSearch(t, e, SearchOptions{Query: "foo", Replace: "Z", InSelection: true})
			tc.edit(e)
			before := e.Text()
			if e.FindNext(1) {
				t.Error("FindNext found a match in an invalidated scope")
			}
			if got := e.ReplaceAll(); got != 0 {
				t.Errorf("ReplaceAll = %d, want 0", got)
			}
			if got := e.Text(); got != before {
				t.Errorf("text = %q, want %q unchanged", got, before)
			}
		})
	}
}

// TestInSelectionScopeSurvivesEditsBelowAndBetween: an edit below the scope, or
// on a row strictly between its first and last, leaves the boundary rows' text
// alone, so the scope holds (the last row renumbered by any change in line count).
func TestInSelectionScopeSurvivesEditsBelowAndBetween(t *testing.T) {
	e := newTestEditor("foo a\nmid\nfoo b\nfoo")
	e.SetBounds(0, 0, 40, 10)
	e.SelectRange(0, 0, 2, 5)
	setSearch(t, e, SearchOptions{Query: "foo", Replace: "Z", InSelection: true})
	e.SelectRange(1, 3, 1, 3)
	e.Paste("\nfoo") // between: "mid", "foo" inside the scope
	// A host draws between the two; without it changedSince merges both into one
	// span covering the last row, which (conservatively) empties the scope.
	e.Draw(tuikittest.NewScreen(40, 10))
	e.SelectRange(4, 3, 4, 3) // the line below the scope
	e.Paste(" foo")
	if got, want := e.ReplaceAll(), 3; got != want {
		t.Fatalf("ReplaceAll = %d, want %d (text %q)", got, want, e.Text())
	}
	if got, want := e.Text(), "Z a\nmid\nZ\nZ b\nfoo foo"; got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
}

// TestFindNextInSelectionStaysInScope: under InSelection, Find and the first
// Replace press stay inside the selection and wrap within it (SSMS's "Look in:
// Selection"). Searching the whole document from the selection's end made the
// first Replace land on the first match after the selection.
func TestFindNextInSelectionStaysInScope(t *testing.T) {
	e := newTestEditor("foo foo;foo")
	e.SelectRange(0, 0, 0, 7)
	setSearch(t, e, SearchOptions{Query: "foo", Replace: "Z", InSelection: true})
	for i, want := range []string{"0:0-3", "0:4-7", "0:0-3"} {
		if got := findAt(t, e, 1); got != want {
			t.Fatalf("FindNext #%d = %s, want %s", i+1, got, want)
		}
	}
	if got, want := findAt(t, e, -1), "0:4-7"; got != want {
		t.Fatalf("FindNext(-1) = %s, want %s (wrapping backwards within the scope)", got, want)
	}

	e.SelectRange(0, 0, 0, 7)
	setSearch(t, e, SearchOptions{Query: "foo", Replace: "Z", InSelection: true})
	for range 4 {
		replacePress(e)
	}
	if got, want := e.Text(), "Z Z;foo"; got != want {
		t.Fatalf("after four Replace presses text = %q, want %q", got, want)
	}
}

// TestReplaceCurrentRefusesOutOfScopeMatch: a match the user selected by hand
// outside the scope is not replaced; Replace moves into the scope instead.
func TestReplaceCurrentRefusesOutOfScopeMatch(t *testing.T) {
	e := newTestEditor("foo foo;foo")
	e.SelectRange(0, 0, 0, 7)
	setSearch(t, e, SearchOptions{Query: "foo", Replace: "Z", InSelection: true})
	e.search.found = true
	e.SelectRange(0, 8, 0, 11)
	if e.ReplaceCurrent() {
		t.Fatalf("ReplaceCurrent replaced an out-of-scope match: %q", e.Text())
	}
}

// TestPrependLinesMovesSearchScope: PrependLines moves the scope down with its
// text, and a trim below it in the same call leaves it alone (synced per splice:
// merged, the two spans would cover the scope and empty it). A trim reaching into
// the scope empties it.
func TestPrependLinesMovesSearchScope(t *testing.T) {
	e := newTestEditor("foo foo\nx\ny")
	e.SelectRange(0, 0, 0, 7)
	setSearch(t, e, SearchOptions{Query: "foo", Replace: "Z", InSelection: true})
	e.PrependLines("foo", 3)
	if got, want := e.ReplaceAll(), 2; got != want {
		t.Fatalf("ReplaceAll = %d, want %d (text %q)", got, want, e.Text())
	}
	if got, want := e.Text(), "foo\nZ Z\nx"; got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}

	e.SetText("a\nfoo\nfoo foo")
	e.SelectRange(1, 0, 2, 7)
	setSearch(t, e, SearchOptions{Query: "foo", Replace: "Z", InSelection: true})
	e.PrependLines("new", 3)
	if got := e.ReplaceAll(); got != 0 {
		t.Fatalf("ReplaceAll = %d after the trim cut into the scope, want 0 (text %q)", got, e.Text())
	}
}
