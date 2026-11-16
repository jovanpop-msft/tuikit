package core

// ClipboardTarget is implemented by any widget that can take part in
// Copy/Cut/Paste (widgets.InputField, editor.Editor, controls.DataGrid,
// propsheet.PropertySheet). It lives in core so a container can hand its focused
// widget back across a package boundary (see ClipboardHost); identical
// interfaces declared in two packages would not be interchangeable.
//
// Cut returns only text it actually removed. A read-only target, or one that
// refuses to hand its text over (a password field), returns "" and changes
// nothing; a host treats "" as "nothing was cut" and leaves the clipboard alone.
//
// The Can methods say which actions would do something right now. A host
// building a right-click Cut/Copy/Paste menu (controls.ClipboardMenuItems) reads
// them to grey what would be a no-op and to leave Cut and Paste off a read-only
// target entirely, so the menu never offers an edit the target would silently
// refuse. They are part of the interface, not an optional companion, so a new
// read-only target cannot be taken as editable by forgetting them.
//
// CanCopy is true when Copy would hand over non-empty text (a password field
// never does); CanCut when Cut would remove some; CanPaste when Paste would
// insert (the target is editable — whether the clipboard holds text is the
// host's to know); CanSelectAll when SelectAll would select something (a
// property sheet's flat row is copied whole and has no selection to make).
type ClipboardTarget interface {
	HasSelection() bool
	SelectedText() string
	Cut() string
	Paste(text string)
	SelectAll()

	CanCopy() bool
	CanCut() bool
	CanPaste() bool
	CanSelectAll() bool
}

// ClipboardHost is implemented by a container (a dialog, a property sheet) that
// owns ClipboardTargets and knows which has keyboard focus.
//
// Returning nil is a real answer: "nothing here can take this key", and the
// application must then do nothing rather than look elsewhere. The application
// resolves Ctrl+C/X/V by asking the frontmost dialog, not by enumerating dialog
// types (Ctrl+X in the Find dialog once cut the query editor's text invisibly
// behind it).
//
// A dialog with no text entry deliberately does not implement this, which makes
// the clipboard inert while it is open.
type ClipboardHost interface {
	FocusedClipboardTarget() ClipboardTarget
}

// ClipboardTargetTokener is an optional companion to ClipboardHost, for a host
// that answers FocusedClipboardTarget with itself and resolves the real field per
// call. The token is an opaque identity of the current field: compare with ==,
// never interpret.
//
// An application that guards an asynchronous clipboard read by checking the
// target is still the active one always passes that check against such a host,
// so a paste aimed at one row but delivered after focus moved lands in another. The token lets the
// guard see inside the host without exposing its rows. A host returning a
// distinct target per field needs none of this.
type ClipboardTargetTokener interface {
	ClipboardTargetToken() any
}

// ClipboardEditHandler is an optional companion to ClipboardHost, for a host that
// must follow up a clipboard edit of one of its fields (e.g. re-filtering an
// autocomplete list). Called after a Cut or Paste changed target, never a Copy.
// The edited target is passed because the application only knows "something
// changed", and acting on that unconditionally would react to a paste into an
// unrelated field as if the watched one were typed in.
type ClipboardEditHandler interface {
	ClipboardEdited(target ClipboardTarget)
}

// ClipboardFocuser is an optional companion to ClipboardHost: a right-click's
// half of click-to-focus. FocusClipboardTargetAt moves keyboard focus to the
// text field (or copyable read-only value) at screen cell (x, y) and reports
// whether there was one; FocusedClipboardTarget then returns it. It must not
// move the caret or drop the selection — the menu it precedes acts on that
// selection — and must not act on anything else at that point: a right-click on
// a button or a list item is not this call's to handle, so it returns false
// and the host routes the click as usual. It must also return false while one
// of the host's own overlays is open (a dropdown's list, a completion popup):
// the overlay is drawn over the fields beneath it, so a geometric hit test
// would focus a field the user cannot see under the pointer. The application
// asks before routing the click to the host, so the host is the only one that
// knows.
//
// An application opens its Cut/Copy/Paste menu (see controls.ClipboardMenuItems)
// when this returns true, without each dialog wiring a right-click handler per
// field.
type ClipboardFocuser interface {
	FocusClipboardTargetAt(x, y int) bool
}
