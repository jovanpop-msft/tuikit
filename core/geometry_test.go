package core

import "testing"

func TestRectRightBottom(t *testing.T) {
	r := Rect{2, 3, 10, 5}
	if got := r.Right(); got != 12 {
		t.Errorf("Right() = %d, want 12", got)
	}
	if got := r.Bottom(); got != 8 {
		t.Errorf("Bottom() = %d, want 8", got)
	}
}

func TestRectInner(t *testing.T) {
	r := Rect{0, 0, 10, 10}
	got := r.Inner(1)
	want := Rect{1, 1, 8, 8}
	if got != want {
		t.Errorf("Inner(1) = %+v, want %+v", got, want)
	}
}

func TestRectContains(t *testing.T) {
	r := Rect{2, 2, 3, 3} // covers x in [2,5), y in [2,5)
	cases := []struct {
		x, y int
		want bool
	}{
		{2, 2, true},
		{4, 4, true},
		{5, 4, false}, // exclusive right edge
		{4, 5, false}, // exclusive bottom edge
		{1, 2, false},
		{2, 1, false},
	}
	for _, c := range cases {
		if got := r.Contains(c.x, c.y); got != c.want {
			t.Errorf("Contains(%d, %d) = %v, want %v", c.x, c.y, got, c.want)
		}
	}
}

func TestRectIsZero(t *testing.T) {
	if !(Rect{}).IsZero() {
		t.Error("zero-value Rect must report IsZero() == true")
	}
	// X/Y alone don't make a rect "zero" per the current implementation —
	// only W and H are considered.
	if !(Rect{1, 1, 0, 0}).IsZero() {
		t.Error("Rect{X:1,Y:1,W:0,H:0} should still report IsZero() == true")
	}
	if (Rect{0, 0, 5, 5}).IsZero() {
		t.Error("a rect with nonzero W/H must not report IsZero() == true")
	}
}

// Columns span the rect exactly, gutters are honoured, and the last column
// takes the remainder rather than leaving dead cells on the right.
func TestSplitColumnsSpansRect(t *testing.T) {
	r := Rect{X: 5, Y: 2, W: 100, H: 4}
	cols := SplitColumns(r, 3, 1, 24)
	if len(cols) != 3 {
		t.Fatalf("len = %d, want 3", len(cols))
	}
	if cols[0].X != r.X || cols[2].Right() != r.Right() {
		t.Errorf("columns %+v don't span %+v", cols, r)
	}
	for i := 1; i < len(cols); i++ {
		if gap := cols[i].X - cols[i-1].Right(); gap != 1 {
			t.Errorf("gap %d..%d = %d, want 1", i-1, i, gap)
		}
	}
	if cols[0].W != 32 || cols[1].W != 32 || cols[2].W != 34 {
		t.Errorf("widths = %d,%d,%d, want 32,32,34", cols[0].W, cols[1].W, cols[2].W)
	}
	for _, c := range cols {
		if c.Y != r.Y || c.H != r.H {
			t.Errorf("column %+v doesn't keep the rect's rows", c)
		}
	}
}

// Fewer columns come back when n won't fit at minW, counting the gutters;
// none at all when even one won't.
func TestSplitColumnsFitsWhatFits(t *testing.T) {
	cases := []struct {
		w, n, want int
	}{
		{2*24 + 1, 2, 2}, // exactly two minimum columns and a gutter
		{2 * 24, 2, 1},   // one short: the gutter counts
		{3*24 + 2, 3, 3},
		{3*24 + 1, 3, 2},
		{23, 2, 0},
		{500, 3, 3}, // never more than asked
	}
	for _, c := range cases {
		got := SplitColumns(Rect{W: c.w, H: 1}, c.n, 1, 24)
		if len(got) != c.want {
			t.Errorf("SplitColumns(W=%d, n=%d) = %d columns, want %d", c.w, c.n, len(got), c.want)
		}
		for _, col := range got {
			if col.W < 24 {
				t.Errorf("SplitColumns(W=%d, n=%d) column %+v below minW", c.w, c.n, col)
			}
		}
	}
	if got := SplitColumns(Rect{W: 100, H: 0}, 2, 1, 24); got != nil {
		t.Errorf("empty rect = %+v, want nil", got)
	}
	if got := SplitColumns(Rect{W: 100, H: 1}, 0, 1, 24); got != nil {
		t.Errorf("n=0 = %+v, want nil", got)
	}
}
