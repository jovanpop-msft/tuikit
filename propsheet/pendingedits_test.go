package propsheet

import (
	"strings"
	"testing"
)

// pendingTestRow is a page row for the PendingEdits tests: a name and one
// edited value.
type pendingTestRow struct {
	PendingState
	name, origName string
	value, orig    int
}

func newPendingTestEdits(same func(a, b string) bool, names ...string) *PendingEdits[*pendingTestRow] {
	loaded := make([]*pendingTestRow, len(names))
	for i, n := range names {
		loaded[i] = &pendingTestRow{name: n, origName: n}
	}
	return NewPendingEdits(loaded, same,
		func(e *pendingTestRow) string { return e.name },
		func(e *pendingTestRow) bool { return e.name != e.origName || e.value != e.orig },
		func(e *pendingTestRow) { e.name, e.value = e.origName, e.orig })
}

func pendingTestNames(rows []*pendingTestRow) string {
	out := make([]string, len(rows))
	for i, e := range rows {
		out[i] = e.name
	}
	return strings.Join(out, ",")
}

// TestPendingEditsLifecycle: an Add is new, a Remove forgets a new row and
// marks a loaded one, the grid lists what is not being removed, Revert puts the
// list back as loaded.
func TestPendingEditsLifecycle(t *testing.T) {
	p := newPendingTestEdits(nil, "a", "b", "c")
	if p.Dirty() {
		t.Fatal("dirty as loaded")
	}
	b := p.All()[1]
	p.Remove(b)
	p.Add(&pendingTestRow{name: "d"})
	d := p.All()[3]
	if !d.IsNew || !b.Removing || !p.Dirty() {
		t.Fatalf("after Remove b and Add d: d.IsNew=%v b.Removing=%v Dirty=%v", d.IsNew, b.Removing, p.Dirty())
	}
	if got := pendingTestNames(p.Visible()); got != "a,c,d" {
		t.Errorf("Visible = %s, want a,c,d", got)
	}
	if got := pendingTestNames(p.All()); got != "a,b,c,d" {
		t.Errorf("All = %s, want a,b,c,d", got)
	}

	// A new row removed is forgotten: nothing for Apply to write.
	p.Remove(d)
	if got := pendingTestNames(p.All()); got != "a,b,c" {
		t.Errorf("All after removing new d = %s, want a,b,c", got)
	}
	p.Restore(b)
	if p.Dirty() {
		t.Error("dirty after the only changes were undone")
	}

	p.All()[0].value = 7
	if !p.RowDirty(p.All()[0]) || !p.IsChanged(p.All()[0]) {
		t.Error("an edited loaded row is not dirty")
	}
	p.All()[2].name = "c2"
	p.Remove(p.All()[1])
	p.Add(&pendingTestRow{name: "e"})
	p.Swap(p.All()[0], p.All()[2])
	p.Revert()
	if got := pendingTestNames(p.All()); got != "a,b,c" {
		t.Errorf("All after Revert = %s, want a,b,c in loaded order", got)
	}
	if p.Dirty() || p.All()[0].value != 0 {
		t.Errorf("after Revert: Dirty=%v a.value=%d, want clean", p.Dirty(), p.All()[0].value)
	}
}

// TestPendingEditsNameChecks: Index and Listed see only the rows the grid
// lists; Taken sees a row being removed too, for a page that creates before it
// drops. All three compare with the host's equality.
func TestPendingEditsNameChecks(t *testing.T) {
	p := newPendingTestEdits(strings.EqualFold, "Ops", "Sales", "Archive")
	p.Remove(p.All()[1])
	for _, tt := range []struct {
		name         string
		index        int
		listed, take bool
	}{
		{"ops", 0, true, true},
		{"ARCHIVE", 1, true, true},
		{"sales", -1, false, true},
		{"other", -1, false, false},
	} {
		if got := p.Index(tt.name); got != tt.index {
			t.Errorf("Index(%q) = %d, want %d", tt.name, got, tt.index)
		}
		if got := p.Listed(tt.name); got != tt.listed {
			t.Errorf("Listed(%q) = %v, want %v", tt.name, got, tt.listed)
		}
		if got := p.Taken(tt.name); got != tt.take {
			t.Errorf("Taken(%q) = %v, want %v", tt.name, got, tt.take)
		}
	}
	if exact := newPendingTestEdits(nil, "Ops"); exact.Listed("ops") || !exact.Listed("Ops") {
		t.Error("nil equality is not byte-wise ==")
	}
}
