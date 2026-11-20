package controls

import (
	"slices"

	"github.com/radix29/tuikit/core"
)

// computeColWidths sizes columns from their header plus up to colWidthSampleRows
// data rows, so a huge result set doesn't slow SetSource. A cell is measured only
// as far as the clamp can see: past max(maxW, 6) columns every width clamps to
// the same result. Measuring whole cells cost 0.3-0.7 s per call for one column
// of 256 KB XML values, on the UI goroutine, and this runs on every SetSource
// and SetBounds. A change of overrides alone (a column drag) goes through
// reapplyColWidths instead, which keeps the measured widths.
func (g *DataGrid) computeColWidths() {
	g.widthsDirty = false
	g.autoAligns = nil
	if g.errorMode {
		g.naturalWidths = nil
		g.colWidths = []int{g.rect.W - 2}
		if w := g.overrideWidth(0); w > 0 {
			g.colWidths[0] = w
		}
		return
	}
	natural := make([]int, len(g.columns))
	for i := range g.columns {
		natural[i] = core.DisplayWidth(g.headerLabel(i)) + 2
	}
	cellLimit := max(g.maxCellWidthOrDefault(), 6) - 2
	var classify func(string) CellClass
	if g.autoAlign != nil {
		classify = g.autoAlign(g.columns)
	}
	var seen []CellClass // per column: CellBlank, CellNumber, or CellText once any text was seen
	if classify != nil {
		seen = make([]CellClass, len(g.columns))
	}
	n := min(g.rows.Len(), colWidthSampleRows)
	for r := range n {
		// A group row's label (cell 0) spills across the row; its other
		// cells are ordinary.
		first := 0
		if g.RowKindAt(r) == RowGroup {
			first = 1
		}
		row := g.rows.Row(r)
		for i, cell := range row {
			if i >= first && i < len(natural) {
				// Measured as drawn: TruncateLine shows a line break as a
				// space, which DisplayWidth counts as nothing.
				if w := core.DisplayWidthAtMost(core.TruncateLine(cell, cellLimit), cellLimit) + 2; w > natural[i] {
					natural[i] = w
				}
				if seen != nil && seen[i] != CellText {
					if c := classify(cell); c != CellBlank {
						seen[i] = max(seen[i], c)
					}
				}
			}
		}
	}
	for i, c := range seen {
		if c == CellNumber {
			if g.autoAligns == nil {
				g.autoAligns = make([]ColumnAlign, len(seen))
			}
			g.autoAligns[i] = AlignRight
		}
	}
	g.naturalWidths = natural
	g.applyColWidths()
}

// reapplyColWidths is computeColWidths for a change that leaves the content
// alone — an override set, the fill flag flipped — reusing the measured widths.
// A separator drag calls it per motion event: rescanning there cost up to
// colWidthSampleRows Row() calls each, 4,000 for a 20-step drag over a
// streamed source that builds every row. It falls back to the full measure
// when there is nothing current to reuse (a RefreshColumnWidths pending, the
// error row, a column count that moved).
func (g *DataGrid) reapplyColWidths() {
	if g.widthsDirty || g.errorMode || len(g.naturalWidths) != len(g.columns) {
		g.computeColWidths()
		return
	}
	g.applyColWidths()
}

// applyColWidths derives colWidths from naturalWidths: the clamp, the dragged
// overrides, then the fill.
func (g *DataGrid) applyColWidths() {
	maxW := g.maxCellWidthOrDefault()
	g.colWidths = append(g.colWidths[:0], g.naturalWidths...)
	for i := range g.colWidths {
		g.colWidths[i] = core.Clamp(g.colWidths[i], 6, maxW)
		if w := g.overrideWidth(i); w > 0 {
			g.colWidths[i] = w
		}
	}
	// A last column dragged to a width of its own keeps it; stretching it back
	// out to the rect would make the drag look ignored.
	if g.fillLastColumn && len(g.colWidths) > 0 && g.overrideWidth(len(g.colWidths)-1) == 0 {
		g.growLastColumnToFill()
	}
}

// overrideWidth returns column i's user-dragged width, or 0 if it has none.
func (g *DataGrid) overrideWidth(i int) int {
	if i < 0 || i >= len(g.colWidthOverride) {
		return 0
	}
	return g.colWidthOverride[i]
}

// setOverrideWidth records column i's dragged width (0 clears it), growing the
// override slice on demand to match the column count.
func (g *DataGrid) setOverrideWidth(i, w int) {
	if i < 0 || i >= len(g.colWidths) {
		return
	}
	for len(g.colWidthOverride) < len(g.colWidths) {
		g.colWidthOverride = append(g.colWidthOverride, 0)
	}
	g.colWidthOverride[i] = w
}

// restoreOverrideWidths reinstates a column set's dragged widths and recomputes
// once, for SetDataPreservingView. Calling SetColumnWidth per column runs a full
// computeColWidths (a rescan of up to colWidthSampleRows rows) per column, and
// that runs on every SetDataPreservingView, which a host may call on every
// keystroke (a grid-backed property page re-rendering as the user edits). Applies SetColumnWidth's minResizeWidth floor so a restored
// width is what a drag would have set.
//
// The recompute is conditional because widths is always column-count long
// (ColumnWidthOverrides pads it), all zeros on an unresized grid; recomputing
// unconditionally doubled the scan on the hot path, since SetData has already
// computed the widths.
func (g *DataGrid) restoreOverrideWidths(widths []int) {
	restored := false
	for i, w := range widths {
		if w > 0 {
			g.setOverrideWidth(i, max(w, minResizeWidth))
			restored = true
		}
	}
	if restored {
		g.reapplyColWidths()
	}
}

