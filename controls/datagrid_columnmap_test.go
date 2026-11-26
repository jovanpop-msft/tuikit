package controls

import (
	"slices"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

// columnMapGrid is a 5-row read-only cell-cursor grid over cols, cells named
// after their column, with column dragW dragged to 40 wide and right-aligned,
// sorted on it, and a block from (1, anchor) to (3, cursor).
func columnMapGrid(cols []string, dragW, anchor, cursor int) *DataGrid {
	g := newTestDataGrid()
	g.SetBounds(0, 0, 200, 10)
	g.SetSource(cols, columnRows(cols))
	g.SetCellCursor(true)
	g.SetColumnWidth(dragW, 40)
	aligns := make([]ColumnAlign, len(cols))
	aligns[dragW] = AlignRight
	g.SetColumnAlignments(aligns)
	g.SetSortIndicator(dragW, true)
	g.SetSelectedCell(1, anchor)
	key := tcell.KeyRight
	if cursor < anchor {
		key = tcell.KeyLeft
	}
	for range max(cursor-anchor, anchor-cursor) {
		g.HandleKey(tuikittest.Key(key, tcell.ModShift))
	}
	for range 2 {
		g.HandleKey(tuikittest.Key(tcell.KeyDown, tcell.ModShift))
	}
	return g
}

func columnRows(cols []string) SliceRowSource {
	rows := make(SliceRowSource, 5)
	for i := range rows {
		rows[i] = slices.Clone(cols)
	}
	return rows
}

// wantColumnState checks which column holds the dragged width, the right
// alignment and the sort arrow (-1: none does), and the block's column span.
func wantColumnState(t *testing.T, g *DataGrid, what string, dragged, c0, c1 int) {
	t.Helper()
	for i := range g.columns {
		want := 0
		if i == dragged {
			want = 40
		}
		if got := g.ColumnWidthOverrides()[i]; got != want {
			t.Errorf("%s: column %d (%s) override = %d, want %d", what, i, g.columns[i], got, want)
		}
		wantAlign := AlignLeft
		if i == dragged {
			wantAlign = AlignRight
		}
		if got := g.alignAt(i); got != wantAlign {
			t.Errorf("%s: column %d (%s) align = %v, want %v", what, i, g.columns[i], got, wantAlign)
		}
	}
	if g.sortCol != dragged {
		t.Errorf("%s: sortCol = %d, want %d", what, g.sortCol, dragged)
	}
	if _, gc0, _, gc1 := g.SelectionBounds(); gc0 != c0 || gc1 != c1 {
		t.Errorf("%s: block columns %d..%d, want %d..%d", what, gc0, gc1, c0, c1)
	}
}

func (g *DataGrid) alignAt(i int) ColumnAlign {
	if i < len(g.aligns) {
		return g.aligns[i]
	}
	return AlignLeft
}

// A field arriving ahead of an action column (the XEvent viewer's case) leaves
// the action's width, alignment, sort arrow and the cursor on the action.
func TestPreservingViewFollowsColumnInsertedMidList(t *testing.T) {
	old := []string{"name", "timestamp", "f1", "act1"}
	cur := []string{"name", "timestamp", "f1", "f2", "act1"}
	g := columnMapGrid(old, 3, 2, 3) // block f1..act1, cursor on act1
	g.SetSourceKeepingSelection(cur, columnRows(cur))
	if _, col := g.SelectedCell(); col != 4 {
		t.Errorf("cursor column = %d, want 4 (act1)", col)
	}
	// f2 fell inside the block, so it collapses onto the cursor.
	wantColumnState(t, g, "insert inside block", 4, 4, 4)

	g = columnMapGrid(old, 3, 3, 1) // anchor on act1, cursor on timestamp
	g.SetSourceKeepingSelection(cur, columnRows(cur))
	wantColumnState(t, g, "insert inside reversed block", 4, 1, 1)

	g = columnMapGrid(old, 3, 0, 1) // block name..timestamp, the insert outside it
	g.SetSourceKeepingSelection(cur, columnRows(cur))
	wantColumnState(t, g, "insert outside block", 4, 0, 1)
	if r0, _, r1, _ := g.SelectionBounds(); r0 != 1 || r1 != 3 {
		t.Errorf("block rows %d..%d, want 1..3", r0, r1)
	}
}

// A hidden column takes its state with it; the cursor on it moves to the
// nearest surviving column to its left, and a block that spanned it collapses.
func TestPreservingViewFollowsColumnRemovedMidList(t *testing.T) {
	old := []string{"name", "timestamp", "f1", "act1"}
	cur := []string{"name", "timestamp", "act1"}
	g := columnMapGrid(old, 3, 2, 3) // block f1..act1
	g.SetSourceKeepingSelection(cur, columnRows(cur))
	wantColumnState(t, g, "remove inside block", 2, 2, 2)

	g = columnMapGrid(old, 2, 2, 2) // f1 dragged, cursor on it
	g.SetSourceKeepingSelection(cur, columnRows(cur))
	if _, col := g.SelectedCell(); col != 1 {
		t.Errorf("cursor column = %d, want 1 (timestamp, left of the gone f1)", col)
	}
	wantColumnState(t, g, "dragged column removed", -1, 1, 1)
}

// A renamed column is a different one: nothing carries over to it.
func TestPreservingViewRenamedColumnStartsAtDefaults(t *testing.T) {
	old := []string{"a", "b", "c"}
	cur := []string{"a", "B", "c"}
	g := columnMapGrid(old, 1, 1, 2)
	g.SetSourcePreservingView(cur, columnRows(cur))
	if _, col := g.SelectedCell(); col != 2 {
		t.Errorf("cursor column = %d, want 2", col)
	}
	wantColumnState(t, g, "renamed", -1, 2, 2)
}

// A repeated header pairs its n-th occurrence with the n-th: the second "x"
// keeps its state when a column is inserted between the two.
func TestPreservingViewDuplicateHeaders(t *testing.T) {
	old := []string{"x", "y", "x"}
	cur := []string{"x", "y", "z", "x"}
	g := columnMapGrid(old, 2, 0, 0)
	g.SetSourceKeepingSelection(cur, columnRows(cur))
	wantColumnState(t, g, "second x", 3, 0, 0)

	g = columnMapGrid(old, 0, 2, 2)
	g.SetSourceKeepingSelection(cur, columnRows(cur))
	if _, col := g.SelectedCell(); col != 3 {
		t.Errorf("cursor column = %d, want 3 (second x)", col)
	}
	wantColumnState(t, g, "first x", 0, 3, 3)

	// One fewer "x": the first keeps its state, the second is gone.
	cur = []string{"x", "y"}
	g = columnMapGrid(old, 2, 2, 2)
	g.SetSourceKeepingSelection(cur, columnRows(cur))
	wantColumnState(t, g, "second x gone", -1, 1, 1)
}

func TestColumnMap(t *testing.T) {
	for _, tc := range []struct {
		old, cur []string
		want     []int
	}{
		{[]string{"a", "b"}, []string{"a", "b"}, []int{0, 1}},
		{[]string{"a", "b"}, []string{"b", "a"}, []int{1, 0}},
		{[]string{"a", "a", "b"}, []string{"b", "a"}, []int{1, -1, 0}},
		{nil, []string{"a"}, []int{}},
		{[]string{"a"}, nil, []int{-1}},
	} {
		if got := columnMap(tc.old, tc.cur); !slices.Equal(got, tc.want) {
			t.Errorf("columnMap(%v, %v) = %v, want %v", tc.old, tc.cur, got, tc.want)
		}
	}
}
