package editor

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"
)

// The tests here pin prefixStates' resume through the splice log: a cache one or
// more versions behind re-lexes the span changedSince names and may stop at a
// matching state only past it. The bug they guard against resumed from dirtyFrom
// and stopped at the first match below it, which is sound for setLine (one line
// changed) but not for a same-count multi-line replaceRange (Undo of Ctrl+/ over
// a selection, Move Lines): the line after the first can enter in its old state
// while its own text changed, and every state below it stayed stale.

// checkToyStates fails unless every cached state of doc matches a full replay.
func checkToyStates(t *testing.T, what string, cache *prefixStates[int], doc *Document) {
	t.Helper()
	for i := range doc.Len() {
		if got, want := cache.at(doc, i, 0, toyStep), toyFullReplay(doc.Lines(), i); got != want {
			t.Fatalf("%s: line %d (%q) begins at comment depth %d, want %d",
				what, i, string(doc.Line(i)), got, want)
		}
	}
}

func runesOf(ss ...string) [][]rune {
	out := make([][]rune, len(ss))
	for i, s := range ss {
		out[i] = []rune(s)
	}
	return out
}

// TestPrefixStatesSameCountSpliceConvergingInsideSpan is the Document-level
// shape: two lines at row 1 replaced by two others, where line 2 enters in the
// same state as before but line 2's own text now closes a comment it didn't.
func TestPrefixStatesSameCountSpliceConvergingInsideSpan(t *testing.T) {
	doc := newDocument()
	doc.setLines(runesOf("SELECT 0", "x = 1", "/* open", "still", "SELECT 2"))
	var cache prefixStates[int]
	checkToyStates(t, "initial", &cache, doc)

	// Line 1 still leaves depth 0 (so line 2 enters as before), but line 2 no
	// longer opens a comment, so lines 3 and 4 change.
	doc.replaceRange(1, 2, runesOf("y = 2", "SELECT 1"))
	checkToyStates(t, "after a same-count splice", &cache, doc)

	// And back: the span reopens the comment.
	doc.replaceRange(1, 2, runesOf("x = 1", "/* open"))
	checkToyStates(t, "after splicing it back", &cache, doc)
}

// TestPrefixStatesUndoOfToggleLineComments is the review probe: Ctrl+/ over two
// lines (each a setLine, so the forward edit replays fully), then Undo, which
// restores both in one splice.
func TestPrefixStatesUndoOfToggleLineComments(t *testing.T) {
	e := NewEditor(nil)
	e.SetBounds(0, 0, 80, 20)
	e.SetLineComment("--")
	e.SetText("SELECT 0\nx = 1\n/* open\nclose */\nSELECT 2")
	doc := e.Document()
	var cache prefixStates[int]
	checkToyStates(t, "initial", &cache, doc)

	e.SelectRange(1, 0, 2, len(doc.Line(2)))
	e.ToggleLineComments()
	checkToyStates(t, "after Ctrl+/", &cache, doc)
	e.Undo()
	checkToyStates(t, "after Undo", &cache, doc)
	e.Redo()
	checkToyStates(t, "after Redo", &cache, doc)
}

// TestPrefixStatesMoveLinesOverCommentOpener moves a block ending in a
// "/* open" line down past a code line: one splice of three lines in which the
// second still enters in its old state (both are code) but the third now opens
// the comment one line later.
func TestPrefixStatesMoveLinesOverCommentOpener(t *testing.T) {
	e := NewEditor(nil)
	e.SetBounds(0, 0, 80, 20)
	e.SetText("SELECT 0\nx = 1\n/* open\ny = 2\nclose */\nSELECT 2")
	doc := e.Document()
	var cache prefixStates[int]
	checkToyStates(t, "initial", &cache, doc)

	e.SelectRange(1, 0, 2, len(doc.Line(2)))
	e.MoveLinesDown()
	checkToyStates(t, "after MoveLinesDown", &cache, doc)
	e.MoveLinesUp()
	checkToyStates(t, "after MoveLinesUp", &cache, doc)
	e.Undo()
	checkToyStates(t, "after Undo", &cache, doc)
}

// prefixStatesMonkey applies a seeded random series of setLine, replaceRange
// and edit calls to a Document, reading the cache after a random number of them
// (so it falls one, several, or more than spliceLogLen versions behind), and
// checks every state against a full replay after each read.
func prefixStatesMonkey[S comparable](t *testing.T, seed uint64, pool []string, zero S, step func([]rune, S) S) {
	t.Helper()
	r := rand.New(rand.NewPCG(seed, 0))
	randLines := func(n int) [][]rune {
		out := make([][]rune, n)
		for i := range out {
			out[i] = []rune(pool[r.IntN(len(pool))])
		}
		return out
	}
	full := func(lines [][]rune, idx int) S {
		in := zero
		for i := range idx {
			in = step(lines[i], in)
		}
		return in
	}
	doc := newDocument()
	doc.setLines(randLines(8))
	var cache prefixStates[S]
	var log []string
	for op := range 400 {
		// Mostly one mutation between reads (the resume path), sometimes a
		// burst, now and then one long enough to outrun the splice log.
		burst := 1
		switch k := r.IntN(10); {
		case k == 0:
			burst = spliceLogLen + 1 + r.IntN(5)
		case k < 4:
			burst = 2 + r.IntN(5)
		}
		for range burst {
			switch k := r.IntN(10); {
			case k < 4:
				i := r.IntN(doc.Len())
				l := randLines(1)[0]
				log = append(log, fmt.Sprintf("setLine(%d, %q)", i, string(l)))
				doc.setLine(i, l)
			case k < 9:
				row := r.IntN(doc.Len() + 1)
				n := r.IntN(min(4, doc.Len()-row) + 1)
				with := randLines(r.IntN(4))
				if doc.Len()-n+len(with) == 0 {
					with = randLines(1)
				}
				log = append(log, fmt.Sprintf("replaceRange(%d, %d, %d lines)", row, n, len(with)))
				doc.replaceRange(row, n, with)
			default:
				log = append(log, "edit(reverse)")
				doc.edit(func(lines [][]rune) [][]rune { slices.Reverse(lines); return lines })
			}
		}
		for range r.IntN(3) {
			cache.at(doc, r.IntN(doc.Len()), zero, step)
		}
		for i := range doc.Len() {
			if got, want := cache.at(doc, i, zero, step), full(doc.Lines(), i); got != want {
				var text strings.Builder
				for _, l := range doc.Lines() {
					fmt.Fprintf(&text, "\n  %q", string(l))
				}
				t.Fatalf("seed %d, op %d: line %d state %v, want %v\nlast ops: %s\ntext:%s",
					seed, op, i, got, want, strings.Join(log[max(0, len(log)-8):], "; "), text.String())
			}
		}
	}
}

func TestPrefixStatesMonkeyToy(t *testing.T) {
	pool := []string{"/*", "*/", "x", "", "-- /*", "'/*'", "a /* b */ c", "/* /*", "*/ */"}
	for seed := range uint64(60) {
		prefixStatesMonkey(t, seed, pool, 0, toyStep)
	}
}

func TestPrefixStatesMonkeyXML(t *testing.T) {
	pool := []string{"<!--", "-->", "<![CDATA[", "]]>", "<a>", "", "<!-- x -->", "]]> <!--", "--> <![CDATA["}
	for seed := range uint64(60) {
		prefixStatesMonkey(t, seed, pool, xmlNone, xmlLineEndState)
	}
}
