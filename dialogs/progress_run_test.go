package dialogs

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/loop"
	"github.com/radix29/tuikit/tuikittest"
)

// runHost is a screenless host for Run: a Poster drained by hand, a Runner
// that counts reported panics, and a dialog on a test screen.
type runHost struct {
	p      *loop.Poster
	r      *loop.Runner
	d      *ProgressDialog
	panics chan string
}

func newRunHost() *runHost {
	h := &runHost{p: new(loop.Poster), panics: make(chan string, 4)}
	// No Post: Run must route the panic repair through p regardless.
	h.r = &loop.Runner{Report: func(what string, _ any, _ []byte) { h.panics <- what }}
	h.d = withScreen(NewProgressDialog(), tuikittest.NewScreen(120, 40))
	return h
}

// drainUntil drains the Poster, as the host's loop would, until cond holds.
func (h *runHost) drainUntil(t *testing.T, cond func() bool, what string) {
	t.Helper()
	for range 400 {
		h.p.Drain()
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func (h *runHost) run(job ProgressJob, work func(context.Context, ProgressReport) error, done func(error, bool)) (context.Context, bool) {
	ctx, cancel := context.WithCancel(context.Background())
	return ctx, h.d.Run(ctx, cancel, h.r, h.p, job, work, done)
}

// A job refused because another holds the dialog runs nothing, releases its
// context, and runs the caller's Repair — the caller latched its busy flag
// before asking, and neither done nor the panic repair will run to clear it.
func TestProgressRunRefusesASecondJobAndRepairs(t *testing.T) {
	h := newRunHost()
	hold := make(chan struct{})
	defer close(hold)
	if _, ok := h.run(ProgressJob{Title: "One", What: "one"},
		func(context.Context, ProgressReport) error { <-hold; return nil }, nil); !ok {
		t.Fatal("the first job was refused")
	}

	repaired := false
	ctx, ok := h.run(ProgressJob{Title: "Two", What: "two", Repair: func() { repaired = true }},
		func(context.Context, ProgressReport) error { t.Error("a refused job ran its work"); return nil },
		func(error, bool) { t.Error("a refused job ran done") })
	if ok {
		t.Fatal("Run accepted a job while another held the dialog")
	}
	if !repaired {
		t.Error("the refusal did not run the job's Repair")
	}
	if ctx.Err() == nil {
		t.Error("the refused job's context was left uncancelled")
	}
	if h.d.Title() != "One" || !h.d.Running() {
		t.Errorf("title = %q, running = %v: the refusal disturbed the job in flight", h.d.Title(), h.d.Running())
	}
}

// Reports land on the dialog while their job holds it; one arriving after the
// job finished (from a goroutine work didn't wait for) is dropped rather than
// rewriting the next job's message.
func TestProgressRunDropsAStaleReport(t *testing.T) {
	h := newRunHost()
	var first ProgressReport
	h.run(ProgressJob{Title: "One", Message: "one", What: "one"},
		func(_ context.Context, report ProgressReport) error {
			first = report
			report("step 2 of 2")
			return nil
		}, nil)
	h.drainUntil(t, func() bool { return !h.d.Running() }, "the first job to finish")

	hold := make(chan struct{})
	defer close(hold)
	h.run(ProgressJob{Title: "Two", Message: "two", What: "two"},
		func(context.Context, ProgressReport) error { <-hold; return nil }, nil)
	first("late")
	h.drainUntil(t, func() bool { return h.p.Pending() == 0 }, "the late report to drain")
	if got := h.d.Message(); got != "two" {
		t.Errorf("message = %q after the first job's late report, want %q", got, "two")
	}
}

// Cancel during the work cancels its context, leaves the dialog up until the
// work returns, and done reports cancelled.
func TestProgressRunCancelReportsCancelled(t *testing.T) {
	h := newRunHost()
	started := make(chan struct{})
	var gotErr error
	gotCancelled, finished := false, false
	h.run(ProgressJob{Title: "T", What: "t"},
		func(ctx context.Context, _ ProgressReport) error {
			close(started)
			<-ctx.Done()
			return ctx.Err()
		},
		func(err error, cancelled bool) { gotErr, gotCancelled, finished = err, cancelled, true })
	<-started
	h.d.RevealDelay = 0
	h.d.HandleKey(tuikittest.Key(tcell.KeyEscape))
	h.drainUntil(t, func() bool { return finished }, "done")
	if !errors.Is(gotErr, context.Canceled) || !gotCancelled {
		t.Errorf("done: err=%v cancelled=%v, want context.Canceled, true", gotErr, gotCancelled)
	}
	if h.d.Visible() || h.d.Running() {
		t.Errorf("visible=%v running=%v after done, want both false", h.d.Visible(), h.d.Running())
	}
}

// A cancel that reaches a job only after its work succeeded is not reported as
// a cancel: the operation happened.
func TestProgressRunCancelAfterSuccessReportsSuccess(t *testing.T) {
	h := newRunHost()
	started, release := make(chan struct{}), make(chan struct{})
	var gotErr error
	gotCancelled, finished := true, false
	h.run(ProgressJob{Title: "T", What: "t"},
		func(context.Context, ProgressReport) error {
			close(started)
			<-release // ignores the context, like a statement already committed
			return nil
		},
		func(err error, cancelled bool) { gotErr, gotCancelled, finished = err, cancelled, true })
	<-started
	h.d.RevealDelay = 0
	h.d.HandleKey(tuikittest.Key(tcell.KeyEscape))
	close(release)
	h.drainUntil(t, func() bool { return finished }, "done")
	if gotErr != nil || gotCancelled {
		t.Errorf("done: err=%v cancelled=%v, want nil, false", gotErr, gotCancelled)
	}
}

// An uninterruptible job ignores Cancel: its context is never cancelled and
// the dialog stays up until the work returns.
func TestProgressRunUninterruptibleIgnoresCancel(t *testing.T) {
	h := newRunHost()
	started, release := make(chan struct{}), make(chan struct{})
	finished := false
	ctx, _ := h.run(ProgressJob{Title: "Fail Over", What: "t", Uninterruptible: "Not now."},
		func(context.Context, ProgressReport) error { close(started); <-release; return nil },
		func(error, bool) { finished = true })
	<-started
	h.d.RevealDelay = 0
	h.d.HandleKey(tuikittest.Key(tcell.KeyEscape))
	h.d.HandleKey(tuikittest.Key(tcell.KeyEnter))
	if ctx.Err() != nil {
		t.Fatalf("the uninterruptible job's context was cancelled: %v", ctx.Err())
	}
	if !h.d.Visible() {
		t.Fatal("the dialog closed while the uninterruptible job ran")
	}
	close(release)
	h.drainUntil(t, func() bool { return finished }, "done")
	if ctx.Err() == nil {
		t.Error("the context was not released once the work returned")
	}
}

// A panicking job releases the dialog — a modal left up would sit over the
// whole application for good — runs Repair, reports the panic, skips done,
// and the next job runs. The Runner here has no Post: the repair still goes
// through the Poster.
func TestProgressRunReleasesOnPanic(t *testing.T) {
	h := newRunHost()
	repaired := false
	h.run(ProgressJob{Title: "T", What: "boom job", Repair: func() { repaired = true }},
		func(context.Context, ProgressReport) error { panic("boom") },
		func(error, bool) { t.Error("done ran for a job that panicked") })
	h.drainUntil(t, func() bool { return repaired }, "the repair")
	if h.d.Running() || h.d.Visible() {
		t.Fatalf("after the panic: running=%v visible=%v, want both false", h.d.Running(), h.d.Visible())
	}
	if what := <-h.panics; what != "boom job" {
		t.Errorf("panic reported as %q, want %q", what, "boom job")
	}

	var gotErr error
	ran := false
	if _, ok := h.run(ProgressJob{Title: "T", What: "t"},
		func(context.Context, ProgressReport) error { return errors.New("second") },
		func(err error, _ bool) { gotErr, ran = err, true }); !ok {
		t.Fatal("the job after the panic was refused")
	}
	h.drainUntil(t, func() bool { return ran }, "the second job")
	if gotErr == nil || gotErr.Error() != "second" {
		t.Errorf("the job after the panic: err=%v, want its own error", gotErr)
	}
}
