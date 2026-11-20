package dialogs

import (
	"fmt"

	"github.com/gdamore/tcell/v3"

	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/loop"
	"github.com/radix29/tuikit/theme"
	"github.com/radix29/tuikit/widgets"
)

// TaskProgressView is what one host's progress view of a loop.Task differs in
// from another's: the rows above the bar, whether a label sits over it, and
// the wording of the message under it.
type TaskProgressView struct {
	Header        []string // "Database : x", …, one row each from inner.Y+1
	ProgressLabel bool     // a Strings.TaskProgress row, and a blank one, above the bar

	// Message words the task's state for the last rows, and says whether it is
	// an error (drawn in the dialog's error style). Nil means TaskMessage. A
	// host's own usually names the operation ("Backup completed
	// successfully.") and adds what only it knows about a failure.
	Message func(t *loop.Task) (msg string, isErr bool)
}

// TaskMessage is TaskProgressView's default wording: Strings.TaskCancelled,
// TaskFailed with the error, or TaskCompleted once t is done, else t.Message,
// or TaskStarting before the first report. Only a cancel or a failure is an
// error.
func TaskMessage(t *loop.Task) (msg string, isErr bool) {
	switch {
	case t.Done && t.Cancelled:
		return Strings.TaskCancelled, true
	case t.Done && t.Err != nil:
		return fmt.Sprintf(Strings.TaskFailed, t.Err), true
	case t.Done:
		return Strings.TaskCompleted, false
	case t.Message == "":
		return Strings.TaskStarting, false
	}
	return t.Message, false
}

// DrawTaskProgress draws a progress view of t into inner, stopping above row
// bottom (a dialog's button separator): v's header rows, then, once there is a
// task, a widgets.DrawProgressBar (full once t succeeded), the elapsed and
// remaining times (core.FormatClock, from t.Times), and v's message wrapped
// over every row left. With t nil it draws only the header.
//
// The message gets every remaining row because a failed task's error is
// usually the server's or the OS's own text, far longer than one line, and the
// only thing on the screen the user still needs: drawn on one clipped line, a
// SQL Server restore failure stopped right before the part that said what went
// wrong.
func DrawTaskProgress(s tcell.Screen, inner core.Rect, bottom int, t *loop.Task, v TaskProgressView) {
	labelStyle := theme.StyleDialog()
	lx := inner.X + 1
	w := inner.W - 2

	for i, h := range v.Header {
		core.DrawTextClipped(s, lx, inner.Y+1+i, w, labelStyle, h)
	}
	if t == nil {
		return
	}
	y := inner.Y + len(v.Header) + 2
	if v.ProgressLabel {
		core.DrawTextClipped(s, lx, y, w, labelStyle, Strings.TaskProgress)
		y += 2
	}
	pct := t.Progress
	if t.Done && t.Err == nil {
		pct = 100
	}
	widgets.DrawProgressBar(s, lx, y, w, pct, labelStyle)

	elapsed, remaining, haveRemaining := t.Times()
	core.DrawTextClipped(s, lx, y+2, w, labelStyle, Strings.TaskElapsed+" "+core.FormatClock(elapsed))
	rem := Strings.TaskNoEstimate
	if haveRemaining {
		rem = core.FormatClock(remaining)
	}
	core.DrawTextClipped(s, lx, y+3, w, labelStyle, Strings.TaskRemaining+" "+rem)

	message := v.Message
	if message == nil {
		message = TaskMessage
	}
	msg, isErr := message(t)
	msgStyle := labelStyle
	if isErr {
		msgStyle = theme.StyleDialogError()
	}
	msgY := y + 5
	for i, ln := range core.WrapTextLimit(msg, w, bottom-msgY) {
		core.DrawTextClipped(s, lx, msgY+i, w, msgStyle, ln)
	}
}
