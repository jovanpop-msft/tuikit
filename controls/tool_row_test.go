package controls

import (
	"slices"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
	"github.com/radix29/tuikit/tuikittest"
)

func newTestToolRow(labels ...string) *ToolRow {
	tr := &ToolRow{}
	for _, l := range labels {
		tr.Cells = append(tr.Cells, ToolCell{Label: l})
	}
	return tr
}

// TestToolRowPlacesEveryCellThatFits: cells get one space of padding either
// side and one gap between, after a one-column margin; no More when nothing is
// hidden, and the returned end is past the last cell plus a gap.
func TestToolRowPlacesEveryCellThatFits(t *testing.T) {
	tr := newTestToolRow("Refresh", "Pause")
	end := tr.Layout(core.Rect{X: 10, Y: 3, W: 40, H: 1}, "")
	if want := (core.Rect{X: 11, Y: 3, W: 9, H: 1}); tr.Cells[0].Rect != want {
		t.Errorf("cell 0 = %+v, want %+v", tr.Cells[0].Rect, want)
	}
	if want := (core.Rect{X: 21, Y: 3, W: 7, H: 1}); tr.Cells[1].Rect != want {
		t.Errorf("cell 1 = %+v, want %+v", tr.Cells[1].Rect, want)
	}
	if end != 29 || len(tr.Hidden) != 0 || !tr.More.Rect.IsZero() {
		t.Errorf("end %d, hidden %v, More %+v; want 29, none, zero", end, tr.Hidden, tr.More.Rect)
	}
}

// TestToolRowMenuAnchorDropsBelowTheCellOrTheFallback: a shown cell and More
// anchor one row under their left edge; a hidden cell, an index past the row,
// and More when nothing overflowed all fall back to the host's rect, so a menu
// never opens at the screen origin.
func TestToolRowMenuAnchorDropsBelowTheCellOrTheFallback(t *testing.T) {
	tr := newTestToolRow("Refresh", "A much longer label", "B")
	tr.Layout(core.Rect{X: 5, Y: 2, W: 30, H: 1}, "")
	if !slices.Equal(tr.Hidden, []int{1, 2}) || tr.More.Rect.IsZero() {
		t.Fatalf("layout: hidden %v, More %+v; want [1 2] behind More", tr.Hidden, tr.More.Rect)
	}
	fb := core.Rect{X: 40, Y: 9, W: 20, H: 10}
	for _, c := range []struct {
		name   string
		i      int
		wx, wy int
	}{
		{"shown cell", 0, tr.Cells[0].Rect.X, 3},
		{"More", -1, tr.More.Rect.X, 3},
		{"hidden cell", 1, 40, 10},
		{"past the row", 7, 40, 10},
	} {
		if x, y := tr.MenuAnchor(c.i, fb); x != c.wx || y != c.wy {
			t.Errorf("%s: anchor (%d,%d), want (%d,%d)", c.name, x, y, c.wx, c.wy)
		}
	}

	tr = newTestToolRow("A")
	tr.Layout(core.Rect{X: 5, Y: 2, W: 30, H: 1}, "")
	if x, y := tr.MenuAnchor(-1, fb); x != 40 || y != 10 {
		t.Errorf("More with nothing hidden: anchor (%d,%d), want the fallback (40,10)", x, y)
	}
}

// TestToolRowCollapsesTheTailIntoMore: what does not fit is the row's tail,
// and More is placed inside the row after the cells that stayed.
func TestToolRowCollapsesTheTailIntoMore(t *testing.T) {
	tr := newTestToolRow("Refresh", "A much longer label", "B", "C")
	r := core.Rect{X: 0, Y: 0, W: 30, H: 1}
	tr.Layout(r, "")
	if !slices.Equal(tr.Hidden, []int{1, 2, 3}) {
		t.Fatalf("hidden = %v, want the tail [1 2 3] — never a later, shorter cell squeezed in", tr.Hidden)
	}
	if tr.More.Label != Strings.More || tr.More.Rect.X != tr.Cells[0].Rect.Right()+ToolRowGap {
		t.Errorf("More = %+v, want it right after Refresh", tr.More)
	}
	if tr.More.Rect.Right() > r.Right() {
		t.Errorf("More runs past the row: %+v", tr.More.Rect)
	}
	for _, i := range tr.Hidden {
		if !tr.Cells[i].Rect.IsZero() {
			t.Errorf("hidden cell %d still has a rect", i)
		}
	}
}

