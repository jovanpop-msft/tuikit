package propsheet

import (
	"slices"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/tuikittest"
)

const testUnset = "(unresolved owner)"

// A PreservingSelectRow still exposes its dropdown's popup through Form's
// core.Overlay lookup, by promotion from the embedded SelectRow.
var _ core.Overlay = (*PreservingSelectRow)(nil)

// The rule PreservingItems exists to enforce: the index it returns always
// points at the value that was asked for. A not-found 0 cannot promise that.
func TestPreservingItemsAlwaysPointsAtTheValue(t *testing.T) {
	base := []string{"alice", "bob", "carol"}
	for _, v := range []string{"alice", "carol", "mallory", testUnset} {
		items, i := PreservingItems(base, v)
		if got := items[i]; got != v {
			t.Errorf("PreservingItems(base, %q) selected %q, want %q", v, got, v)
		}
	}
}

func TestPreservingItemsDoesNotDisturbTheBaseList(t *testing.T) {
	base := []string{"alice", "bob"}
	items, _ := PreservingItems(base, "mallory")
	if !slices.Equal(base, []string{"alice", "bob"}) {
		t.Errorf("base list was mutated: %v", base)
	}
	// A widened list must keep every real option offered, not replace them.
	if !slices.Equal(items, []string{"alice", "bob", "mallory"}) {
		t.Errorf("widened list = %v, want the real options plus the value", items)
	}
}

// A value the list does have must not be duplicated onto the end: a second
// "alice" would make the same name selectable at two indices, and Dirty()
// compares indices.
func TestPreservingItemsReusesAPresentValue(t *testing.T) {
	base := []string{"alice", "bob"}
	items, i := PreservingItems(base, "bob")
	if len(items) != 2 || i != 1 {
		t.Errorf("PreservingItems(base, \"bob\") = %v, %d; want the original list and index 1", items, i)
	}
}

func TestSelectPreservingShowsAValueTheListLacks(t *testing.T) {
	// An object owned by a principal since dropped: the owner is not in the
	// list, and must still be what the row displays.
	row := SelectPreserving("Owner", []string{"alice", "bob"}, "ghost", testUnset)
	if got := row.Value(); got != "ghost" {
		t.Errorf("Value() = %q, want %q", got, "ghost")
	}
	if got := row.PreservedValue(); got != "ghost" {
		t.Errorf("PreservedValue() = %q, want %q", got, "ghost")
	}
	if row.Dirty() {
		t.Error("a freshly built row must not report itself dirty")
	}
}

func TestSelectPreservingStandsInForABlank(t *testing.T) {
	row := SelectPreserving("Owner", []string{"alice", "bob"}, "", testUnset)
	if got := row.Value(); got != testUnset {
		t.Errorf("Value() = %q, want %q", got, testUnset)
	}
	if got := row.PreservedValue(); got != "" {
		t.Errorf("PreservedValue() = %q, want \"\": the stand-in must not read back as a value", got)
	}
	if row.Dirty() {
		t.Error("a freshly built row must not report itself dirty")
	}
}

// ChangedTo is the write gate. These are the answers that must be "no": an
// untouched row, one still sitting on its stand-in, and a nil row.
func TestChangedToRefusesAnUntouchedRow(t *testing.T) {
	row := SelectPreserving("Owner", []string{"alice", "bob"}, "alice", testUnset)
	if v, ok := row.ChangedTo(); ok {
		t.Errorf("ChangedTo on an untouched row = %q, true; want no write", v)
	}
}

func TestChangedToRefusesANilRow(t *testing.T) {
	var row *PreservingSelectRow
	if v, ok := row.ChangedTo(); ok {
		t.Errorf("ChangedTo on a nil row = %q, true; want no write", v)
	}
}

func TestChangedToRefusesTheStandIn(t *testing.T) {
	// The row must be dirty *and* on the stand-in, which needs the stand-in
	// offered alongside a real value: the shape a host produces if it ever
	// lists one unconditionally. SetSelected/SetItems reset the dirty
	// baseline and would test nothing, so this drives real keys.
	row := SelectPreserving("Owner", []string{testUnset, "alice"}, "alice", testUnset)
	focusRow(row)

	row.HandleKey(tuikittest.Key(tcell.KeyEnter)) // open
	row.HandleKey(tuikittest.Key(tcell.KeyUp))    // "alice" -> the stand-in
	row.HandleKey(tuikittest.Key(tcell.KeyEnter)) // close

	if !row.Dirty() {
		t.Fatal("the row did not go dirty: test premise is wrong, the rest proves nothing")
	}
	if row.Value() != testUnset {
		t.Fatalf("row landed on %q, want the stand-in %q", row.Value(), testUnset)
	}
	if v, ok := row.ChangedTo(); ok {
		t.Errorf("ChangedTo returned %q for a row edited onto its stand-in; want no write", v)
	}
}

func TestChangedToReportsARealEdit(t *testing.T) {
	row := SelectPreserving("Owner", []string{"alice", "bob"}, "", testUnset)
	focusRow(row)

	// Up, not Down: the stand-in is appended, so it is the last item and
	// Down has nowhere to go.
	row.HandleKey(tuikittest.Key(tcell.KeyEnter)) // open
	row.HandleKey(tuikittest.Key(tcell.KeyUp))    // move off the stand-in
	row.HandleKey(tuikittest.Key(tcell.KeyEnter)) // close

	if !row.Dirty() {
		t.Fatal("the row did not go dirty: test premise is wrong, the rest proves nothing")
	}
	v, ok := row.ChangedTo()
	if !ok || v != "bob" {
		t.Errorf("ChangedTo = %q, %v; want \"bob\", true", v, ok)
	}
}

// SetChoices is the repoint form: it widens and stands in as the constructor
// does, and never leaves the row dirty against a value nobody picked.
func TestSetChoicesPreservesAndStaysClean(t *testing.T) {
	modes := []string{"AUTOMATIC", "MANUAL"}
	row := SelectPreserving("Seeding mode", modes, modes[0], "(unknown)")

	row.SetChoices(modes, "")
	if row.Value() != "(unknown)" || row.PreservedValue() != "" {
		t.Errorf("blank value shows %q, reads back %q; want the stand-in, \"\"", row.Value(), row.PreservedValue())
	}
	if row.Dirty() {
		t.Error("a row repointed at the stand-in reports dirty, so Apply would write it")
	}

	row.SetChoices(modes, "SOMETHING_NEW")
	if row.Value() != "SOMETHING_NEW" || row.PreservedValue() != "SOMETHING_NEW" {
		t.Errorf("unknown value shows %q, reads back %q; want it verbatim", row.Value(), row.PreservedValue())
	}

	row.SetChoices([]string{"EXTERNAL"}, "EXTERNAL")
	if got := row.Items(); !slices.Equal(got, []string{"EXTERNAL"}) {
		t.Errorf("items %v, want only the new list", got)
	}
	if row.Dirty() {
		t.Error("repointing left the row dirty")
	}
}

// focusRow gives row's dropdown keyboard focus. SelectRow.Draw is what hands
// focus down to the widget, and DropDown.HandleKey ignores every key while
// unfocused, so drawing once is unavoidable here.
func focusRow(row *PreservingSelectRow) {
	row.Layout(0, 0, 40)
	row.Draw(tuikittest.NewScreen(100, 40), true)
}
