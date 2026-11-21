package dialogs

import (
	"sync"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/controls/editor"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// ---------------------------------------------------------------------------
// LogDialog — a read-only, newest-first log in a selectable editor
// ---------------------------------------------------------------------------

// LogDialogSpec configures a LogDialog.
type LogDialogSpec struct {
	Title string
	W, H  int

	// Max is the number of lines (Record calls) kept; the oldest are dropped
	// first. A line holding newlines is still one. Zero or less keeps
	// everything.
	Max int

	// Empty is drawn on the first row while the log holds no lines.
	Empty string

	// Gutter shows the editor's line numbers. A log whose lines are already
	// wide (a key trace) is better off without them.
	Gutter bool

	// ResetOnShow empties the log on every Show, so each opening is a fresh
	// session (a key trace). Otherwise the log accumulates across showings (a
	// status history).
	ResetOnShow bool

	// Buttons is the button row, in order; nil means {"Close"}. The last is the
	// default (highlighted) and closes the dialog without calling OnButton.
	Buttons []string

	// OnButton runs button b, any but the last, on a click. The dialog stays
	// open.
	OnButton func(b int)
}

// LogDialog is a modal showing a capped, newest-first log in a read-only
// editor.Editor, so the text is selectable and copyable (it is the host's
// core.ClipboardHost target while on top). Record is safe from any goroutine;
// the editor catches up on the UI goroutine, in Show and before each Draw.
//
// Keys go to the editor first (arrows, PgUp/PgDn, Home/End, Shift-selection,
// Ctrl+A); Escape and Enter, which a read-only editor refuses, close the
// dialog.
type LogDialog struct {
	ModalDialog
	spec    LogDialogSpec
	buttons []string
	editor  *editor.Editor

	// mu guards feed: Record may run on a background goroutine (a status
	// history fed by a loader) while the UI goroutine draws. The editor is
	// only touched on the UI goroutine.
	mu sync.Mutex

	// feed holds the log and the lines recorded since the last sync, so a sync
	// while the dialog is open prepends just those (editor.LogFeed) instead of
	// rebuilding with SetText, which resets cursor, scroll and selection. A full
	// rebuild shipped twice in goSSMS and dropped the selection on every new
	// line: in a status history the "Copied to clipboard" of the user's own
	// Ctrl+C was itself a line, so a second Copy copied nothing, and a
	// scrolled-down view snapped back to line 1 whenever a background load
	// recorded a status; in a key trace the Ctrl+A that made the selection was
	// itself recorded.
	feed *editor.LogFeed
}

// NewLogDialog creates a LogDialog from spec.
func NewLogDialog(spec LogDialogSpec) *LogDialog {
	buttons := spec.Buttons
	if len(buttons) == 0 {
		buttons = []string{Strings.Close}
	}
	d := &LogDialog{spec: spec, buttons: buttons, feed: editor.NewLogFeed(spec.Max)}
	d.InitModal(spec.Title, spec.W, spec.H)
	d.editor = editor.NewEditor(nil)
	d.editor.SetReadOnly(true)
	d.editor.Focus(true)
	d.editor.SetGutterVisible(spec.Gutter)
	return d
}

// Record prepends line to the log. Newest-first, so the most recent line is
// the first on open, and a sync while open is a prepend that keeps a viewer at
// the top seeing the newest line.
//
// Safe from any goroutine, and it deliberately does no more than record:
// updating the editor is the UI goroutine's job, done by Show and Draw.
// Calling Editor.SetText here would write the widget the UI goroutine is
// drawing. Deferring it also keeps it off the hot path: a busy log pays once
// per frame the dialog is open, not once per line nobody is looking at.
func (d *LogDialog) Record(line string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.feed.Record(line)
}

// Len reports how many lines are held.
func (d *LogDialog) Len() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.feed.Len()
}

// Lines returns a copy of the held lines, newest first.
func (d *LogDialog) Lines() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.feed.Lines()
}

// Dirty reports whether lines were recorded that the editor does not show yet.
func (d *LogDialog) Dirty() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.feed.Dirty()
}

// Editor is the read-only editor holding the log, for a host that selects or
// copies from it itself (a Copy button: SelectAll, copy, ClearSelection).
func (d *LogDialog) Editor() *editor.Editor { return d.editor }

