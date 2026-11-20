package propsheet

// FilterIndex is the filtered view a filter box keeps over a grid's list: the
// grid shows Rows, and its row i addresses At(i), never the full list. The
// full list (SetItems) stays the host's; Apply narrows it by a term through
// the host's match (core.MatchesAny suits most).
//
// The trap it exists to name: a filter that matches nothing leaves the grid
// empty, and a detail the page shows for the grid's selection (a second grid,
// a set of editor rows) must be cleared then, not left on the previously
// selected item. Otherwise an edit there lands on an item the page no longer
// lists, and Apply writes it. FilterRow's onFilter is where the host re-points
// that detail, at At(0) or, when Len is 0, at nothing.
//
// The zero value is not usable; NewFilterIndex.
type FilterIndex[T any] struct {
	match   func(item T, term string) bool
	items   []T
	term    string
	visible []T
}

// NewFilterIndex returns an empty index filtering by match, which reports
// whether item matches term. match must accept an empty term (normally as
// matching everything): the index starts with one.
func NewFilterIndex[T any](match func(item T, term string) bool) *FilterIndex[T] {
	return &FilterIndex[T]{match: match}
}

// SetItems replaces the full list and re-filters it by the current term. The
// slice is held, not copied, and never written to.
func (f *FilterIndex[T]) SetItems(items []T) {
	f.items = items
	f.refilter()
}

// Apply sets the term and re-filters the list by it.
func (f *FilterIndex[T]) Apply(term string) {
	f.term = term
	f.refilter()
}

// refilter rebuilds visible into a fresh slice, so a Visible result taken
// earlier is never rewritten underneath its holder.
func (f *FilterIndex[T]) refilter() {
	f.visible = make([]T, 0, len(f.items))
	for _, it := range f.items {
		if f.match(it, f.term) {
			f.visible = append(f.visible, it)
		}
	}
}

// Term returns the term last applied.
func (f *FilterIndex[T]) Term() string { return f.term }

// Len returns how many items pass the filter: the grid's row count.
func (f *FilterIndex[T]) Len() int { return len(f.visible) }

// At returns the item at grid row row, and false (with a zero T) when row is
// outside the filtered list.
func (f *FilterIndex[T]) At(row int) (T, bool) {
	if row < 0 || row >= len(f.visible) {
		var zero T
		return zero, false
	}
	return f.visible[row], true
}

// Visible returns the filtered list in grid-row order. It is the index's own
// slice: read it, don't write to it. A later SetItems or Apply builds a new
// one rather than changing this one.
func (f *FilterIndex[T]) Visible() []T { return f.visible }

// Rows renders the filtered list as grid rows, one cells call per item.
func (f *FilterIndex[T]) Rows(cells func(T) []string) [][]string {
	rows := make([][]string, len(f.visible))
	for i, it := range f.visible {
		rows[i] = cells(it)
	}
	return rows
}

// FilterRow returns the filter box driving the index: a Text row, width
// columns wide, that is a view control (SetDirtyTracked(false): typing in it
// is not an unsaved change) and on every edit applies its text and then calls
// onFilter, where the host redraws the grid and re-points any detail (see
// FilterIndex for why an empty result must clear it). onFilter may be nil.
func (f *FilterIndex[T]) FilterRow(label string, width int, onFilter func()) *TextRow {
	r := Text(label, "", width)
	r.SetDirtyTracked(false)
	r.SetOnChange(func(term string) {
		f.Apply(term)
		if onFilter != nil {
			onFilter()
		}
	})
	return r
}
