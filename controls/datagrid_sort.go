package controls

import "slices"

// datagrid_sort.go is the mechanics of a host's column sort, shared by every
// grid that sorts on a header click: a source shown through a permutation, the
// swap that keeps the cursor on its row, and the cell menu's keyboard route.
// What orders the rows (the comparison, sorting off the UI goroutine, which
// order each view remembers) stays the host's; the grid only draws the arrow
// (SetSortIndicator) and reports the click (OnHeaderClick).

// PermutedSource shows Src in Perm's order: Perm[i] is Src's index of the i-th
// row shown. The rows are not copied, so a host writing cells of Src in place
// reaches the permuted view too.
//
// It passes NullSource and RowKindSource through when Src implements them, so
// reordered NULLs still draw dimmed and marked rows stay marked.
type PermutedSource struct {
	Src  RowSource
	Perm []int
}

func (s PermutedSource) Len() int           { return len(s.Perm) }
func (s PermutedSource) Row(i int) []string { return s.Src.Row(s.Perm[i]) }

// IsNull is Src's IsNull for the row shown at row, false when Src is not a
// NullSource.
func (s PermutedSource) IsNull(row, col int) bool {
	if n, ok := s.Src.(NullSource); ok {
		return n.IsNull(s.Perm[row], col)
	}
	return false
}

// RowKind is Src's RowKind for the row shown at i, RowNormal when Src is not a
// RowKindSource.
func (s PermutedSource) RowKind(i int) RowKind {
	if k, ok := s.Src.(RowKindSource); ok {
		return k.RowKind(s.Perm[i])
	}
	return RowNormal
}

// SetSourceReordered installs rows, the same rows as the grid's in another
// order, keeping the cursor on the row it was on rather than on its old index
// (which now holds another row), plus the scroll, the dragged column widths and
// the sort indicator, as SetSourcePreservingView does. oldPerm and newPerm map
// each shown index to the row's index in the host's underlying order — the
// order before and after; a nil oldPerm is the identity over the rows shown
// now (nothing was sorted). When the cursor's row is not in newPerm the cursor
// stays on its index.
//
// Ctrl+click marks follow their rows too, so a user who marks rows and sorts
// to check them still copies those rows; a marked row missing from newPerm is
// dropped. A Shift-extended block is collapsed: after a re-sort its rows are no
// longer contiguous, and its column range is something marks cannot express.
//
// The host sets the indicator for the new order (SetSortIndicator) before or
// after; this keeps the old one.
func (g *DataGrid) SetSourceReordered(columns []string, rows RowSource, oldPerm, newPerm []int) {
	row, col := g.SelectedCell()
	shown := g.rows.Len()
	if oldPerm != nil {
		shown = len(oldPerm)
	}
	under := func(i int) int { // a shown index of the old order -> underlying row
		if oldPerm == nil {
			return i
		}
		return oldPerm[i]
	}
	orig := -1
	if row >= 0 && row < shown {
		orig = under(row)
	}
	// A base run covering every row shown (Ctrl+A, then a Ctrl+click) is carried
	// as a run over the new order, with only its flips moved: materialising it
	// cost a million-int slice and two million-entry maps per sort.
	marking, flips := g.marking, g.markFlips
	allRun := marking && shown > 0 && g.markLo <= 0 && g.markHi >= shown-1
	var marked []int // the marked rows, as indexes in the underlying order
	if marking && !allRun {
		for _, r := range g.SelectedRows() {
			if r >= 0 && r < shown {
				marked = append(marked, under(r))
			}
		}
	}
	g.SetSourcePreservingView(columns, rows)
	if !marking {
		if i := slices.Index(newPerm, orig); i >= 0 {
			g.SetSelectedCell(i, col)
		}
		return
	}
	// at[r] is 1 + the shown index of underlying row r in the new order, 0 when
	// the new order lacks it: a dense inverse, no hashing per row.
	top := -1
	for _, r := range newPerm {
		top = max(top, r)
	}
	at := make([]int, top+1)
	for i, r := range newPerm {
		at[r] = i + 1
	}
	pos := func(r int) int {
		if r < 0 || r >= len(at) {
			return -1
		}
		return at[r] - 1
	}
	if i := pos(orig); i >= 0 {
		g.SetSelectedCell(i, col)
	}
	if allRun {
		g.setAllMarkedReordered(flips, shown, under, pos, oldPerm, newPerm)
		return
	}
	rowsNow := make([]int, 0, len(marked))
	for _, r := range marked {
		if i := pos(r); i >= 0 {
			rowsNow = append(rowsNow, i)
		}
	}
	g.setMarkedRows(rowsNow)
}

// setAllMarkedReordered is SetSourceReordered's marks when the base run covered
// every row shown: the new base run covers every row of newPerm, and the flips
// are the old flips that still name a row (each unmarked it) plus the rows
// newPerm holds that the old order did not show (never marked). No rows left
// marked leaves no marks, as setMarkedRows does.
func (g *DataGrid) setAllMarkedReordered(flips map[int]struct{}, shown int, under, pos func(int) int, oldPerm, newPerm []int) {
	newFlips := make(map[int]struct{}, len(flips))
	for f := range flips {
		if f < shown { // a flip past the rows shown named no row
			if i := pos(under(f)); i >= 0 {
				newFlips[i] = struct{}{}
			}
		}
	}
	if oldPerm == nil {
		for i, r := range newPerm {
			if r >= shown {
				newFlips[i] = struct{}{}
			}
		}
	} else {
		was := make([]bool, len(newPerm)) // indexed by new shown index
		for _, r := range oldPerm {
			if i := pos(r); i >= 0 {
				was[i] = true
			}
		}
		for i, ok := range was {
			if !ok {
				newFlips[i] = struct{}{}
			}
		}
	}
	if len(newFlips) >= len(newPerm) {
		g.ClearMarkedRows()
		return
	}
	g.markLo, g.markHi, g.markFlips, g.marking = 0, len(newPerm)-1, newFlips, true
}

// SortMenuItems are a grid cell menu's keyboard route to a header click (an
// OnMenuItems host's menu opens on the cursor's cell): Sort Ascending and Sort
// Descending of the column the host captured in the closures. isCurrent reports
// whether that column is already sorted in that direction, which greys the
// entry; sort applies it. Enabled is evaluated when the menu draws.
func SortMenuItems(isCurrent func(desc bool) bool, sort func(desc bool)) []MenuItem {
	item := func(label string, desc bool) MenuItem {
		return MenuItem{
			Label:   label,
			Enabled: func() bool { return !isCurrent(desc) },
			Action:  func() { sort(desc) },
		}
	}
	return []MenuItem{item(Strings.SortAscending, false), item(Strings.SortDescending, true)}
}
