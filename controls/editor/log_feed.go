package editor

import (
	"slices"
	"strings"
)

// LogFeed is the bookkeeping behind a newest-first, capped log shown in a
// read-only Editor (a status history, a key trace): the retained lines, and
// the lines recorded since the editor was last synced. A "line" is one record
// and may itself hold newlines (an error message often does).
//
// Sync prepends only those new lines with PrependLines, so a log that updates
// while someone is reading it keeps their cursor, scroll and selection. The
// obvious alternative, SetText with the whole log on every change, shipped
// twice in goSSMS and dropped the selection each time: in a status history the
// "Copied to clipboard" of the user's own Ctrl+C was itself a new line, and in
// a key trace the Ctrl+A that made the selection was, so the second Copy copied
// nothing. Rebuild is the SetText form, for a fresh open.
//
// Not safe for concurrent use (tuikit does no locking). A host recording from
// background goroutines guards the feed with its own mutex and must not hold
// it across the editor update: TakePending under the lock, then PrependLines
// with its keepLines outside it — Sync is that pair for a single goroutine.
//
// The cap counts records, but the editor holds document lines, so the trim
// handed to PrependLines is the document lines the retained records take, not
// Max: trimming to Max lines let one multi-line record push older records out
// of an open view (cutting the oldest mid-text) that Rebuild then brought back.
type LogFeed struct {
	lines   []string // newest first, at most max
	rows    []int    // document lines each of lines takes, parallel to lines; nil when uncapped
	pending []string // recorded since the last Sync/Rebuild, newest first, at most max
	max     int
}

// NewLogFeed returns an empty feed that keeps at most max lines (max <= 0
// keeps everything).
func NewLogFeed(max int) *LogFeed {
	return &LogFeed{max: max}
}

// Record prepends line to the log and to the lines awaiting the next sync,
// dropping the oldest beyond Max. Pending is capped too: more than Max new
// lines would all be trimmed off by PrependLines anyway.
//
// Line endings are normalized to LF first. Records are joined with "\n", so a
// record ending in a bare CR would fuse with the separator into one CRLF: the
// editor would get a line fewer than the records count, and keepLines would
// keep a line of an evicted record that Rebuild doesn't show.
func (f *LogFeed) Record(line string) {
	line = normalizeEOL(line)
	f.lines = prepend(f.lines, line, f.max)
	if f.max > 0 { // uncapped, keepLines is 0 and rows is never read
		f.rows = prepend(f.rows, docLines(line), f.max)
	}
	f.pending = prepend(f.pending, line, f.max)
}

func prepend[T any](s []T, v T, max int) []T {
	s = append([]T{v}, s...)
	if max > 0 && len(s) > max {
		s = s[:max]
	}
	return s
}

// docLines counts the document lines an Editor splits line into, line
// already LF-normalized (Record): each LF ends one; tab expansion adds none.
func docLines(line string) int {
	return strings.Count(line, "\n") + 1
}

// Len reports how many lines are retained.
func (f *LogFeed) Len() int { return len(f.lines) }

// Max reports the cap given to NewLogFeed, in records. It is not the keepMax
// to pass PrependLines when a record may span lines; TakePending gives that.
func (f *LogFeed) Max() int { return f.max }

// Lines returns a copy of the retained lines, newest first.
func (f *LogFeed) Lines() []string { return slices.Clone(f.lines) }

// Dirty reports whether lines were recorded since the last sync.
func (f *LogFeed) Dirty() bool { return len(f.pending) > 0 }

// TakePending returns the lines recorded since the last sync, newest first and
// joined for PrependLines, and marks them synced. keepLines is the keepMax to
// pass with them: the document lines the retained records take, or 0 (keep
// everything) for an uncapped feed. ok is false when there were none.
func (f *LogFeed) TakePending() (text string, keepLines int, ok bool) {
	if len(f.pending) == 0 {
		return "", 0, false
	}
	text = strings.Join(f.pending, "\n")
	f.pending = nil
	if f.max > 0 {
		for _, n := range f.rows {
			keepLines += n
		}
	}
	return text, keepLines, true
}

// TakeAll returns the whole log joined for SetText and marks everything
// synced.
func (f *LogFeed) TakeAll() string {
	f.pending = nil
	return strings.Join(f.lines, "\n")
}

// Sync prepends the lines recorded since the last sync to e, trimming e to
// the retained records, and keeps e's cursor, scroll and selection on the text they were on. A
// no-op when nothing is pending. Call it before each draw of e.
func (f *LogFeed) Sync(e *Editor) {
	if text, keep, ok := f.TakePending(); ok {
		e.PrependLines(text, keep)
	}
}

// Rebuild replaces e's text with the whole log, resetting cursor, scroll and
// selection: for opening the view afresh, never for an update while it is
// being read (see LogFeed).
func (f *LogFeed) Rebuild(e *Editor) {
	e.SetText(f.TakeAll())
}

// Reset empties the log. An editor showing it is not touched; clear it with
// SetText("") (or Rebuild) — a Sync would keep the old text below the new.
func (f *LogFeed) Reset() {
	f.lines = nil
	f.rows = nil
	f.pending = nil
}