// TestToolRowMoreTakesThePrefixsPlace on a row too narrow for both, and
// PrefixShown says so.
func TestToolRowMoreTakesThePrefixsPlace(t *testing.T) {
	tr := newTestToolRow("1 s", "5 s")
	r := core.Rect{X: 0, Y: 0, W: 12, H: 1}
	tr.Layout(r, "Refresh rate:")
	if tr.More.Rect.X != 1 || tr.PrefixShown(r) {
		t.Errorf("More at %d, prefix shown %v; want More at 1 in the prefix's place", tr.More.Rect.X, tr.PrefixShown(r))
	}
	tr.Layout(core.Rect{W: 60, H: 1}, "Refresh rate:")
	if !tr.PrefixShown(core.Rect{W: 60, H: 1}) {
		t.Error("a row with room for everything hid the prefix")
	}
}

// TestToolRowNoOverflowDropsWithoutMore: the drop-only row hides the same
// tail but puts no More cell up.
func TestToolRowNoOverflowDropsWithoutMore(t *testing.T) {
	tr := newTestToolRow("A: statement 1/2", "B: statement 1/2")
	tr.NoOverflow = true
	tr.Layout(core.Rect{W: 25, H: 1}, "")
	if !slices.Equal(tr.Hidden, []int{1}) || !tr.More.Rect.IsZero() {
		t.Errorf("hidden %v, More %+v; want [1] and no More", tr.Hidden, tr.More.Rect)
	}
}

// TestToolRowWithNoRowHidesEverything: a host with no room for the row passes
// a zero rect; nothing is placed and there is nothing to collapse into.
func TestToolRowWithNoRowHidesEverything(t *testing.T) {
	tr := newTestToolRow("Refresh")
	tr.Layout(core.Rect{W: 40, H: 1}, "")
	tr.Layout(core.Rect{}, "")
	if !tr.Cells[0].Rect.IsZero() || !tr.More.Rect.IsZero() || len(tr.Hidden) != 0 {
		t.Errorf("cell %+v, More %+v, hidden %v; want all empty", tr.Cells[0].Rect, tr.More.Rect, tr.Hidden)
	}
	if tr.CellAt(1, 0) != -1 {
		t.Error("a cell that is not on screen was hit")
	}
}

// TestToolRowOverflowItemsCarryTheCellsGate: with nil predicates the menu
// reads each cell's Disabled and Reason; the note is set only for a cell
// withheld now, and a selected cell keeps its bullet.
func TestToolRowOverflowItemsCarryTheCellsGate(t *testing.T) {
	ran := -1
	tr := &ToolRow{Cells: []ToolCell{
		{Label: "Refresh"},
		{Label: "5 s", Selected: true},
		{Label: "Install", Disabled: true, Reason: "Requires CONTROL SERVER"},
	}, Hidden: []int{1, 2}}
	items := tr.OverflowItems(nil, nil, func(i int) { ran = i })
	if len(items) != 2 || items[0].Label != "• 5 s" || items[1].Label != "Install" {
		t.Fatalf("items = %+v", items)
	}
	if !items[0].enabled() || items[0].Note != "" {
		t.Errorf("an enabled cell's item: enabled %v, note %q", items[0].enabled(), items[0].Note)
	}
	if items[1].enabled() || items[1].Note != "Requires CONTROL SERVER" {
		t.Errorf("a disabled cell's item: enabled %v, note %q", items[1].enabled(), items[1].Note)
	}
	items[0].Action()
	if ran != 1 {
		t.Errorf("run got %d, want the cell's index 1", ran)
	}
}

