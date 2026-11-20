package controls

import (
	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// ---------------------------------------------------------------------------
// MenuItem / Menu — shared by MenuBar and ContextMenu
// ---------------------------------------------------------------------------

// MenuItem is a single entry in a menu.
type MenuItem struct {
	Label    string
	Shortcut string
	Divider  bool        // renders as a ──── separator
	Action   func()      // called when the item is activated
	Enabled  func() bool // nil means always enabled

	// Sub, when non-empty, makes this item a cascade: it opens a submenu
	// instead of firing Action, and draws a ▸ marker where a shortcut would
	// otherwise go. See menuCascade (menu_box.go).
	Sub []MenuItem

	// Note is shown in the shortcut column *while the item is disabled*, to say
	// why. A disabled item can't be selected or activated, so no keypress or click
	// is left to answer with a status message; without this, an item withheld for
	// lack of a permission is grey and silent. Its own Shortcut shows again once
	// enabled.
	Note string

	// NoteWhen narrows Note to the reason it describes. nil shows Note whenever the
	// item is disabled; a predicate shows it only when it answers true.
	//
	// An item can be withheld for more than one reason and Note describes one of
	// them. "Fail Over to This Replica..." is disabled on the primary because a
	// replica cannot fail over to itself; labelling it "needs ALTER ANY
	// AVAILABILITY GROUP" would send the user to ask for a permission that changes
	// nothing.
	NoteWhen func() bool

	// KeepOpen makes activating the item (Enter, or a click) run Action and
	// leave the menu open with the highlight where it was, instead of closing
	// it first — a checklist row, ticked several times in one showing. The
	// labels are the host's to bring up to date: a ContextMenu re-reads its
	// items from SetRefresh after the action, a MenuBar takes them from the
	// host's SetMenus. A KeepOpen item that opens a submenu ignores the flag.
	KeepOpen bool

	// Checked, when non-nil, makes the item an on/off toggle: the menu box
	// draws MenuTick in a mark column before the label while it answers true,
	// and a blank there while false. It is read at draw time, so the label
	// stays fixed and the menu shows the state without the host rebuilding
	// its items after each flip. Any toggle in a box reserves the column for
	// every row of that box, keeping the labels aligned and the box the same
	// width in both states; a box with no toggle has no column.
	Checked func() bool
}

// MenuTick is the mark a Checked menu item draws while on. It is lighter than
// CheckOn/CheckOff by design: a menu toggle reads as a check column, as in
// desktop menus, not as a checklist row.
const MenuTick = "✓"

// menuMarkWidth is the mark column a box with a Checked item reserves: the
// tick and one column of space before the label.
const menuMarkWidth = 2

// enabled reports whether it can be selected or activated right now.
func (it MenuItem) enabled() bool {
	return it.Enabled == nil || it.Enabled()
}

// checked reports whether a toggle item is on; false for a non-toggle.
func (it MenuItem) checked() bool {
	return it.Checked != nil && it.Checked()
}

// menuMarkColumn is the width of the mark column items need: menuMarkWidth
// when any of them is a toggle, else 0.
func menuMarkColumn(items []MenuItem) int {
	for _, it := range items {
		if it.Checked != nil && !it.Divider {
			return menuMarkWidth
		}
	}
	return 0
}

// Menu is a top-level menu header with its items.
type Menu struct {
	Label string
	Items []MenuItem
}

// outsideMenu reports what an open menu (a MenuBar dropdown, a ContextMenu)
// does with an event outside it, per core.MouseHandler: any button press
// dismisses the menu; a right-click then falls through (consumed false) so the
// host can open a new menu where it landed, and anything else is consumed. A
// wheel tick is consumed without dismissing. Plain motion is left to the
// caller (dismiss and consumed both false).
func outsideMenu(ev *tcell.EventMouse) (dismiss, consumed bool) {
	switch b := ev.Buttons(); {
	case b == tcell.ButtonNone:
		return false, false
	case b&(tcell.WheelUp|tcell.WheelDown|tcell.WheelLeft|tcell.WheelRight) != 0:
		return false, true
	default:
		return true, b != tcell.Button2
	}
}

// menuItemSkippable reports whether an item must be skipped by keyboard/
// mouse selection — a divider, or one whose Enabled predicate says no.
func menuItemSkippable(it MenuItem) bool {
	return it.Divider || !it.enabled()
}

// firstSelectableItem returns the index of the first selectable item, or
// -1 if the menu has none.
func firstSelectableItem(items []MenuItem) int {
	for i, it := range items {
		if !menuItemSkippable(it) {
			return i
		}
	}
	return -1
}

// stepSelectableItem returns the next selectable item index starting from
// `from`, moving by `dir` (+1 or -1) and wrapping around. Returns -1 if the
// menu has no selectable items at all. With nothing highlighted (from < 0),
// Down gives the first item and Up the last, as Windows and GTK menus do.
func stepSelectableItem(items []MenuItem, from, dir int) int {
	n := len(items)
	if n == 0 {
		return -1
	}
	i := from
	if i < 0 {
		i = -1
		if dir < 0 {
			i = n
		}
	}
	for range n {
		i = (i + dir + n) % n
		if !menuItemSkippable(items[i]) {
			return i
		}
	}
	return -1
}

// menuRowSuffix returns the right-aligned text of an item: the cascade marker
// for a submenu, otherwise its shortcut. An item with a submenu has no shortcut
// of its own (activating it only opens the submenu), but a disabled one shows
// its note in place of the marker.
func menuRowSuffix(it MenuItem) string { return rowSuffix(it, it.enabled()) }

// rowSuffix is menuRowSuffix with the item's Enabled already evaluated, so a
// draw calls each predicate once (menuBox.measure).
func rowSuffix(it MenuItem, enabled bool) string {
	// The note outranks the ▸: a disabled cascade cannot be opened, so its
	// marker points at a submenu the user will never see and the note is the
	// only thing left that says why the item is grey.
	if it.Note != "" && !enabled && (it.NoteWhen == nil || it.NoteWhen()) {
		return it.Note
	}
	if len(it.Sub) > 0 {
		return "▸"
	}
	return it.Shortcut
}

// menuRowStyles returns the label and suffix styles for one item — disabled
// beats selected/hovered, which beats the plain default — so every menu box
// greys out a disabled item identically.
func menuRowStyles(enabled, selected bool) (label, suffix tcell.Style) {
	switch {
	case !enabled:
		st := theme.StyleDisabled()
		return st, st
	case selected:
		st := theme.StyleMenuSelected()
		return st, st
	default:
		return theme.StyleMenuBar(),
			theme.StyleMenuDim()
	}
}

// drawMenuRow renders one row of a menu box at (x,y) spanning w columns: a
// divider line, or a label plus right-aligned suffix styled per menuRowStyles,
// from the state menuBox.measure evaluated for this draw. mark is the box's
// mark column width (menuMarkColumn): the label starts after it, and a
// checked toggle draws MenuTick in it.
func drawMenuRow(s tcell.Screen, x, y, w, mark int, item MenuItem, st menuRowState, selected bool, borderStyle tcell.Style) {
	if item.Divider {
		for cx := x + 1; cx < x+w-1; cx++ {
			core.PutRune(s, cx, y, '─', borderStyle)
		}
		core.PutRune(s, x, y, '├', borderStyle)
		core.PutRune(s, x+w-1, y, '┤', borderStyle)
		return
	}
	labelStyle, suffixStyle := menuRowStyles(st.enabled, selected)
	core.FillRect(s, core.Rect{X: x + 1, Y: y, W: w - 2, H: 1}, ' ', labelStyle)
	if st.checked && mark > 0 {
		core.DrawTextClipped(s, x+2, y, w-4, labelStyle, MenuTick)
	}
	core.DrawTextClipped(s, x+2+mark, y, w-4-mark, labelStyle, item.Label)
	if st.suffix != "" {
		sx := x + w - 1 - core.DisplayWidth(st.suffix) - 1
		core.DrawText(s, sx, y, suffixStyle, st.suffix)
	}
}
