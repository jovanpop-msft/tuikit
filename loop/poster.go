package loop

import (
	"sync"
	"sync/atomic"

	"github.com/gdamore/tcell/v3"
)

// Poster is the queue of callbacks background goroutines have handed to the
// UI goroutine, and the wakeup that makes the event loop run them. The zero
// value is ready to use and inert until SetScreen: posts queue, wakes are
// dropped, and Drain runs the queue (a test host without a screen drains by
// hand).
//
// The host's event loop calls Drain at the top of every iteration, whatever
// woke it — a key, a resize, or the *tcell.EventInterrupt Wake sends — and
// shuts down through Close:
//
//	p.SetScreen(s)
//	for ev := range s.EventQ() {
//		p.Drain()
//		... dispatch ev, draw ...
//	}
//
//	// quitting, on the UI goroutine:
//	p.Close(s.Fini)
//
// Every method is safe from any goroutine, except that Drain belongs to the
// UI goroutine (it runs the callbacks there) and Wake must not be called from
// it (see Wake).
type Poster struct {
	mu      sync.Mutex
	pending []func()

	// sent coalesces wakes: true from the moment an EventInterrupt is sent
	// until Drain next clears it. A goroutine finding it already true skips
	// its own interrupt; its queued callback still runs, since the next Drain
	// takes the whole queue. Without this, a burst of near-simultaneous
	// completions (one per row of a list loading in parallel) each cost the
	// host a full drain and frame.
	sent atomic.Bool

	// gate serializes Wake's channel send against Close's fini, which closes
	// EventQ(). A flag checked before sending still races — Fini() can close
	// the channel between check and send. Holding gate across flag-and-send
	// and across set-flag-and-fini makes the two mutually exclusive: a send
	// either completes before the close or never happens.
	gate   sync.Mutex
	closed bool
	screen tcell.Screen
}

// SetScreen makes s the screen whose EventQ() Wake sends to. Until it is
// called (or with nil) Wake is a no-op, so a background goroutine outliving a
// screenless test never panics on it.
func (p *Poster) SetScreen(s tcell.Screen) {
	p.gate.Lock()
	p.screen = s
	p.gate.Unlock()
}

// Post queues fn to run on the UI goroutine and wakes the event loop, without
// waiting for an unrelated key or mouse event. Call it from a background
// goroutine, never from the UI goroutine (see Wake).
//
// This is the one call a background operation should report through: writing
// Queue and Wake out by hand invites nesting the Wake inside the very callback
// waiting to be drained, which never fires — the result sits queued until an
// unrelated keypress.
func (p *Poster) Post(fn func()) {
	p.Queue(fn)
	p.Wake()
}

// Queue adds fn to the queue without waking the loop: it runs at the next
// Drain, whatever prompts it. Post is Queue followed by Wake.
func (p *Poster) Queue(fn func()) {
	p.mu.Lock()
	p.pending = append(p.pending, fn)
	p.mu.Unlock()
}

// Wake nudges the event loop to run one more iteration, which drains the queue
// and redraws. Call it from a background goroutine after Queue, or alone for a
// redraw clock with nothing to post (a spinner's tick: Runner.Tick). From the UI goroutine
// it is pointless at best: the loop can't read EventQ() mid-dispatch.
//
// A no-op without a screen, after Close, or while an earlier wake has not yet
// been drained (the coalescing above). The wakeup has to be sent after Queue
// and outside the callback: the loop drains only when it wakes, so a wakeup
// nested inside the callback waiting to be drained never fires.
//
// The send is non-blocking. gate is held across it and Close takes the same
// lock from a UI goroutine that is then not reading EventQ(), so a blocking
// send on a full queue (tcell buffers 128 events, which all-motion mouse
// tracking fills fast during a slow frame) would hang the host's quit. Giving
// up on a full queue loses nothing: the loop is about to wake, and every
// iteration's Drain clears the coalesce flag and runs the queue regardless of
// what woke it.
func (p *Poster) Wake() {
	p.gate.Lock()
	defer p.gate.Unlock()
	// Checked before the coalesce flag, not after: a wake dropped for want of
	// a screen must not leave the flag set, or the first wake after SetScreen
	// would be swallowed too.
	if p.closed || p.screen == nil {
		return
	}
	if !p.sent.CompareAndSwap(false, true) {
		return
	}
	select {
	case p.screen.EventQ() <- tcell.NewEventInterrupt(nil):
	default:
	}
}

// Drain runs every queued callback, in the order posted, on the calling (UI)
// goroutine. The host's event loop calls it at the top of every iteration.
//
// It clears the coalesce flag before taking the queue, not after, so a Post
// racing this instant still gets its own wake: if its append lands after the
// queue is taken, the flag is already false and its wake goes out.
//
// A callback that panics loses the callbacks after it in this batch (they were
// already taken off the queue); recovering is the host's, as for any panic on
// its UI goroutine.
func (p *Poster) Drain() {
	p.sent.Store(false)
	p.mu.Lock()
	fns := p.pending
	p.pending = nil
	p.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
}

// Pending reports how many callbacks are queued and not yet drained.
func (p *Poster) Pending() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.pending)
}

// Close marks the Poster closed — every later Wake is a no-op — and then
// calls fini (typically the screen's Fini, which closes EventQ()), both under
// the lock Wake sends under, so a racing Wake either completes its send first
// or sees Close and skips it, never sending on the closed channel. fini may be
// nil. Posts after Close still queue; a host that drains again runs them.
func (p *Poster) Close(fini func()) {
	p.gate.Lock()
	defer p.gate.Unlock()
	p.closed = true
	if fini != nil {
		fini()
	}
}

// Closed reports whether Close has been called.
func (p *Poster) Closed() bool {
	p.gate.Lock()
	defer p.gate.Unlock()
	return p.closed
}
