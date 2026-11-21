package editor

import (
	"slices"
	"strconv"
	"testing"
)

func TestLogFeedRecordIsNewestFirstAndCapped(t *testing.T) {
	f := NewLogFeed(3)
	for i := range 5 {
		f.Record("line" + strconv.Itoa(i))
	}
	if got, want := f.Lines(), []string{"line4", "line3", "line2"}; !slices.Equal(got, want) {
		t.Fatalf("Lines = %q, want %q", got, want)
	}
	if f.Len() != 3 {
		t.Fatalf("Len = %d, want 3", f.Len())
	}
	text, _, ok := f.TakePending()
	if !ok || text != "line4\nline3\nline2" {
		t.Fatalf("TakePending = %q, %v; want the capped newest three", text, ok)
	}
	if f.Dirty() {
		t.Fatal("still dirty after TakePending")
	}
	if _, _, ok := f.TakePending(); ok {
		t.Fatal("TakePending reported lines a second time")
	}
}

func TestLogFeedUncapped(t *testing.T) {
	f := NewLogFeed(0)
	for i := range 10 {
		f.Record(strconv.Itoa(i))
	}
	if f.Len() != 10 {
		t.Fatalf("Len = %d, want 10 with no cap", f.Len())
	}
}

// The reason LogFeed exists: a line recorded while the log is being read must
// not drop the reader's selection (the second-Copy-copies-nothing bug).
func TestLogFeedSyncKeepsTheSelection(t *testing.T) {
	f := NewLogFeed(10)
	e := NewEditor(nil)
	e.SetReadOnly(true)
	f.Record("first")
	f.Record("second")
	f.Rebuild(e)
	e.SelectAll()
	before := e.SelectedText()

	f.Record("third")
	f.Sync(e)

	if !e.HasSelection() || e.SelectedText() != before {
		t.Fatalf("selection after Sync = %q (has %v), want %q kept", e.SelectedText(), e.HasSelection(), before)
	}
	if got := e.Text(); got != "third\nsecond\nfirst" {
		t.Fatalf("Text = %q, want the new line on top", got)
	}
	if f.Dirty() {
		t.Fatal("dirty after Sync")
	}
}

func TestLogFeedSyncTrimsTheEditorToMax(t *testing.T) {
	f := NewLogFeed(2)
	e := NewEditor(nil)
	f.Record("a")
	f.Record("b")
	f.Sync(e)
	f.Record("c")
	f.Sync(e)
	if got := e.Text(); got != "c\nb" {
		t.Fatalf("Text = %q, want the editor trimmed to Max", got)
	}
}

func TestLogFeedRebuildClearsPending(t *testing.T) {
	f := NewLogFeed(10)
	e := NewEditor(nil)
	f.Record("a")
	f.Record("b")
	f.Rebuild(e)
	if f.Dirty() {
		t.Fatal("dirty after Rebuild")
	}
	f.Sync(e) // must not prepend a and b a second time
	if got := e.Text(); got != "b\na" {
		t.Fatalf("Text = %q, want each line once", got)
	}
}

func TestLogFeedReset(t *testing.T) {
	f := NewLogFeed(10)
	f.Record("a")
	f.Reset()
	if f.Len() != 0 || f.Dirty() {
		t.Fatalf("after Reset: Len %d, Dirty %v; want empty", f.Len(), f.Dirty())
	}
}

// The cap counts records, and the editor's trim must follow it: trimmed to Max
// document lines, one multi-line record pushed the older records out of an
// open view (and Rebuild brought them back), so Sync and Rebuild disagreed.
func TestLogFeedSyncMatchesRebuildWithMultiLineRecords(t *testing.T) {
	cases := []struct {
		name    string
		max     int
		records []string
	}{
		{"multi-line newest", 3, []string{"one", "two", "err A\nerr B\nerr C", "four", "five"}},
		{"one record longer than the rest", 2, []string{"a", "1\n2\n3\n4\n5\n6", "b", "c"}},
		{"CRLF and lone CR", 3, []string{"x", "p\r\nq", "r\rs", "y", "t\r\nu\r\nv"}},
		{"record ending in CR", 2, []string{"old", "err\r", "new", "x\r\r"}},
		{"uncapped", 0, []string{"one", "a\nb", "two", "c\nd\ne"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewLogFeed(tc.max)
			synced, rebuilt := NewEditor(nil), NewEditor(nil)
			for i, r := range tc.records {
				f.Record(r)
				f.Sync(synced)
				g := NewLogFeed(tc.max)
				for _, r := range tc.records[:i+1] {
					g.Record(r)
				}
				g.Rebuild(rebuilt)
				if synced.Text() != rebuilt.Text() {
					t.Fatalf("after record %d: Sync = %q, Rebuild = %q", i, synced.Text(), rebuilt.Text())
				}
			}
		})
	}
}

func TestLogFeedTakePendingKeepLines(t *testing.T) {
	f := NewLogFeed(2)
	f.Record("a\nb")
	f.Record("c")
	f.Record("d\r\ne\nf")
	if text, keep, ok := f.TakePending(); !ok || keep != 4 || text != "d\ne\nf\nc" {
		t.Fatalf("TakePending = %q, %d, %v; want the two newest records and 4 lines", text, keep, ok)
	}
	g := NewLogFeed(0)
	g.Record("a\nb")
	if _, keep, _ := g.TakePending(); keep != 0 {
		t.Fatalf("uncapped keepLines = %d, want 0 (keep everything)", keep)
	}
	if g.rows != nil {
		t.Fatalf("uncapped feed keeps line counts %v it never reads", g.rows)
	}
}
