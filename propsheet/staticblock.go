package propsheet

// StaticBlock is a group of read-only Static rows filled from a grid's selected
// row: the "selected item" detail half of a grid-plus-detail page.
//
// It exists for the no-selection branch: clearing one SetValue("") per row
// risks a missed row that keeps describing the item the selection just left.
// Set with no values clears every row, and a short list clears the rest.
//
// The block holds rows the caller constructed and added to its Form, so each
// label stays a literal argument to Static at the call site, where a host's
// label-width check can read it.
type StaticBlock struct {
	rows []*StaticRow
}

// NewStaticBlock groups rows, in the order Set fills them.
func NewStaticBlock(rows ...*StaticRow) *StaticBlock {
	return &StaticBlock{rows: rows}
}

// Set fills the block from values, one per row in construction order; rows
// past the end of values are cleared.
func (b *StaticBlock) Set(values ...string) {
	for i, r := range b.rows {
		if i < len(values) {
			r.SetValue(values[i])
		} else {
			r.SetValue("")
		}
	}
}
