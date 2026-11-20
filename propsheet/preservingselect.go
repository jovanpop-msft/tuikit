package propsheet

import "slices"

// ---------------------------------------------------------------------------
// PreservingSelectRow: a dropdown that never misreports the value it was seeded
// with
// ---------------------------------------------------------------------------

// PreservingItems returns base and the index of value within it, widening the
// list with value itself when base lacks it, so the index always points at
// value. base is never modified.
//
// Use it for any externally sourced value a Select row shows. A not-found
// fallback to index 0 would display items[0] as fact (an object owned by a
// principal since dropped, shown as owned by whichever name sorts first) on
// exactly the objects a user opened the sheet to investigate.
//
// value must be non-empty for the result to mean anything; SelectPreserving
// substitutes its stand-in for a blank.
func PreservingItems(base []string, value string) ([]string, int) {
	if i := slices.Index(base, value); i >= 0 {
		return base, i
	}
	items := append(slices.Clone(base), value)
	return items, len(items) - 1
}

// PreservingSelectRow is a SelectRow built by SelectPreserving: it displays
// the value it was seeded with whether or not its list offers it, and a
// stand-in ("unset") when the source reported nothing. It is a distinct type
// so that only rows built this way expose PreservedValue and ChangedTo, which
// mean nothing on a plain Select.
type PreservingSelectRow struct {
	*SelectRow
	unset string
}

// SelectPreserving builds a Select row that can never misreport: it shows
// value whether or not base offers it, and unset when value is empty.
//
// Read it back with PreservedValue or ChangedTo, which map unset to "" again:
// a stand-in must never be written back as a real value.
func SelectPreserving(label string, base []string, value, unset string) *PreservingSelectRow {
	items, i := PreservingItems(base, orUnset(value, unset))
	return &PreservingSelectRow{SelectRow: Select(label, items, i), unset: unset}
}

// SetChoices replaces the row's list with base and selects value, widened and
// stood-in exactly as SelectPreserving does, and resets the dirty baseline: the
// repoint form, for a row whose list or value arrives after it is built.
func (r *PreservingSelectRow) SetChoices(base []string, value string) {
	items, i := PreservingItems(base, orUnset(value, r.unset))
	r.SetItems(items)
	r.SetSelected(i)
}

// Unset returns the row's stand-in for "no value".
func (r *PreservingSelectRow) Unset() string { return r.unset }

// PreservedValue reads the row back as a value to write, undoing its
// stand-in: "" while the row sits on unset.
func (r *PreservingSelectRow) PreservedValue() string {
	if v := r.Value(); v != r.unset {
		return v
	}
	return ""
}

// ChangedTo reports the real value the row was edited to, and whether there is
// one: dirty, and not sitting on its stand-in.
//
// Gate every write behind it rather than on Dirty() alone. Dirty() suffices only
// while the stand-in is listed solely as the original selection, a property of
// how the list was built far from the write; a list that also offers the
// stand-in as a choice would otherwise send it as a real value.
//
// A nil row reports no change, for a host that drew the value read-only for
// this object and never built the row.
func (r *PreservingSelectRow) ChangedTo() (string, bool) {
	if r == nil || !r.Dirty() {
		return "", false
	}
	v := r.PreservedValue()
	return v, v != ""
}

func orUnset(value, unset string) string {
	if value == "" {
		return unset
	}
	return value
}
