package dialogs

import (
	"context"

	"github.com/radix29/tuikit/loop"
)

// ProgressJob describes one operation ProgressDialog.Run runs behind the dialog.
type ProgressJob struct {
	Title   string // the dialog's title, normally the confirmation's
	Message string // what the dialog says is happening; Report replaces it
	What    string // names the worker goroutine in a panic report (loop.Label)

	// Uninterruptible, when set, greys Cancel and is the reason the dialog gives
	// (ShowUninterruptible): for an operation that stopping halfway leaves worse
	// off than waiting. Otherwise Cancel calls Run's cancel.
	Uninterruptible string

	// Repair, when set, runs on the UI goroutine if Run refuses the job, or after
	// the dialog has been released if the work panics: for a caller that latched a
	// busy flag of its own before asking, which only done would otherwise clear.
	Repair func()
}

// ProgressReport replaces the dialog's message from the work goroutine (the
// object a batch has reached). Safe from any goroutine.
type ProgressReport func(message string)

// Run runs work on a background goroutine behind the dialog, then hands its
// error to done on the UI goroutine once the dialog has closed. Call it from
// the UI goroutine.
//
// The dialog is modal and stays up until work returns: Cancel calls cancel and
// leaves the dialog saying so, because only the returned error says how far
// the operation got. done's cancelled is true when Cancel was pressed and work
// failed; a run that finished before the cancel reached it reports its
// success, since that is what happened, and saying "cancelled" would send the
// user to retry something already done.
//
// Run owns cancel: it is called when work returns, panic included, or at once
// if the job is refused. A refusal happens while an earlier Run on this dialog
// is still going (Running): Run then calls job.Repair, starts nothing and
// returns false, and the host says why. Unreachable from the keyboard, since
// the dialog takes every key while a job runs; a job started from a callback
// must not run unseen behind another job's dialog.
//
// r recovers and reports a panic in work; the dialog is then released and
// job.Repair run, and done is not called. Its Post is ignored: posts, the
// panic repair included, go through p, whose Wake is also the spinner's redraw
// clock (loop.Runner.Tick at Spinner.Interval). A report arriving after its job
// has finished — from a goroutine work started and didn't wait for — is
// dropped rather than rewriting a later job's message.
func (d *ProgressDialog) Run(ctx context.Context, cancel context.CancelFunc, r *loop.Runner, p *loop.Poster,
	job ProgressJob, work func(ctx context.Context, report ProgressReport) error, done func(err error, cancelled bool)) bool {
	if d.running {
		cancel()
		if job.Repair != nil {
			job.Repair()
		}
		return false
	}
	d.running = true
	d.runSeq++
	seq := d.runSeq
	if job.Uninterruptible != "" {
		d.ShowUninterruptible(job.Title, job.Message, job.Uninterruptible)
	} else {
		d.ShowProgress(job.Title, job.Message, cancel)
	}
	report := func(message string) {
		p.Post(func() {
			if d.runSeq == seq && d.running {
				d.SetMessage(message)
			}
		})
	}
	release := func() {
		d.running = false
		d.Hide()
	}

	// A copy, so the repair is posted through p whatever r.Post is: a Runner
	// without Post would drop it and leave the modal dialog over the whole
	// application for good.
	run := *r
	run.Post = p.Post
	finished := make(chan struct{})
	run.Tick("animating the progress dialog", d.Spinner.Interval(), finished, p.Wake)
	run.GoRepair(job.What, func() {
		release()
		if job.Repair != nil {
			job.Repair()
		}
	}, func() {
		defer close(finished)
		defer cancel()
		err := work(ctx, report)
		p.Post(func() {
			cancelled := err != nil && d.Cancelling()
			release()
			if done != nil {
				done(err, cancelled)
			}
		})
	})
	return true
}

// Running reports whether a Run job holds the dialog: from Run until the
// callback that releases it, on finishing or on a panic.
func (d *ProgressDialog) Running() bool { return d.running }
