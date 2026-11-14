package controls

import "github.com/radix29/tuikit/core"

// ClipboardActions are a host's handlers behind ClipboardMenuItems' entries.
// The host owns the clipboard itself (an OS tool, OSC 52), so the menu only
// calls back; each handler should act on the same target the host's
// Ctrl+X/C/V/A would, which the right-click has just focused. A nil handler
// leaves its entry out.
type ClipboardActions struct {
	Cut, Copy, Paste, SelectAll func()
}

// ClipboardMenuItems builds the standard right-click menu for a text field
// (core.ClipboardTarget): Cut, Copy, Paste, a divider and Select All, for a
// ContextMenu.Show.
//
// A target that is read-only now — a core.ClipboardAccess answering CanPaste
// false (a read-only editor, a flat property row, a grid's value viewer) — gets
// Copy and Select All only: Cut and Paste are not offered on what can never take
// them, rather than shown greyed on every read-only view. Otherwise each entry is
// greyed while it would do nothing: Copy and Cut without a selection (or, for a
// password field, ever), via core.ClipboardAccess where the target has it and
// HasSelection where it doesn't. Paste stays enabled: whether the clipboard
// holds text is not known until it is read.
//
// Select All is greyed on an empty field, and left off a read-only target
// that has nothing to select (CanSelectAll false: a value copied whole).
//
// Enabled is evaluated when the menu draws, against target, so the entries
// follow the selection while the menu is open.
func ClipboardMenuItems(target core.ClipboardTarget, act ClipboardActions) []MenuItem {
	access, _ := target.(core.ClipboardAccess)
	canCopy := target.HasSelection
	canCut := target.HasSelection
	var canSelectAll func() bool
	editable := true
	if access != nil {
		canCopy, canCut, canSelectAll = access.CanCopy, access.CanCut, access.CanSelectAll
		editable = access.CanPaste()
		if !canSelectAll() && !editable {
			// A read-only value copied whole (a flat property row) has nothing for
			// Select All to select: a menu item that does nothing is not offered.
			act.SelectAll = nil
		}
	}
	var items []MenuItem
	add := func(label, key string, action func(), enabled func() bool) {
		if action != nil {
			items = append(items, MenuItem{Label: label, Shortcut: key, Action: action, Enabled: enabled})
		}
	}
	if editable {
		add(Strings.Cut, Strings.CutKey, act.Cut, canCut)
	}
	add(Strings.Copy, Strings.CopyKey, act.Copy, canCopy)
	if editable {
		add(Strings.Paste, Strings.PasteKey, act.Paste, nil)
	}
	if act.SelectAll != nil && len(items) > 0 {
		items = append(items, MenuItem{Divider: true})
	}
	add(Strings.SelectAll, Strings.SelectAllKey, act.SelectAll, canSelectAll)
	return items
}
