package dialogs

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
	"github.com/radix29/tuikit/widgets"
)

// ---------------------------------------------------------------------------
// TypedConfirmDialog — retype-to-confirm
// ---------------------------------------------------------------------------

// typedConfirmFocus tracks which element has focus: the input, then one
// position per button (focus-1 indexes d.buttons), so a Script button adds a
// position rather than renumbering.
type typedConfirmFocus int

const (
	typedConfirmFocusInput typedConfirmFocus = iota
	typedConfirmFocusConfirm
	typedConfirmFocusCancel
)

const (
	typedConfirmW = 60
	typedConfirmH = 10
)

// TypedConfirmDialog gates an action behind retyping a short confirmation
// string rather than a plain Yes/No, for actions where a misclick shouldn't
// suffice. Confirm fires only once the typed text matches Required,
// case-insensitively.
type TypedConfirmDialog struct {
	ModalDialog
	message  string
	msgLines []string
	required string
	// promptLines is the "Type … to confirm" prompt, wrapped to the dialog: the
	// user must read the whole required text to type it, so it is never clipped.
	promptLines []string
	status      string
	input       *widgets.InputField
	focus       typedConfirmFocus

	// drag owns the text-selection gesture a press in input starts; see
	// FieldGesture.
	drag FieldGesture

	// buttons is what the showing renders and hit-tests; answers is what each
	// means, parallel rather than the button index (as in ConfirmDialog).
	buttons []string
	answers []ConfirmAnswer

	// onAnswer is the showing's handler. OnConfirm is the two-button form's.
	onAnswer  func(ConfirmAnswer)
	OnConfirm func(confirmed bool)
}

// NewTypedConfirmDialog creates a TypedConfirmDialog.
func NewTypedConfirmDialog() *TypedConfirmDialog {
	d := &TypedConfirmDialog{}
	d.InitModal(Strings.ConfirmTitle, typedConfirmW, typedConfirmH)
	return d
}

// ShowTypedConfirm shows the dialog: message explains the action, required is
// the exact text (case-insensitive, surrounding whitespace ignored) the user
// must type before Confirm proceeds. Sizing follows fitMessage, widened for the
// prompt and input (see layout): one line where it fits within 2/3 of the screen
// width, else wrapped and taller, pushing the prompt and input down.
func (d *TypedConfirmDialog) ShowTypedConfirm(title, message, required string, onConfirm func(bool)) {
	d.OnConfirm = onConfirm
	d.show(title, message, required, []string{Strings.Confirm, Strings.Cancel},
		[]ConfirmAnswer{ConfirmYes, ConfirmNo}, func(a ConfirmAnswer) {
			if d.OnConfirm != nil {
				d.OnConfirm(a == ConfirmYes)
			}
		})
}

// ShowTypedConfirmScript is ShowTypedConfirm with a third button answering
// ConfirmScript, the counterpart of ConfirmDialog.ShowConfirmScript for a delete
// serious enough to be typed out that the user may want to read as SQL first.
// Script is not gated on the typed text: it runs nothing for the retyping to
// protect. Escape still answers No.
func (d *TypedConfirmDialog) ShowTypedConfirmScript(title, message, required string, onAnswer func(ConfirmAnswer)) {
	d.OnConfirm = nil
	d.show(title, message, required, []string{Strings.Confirm, Strings.Cancel, Strings.Script},
		[]ConfirmAnswer{ConfirmYes, ConfirmNo, ConfirmScript}, onAnswer)
}

func (d *TypedConfirmDialog) show(title, message, required string, buttons []string,
	answers []ConfirmAnswer, onAnswer func(ConfirmAnswer)) {
	d.SetTitle(title)
	d.message = message
	d.required = required
	d.status = ""
	d.input = widgets.NewInputField("", max(20, core.DisplayWidth(required)+16), false)
	d.focus = typedConfirmFocusInput
	d.syncFocus()
	// input is rebuilt above, so a gesture held from the last showing points at a
	// discarded widget and would route every click there.
	d.drag.Clear()
	d.buttons, d.answers = buttons, answers
	d.onAnswer = onAnswer
	d.layout()
	d.ModalDialog.Show()
}

// Message returns the question the dialog asks.
func (d *TypedConfirmDialog) Message() string { return d.message }

// Relayout re-wraps the message and prompt for the new screen width, then
// recentres.
func (d *TypedConfirmDialog) Relayout() { d.layout() }

