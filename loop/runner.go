package loop

import (
	"context"
	"runtime/debug"
	"runtime/pprof"
	"sync"
	"time"
)

// Runner starts a host's background goroutines so that a panic on one becomes
// a report instead of a process-wide crash. It is the background half of the
// protocol Poster is the UI-goroutine half of.
//
// The UI goroutine cannot recover a panic on another goroutine, so an
// unrecovered one takes the process down: the host's deferred screen.Fini()
// never runs, the terminal is left in raw mode on the alternate screen, and
// whatever the user had unsaved is gone. A driver or library panicking on an
// input newer than it knows is enough to reach it.
//
// The zero value is safe: a nil Post skips the repair step, a nil Report drops
// the report (the panic is still recovered), and Workers <= 0 means one. Every
// method is safe from any goroutine; Runner holds no state of its own.
type Runner struct {
	// Post queues a callback on the UI goroutine, e.g. (*Poster).Post.
	// GoRepair posts its repair through it.
	Post func(func())

	// Report is told about every recovered panic: what names the operation
	// (the what passed to Go, GoRepair, Recover or FanOut), r is the
	// recovered value and stack the panicking goroutine's stack. It runs on
	// the goroutine that panicked — debug.Stack() is only meaningful there,
	// so the stack is captured before the call — and must not touch a widget:
	// a host logs the stack and posts a status message through its Poster.
	Report func(what string, r any, stack []byte)

	// Workers bounds FanOut's pool. <= 0 means 1.
	Workers int
}

// Go runs fn on a new goroutine labelled with what (see Label), recovering a
// panic and handing it to Report.
func (r *Runner) Go(what string, fn func()) {
	r.GoRepair(what, nil, fn)
}

// GoRepair is Go for an operation that latched UI state before starting: a
// busy flag, a "loading" placeholder, a dimmed toolbar. repair is posted to
// the UI goroutine if, and only if, fn panicked; nil is Go.
//
// The latch is normally cleared by the callback fn posts on finishing, and a
// panic unwinds past it, so without a repair the flag stays set for the
// object's lifetime: the toolbar stays dimmed, the tab says "Running..."
// forever. The repair must not run after a clean return either, where it
// would clear a latch a newer operation has since taken for itself.
//
// repair is posted before Report runs, so a status message Report posts lands
// after whatever repair sets: the status bar's last word is the panic.
func (r *Runner) GoRepair(what string, repair, fn func()) {
	go func() {
		Label(what)
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if repair != nil && r.Post != nil {
				r.Post(repair)
			}
			r.report(what, v)
		}()
		fn()
	}()
}

// GoOnce is GoRepair for a single-flight action — a button that must not start
// a second run while the first is out — with the busy latch kept here, so the
// one rule a host writing it by hand forgets cannot be forgotten: a panic must
// release the latch too, or the button refuses every later click for the
// object's lifetime.
//
// If *busy is set, GoOnce does nothing and returns false. Otherwise it sets
// *busy and runs fn on a new goroutine labelled with what. fn returns the
// callback for its result (nil for none); GoOnce posts one callback to the UI
// goroutine that clears *busy and then runs it. On a panic in fn the posted
// repair clears *busy and the panic goes to Report, as for GoRepair. Either
// way the latch is released on the UI goroutine, in the same callback that
// reports the outcome, so busy is a plain bool: call GoOnce from the UI
// goroutine only, with Post set (without it the latch is never released).
//
// What fn needs from the UI goroutine — a context, the object it writes into
// — it captures before GoOnce is called; whether the result is still wanted
// when it arrives (the dialog closed, a newer showing) is the callback's to
// check.
func (r *Runner) GoOnce(busy *bool, what string, fn func() (done func())) bool {
	if *busy {
		return false
	}
	*busy = true
	release := func() { *busy = false }
	r.GoRepair(what, release, func() {
		done := fn()
		r.Post(func() {
			release()
			if done != nil {
				done()
			}
		})
	})
	return true
}

