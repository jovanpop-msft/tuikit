package controls

// ChoiceMark prefixes the menu entry already in force — a pick-one selector's
// current value, a selected ToolRow cell in the More menu — so a user sees
// which one applies without reading the button the menu covers.
const ChoiceMark = "• "

// ChoiceItems builds a pick-one selector's menu: one entry per value, labelled
// by label, the one equal to current marked with ChoiceMark, each choosing its
// value. The items are plain; a host that withholds some sets their Enabled and
// Note afterwards (the slice is in values order).
func ChoiceItems[T comparable](values []T, current T, label func(T) string, choose func(T)) []MenuItem {
	items := make([]MenuItem, 0, len(values))
	for _, v := range values {
		text := label(v)
		if v == current {
			text = ChoiceMark + text
		}
		items = append(items, MenuItem{Label: text, Action: func() { choose(v) }})
	}
	return items
}
