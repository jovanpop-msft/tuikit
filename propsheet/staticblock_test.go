package propsheet

import "testing"

// TestStaticBlockSetFillsAndClears: Set fills rows in order, a short list
// clears the rest, and no values clears every row, so no row keeps describing
// the item the selection left.
func TestStaticBlockSetFillsAndClears(t *testing.T) {
	a, b, c := Static("A", ""), Static("B", ""), Static("C", "")
	block := NewStaticBlock(a, b, c)

	block.Set("1", "2", "3")
	if a.Value() != "1" || b.Value() != "2" || c.Value() != "3" {
		t.Fatalf("after Set(1,2,3): %q %q %q", a.Value(), b.Value(), c.Value())
	}
	block.Set("x")
	if a.Value() != "x" || b.Value() != "" || c.Value() != "" {
		t.Errorf("after Set(x): %q %q %q, want x and two blanks", a.Value(), b.Value(), c.Value())
	}
	block.Set("1", "2", "3")
	block.Set()
	for i, r := range []*StaticRow{a, b, c} {
		if r.Value() != "" {
			t.Errorf("row %d = %q after Set(), want blank", i, r.Value())
		}
	}
}
