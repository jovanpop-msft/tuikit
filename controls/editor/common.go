package editor

import (
	"slices"
	"strings"
)

// ---------------------------------------------------------------------------
// Helpers shared across more than one file in this package
// ---------------------------------------------------------------------------

// prefixStates caches, for one version of one Document, the lexer state each line
// *begins* in — whether it opens inside a block comment, a CDATA section, and so
// on. XMLHighlighter keeps one, and LineStateAt keeps one per owner on the
// Document for highlighters outside this package; see XMLHighlighter for the
// measurements.
//
// A line's starting state depends on every line before it, so answering per line
// is O(document) and for a whole viewport O(H * document), on every event since
// Draw runs on all of them. One replay fills the whole array instead, reused
// until the text changes.
//
// Keyed on the *Document as well as its version: two Documents number versions
// independently from zero, so the pointer stops a highlighter handed a second
// document from answering with the first one's states.
type prefixStates[S comparable] struct {
	doc     *Document
	version uint64
	valid   bool
	states  []S
}

// at returns the state line idx begins in.
//
// zero is the state the first line starts in; step is the one-line transition —
// given a line and the state carried into it, the state carried out. step must be
// the function the highlighter's own per-line scan agrees with, or a line's colour
// depends on which cache generation drew it (flicker while scrolling).
func (c *prefixStates[S]) at(doc *Document, idx int, zero S, step func(line []rune, in S) S) S {
	switch {
	case !c.valid || c.doc != doc:
		c.replay(doc, 0, zero, step, doc.Len())
	case c.version == doc.Version():
		// Nothing has changed since the last replay.
	default:
		c.resume(doc, zero, step)
	}
	if idx < 0 || idx >= len(c.states) {
		return zero
	}
	return c.states[idx]
}

// resume brings states[] up to date through the document's splice log.
//
// changedSince names one span covering every mutation since the cached version:
// lines [row, row+oldN) became [row, row+newN), and every line outside it is the
// same text. states[] is spliced to match, so the entries after the span keep
// their (shifted) old values, and the replay starts at row. It may stop where a
// recomputed state matches the stored one only past the span, at row+newN or
// later: a line inside it is new text, so a state that matches on entering it
// says nothing about the lines below. Resuming from the mutation's first line
// alone (the old dirtyFrom rule) stopped there, and a same-count multi-line splice
// (the undo of Ctrl+/ over a selection, Move Lines) left every state below it
// stale.
func (c *prefixStates[S]) resume(doc *Document, zero S, step func(line []rune, in S) S) {
	row, oldN, newN, ok := doc.changedSince(c.version)
	if !ok || row+oldN > len(c.states) || len(c.states)-oldN+newN != doc.Len() {
		c.replay(doc, 0, zero, step, doc.Len())
		return
	}
	// The state line row begins in depends only on the lines above it, none of
	// which changed. Read it before the splice overwrites the slot.
	in := zero
	switch {
	case row == 0:
	case row < len(c.states):
		in = c.states[row]
	default:
		// The span starts past the old last line (an append): carry the
		// unchanged last line's state through it.
		in = step(doc.Line(row-1), c.states[row-1])
	}
	c.states = slices.Replace(c.states, row, row+oldN, make([]S, newN)...)
	c.replay(doc, row, in, step, row+newN)
}

// replay recomputes states[from:], with in the state line from begins in.
//
// From line convergeFrom on, the walk ends as soon as a recomputed state matches
// the stored one: the carried state has rejoined the previous scan, so no later
// line can differ. Typing in a large script then costs the lines whose meaning
// changed, not the whole document: outside a comment it converges on the next
// line, and only an edit that opens or closes one propagates further.
// convergeFrom must be past every line whose text changed since states[] was last
// right; doc.Len() disables the stop.
func (c *prefixStates[S]) replay(doc *Document, from int, in S, step func(line []rune, in S) S, convergeFrom int) {
	n := doc.Len()
	if cap(c.states) < n {
		grown := make([]S, n)
		copy(grown, c.states)
		c.states = grown
	}
	c.states = c.states[:n]

	for i := from; i < n; i++ {
		if i >= convergeFrom && c.states[i] == in {
			break
		}
		c.states[i] = in
		in = step(doc.Line(i), in)
	}
	c.doc, c.version, c.valid = doc, doc.Version(), true
}

// normalizeEOL folds every line ending to "\n": "\r\n" first, then any lone
// "\r" (a classic Mac file). Every entry point that splits text into lines —
// SetText/ReplaceText (splitText), Paste, blockPaste and LineBuffer.AppendText —
// goes through it, so they cannot drift apart again. Before it they each folded
// only "\r\n": a CR-only file loaded as one line with embedded CRs, which the
// host's batch splitter and formatter (which treat a lone CR as ending a "--"
// comment, as SQL Server does) then disagreed with.
func normalizeEOL(s string) string {
	if strings.IndexByte(s, '\r') < 0 {
		return s
	}
	return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
}
