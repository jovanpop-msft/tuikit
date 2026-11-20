package propsheet

import "slices"

// PendingState is a list-editor row's place in its page's pending-edit list:
// added since load (IsNew), or loaded and to be removed on Apply (Removing). A
// row type embeds it and so satisfies PendingRow.
type PendingState struct {
	IsNew, Removing bool
}

// Pending returns s; it is what makes a row type embedding PendingState a
// PendingRow.
func (s *PendingState) Pending() *PendingState { return s }

// PendingRow is a pointer to a row type embedding PendingState.
type PendingRow interface {
	comparable
	Pending() *PendingState
}

// PendingEdits is a master-detail page's pending-edit list: the rows as loaded
// and as the page now says, for the host's Apply to write. It is the model a
// ListEditor's Items, Add, Remove, Dirty and Revert read and change; it draws
// nothing and runs nothing.
//
// changed reports an edit to a loaded row's values, and reset undoes it; both
// nil on a page whose rows are only added and removed. nameOf is the row's name
// for the Add duplicate check (Taken, Index); nil on a page with no name to
// type. same is the name equality those checks use: the host's, since whether
// "Ops" and "ops" are one name is the data's rule (a collation, a file system),
// not the list's; nil is byte-wise ==.
type PendingEdits[E PendingRow] struct {
	same    func(a, b string) bool
	nameOf  func(E) string
	changed func(E) bool
	reset   func(E)

	loaded, rows []E
}

// NewPendingEdits is the list for loaded; see PendingEdits for the callbacks.
// The list keeps loaded for Revert and works on its own copy of the slice.
func NewPendingEdits[E PendingRow](loaded []E, same func(a, b string) bool, nameOf func(E) string, changed func(E) bool, reset func(E)) *PendingEdits[E] {
	if same == nil {
		same = func(a, b string) bool { return a == b }
	}
	return &PendingEdits[E]{same: same, nameOf: nameOf, changed: changed, reset: reset,
		loaded: loaded, rows: slices.Clone(loaded)}
}

// All is every row, those being removed included, in loaded-then-added order:
// Apply's input.
func (p *PendingEdits[E]) All() []E { return p.rows }

// Visible is the rows the grid lists: all but those being removed.
func (p *PendingEdits[E]) Visible() []E {
	out := make([]E, 0, len(p.rows))
	for _, e := range p.rows {
		if !e.Pending().Removing {
			out = append(out, e)
		}
	}
	return out
}

// IsChanged reports an edit to loaded row e's values.
func (p *PendingEdits[E]) IsChanged(e E) bool { return p.changed != nil && p.changed(e) }

// RowDirty reports whether e has anything for Apply to write.
func (p *PendingEdits[E]) RowDirty(e E) bool {
	s := e.Pending()
	return s.IsNew || s.Removing || p.IsChanged(e)
}

// Dirty reports whether any row has anything for Apply to write.
func (p *PendingEdits[E]) Dirty() bool { return slices.ContainsFunc(p.rows, p.RowDirty) }

// Revert puts the list back as loaded: added rows gone, removals and edits
// undone.
func (p *PendingEdits[E]) Revert() {
	p.rows = p.rows[:0]
	for _, e := range p.loaded {
		e.Pending().Removing = false
		if p.reset != nil {
			p.reset(e)
		}
		p.rows = append(p.rows, e)
	}
}

// Swap exchanges rows a and b; a page that reorders moves them in All, not in
// Visible, which skips rows being removed.
func (p *PendingEdits[E]) Swap(a, b E) {
	i, j := slices.Index(p.rows, a), slices.Index(p.rows, b)
	p.rows[i], p.rows[j] = p.rows[j], p.rows[i]
}

// Add appends e as a new row.
func (p *PendingEdits[E]) Add(e E) {
	e.Pending().IsNew = true
	p.rows = append(p.rows, e)
}

// Remove takes e off the list: a new row is forgotten, a loaded one marked for
// removal on Apply.
func (p *PendingEdits[E]) Remove(e E) {
	if e.Pending().IsNew {
		p.rows = slices.DeleteFunc(p.rows, func(x E) bool { return x == e })
		return
	}
	e.Pending().Removing = true
}

// Restore takes back e's pending removal: an Add of a row being removed, on a
// page that would otherwise drop and re-create it.
func (p *PendingEdits[E]) Restore(e E) { e.Pending().Removing = false }

// Name is e's name as the list's nameOf gives it: for a host check over All
// (one that also needs each row's stored name, say).
func (p *PendingEdits[E]) Name(e E) string { return p.nameOf(e) }

// Index is the index in Visible of the row named name, or -1: the Add duplicate
// check of a page that selects the row it finds.
func (p *PendingEdits[E]) Index(name string) int {
	for i, e := range p.Visible() {
		if p.same(p.nameOf(e), name) {
			return i
		}
	}
	return -1
}

// Listed reports whether a visible row is named name. A row being removed is
// not: on a page whose Apply drops before it creates, its name is free, and on
// one that refuses a clash before Apply, a message explaining that the removal
// frees the name only on Apply beats "already listed" about a row no longer
// shown.
func (p *PendingEdits[E]) Listed(name string) bool { return p.Index(name) >= 0 }

// Taken reports whether any row is named name, those being removed included: the
// Add check of a page whose Apply creates before it drops and has no refusal to
// catch the clash.
func (p *PendingEdits[E]) Taken(name string) bool {
	return slices.ContainsFunc(p.rows, func(e E) bool { return p.same(p.nameOf(e), name) })
}
