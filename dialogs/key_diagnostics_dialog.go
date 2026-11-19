package dialogs

import (
	"fmt"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
)

// ---------------------------------------------------------------------------
// KeyDiagnosticsDialog — a live trace of the key and mouse events tcell decodes
// ---------------------------------------------------------------------------

// KeyDiagnosticsMax caps the diagnostics log so a long session doesn't grow it
// without bound; oldest entries are dropped first.
const KeyDiagnosticsMax = 200

// KeyDiagnosticsSpec configures a KeyDiagnosticsDialog.
type KeyDiagnosticsSpec struct {
	// OnCopy receives the whole log, newest line first, when the Copy button is
	// pressed. The host writes it to the clipboard (and reports it): tuikit's
	// dialogs never touch the clipboard themselves. Nil leaves Copy inert.
	OnCopy func(text string)
}

// KeyDiagnosticsDialog is a small modal that shows what tcell decoded for every
// key event while it's open: the raw Key/Modifiers/rune values plus ev.Name()'s
// decode (e.g. "Shift+Left"). It turns "a shortcut doesn't seem to work" into a
// 10-second check of whether the terminal delivers the modifier bits the
// application expects. Mouse events are logged the same way, since a terminal
// can keep Shift+click for itself. The Shift+Arrow selection handling of
// controls/editor and widgets.InputField is what it most often debugs.
//
// The host feeds it: while it is visible, call RecordKey for every key event
// and RecordMouse for every mouse event, before anything else consumes them
// (clipboard shortcuts and the keys that close the dialog included).
//
// Newest entries are prepended, so the most recent key press is the first line.
//
// The log is a LogDialog: a read-only editor rather than a hand-rolled line
// list, so it can be selected and copied — what the dialog shows is the text
// pasted into a bug report. Its prepend-only sync matters doubly here: the
// dialog records the very keys used to copy from it (Ctrl+A is itself
// recorded), so a SetText rebuild on the next frame would drop the selection
// and the following Ctrl+C would copy nothing. Each Show starts a fresh session
// (ResetOnShow).
type KeyDiagnosticsDialog struct {
	*LogDialog
	onCopy func(text string)

	// lastMouse is the buttons/modifiers of the last mouse event recorded, so
	// RecordMouse can drop the resends tcell sends throughout a drag. -1 means
	// nothing recorded yet, which no real event matches.
	lastMouse int
}

// NewKeyDiagnosticsDialog creates the key diagnostics dialog.
func NewKeyDiagnosticsDialog(spec KeyDiagnosticsSpec) *KeyDiagnosticsDialog {
	d := &KeyDiagnosticsDialog{onCopy: spec.OnCopy, lastMouse: -1}
	d.LogDialog = NewLogDialog(LogDialogSpec{
		Title: Strings.KeyDiagnosticsTitle, W: 70, H: 24,
		Max:   KeyDiagnosticsMax,
		Empty: Strings.KeyDiagnosticsEmpty,
		// No gutter: line numbers say nothing about a key log, and every column
		// they take comes off lines that are already 60-odd wide.
		Gutter:      false,
		ResetOnShow: true,
		Buttons:     []string{Strings.Copy, Strings.Close}, // Close, the last, is the default
		OnButton:    func(int) { d.CopyAll() },             // Copy is the only non-Close button
	})
	return d
}

// RecordKey appends a formatted description of ev to the log. Call only while
// the dialog is visible; recording unseen keys is wasted work.
func (d *KeyDiagnosticsDialog) RecordKey(ev *tcell.EventKey) {
	d.Record(fmt.Sprintf("%-20s  Key=%-4d Mod=%-3d Str=%-8q Repeat=%d",
		ev.Name(), ev.Key(), ev.Modifiers(), ev.Str(), ev.Repeat()))
}

// RecordMouse appends a description of a mouse event: the buttons, the
// modifiers, and where it landed. Same call rule as RecordKey.
//
// A mouse gesture is not one event. tcell resends the held button for every
// pixel of a drag, and a hundred identical lines would bury the press that
// started it, so only a change of buttons or modifiers is recorded (a press, a
// release, a wheel tick). The position is wherever that change happened.
//
// A modifier the terminal keeps for itself is invisible from inside the app:
// xfce4-terminal (VTE) handles Shift+click as its own text selection while
// mouse reporting is on and never forwards it, which reads like a broken
// Shift+click. One line in this log, or none, tells the two apart.
func (d *KeyDiagnosticsDialog) RecordMouse(ev *tcell.EventMouse) {
	state := int(ev.Buttons())<<8 | int(ev.Modifiers())
	if state == d.lastMouse {
		return
	}
	d.lastMouse = state
	x, y := ev.Position()
	d.Record(fmt.Sprintf("%-20s  Btn=%-4d Mod=%-3d At=%d,%d",
		core.MouseEventName(ev), ev.Buttons(), ev.Modifiers(), x, y))
}

// Show starts a fresh session: LogDialog empties the log (ResetOnShow), and the
// mouse collapse forgets the previous showing's last event.
func (d *KeyDiagnosticsDialog) Show() {
	d.lastMouse = -1
	d.LogDialog.Show()
}

// CopyAll hands the whole log to OnCopy — the Copy button. The keyboard route
// (Ctrl+A then Ctrl+C) records two more lines before it completes, and on a
// narrow terminal the selection is fiddly. UI goroutine only.
//
// It reads the editor's text rather than selecting all and copying the
// selection: a read-only Editor ignores every plain rune key without clearing
// a selection, so a select-all left behind would sit over the log while the
// keys this dialog exists to decode arrive above it, and the next Ctrl+C would
// copy the stale whole-log selection. The user's own selection is untouched.
func (d *KeyDiagnosticsDialog) CopyAll() {
	if d.onCopy == nil || d.Len() == 0 {
		return
	}
	d.Sync() // what is copied is what the dialog shows, the lines since the last frame included
	d.onCopy(d.Editor().Text())
}
