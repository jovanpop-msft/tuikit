package controls

import "testing"

func TestChoiceItemsMarksCurrentAndChoosesEachValue(t *testing.T) {
	var chosen string
	items := ChoiceItems([]string{"a", "b", "c"}, "b",
		func(s string) string { return "x" + s },
		func(s string) { chosen = s })
	want := []string{"xa", ChoiceMark + "xb", "xc"}
	if len(items) != len(want) {
		t.Fatalf("got %d items, want %d", len(items), len(want))
	}
	for i, it := range items {
		if it.Label != want[i] {
			t.Errorf("item %d label = %q, want %q", i, it.Label, want[i])
		}
		if !it.enabled() {
			t.Errorf("item %d disabled, want enabled", i)
		}
	}
	items[2].Action()
	if chosen != "c" {
		t.Errorf("chose %q, want c", chosen)
	}
}

func TestChoiceItemsNoCurrentMatchMarksNothing(t *testing.T) {
	items := ChoiceItems([]int{1, 2}, 9, func(i int) string { return "n" }, func(int) {})
	for i, it := range items {
		if it.Label != "n" {
			t.Errorf("item %d label = %q, want unmarked", i, it.Label)
		}
	}
}
