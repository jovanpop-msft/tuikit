package editor

import (
	"slices"
	"strings"
)

// LogFeed is the bookkeeping behind a newest-first, capped log shown in a
// read-only Editor (a status history, a key trace): the retained lines, and
// the lines recorded since the editor was last synced.
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
// with Max outside it — Sync is that pair for a single goroutine.
type LogFeed struct {
	lines   []string // newest first, at most max
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
func (f *LogFeed) Record(line string) {
	f.lines = f.prepend(f.lines, line)
	f.pending = f.prepend(f.pending, line)
}

func (f *LogFeed) prepend(s []string, line string) []string {
	s = append([]string{line}, s...)
	if f.max > 0 && len(s) > f.max {
		s = s[:f.max]
	}
	return s
}

// Len reports how many lines are retained.
func (f *LogFeed) Len() int { return len(f.lines) }

// Max reports the cap given to NewLogFeed, the keepMax to pass PrependLines.
func (f *LogFeed) Max() int { return f.max }

// Lines returns a copy of the retained lines, newest first.
func (f *LogFeed) Lines() []string { return slices.Clone(f.lines) }

// Dirty reports whether lines were recorded since the last sync.
func (f *LogFeed) Dirty() bool { return len(f.pending) > 0 }

// TakePending returns the lines recorded since the last sync, newest first and
// joined for PrependLines, and marks them synced. ok is false when there were
// none.
func (f *LogFeed) TakePending() (text string, ok bool) {
	if len(f.pending) == 0 {
		return "", false
	}
	text = strings.Join(f.pending, "\n")
	f.pending = nil
	return text, true
}

// TakeAll returns the whole log joined for SetText and marks everything
// synced.
func (f *LogFeed) TakeAll() string {
	f.pending = nil
	return strings.Join(f.lines, "\n")
}

// Sync prepends the lines recorded since the last sync to e, trimming e to
// Max, and keeps e's cursor, scroll and selection on the text they were on. A
// no-op when nothing is pending. Call it before each draw of e.
func (f *LogFeed) Sync(e *Editor) {
	if text, ok := f.TakePending(); ok {
		e.PrependLines(text, f.max)
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
	f.pending = nil
}
