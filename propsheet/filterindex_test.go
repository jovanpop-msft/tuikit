package propsheet

import (
	"slices"
	"testing"

	"github.com/radix29/tuikit/core"
)

func newNameIndex() *FilterIndex[string] {
	return NewFilterIndex(func(s, term string) bool { return core.MatchesAny(term, s) })
}

func TestFilterIndexNarrowsAndMapsRows(t *testing.T) {
	f := newNameIndex()
	if f.Len() != 0 {
		t.Fatalf("new index Len = %d, want 0", f.Len())
	}
	f.SetItems([]string{"alice", "bob", "alfred"})
	if f.Len() != 3 {
		t.Fatalf("empty term Len = %d, want 3", f.Len())
	}
	f.Apply("AL")
	if got := f.Visible(); !slices.Equal(got, []string{"alice", "alfred"}) {
		t.Fatalf("Visible = %q", got)
	}
	if v, ok := f.At(1); !ok || v != "alfred" {
		t.Errorf("At(1) = %q, %v; want alfred, true", v, ok)
	}
	for _, row := range []int{-1, 2} {
		if v, ok := f.At(row); ok || v != "" {
			t.Errorf("At(%d) = %q, %v; want zero, false", row, v, ok)
		}
	}
	if f.Term() != "AL" {
		t.Errorf("Term = %q", f.Term())
	}
	rows := f.Rows(func(s string) []string { return []string{s, "x"} })
	if len(rows) != 2 || rows[1][0] != "alfred" || rows[1][1] != "x" {
		t.Errorf("Rows = %q", rows)
	}
}

// SetItems keeps the term: a list that grows or is swapped (another selection's
// detail) is shown through the filter already typed.
func TestFilterIndexSetItemsKeepsTerm(t *testing.T) {
	f := newNameIndex()
	f.Apply("b")
	f.SetItems([]string{"alice", "bob"})
	if got := f.Visible(); !slices.Equal(got, []string{"bob"}) {
		t.Fatalf("Visible = %q, want [bob]", got)
	}
	f.SetItems(nil)
	if f.Len() != 0 {
		t.Errorf("nil items Len = %d", f.Len())
	}
}

// A Visible slice taken before a re-filter is not rewritten by it.
func TestFilterIndexVisibleNotRewritten(t *testing.T) {
	f := newNameIndex()
	f.SetItems([]string{"alice", "bob"})
	before := f.Visible()
	f.Apply("bob")
	if !slices.Equal(before, []string{"alice", "bob"}) {
		t.Errorf("earlier Visible changed to %q", before)
	}
}

func TestFilterIndexFilterRow(t *testing.T) {
	f := newNameIndex()
	f.SetItems([]string{"alice", "bob"})
	var seen []int
	r := f.FilterRow("Filter", 20, func() { seen = append(seen, f.Len()) })
	r.Edit("zzz")
	r.Edit("")
	if !slices.Equal(seen, []int{0, 2}) {
		t.Errorf("onFilter saw Len %v, want [0 2]", seen)
	}
	if r.Dirty() {
		t.Error("a filter box reported dirty")
	}
	// nil onFilter is allowed.
	f.FilterRow("Filter", 20, nil).Edit("a")
	if f.Len() != 1 {
		t.Errorf("Len after nil-onFilter edit = %d, want 1", f.Len())
	}
}
