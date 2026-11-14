package controls

import (
	"slices"
	"testing"

	"github.com/radix29/tuikit/controls/editor"
)

func menuLabels(items []MenuItem) []string {
	var out []string
	for _, it := range items {
		if it.Divider {
			out = append(out, "-")
			continue
		}
		out = append(out, it.Label)
	}
	return out
}

func itemEnabled(t *testing.T, items []MenuItem, label string) bool {
	t.Helper()
	for _, it := range items {
		if it.Label == label {
			return it.enabled()
		}
	}
	t.Fatalf("no %q item in %v", label, menuLabels(items))
	return false
}

var allActions = ClipboardActions{Cut: func() {}, Copy: func() {}, Paste: func() {}, SelectAll: func() {}}

// TestClipboardMenuOffersOnlyWhatTheTargetTakes: an editable field gets the
// full menu, greyed per selection; a read-only one gets Copy and Select All,
// never a Cut or Paste it would silently refuse.
func TestClipboardMenuOffersOnlyWhatTheTargetTakes(t *testing.T) {
	ed := editor.NewEditor(nil)
	ed.SetText("select 1")
	items := ClipboardMenuItems(ed, allActions)
	if got, want := menuLabels(items), []string{"Cut", "Copy", "Paste", "-", "Select All"}; !slices.Equal(got, want) {
		t.Fatalf("editable menu = %v, want %v", got, want)
	}
	if itemEnabled(t, items, "Cut") || itemEnabled(t, items, "Copy") {
		t.Error("Cut/Copy enabled with no selection")
	}
	if !itemEnabled(t, items, "Paste") {
		t.Error("Paste disabled on an editable field")
	}
	ed.SelectAll()
	if !itemEnabled(t, items, "Cut") || !itemEnabled(t, items, "Copy") {
		t.Error("Cut/Copy still disabled after the selection the open menu follows")
	}

	ed.SetReadOnly(true)
	items = ClipboardMenuItems(ed, allActions)
	if got, want := menuLabels(items), []string{"Copy", "-", "Select All"}; !slices.Equal(got, want) {
		t.Fatalf("read-only menu = %v, want %v", got, want)
	}

	// A read-only value with nothing to select (a flat property row) gets Copy
	// alone: a Select All doing nothing is not offered.
	vv := &ValueViewer{}
	items = ClipboardMenuItems(vv, allActions)
	if got, want := menuLabels(items), []string{"Copy"}; !slices.Equal(got, want) {
		t.Errorf("closed viewer menu = %v, want %v", got, want)
	}

	items = ClipboardMenuItems(ed, ClipboardActions{Copy: func() {}})
	if got, want := menuLabels(items), []string{"Copy"}; !slices.Equal(got, want) {
		t.Errorf("menu with only Copy wired = %v, want %v", got, want)
	}
}