// layout sizes the dialog to the widest of the message, the prompt and the input,
// capped at fitMessage's fraction of the screen. Sizing from the message alone let
// a long required name (SQL Server names run to 128 characters) push the input
// through the right border and clip the prompt mid-name, so the user could not
// read what this gate asks them to type. Past the cap the input is narrowed (it
// scrolls) and the prompt wraps onto extra rows, which push the input down as
// extra message lines do.
func (d *TypedConfirmDialog) layout() {
	prompt := fmt.Sprintf(Strings.TypeToConfirm, d.required)
	inputW := max(20, core.DisplayWidth(d.required)+16)
	// The input draws inputW+2 columns: its brackets.
	minW := max(typedConfirmW, core.DisplayWidth(prompt)+messageBoxOverhead, inputW+2+messageBoxOverhead)
	sw := 0
	if d.screen != nil {
		sw, _ = d.screen.Size()
	}
	if sw > 0 {
		minW = min(minW, max(typedConfirmW, sw*maxMessageWidthNum/maxMessageWidthDen))
	}
	// The width does not depend on the height, so the first fit only finds it; the
	// second caps the message's lines with the prompt's extra rows counted.
	w, _, _ := d.fitMessage(d.message, minW, typedConfirmH)
	if sw > 0 {
		w = min(w, sw) // recentre's clamp
	}
	contentW := w - messageBoxOverhead
	d.promptLines = wrapPrompt(prompt, d.required, contentW)
	w, h, lines := d.fitMessage(d.message, minW, typedConfirmH+len(d.promptLines)-1)
	d.msgLines = lines
	d.input.SetWidth(min(inputW, contentW-2))
	d.SetSize(w, h)
}

// wrapPrompt wraps prompt to w columns. WrapText breaks at spaces and hard-breaks
// an unbroken token, but it collapses whitespace runs, so a required text with a
// doubled, leading or trailing space (legal in a SQL Server name) is instead cut
// at exact grapheme widths: a collapsed space would show the user a name they
// cannot type to a match.
func wrapPrompt(prompt, required string, w int) []string {
	if w <= 0 || core.DisplayWidth(prompt) <= w {
		return []string{prompt}
	}
	if strings.Join(strings.Fields(required), " ") == required {
		return core.WrapText(prompt, w)
	}
	var lines []string
	runes := []rune(prompt)
	start, col := 0, 0
	for i := 0; i < len(runes); {
		end, gw := core.GraphemeAt(runes, i)
		if col+gw > w && i > start {
			lines = append(lines, string(runes[start:i]))
			start, col = i, 0
		}
		col += gw
		i = end
	}
	return append(lines, string(runes[start:]))
}

func (d *TypedConfirmDialog) syncFocus() {
	d.input.Focus(d.focus == typedConfirmFocusInput)
}

// matched reports whether the typed text answers required, case-insensitively,
// with surrounding whitespace ignored on both sides: trimming only the input made
// a required name with a leading or trailing space (legal in a SQL Server name)
// impossible to confirm. An exact match still counts, so a required text that is
// all whitespace can be confirmed without an empty input matching it.
func (d *TypedConfirmDialog) matched() bool {
	if d.required == "" {
		return false
	}
	v := d.input.Value()
	if strings.EqualFold(v, d.required) {
		return true
	}
	req := strings.TrimSpace(d.required)
	return req != "" && strings.EqualFold(strings.TrimSpace(v), req)
}

// finish resolves the dialog: every answer proceeds, but a confirm whose typed
// text doesn't match is refused in place with a status message rather than
// treated as a cancel, so the user must fix the input or explicitly back out.
func (d *TypedConfirmDialog) finish(answer ConfirmAnswer) {
	if answer == ConfirmYes && !d.matched() {
		d.status = Strings.TextMismatch
		return
	}
	d.Hide()
	// Read and cleared before it runs: it commonly opens something that can route
	// back here, and a stale handler would fire twice.
	onAnswer := d.onAnswer
	d.onAnswer = nil
	if onAnswer != nil {
		onAnswer(answer)
	}
}

// answerAt is what the button at index i answers.
func (d *TypedConfirmDialog) answerAt(i int) ConfirmAnswer {
	if i < 0 || i >= len(d.answers) {
		return ConfirmNo
	}
	return d.answers[i]
}

