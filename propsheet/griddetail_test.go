package propsheet

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/controls"
)

// BindGridDetail redraws from inside OnSelectRow, and SetData resets the
// selection to 0,0; without SetDataPreservingView the cursor snapped back every
// arrow key. These pin the restore, the commit/show order and reload.

// gridDetailHarness is three rows with a one-field detail editor.
type gridDetailHarness struct {
	grid    *controls.DataGrid
	values  []string // the edits behind the grid
	field   string   // the detail editor's single field
	sel     int
	commits int
	reload  func()
}

func newGridDetailHarness() *gridDetailHarness {
	h := &gridDetailHarness{
		grid:   controls.NewDataGrid(),
		values: []string{"a", "b", "c"},
		sel:    -1,
	}
	h.grid.SetCellCursor(true)
	headers := []string{"Name", "Value"}
	rows := func() [][]string {
		out := make([][]string, len(h.values))
		for i, v := range h.values {
			out[i] = []string{string(rune('A' + i)), v}
		}
		return out
	}
	h.grid.SetData(headers, rows())
	commit := func() {
		h.commits++
		if h.sel >= 0 && h.sel < len(h.values) {
			h.values[h.sel] = h.field
		}
	}
	show := func() {
		h.sel = h.grid.SelectedRow()
		h.field = ""
		if h.sel >= 0 && h.sel < len(h.values) {
			h.field = h.values[h.sel]
		}
	}
	h.reload = BindGridDetail(h.grid, headers, rows, commit, show)
	return h
}

func (h *gridDetailHarness) down(t *testing.T) {
	t.Helper()
	if !h.grid.HandleKey(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone)) {
		t.Fatal("DataGrid refused Down")
	}
}

func TestBindGridDetailKeepsSelectionAcrossItsOwnRedraw(t *testing.T) {
	h := newGridDetailHarness()
	if h.sel != 0 || h.field != "a" {
		t.Fatalf("initial show: row %d field %q, want row 0 field %q", h.sel, h.field, "a")
	}
	for want := 1; want <= 2; want++ {
		h.down(t)
		if got := h.grid.SelectedRow(); got != want {
			t.Fatalf("after %d Down keypresses the grid selects row %d, want %d — the redraw reset it", want, got, want)
		}
		if h.sel != want {
			t.Fatalf("detail is showing row %d, want %d", h.sel, want)
		}
	}
}

func TestBindGridDetailCommitsBeforeShowingTheNewRow(t *testing.T) {
	h := newGridDetailHarness()
	h.field = "edited" // typed while row 0 was selected
	h.down(t)

	if h.values[0] != "edited" {
		t.Fatalf("row 0 kept %q, want the edit committed on the way out", h.values[0])
	}
	if h.field != "b" {
		t.Fatalf("detail shows %q after moving to row 1, want %q", h.field, "b")
	}
	if got := h.grid.Row(0)[1]; got != "edited" {
		t.Fatalf("grid cell (0,1) reads %q, want the committed edit redrawn", got)
	}
}

func TestBindGridDetailReloadShowsWithoutCommitting(t *testing.T) {
	h := newGridDetailHarness()
	h.down(t)
	commits := h.commits
	h.field = "typed, never committed"
	h.values[1] = "reverted" // what a RevertFn does behind the grid
	h.reload()

	if h.commits != commits {
		t.Fatalf("reload committed %d times, want none: it would write the pre-revert detail back", h.commits-commits)
	}
	if got := h.grid.SelectedRow(); got != 1 {
		t.Fatalf("reload left the grid on row %d, want 1", got)
	}
	if h.field != "reverted" || h.grid.Row(1)[1] != "reverted" {
		t.Fatalf("reload: detail %q, cell %q; want both %q", h.field, h.grid.Row(1)[1], "reverted")
	}
}