// Recover is the bare deferred recovery, for a goroutine the host starts
// itself: `defer r.Recover("what this was doing")`. It must be deferred
// directly — recover only stops a panic when called by the deferred function
// itself.
func (r *Runner) Recover(what string) {
	if v := recover(); v != nil {
		r.report(what, v)
	}
}

// FanOut runs work(i) for every i in [0, n) across a pool of at most Workers
// goroutines, each labelled with what, and returns once every call has
// finished.
//
// Each call is recovered on its own: a panic in one item is reported and the
// worker moves on, rather than dying and taking every item it still owed with
// it. onPanic (optional) runs on the worker for the panicked item, before
// Report and before FanOut can return, because a caller acting on the results
// at once needs the item's repair already done — a row whose fetch died would
// otherwise be cached with its placeholder.
//
// The queue is filled and closed before a worker exists, so no send can block
// on one. Handing indices out from the calling goroutine would hang it forever
// if a panic escaped the per-item recovery and took every worker down.
//
// A fixed set of workers pulling indices off a channel, rather than n
// goroutines each waiting on a token: the token form parks hundreds of idle
// goroutines on a list with hundreds of entries.
func (r *Runner) FanOut(n int, what string, work func(i int), onPanic func(i int)) {
	if n <= 0 {
		return
	}
	idx := make(chan int, n)
	for i := range n {
		idx <- i
	}
	close(idx)

	one := func(i int) {
		defer func() {
			v := recover()
			if v == nil {
				return
			}
			if onPanic != nil {
				onPanic(i)
			}
			r.report(what, v)
		}()
		work(i)
	}

	var wg sync.WaitGroup
	for range min(n, max(r.Workers, 1)) {
		wg.Go(func() {
			// The per-item recovery keeps a worker alive; this is the belt
			// under it, for a panic in onPanic or Report itself. One escaping
			// both is on a background goroutine nothing else can catch.
			Label(what)
			defer r.Recover(what)
			for i := range idx {
				one(i)
			}
		})
	}
	wg.Wait()
}

// Tick calls tick every period on a new goroutine labelled with what, until
// done closes. It is the redraw clock a busy indicator drawn from elapsed time
// needs and no control may start itself (widgets.Spinner,
// dialogs.ProgressDialog, propsheet.ApplyingSpinner): pass (*Poster).Wake as
// tick and the spinner's Period.
//
// Close done in a defer inside the worker the clock stands for, so a panicking
// worker stops it too; closed on the worker's happy path only, the clock
// outlives the panic and wakes the loop every period for the process's
// lifetime. A panic in tick itself is recovered and reported, as for Go, and
// stops the clock. The first tick comes one period after the call, not at it.
func (r *Runner) Tick(what string, period time.Duration, done <-chan struct{}, tick func()) {
	r.Go(what, func() {
		ticker := time.NewTicker(period)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				tick()
			}
		}
	})
}

// report captures the stack on the panicking goroutine and hands it to Report.
func (r *Runner) report(what string, v any) {
	if r.Report == nil {
		return
	}
	r.Report(what, v, debug.Stack())
}

// LabelKey is the pprof label key Label sets.
const LabelKey = "op"

// Label tags the calling goroutine with the pprof label op=what, which Go
// prints in the header of every traceback the goroutine appears in (a
// recovered stack, a goroutine dump, the goroutineleak profile). Without it
// each is headed by an anonymous func literal naming no operation. Go,
// GoRepair and FanOut call it; a host starting a goroutine itself can too.
//
// Set directly, not through pprof.Do, whose deferred restore runs while a
// panic is still unwinding — before the recover — and would strip the label
// from the one stack that has to carry it.
func Label(what string) {
	pprof.SetGoroutineLabels(pprof.WithLabels(context.Background(), pprof.Labels(LabelKey, what)))
}