// Sync prepends the lines recorded since the last sync to the editor, keeping
// its cursor, selection and scroll on the text they were on (see feed), and
// trims it to the lines the retained records take. Draw calls it; call it directly only to bring the editor up
// to date without drawing. UI goroutine only.
//
// LogFeed.Sync is not used: it would hold mu across the editor update, the
// expensive half, and Record must not block behind it on a background
// goroutine. The pending lines are taken under the lock and applied outside.
func (d *LogDialog) Sync() {
	d.mu.Lock()
	text, keep, ok := d.feed.TakePending()
	d.mu.Unlock()
	if ok {
		d.editor.PrependLines(text, keep)
	}
}

// Show replaces the editor's content with the whole log, resetting cursor,
// scroll and selection, so a fresh open starts at the top on the newest line;
// with ResetOnShow it empties the log first. Then it displays the dialog.
//
// The editor is replaced with SetText rather than synced: a sync would keep
// the previous showing's text (or, after a reset, the old session) below the
// new, and a selection left from the last showing.
func (d *LogDialog) Show() {
	d.mu.Lock()
	if d.spec.ResetOnShow {
		d.feed.Reset()
	}
	text := d.feed.TakeAll()
	d.mu.Unlock()
	d.editor.SetText(text)
	d.ModalDialog.Show()
}

// Draw renders the dialog. Lines recorded since the last frame are picked up
// here rather than in Record, which may be on a background goroutine.
func (d *LogDialog) Draw(s tcell.Screen) {
	if !d.visible {
		return
	}
	d.Sync()

	d.DrawBase(s)
	inner := d.InnerRect()
	dataH := inner.H - 2 // the editor ends on the row above the button row

	if d.Len() == 0 {
		p := theme.Active()
		core.DrawText(s, inner.X+1, inner.Y+1, theme.On(p.DialogBg, p.DialogTextDim), d.spec.Empty)
	} else {
		d.editor.SetBounds(inner.X, inner.Y, inner.W, dataH)
		d.editor.Draw(s)
	}
	d.DrawButtons(s, d.buttons, len(d.buttons)-1)
}

// HandleKey processes keyboard events: the editor first, then Escape/Enter
// close. Every key is consumed while visible.
func (d *LogDialog) HandleKey(ev *tcell.EventKey) bool {
	if !d.visible {
		return false
	}
	if d.editor.HandleKey(ev) {
		return true
	}
	switch ev.Key() {
	case tcell.KeyEscape, tcell.KeyEnter:
		d.Hide()
	}
	return true
}

// HandleMouse handles mouse events: the button row, then the editor.
func (d *LogDialog) HandleMouse(ev *tcell.EventMouse) bool {
	if !d.visible {
		return false
	}
	// A release must reach the editor even when it lands outside the dialog
	// (consumed below) — otherwise its next press is swallowed as a
	// continuation of the stale drag. Editor.HandleMouse returns false on
	// ButtonNone, so this has no effect beyond resetting the latch.
	if ev.Buttons() == tcell.ButtonNone {
		d.editor.HandleMouse(ev)
	}
	if d.ConsumeOutsideClick(ev) {
		return true
	}
	if b := d.ButtonClicked(ev, d.buttons); b >= 0 {
		if b == len(d.buttons)-1 {
			d.Hide()
		} else if d.spec.OnButton != nil {
			d.spec.OnButton(b)
		}
		return true
	}
	if ev.Buttons() == tcell.ButtonNone {
		return true
	}
	d.editor.HandleMouse(ev)
	return true
}

// FocusClipboardTargetAt implements core.ClipboardFocuser: the editor is the
// only field, always focused, so a right-click on it is all it takes.
func (d *LogDialog) FocusClipboardTargetAt(x, y int) bool {
	return d.editor.Bounds().Contains(x, y)
}

// FocusedClipboardTarget implements core.ClipboardHost. The editor is
// read-only, so Copy works and Cut/Paste are no-ops on it. Without it a host
// routes Ctrl+C past the dialog to whatever is underneath.
func (d *LogDialog) FocusedClipboardTarget() core.ClipboardTarget {
	return d.editor
}