// TestToolRowRunGatesAndExplains: Run acts only on an allowed cell, reports a
// withheld one as "Label: reason" (the label without its "..."), stays silent without a reason, reads the
// cell's own fields when given nil, and ignores an index outside the row.
func TestToolRowRunGatesAndExplains(t *testing.T) {
	ran := -1
	tr := &ToolRow{Cells: []ToolCell{
		{Label: "Refresh", Action: func() { ran = 0 }},
		{Label: "Recycle...", Disabled: true, Reason: "Requires sysadmin", Action: func() { ran = 1 }},
		{Label: "Pause", Disabled: true, Action: func() { ran = 2 }},
		{Label: "Inert"},
	}}
	var said []string
	status := func(s string) { said = append(said, s) }

	tr.Run(0, nil, nil, status)
	if ran != 0 || len(said) != 0 {
		t.Fatalf("allowed cell: ran %d, said %q", ran, said)
	}
	ran = -1
	tr.Run(1, nil, nil, status)
	if ran != -1 || !slices.Equal(said, []string{"Recycle: Requires sysadmin"}) {
		t.Fatalf("withheld cell: ran %d, said %q", ran, said)
	}
	said = nil
	tr.Run(2, nil, nil, status)
	if ran != -1 || len(said) != 0 {
		t.Fatalf("withheld cell without a reason: ran %d, said %q", ran, said)
	}
	for _, i := range []int{-1, 3, 4} {
		tr.Run(i, nil, nil, status)
	}
	if ran != -1 || len(said) != 0 {
		t.Fatalf("out of range / no action: ran %d, said %q", ran, said)
	}

	// The host's predicates override the fields, both ways.
	tr.Run(0, func(int) bool { return true }, func(int) string { return "busy" }, status)
	tr.Run(1, func(int) bool { return false }, nil, status)
	if ran != 1 || !slices.Equal(said, []string{"Refresh: busy"}) {
		t.Fatalf("host predicates: ran %d, said %q", ran, said)
	}
}

// recordingMenu is a MenuShower that keeps what it was asked to show.
type recordingMenu struct {
	x, y  int
	items []MenuItem
}

func (m *recordingMenu) Show(x, y int, items []MenuItem) { m.x, m.y, m.items = x, y, items }

// TestToolRowGateSetOnceReachesEveryPath: with Disabled, Reason and Status set
// on the row, RunCell, Run with nil arguments and the More menu all ask the
// row's gate over the cells' fields, so the click and the menu cannot disagree;
// an argument passed to Run still wins over the row's.
func TestToolRowGateSetOnceReachesEveryPath(t *testing.T) {
	ran := -1
	var said []string
	tr := &ToolRow{
		Cells: []ToolCell{
			{Label: "Refresh", Disabled: true, Action: func() { ran = 0 }},
			{Label: "Export...", Action: func() { ran = 1 }},
			{Label: "Find", Action: func() { ran = 2 }},
		},
		Disabled: func(i int) bool { return i == 1 },
		Reason:   func(i int) string { return "no events" },
		Status:   func(s string) { said = append(said, s) },
	}

	tr.RunCell(0)
	if ran != 0 || len(said) != 0 {
		t.Fatalf("row-allowed cell (field says disabled): ran %d, said %q", ran, said)
	}
	ran = -1
	tr.RunCell(1)
	if ran != -1 || !slices.Equal(said, []string{"Export: no events"}) {
		t.Fatalf("row-withheld cell: ran %d, said %q", ran, said)
	}
	said = nil
	var other []string
	tr.Run(1, func(int) bool { return true }, func(int) string { return "busy" }, func(s string) { other = append(other, s) })
	if ran != -1 || len(said) != 0 || !slices.Equal(other, []string{"Export: busy"}) {
		t.Fatalf("arguments over the row: ran %d, row status %q, arg status %q", ran, said, other)
	}

	tr.Layout(core.Rect{X: 2, Y: 4, W: 22, H: 1}, "")
	if !slices.Equal(tr.Hidden, []int{1, 2}) || tr.More.Rect.IsZero() {
		t.Fatalf("layout: hidden %v, More %+v; want [1 2] behind More", tr.Hidden, tr.More.Rect)
	}
	var m recordingMenu
	tr.OverflowMenu(&m, core.Rect{X: 30, Y: 9})
	if m.x != tr.More.Rect.X || m.y != 5 || len(m.items) != 2 {
		t.Fatalf("menu at (%d,%d) with %d items; want under More with 2", m.x, m.y, len(m.items))
	}
	if m.items[0].enabled() || m.items[0].Note != "no events" || !m.items[1].enabled() || m.items[1].Note != "" {
		t.Errorf("items' gate: %v %q / %v %q", m.items[0].enabled(), m.items[0].Note, m.items[1].enabled(), m.items[1].Note)
	}
	m.items[0].Action()
	m.items[1].Action()
	if ran != 2 || !slices.Equal(said, []string{"Export: no events"}) {
		t.Errorf("menu actions go through RunCell: ran %d, said %q", ran, said)
	}

	said = nil
	tr.ReasonOnly = true
	tr.RunCell(1)
	if !slices.Equal(said, []string{"no events"}) {
		t.Errorf("ReasonOnly: said %q, want the reason without the label", said)
	}
}