// Draw renders the message, the required confirmation text, the input, and
// the showing's button row.
func (d *TypedConfirmDialog) Draw(s tcell.Screen) {
	if !d.visible {
		return
	}
	d.DrawBase(s)
	msgStyle := theme.StyleDialog()
	inner := d.InnerRect()
	lx, w := inner.X+1, inner.W-2

	for i, line := range d.msgLines {
		x := lx + core.CenterOffset(w, core.DisplayWidth(line))
		core.DrawTextClipped(s, x, inner.Y+1+i, w, msgStyle, line)
	}
	extra := len(d.msgLines) - 1
	for i, line := range d.promptLines {
		core.DrawTextClipped(s, lx, inner.Y+2+extra+i, w, msgStyle, line)
	}
	extra += len(d.promptLines) - 1
	d.input.SetBounds(lx, inner.Y+3+extra)
	d.input.Draw(s)

	if d.status != "" {
		errStyle := theme.StyleDialogError()
		core.DrawTextClipped(s, lx, d.ButtonRowY()-2, w, errStyle, d.status)
	}

	d.DrawSeparator(s)
	activeIdx := -1
	if d.focus != typedConfirmFocusInput {
		activeIdx = int(d.focus) - 1
	}
	d.DrawButtons(s, d.buttons, activeIdx)
}

// HandleKey routes keyboard events.
func (d *TypedConfirmDialog) HandleKey(ev *tcell.EventKey) bool {
	if !d.visible {
		return false
	}
	n := typedConfirmFocus(len(d.buttons) + 1)
	switch ev.Key() {
	case tcell.KeyEscape:
		d.finish(ConfirmNo)
		return true
	case tcell.KeyTab:
		d.focus = (d.focus + 1) % n
		d.syncFocus()
		return true
	case tcell.KeyBacktab:
		d.focus = (d.focus + n - 1) % n
		d.syncFocus()
		return true
	case tcell.KeyEnter:
		// Enter with the keyboard in the input answers the question the dialog is
		// about.
		if d.focus == typedConfirmFocusInput {
			d.finish(ConfirmYes)
		} else {
			d.finish(d.answerAt(int(d.focus) - 1))
		}
		return true
	}
	if d.focus == typedConfirmFocusInput {
		d.input.HandleKey(ev)
	}
	return true
}

// HandleMouse routes mouse events.
func (d *TypedConfirmDialog) HandleMouse(ev *tcell.EventMouse) bool {
	if !d.visible {
		return false
	}
	// A release must reach d.input even when it lands outside the dialog (consumed
	// below), or its next press is swallowed as a continuation of the stale drag.
	if ev.Buttons() == tcell.ButtonNone {
		d.drag.Release(ev)
	}
	if d.ConsumeOutsideClick(ev) {
		return true
	}
	// The gesture belongs to the field that claimed the press, so motion is
	// replayed there without hit-testing, ahead of ButtonClicked, which would answer
	// the confirmation when a selection drag wandered onto the button row.
	if d.drag.Replay(ev) {
		return true
	}
	if i := d.ButtonClicked(ev, d.buttons); i >= 0 {
		d.finish(d.answerAt(i))
		return true
	}
	if ev.Buttons() == tcell.ButtonNone {
		return true
	}
	// A visible modal owns every event: a wheel or right-click it has no use for
	// must not reach the screen beneath it.
	if ev.Buttons() != tcell.Button1 {
		return true
	}
	mx, my := ev.Position()
	if d.input.HitTest(mx, my) {
		d.focus = typedConfirmFocusInput
		d.syncFocus()
		d.drag.Claim(d.input, ev)
	}
	return true
}

// FocusClipboardTargetAt implements core.ClipboardFocuser: a right-click on the
// confirmation field focuses it, caret and selection untouched.
func (d *TypedConfirmDialog) FocusClipboardTargetAt(x, y int) bool {
	if !d.input.HitTest(x, y) {
		return false
	}
	d.focus = typedConfirmFocusInput
	d.syncFocus()
	return true
}

// FocusedClipboardTarget implements core.ClipboardHost: the confirmation
// field while it has focus, nothing while a button does.
func (d *TypedConfirmDialog) FocusedClipboardTarget() core.ClipboardTarget {
	if d.focus == typedConfirmFocusInput {
		return d.input
	}
	return nil
}