// ColumnWidthOverrides returns each column's dragged width, 0 for one at its
// computed default: the inverse of SetColumnWidth, and the only way to carry
// drags across a SetData that discards them (see SetDataPreservingView). The
// slice is a copy, as long as the column count.
func (g *DataGrid) ColumnWidthOverrides() []int {
	out := make([]int, len(g.colWidths))
	copy(out, g.colWidthOverride)
	return out
}

// SetColumnWidth sets column i's width explicitly, as a separator drag would —
// bypassing the max-default clamp and surviving later recomputes. w <= 0
// restores the computed default.
func (g *DataGrid) SetColumnWidth(i, w int) {
	if i < 0 || i >= len(g.colWidths) {
		return
	}
	if w > 0 {
		g.setOverrideWidth(i, max(w, minResizeWidth))
	} else {
		g.setOverrideWidth(i, 0)
	}
	g.reapplyColWidths()
}

// ColumnWidth returns column i's current on-screen width, or 0 if i isn't a
// column.
func (g *DataGrid) ColumnWidth(i int) int {
	if i < 0 || i >= len(g.colWidths) {
		return 0
	}
	return g.colWidths[i]
}

// growLastColumnToFill widens the last column to consume the width left once
// every other column is accounted for, bypassing maxCellWidthOrDefault: that
// cap keeps an ordinary column from swallowing the grid, which here is the
// point.
func (g *DataGrid) growLastColumnToFill() {
	avail := g.rect.W - g.gutterWidth()
	last := len(g.colWidths) - 1
	used := 0
	for _, w := range g.colWidths[:last] {
		used += w
	}
	if rem := avail - used; rem > g.colWidths[last] {
		g.colWidths[last] = rem
	}
}

// gutterWidth returns the row-number column's on-screen width — the highest row
// number plus a padding column each side — or 0 unless SetRowNumbers is on.
func (g *DataGrid) gutterWidth() int {
	if !g.showRowNumbers {
		return 0
	}
	// Counted, not formatted: every draw and mouse event asks, and Itoa allocated.
	digits := 1
	for n := g.rows.Len(); n >= 10; n /= 10 {
		digits++
	}
	return digits + 2
}

// ColumnAlign is where a column's data cells sit within it.
type ColumnAlign int

const (
	// AlignLeft is the default: text starts after the column's left padding.
	AlignLeft ColumnAlign = iota
	// AlignRight ends the text before the column's right padding — numbers,
	// so their digits line up by place value.
	AlignRight
)

// SetColumnAlignments sets each column's data-cell alignment by index; a column
// past the slice's end is AlignLeft. Headers stay left-aligned, as SSMS draws
// them, and so does a cell too wide for its column: it is cut at the right with
// an ellipsis, since cutting its start would hide a number's leading digits.
// SetData and SetSource clear the alignments, which name the old set's columns;
// SetDataPreservingView keeps them. While set they override SetAutoAlign.
func (g *DataGrid) SetColumnAlignments(aligns []ColumnAlign) {
	g.aligns = slices.Clone(aligns)
}

// CellClass is what a SetAutoAlign classifier reads one cell as.
type CellClass int

// The order is load-bearing: computeColWidths keeps the highest class a
// column's cells reached, so one text cell outranks any number of numbers.
const (
	// CellBlank holds no value (empty, a placeholder): it decides nothing.
	CellBlank CellClass = iota
	// CellNumber is a number, as the host renders them.
	CellNumber
	// CellText is anything else.
	CellText
)

// SetAutoAlign right-aligns, by content, every column whose cells are numbers:
// at least one CellNumber and no CellText among the cells the width measure
// samples (the first colWidthSampleRows rows). What a number looks like is the
// host's ("1,234", "12 MB"), so it supplies the test: classify is called with
// the header row each time the widths are measured, and returns the cell
// classifier for that column set, or nil to leave every column left-aligned
// (a shape whose columns are not lists, such as a Property/Value record).
//
// The derivation reruns wherever the widths are measured (SetSource,
// SetBounds, RefreshColumnWidths), so cells written in place follow it. The
// classifier is kept across row sets; SetColumnAlignments, while set,
// overrides it. Nil turns it off.
func (g *DataGrid) SetAutoAlign(classify func(columns []string) func(cell string) CellClass) {
	g.autoAlign = classify
	g.widthsDirty = true
}

// columnAlign is column i's alignment: SetColumnAlignments' while it is set,
// else SetAutoAlign's.
func (g *DataGrid) columnAlign(i int) ColumnAlign {
	if g.aligns != nil {
		if i < len(g.aligns) {
			return g.aligns[i]
		}
		return AlignLeft
	}
	if i < len(g.autoAligns) {
		return g.autoAligns[i]
	}
	return AlignLeft
}