// TestToolRowRunLabelDropsTheMenuAndDialogMarks: "Label: reason" reads the
// label without the marks that say a dialog or menu follows.
func TestToolRowRunLabelDropsTheMenuAndDialogMarks(t *testing.T) {
	for in, want := range map[string]string{
		"Recycle...":        "Recycle",
		"Export…":           "Export",
		"Log: SQL Server ▾": "Log: SQL Server",
		"Refresh":           "Refresh",
		"Install in master": "Install in master",
	} {
		if got := runLabel(in); got != want {
			t.Errorf("runLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

// toolRowDrawn lays out a row of A, B, C and a long tail in a row narrow
// enough for More, and draws it with disabled and moreDim.
func toolRowDrawn(t *testing.T, mutate func(*ToolRow), disabled func(int) bool, moreDim bool) (*ToolRow, *tuikittest.Screen) {
	t.Helper()
	tr := newTestToolRow("A", "B", "C", "A much longer tail label")
	if mutate != nil {
		mutate(tr)
	}
	r := core.Rect{X: 0, Y: 0, W: 24, H: 1}
	tr.Layout(r, "")
	if tr.More.Rect.IsZero() || len(tr.Hidden) == 0 {
		t.Fatalf("setup: want an overflowed row, got hidden %v More %+v", tr.Hidden, tr.More.Rect)
	}
	s := tuikittest.NewScreen(24, 1)
	tr.Draw(s, r, disabled, moreDim)
	return tr, s
}

func cellStyle(s *tuikittest.Screen, c ToolCell) tcell.Style { return s.StyleAt(c.Rect.X+1, c.Rect.Y) }

// TestToolRowDrawPaintsEachCellsState: the bar, a plain cell in the tooltip
// scheme, a disabled one dimmed, a selected one in the selected look, and a
// cell both disabled and selected dimmed.
func TestToolRowDrawPaintsEachCellsState(t *testing.T) {
	tr, s := toolRowDrawn(t, func(tr *ToolRow) {
		tr.Cells[1].Disabled = true
		tr.Cells[2].Selected = true
	}, nil, false)
	dim := theme.StyleTooltip().Foreground(theme.Active().TextDim)
	if got := s.StyleAt(tr.Cells[0].Rect.X-1, 0); got != theme.StyleMenuBar() {
		t.Errorf("bar margin style = %v, want StyleMenuBar", got)
	}
	if got := cellStyle(s, tr.Cells[0]); got != theme.StyleTooltip() {
		t.Errorf("plain cell style = %v, want StyleTooltip", got)
	}
	if got := cellStyle(s, tr.Cells[1]); got != dim {
		t.Errorf("disabled cell style = %v, want dimmed", got)
	}
	if got := cellStyle(s, tr.Cells[2]); got != theme.StyleMenuSelected().Bold(true) {
		t.Errorf("selected cell style = %v, want StyleMenuSelected bold", got)
	}
	if got := cellStyle(s, tr.More); got != theme.StyleTooltip() {
		t.Errorf("More style = %v, want undimmed without moreDim", got)
	}
	if r := s.RuneAt(tr.Cells[0].Rect.X+1, 0); r != 'A' {
		t.Errorf("cell 0 label rune = %q, want 'A' after one space of padding", r)
	}

	tr, s = toolRowDrawn(t, func(tr *ToolRow) {
		tr.Cells[0].Disabled, tr.Cells[0].Selected = true, true
	}, nil, false)
	if got := cellStyle(s, tr.Cells[0]); got != dim {
		t.Errorf("disabled+selected cell style = %v, want dimmed: Disabled wins", got)
	}
}

// TestToolRowDrawAsksThePredicateOverTheField: a non-nil disabled replaces
// the Disabled field, as in Run.
func TestToolRowDrawAsksThePredicateOverTheField(t *testing.T) {
	dim := theme.StyleTooltip().Foreground(theme.Active().TextDim)
	tr, s := toolRowDrawn(t, func(tr *ToolRow) { tr.Cells[0].Disabled = true },
		func(i int) bool { return i == 1 }, false)
	if got := cellStyle(s, tr.Cells[0]); got != theme.StyleTooltip() {
		t.Errorf("field-disabled cell = %v, want undimmed: the predicate decides", got)
	}
	if got := cellStyle(s, tr.Cells[1]); got != dim {
		t.Errorf("predicate-disabled cell = %v, want dimmed", got)
	}
}

// TestToolRowDrawAsksTheRowsDisabled: with nil passed, the row's Disabled
// replaces the cell field, as in Run, so a cell that looks inert is.
func TestToolRowDrawAsksTheRowsDisabled(t *testing.T) {
	dim := theme.StyleTooltip().Foreground(theme.Active().TextDim)
	tr, s := toolRowDrawn(t, func(tr *ToolRow) {
		tr.Cells[0].Disabled = true
		tr.Disabled = func(i int) bool { return i == 1 }
	}, nil, false)
	if got := cellStyle(s, tr.Cells[0]); got != theme.StyleTooltip() {
		t.Errorf("field-disabled cell = %v, want undimmed: the row's predicate decides", got)
	}
	if got := cellStyle(s, tr.Cells[1]); got != dim {
		t.Errorf("row-disabled cell = %v, want dimmed", got)
	}
}

// TestToolRowDrawDimsMoreOnlyOnMoreDim, whatever the cells' state.
func TestToolRowDrawDimsMoreOnlyOnMoreDim(t *testing.T) {
	dim := theme.StyleTooltip().Foreground(theme.Active().TextDim)
	tr, s := toolRowDrawn(t, nil, func(int) bool { return true }, false)
	if got := cellStyle(s, tr.More); got != theme.StyleTooltip() {
		t.Errorf("More with every cell disabled = %v, want undimmed", got)
	}
	tr, s = toolRowDrawn(t, nil, nil, true)
	if got := cellStyle(s, tr.More); got != dim {
		t.Errorf("More with moreDim = %v, want dimmed", got)
	}
}

// TestToolRowDrawNeedsOneRow: a rect that is not one row high paints nothing.
func TestToolRowDrawNeedsOneRow(t *testing.T) {
	tr := newTestToolRow("A")
	s := tuikittest.NewScreen(10, 2)
	tr.Draw(s, core.Rect{X: 0, Y: 0, W: 10, H: 2}, nil, false)
	if got := s.StyleAt(0, 0); got != tcell.StyleDefault {
		t.Errorf("style at 0,0 = %v, want untouched", got)
	}
}
