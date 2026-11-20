package loop

import (
	"context"
	"errors"
	"time"
)

// ErrTaskPanicked is what a task is finished with when its goroutine
// panicked: a GoRepair repair passes it to Tasks.Finish. The panic itself is
// already reported by the Runner, so this only gives the task a terminal state
// and a reason a history list can show. A task finished with it is never
// Cancelled, even if Cancel was asked for first — a panic is a failure.
var ErrTaskPanicked = errors.New("stopped unexpectedly — see the log for details")

// Task is one long-running background operation a host reports in its status
// bar and lets the user check on or cancel later, rather than blocking the UI:
// the non-modal counterpart of dialogs.ProgressDialog.Run.
//
// Its fields are read and written only on the UI goroutine. The goroutine
// doing the work never touches one: it reports through Tasks.Progress and
// Tasks.Done, which post the mutation to the UI goroutine. No mutex is needed
// while that holds.
type Task struct {
	ID       int
	Label    string
	Progress int // 0-100; -1 = indeterminate (no percentage to show)
	Message  string
	Done     bool
	Err      error
	Started  time.Time
	Finished time.Time
	// Cancelled is set when the task finished with an error after Cancel was
	// asked for while it ran. The error is then the cancel itself, as the
	// driver or server words it ("context canceled", "operation terminating
	// abnormally"), and reporting it as a failure blames the server for what
	// the user did. A run that finished despite the cancel keeps its success;
	// a panic (ErrTaskPanicked) is still a failure. Err keeps the error.
	Cancelled bool

	cancel context.CancelFunc
	// cancelAsked records a Cancel while the task was running — Finish's own
	// Cancel, releasing a finished task's context, does not count.
	cancelAsked bool
}

// Cancel cancels the task's context. Safe on a finished task (it only
// releases the context again), on one built without Tasks.Start (no context),
// and more than once. UI goroutine only, like the fields.
func (t *Task) Cancel() {
	if !t.Done {
		t.cancelAsked = true
	}
	if t.cancel != nil {
		t.cancel()
	}
}

// Times is the task's elapsed time — up to now while it runs, up to Finished
// once done — and, while it runs with a percentage strictly between 0 and 100,
// a linear estimate of the time remaining (ok reports whether there is one).
func (t *Task) Times() (elapsed, remaining time.Duration, ok bool) {
	end := time.Now()
	if t.Done {
		end = t.Finished
	}
	elapsed = end.Sub(t.Started)
	if !t.Done && t.Progress > 0 && t.Progress < 100 {
		remaining = elapsed * time.Duration(100-t.Progress) / time.Duration(t.Progress)
		ok = true
	}
	return elapsed, remaining, ok
}

// Tasks is a host's registry of background tasks, running and finished, for
// a status-bar count and a history list. Start, Finish, Running and All
// belong to the UI goroutine; Progress and Done are called from the worker.
//
// The zero value holds no tasks and keeps every finished one; Post must be
// set before the first Progress or Done. Set the fields before the first
// Start and leave them: a worker reads Post.
type Tasks struct {
	// Post queues a callback on the UI goroutine, e.g. (*Poster).Post — the
	// same hook Runner takes. Progress and Done deliver through it.
	Post func(func())

	// Keep caps the registry's length for history. Start evicts the oldest
	// finished tasks past it; a running task is never evicted, so enough of
	// them can hold it past Keep. <= 0 keeps everything.
	Keep int

	// OnFinish, if set, runs on the UI goroutine after Finish has settled t
	// (Done, Err, Cancelled, Finished): the host's own wording for the status
	// bar.
	OnFinish func(t *Task)

	tasks []*Task
	seq   int
}

// Start registers a running task under label and returns it with the context
// its worker runs under. The context is cancelled by Task.Cancel, when parent
// is, and by Finish, which releases it. Pass a parent scoped to whatever the
// work depends on (a connection, say) so losing that stops the work too.
func (r *Tasks) Start(parent context.Context, label string) (*Task, context.Context) {
	r.seq++
	ctx, cancel := context.WithCancel(parent)
	t := &Task{ID: r.seq, Label: label, Progress: -1, Started: time.Now(), cancel: cancel}
	r.tasks = append(r.tasks, t)
	r.prune()
	return t, ctx
}

// Progress posts a progress update for t (-1 for no percentage). Any
// goroutine; the fields change on the UI goroutine.
func (r *Tasks) Progress(t *Task, pct int, msg string) {
	r.Post(func() {
		t.Progress = pct
		t.Message = msg
	})
}

// Done posts t's completion (err nil on success) to Finish. Any goroutine
// but the UI one; a caller already there calls Finish directly, and a GoRepair
// repair must, since a second hop would land after the panic report.
func (r *Tasks) Done(t *Task, err error) {
	r.Post(func() { r.Finish(t, err) })
}

// Finish settles t on the UI goroutine: Done, Err, Finished, and Cancelled
// when err followed a Cancel asked for while it ran (never for
// ErrTaskPanicked). It releases t's context — the work is over whether or not
// Cancel was called — and then calls OnFinish.
func (r *Tasks) Finish(t *Task, err error) {
	t.Done = true
	t.Err = err
	t.Cancelled = err != nil && !errors.Is(err, ErrTaskPanicked) && t.cancelAsked
	t.Finished = time.Now()
	t.Cancel()
	if r.OnFinish != nil {
		r.OnFinish(t)
	}
}

// Running is how many tasks have not finished.
func (r *Tasks) Running() int {
	n := 0
	for _, t := range r.tasks {
		if !t.Done {
			n++
		}
	}
	return n
}

// All is every registered task, oldest first. The slice is the registry's
// own, valid until the next Start; don't modify it.
func (r *Tasks) All() []*Task { return r.tasks }

// prune evicts the oldest finished tasks while the registry is past Keep.
func (r *Tasks) prune() {
	if r.Keep <= 0 || len(r.tasks) <= r.Keep {
		return
	}
	excess := len(r.tasks) - r.Keep
	kept := r.tasks[:0]
	for _, t := range r.tasks {
		if !t.Done || excess == 0 {
			kept = append(kept, t)
		} else {
			excess--
		}
	}
	clear(r.tasks[len(kept):])
	r.tasks = kept
}
