package controls

// CheckOn and CheckOff prefix a pick-many entry: ticked, and not.
const (
	CheckOn  = "☑ "
	CheckOff = "☐ "
)

// CheckMark returns the prefix for an entry that is on or off — CheckOn or
// CheckOff — for a single toggle in a menu as much as a checklist row.
func CheckMark(on bool) string {
	if on {
		return CheckOn
	}
	return CheckOff
}

// CheckItems builds a pick-many checklist's menu, the sibling of ChoiceItems:
// one entry per value, labelled CheckMark(checked(v)) + label(v), each
// toggling its value. label sees the value, so a row can carry more of its
// state than the tick (an order, a level). The items are KeepOpen: give the
// ContextMenu a SetRefresh that rebuilds them, and every tick redraws in place
// with the highlight kept. The slice is in values order; a host appends its
// own rows (Select All, Clear) after it.
func CheckItems[T any](values []T, checked func(T) bool, label func(T) string, toggle func(T)) []MenuItem {
	items := make([]MenuItem, 0, len(values))
	for _, v := range values {
		items = append(items, MenuItem{
			Label:    CheckMark(checked(v)) + label(v),
			Action:   func() { toggle(v) },
			KeepOpen: true,
		})
	}
	return items
}
