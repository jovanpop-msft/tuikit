package propsheet

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

// These pin the two things BindCellToggle is for: the guard that stops an
// activation outside the editable column or past the last row from writing to
// an edit, and the redraw that leaves the grid where the user is standing.

func toggleCell(on bool) string {
	if on {
		return "[x]"
	}
	return "[ ]"
}

// toggleRows renders the page's own state (one bool per row) as grid rows.
func toggleRows(on []bool) func() [][]string {
	return func() [][]string {
		rows := make([][]string, len(on))
		for i, v := range on {
			rows[i] = []string{toggleCell(v), string(rune('a' + i%26))}
		}
		return rows
	}
}

// TestCellToggleGuardsColumnAndRow: only column 0 toggles, and a row outside
// the slice is ignored. Without the guard a click on the blank space past the
// last row wrote to the final edit.
func TestCellToggleGuardsColumnAndRow(t *testing.T) {
	on := make([]bool, 3)
	headers := []string{"Linked", "Name"}
	grid := NewCellToggleGrid(headers, 0, func() int { return len(on) },
		func(row int) { on[row] = !on[row] }, toggleRows(on))

	grid.OnActivateCell(1, 1) // not the editable column
	grid.OnActivateCell(-1, 0)
	grid.OnActivateCell(3, 0) // past the last row
	for i, v := range on {
		if v {
			t.Errorf("edit %d was toggled by a guarded activation", i)
		}
	}

	grid.OnActivateCell(1, 0)
	if !on[1] || on[0] || on[2] {
		t.Errorf("after toggling row 1, on = %v, want only row 1 set", on)
	}
	if got := grid.Row(1)[0]; got != toggleCell(true) {
		t.Errorf("row 1 cell = %q, want %q — the grid did not re-render", got, toggleCell(true))
	}
}

// TestCellToggleReadsCountLate: count is read per activation, so a visible
// slice that shrank after wiring guards against its new length.
func TestCellToggleReadsCountLate(t *testing.T) {
	on := make([]bool, 3)
	visible := 3
	grid := NewCellToggleGrid([]string{"Linked", "Name"}, 0, func() int { return visible },
		func(row int) { on[row] = !on[row] }, toggleRows(on))
	visible = 1
	grid.OnActivateCell(2, 0)
	if on[2] {
		t.Error("an activation past the shrunken count toggled its row")
	}
}

// TestCellToggleKeepsCursorAndScroll: toggling a cell half way down a long grid
// must not move the grid. SetData would reset the cursor to 0,0 and the scroll
// to the top, which from inside a grid callback also makes every row but the
// first unreachable.
func TestCellToggleKeepsCursorAndScroll(t *testing.T) {
	on := make([]bool, 40)
	grid := NewCellToggleGrid([]string{"Linked", "Name"}, 0, func() int { return len(on) },
		func(row int) { on[row] = !on[row] }, toggleRows(on))
	grid.SetBounds(0, 0, 40, 10)

	for range 20 {
		grid.HandleKey(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone))
	}
	wantRow, wantCol := grid.SelectedCell()
	wantScroll := grid.ScrollRow()
	if wantRow == 0 || wantScroll == 0 {
		t.Fatalf("setup: cursor at row %d, scroll %d — the grid never moved, so this proves nothing", wantRow, wantScroll)
	}

	if !grid.HandleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone)) {
		t.Fatal("the grid refused Enter on a cell-cursor grid")
	}
	if !on[wantRow] {
		t.Fatalf("Enter did not toggle row %d", wantRow)
	}
	if row, col := grid.SelectedCell(); row != wantRow || col != wantCol {
		t.Errorf("cursor moved to %d,%d after a toggle, want %d,%d", row, col, wantRow, wantCol)
	}
	if got := grid.ScrollRow(); got != wantScroll {
		t.Errorf("scroll moved to %d after a toggle, want %d", got, wantScroll)
	}
}
