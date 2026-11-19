package propsheet

import "github.com/radix29/tuikit/controls"

// BindCellToggle wires the "activate a cell in one column to change that row's
// value" idiom onto grid: bounds-check the activation, run change for that row
// (which mutates the page's own edit state), then re-render the grid in place.
//
// col is the one editable column; every other column and an out-of-range row
// (header area, blank space past the last row) is ignored, which stops a write
// to edits[len(edits)-1]. count is read at activation time, not captured,
// because a filter row can shrink the visible slice after wiring.
//
// The redraw is DataGrid.SetDataPreservingView, never SetData: these grids are
// navigated while toggled, and SetData would reset the cursor to 0,0 and the
// scroll to the top (from inside a grid callback, a GridRow then reports the
// arrow keys unhandled and every row but the first becomes unreachable). It is
// packaged because hand-rolled copies kept getting that wrong.
//
// It replaces grid.OnActivateCell.
func BindCellToggle(grid *controls.DataGrid, headers []string, col int,
	count func() int, change func(row int), rowsFor func() [][]string) {
	grid.OnActivateCell = func(row, c int) {
		if c != col || row < 0 || row >= count() {
			return
		}
		change(row)
		grid.SetDataPreservingView(headers, rowsFor())
	}
}

// NewCellToggleGrid is BindCellToggle for a page that builds the grid too: a
// cell-cursor DataGrid seeded from rowsFor, with column col editable.
//
// A page whose grid other closures reference before the toggle can be
// described builds the grid itself and calls BindCellToggle.
func NewCellToggleGrid(headers []string, col int,
	count func() int, change func(row int), rowsFor func() [][]string) *controls.DataGrid {
	grid := controls.NewDataGrid()
	grid.SetData(headers, rowsFor())
	grid.SetCellCursor(true)
	BindCellToggle(grid, headers, col, count, change, rowsFor)
	return grid
}
