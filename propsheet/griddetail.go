package propsheet

import "github.com/radix29/tuikit/controls"

// BindGridDetail connects detail rows to a DataGrid's selection: the
// grid-plus-detail shape of a page whose rows each carry settings edited below
// the grid. ListEditor is built on it; a page with no Add/Remove, or rows that
// are not items of one type, calls it directly.
//
// Moving the grid cursor runs commit (fold the detail rows into the row left),
// then show (load the row reached, read from grid.SelectedRow), then redraws the
// grid from rows so the committed edit shows. The redraw is
// DataGrid.SetDataPreservingView, never SetData: SetData would put the cursor
// back on row 0 from inside OnSelectRow, GridRow (which diffs the selection
// around the key) would report the arrow not handled, Form would move focus out
// of the grid, and only the first row would be reachable.
//
// show runs once before BindGridDetail returns, for the initial selection. The
// returned reload redraws the grid and runs show without committing first: what
// a RevertFn calls after putting the rows back, since a commit there would write
// the detail rows' pre-revert values straight back over what Revert undid.
//
// It replaces grid.OnSelectRow.
func BindGridDetail(grid *controls.DataGrid, headers []string, rows func() [][]string, commit, show func()) (reload func()) {
	redraw := func() { grid.SetDataPreservingView(headers, rows()) }
	grid.OnSelectRow = func(int) {
		commit()
		show()
		redraw()
	}
	show()
	return func() {
		redraw()
		show()
	}
}
