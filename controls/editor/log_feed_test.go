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
	text, ok := f.TakePending()
	if !ok || text != "line4\nline3\nline2" {
		t.Fatalf("TakePending = %q, %v; want the capped newest three", text, ok)
	}
	if f.Dirty() {
		t.Fatal("still dirty after TakePending")
	}
	if _, ok := f.TakePending(); ok {
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
