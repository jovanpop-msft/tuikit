package dialogs

import (
	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// ---------------------------------------------------------------------------
// MessageDialog — static text lines over a row of the host's own buttons
// ---------------------------------------------------------------------------

// MessageLine is one line of a MessageDialog's text.
type MessageLine struct {
	Text string
	// Style draws the line; the zero Style means theme.StyleDialog().
	Style tcell.Style
	// Indent is the line's extra columns from the left margin.
	Indent int
	// Wrap word-wraps Text to the content width at draw time, taking as many
	// rows as it needs of those left. Unwrapped lines clip.
	Wrap bool
}

// MessageDialogSpec is what a MessageDialog shows and does.
type MessageDialogSpec struct {
	Title   string
	Lines   []MessageLine
	Buttons []string
	// Focus is the button focused when the dialog opens.
	Focus int
	// OnButton runs when button i is clicked or Enter-ed. Nil does nothing.
	OnButton func(i int)
	// OnEscape runs on Escape; nil hides the dialog.
	OnEscape func()
	// W and H are the dialog's size (see ModalDialog.SetSize).
	W, H int
}

// MessageDialog is static text lines, each with its own style, above a row of
// buttons the host names and answers. Unlike AlertDialog (one OK) and
// ConfirmDialog (a fixed answer set), the buttons and Escape do what the host
// says. Its keyboard is ButtonsKey and its mouse ConsumeOutsideClick plus
// ButtonClicked: everything else is swallowed, the dialog being modal.
//
// A host embeds it by value and calls InitMessage from its constructor, or
// uses NewMessageDialog. SetLines and SetButtons change the content while it
// is open (a loading state replaced by a result, say).
type MessageDialog struct {
	ModalDialog
	lines    []MessageLine
	buttons  []string
	btnFocus int
	onButton func(i int)
	onEscape func()
}

// NewMessageDialog creates a MessageDialog from spec.
func NewMessageDialog(spec MessageDialogSpec) *MessageDialog {
	d := new(MessageDialog{})
	d.InitMessage(spec)
	return d
}

// InitMessage sets the dialog up from spec. Call it from the embedding type's
// constructor.
func (d *MessageDialog) InitMessage(spec MessageDialogSpec) {
	d.InitModal(spec.Title, spec.W, spec.H)
	d.lines = spec.Lines
	d.SetButtons(spec.Buttons, spec.Focus)
	d.onButton, d.onEscape = spec.OnButton, spec.OnEscape
}

// SetLines replaces the dialog's text.
func (d *MessageDialog) SetLines(lines []MessageLine) { d.lines = lines }

// Lines returns the dialog's text.
func (d *MessageDialog) Lines() []MessageLine { return d.lines }

// SetButtons replaces the button row and focuses button focus (clamped).
func (d *MessageDialog) SetButtons(labels []string, focus int) {
	d.buttons = labels
	d.btnFocus = core.Clamp(focus, 0, max(0, len(labels)-1))
}

// Buttons returns the button row's labels.
func (d *MessageDialog) Buttons() []string { return d.buttons }

// ButtonFocus is the index of the focused button.
func (d *MessageDialog) ButtonFocus() int { return d.btnFocus }

// Draw renders the dialog.
func (d *MessageDialog) Draw(s tcell.Screen) {
	if !d.visible {
		return
	}
	d.DrawBase(s)
	inner := d.InnerRect()
	// Rows between the top padding row and DrawSeparator's line (one above
	// ButtonRowY), which is drawn after the text and would paint over it.
	y, bottom := inner.Y+1, d.ButtonRowY()-1
	for _, l := range d.lines {
		if y >= bottom {
			break
		}
		st := l.Style
		if st == (tcell.Style{}) {
			st = theme.StyleDialog()
		}
		x, w := inner.X+1+l.Indent, inner.W-2-l.Indent
		if !l.Wrap {
			core.DrawTextClipped(s, x, y, w, st, l.Text)
			y++
			continue
		}
		for _, row := range core.WrapTextLimit(l.Text, w, bottom-y) {
			core.DrawTextClipped(s, x, y, w, st, row)
			y++
		}
	}
	d.DrawSeparator(s)
	d.DrawButtons(s, d.buttons, d.btnFocus)
}

func (d *MessageDialog) fire() {
	if d.onButton != nil && d.btnFocus < len(d.buttons) {
		d.onButton(d.btnFocus)
	}
}

func (d *MessageDialog) escape() {
	if d.onEscape != nil {
		d.onEscape()
		return
	}
	d.Hide()
}

// HandleKey handles keyboard events (ButtonsKey).
func (d *MessageDialog) HandleKey(ev *tcell.EventKey) bool {
	if !d.visible {
		return false
	}
	return ButtonsKey(ev, &d.btnFocus, len(d.buttons), nil, d.fire, d.escape)
}

// HandleMouse handles mouse events: a click on a button focuses and fires it.
func (d *MessageDialog) HandleMouse(ev *tcell.EventMouse) bool {
	if !d.visible {
		return false
	}
	if d.ConsumeOutsideClick(ev) {
		return true
	}
	if i := d.ButtonClicked(ev, d.buttons); i >= 0 {
		d.btnFocus = i
		d.fire()
	}
	return true
}
