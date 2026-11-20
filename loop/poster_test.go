package loop

import (
	"sync"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
)

// fakeScreen is a tcell.Screen whose only working method is EventQ — the one
// Poster calls. Every other method panics on the nil embedded interface.
type fakeScreen struct {
	tcell.Screen
	q chan tcell.Event
}

func (f *fakeScreen) EventQ() chan tcell.Event { return f.q }

func newFake(buf int) *fakeScreen { return &fakeScreen{q: make(chan tcell.Event, buf)} }

func interrupts(q chan tcell.Event) int {
	n := 0
	for {
		select {
		case ev := <-q:
			if _, ok := ev.(*tcell.EventInterrupt); ok {
				n++
			}
		default:
			return n
		}
	}
}

func TestPostBurstSendsOneWake(t *testing.T) {
	var p Poster
	s := newFake(128)
	p.SetScreen(s)
	var wg sync.WaitGroup
	ran := 0
	for range 50 {
		wg.Go(func() { p.Post(func() { ran++ }) })
	}
	wg.Wait()
	if n := interrupts(s.q); n != 1 {
		t.Fatalf("a burst of 50 posts sent %d wakes, want 1", n)
	}
	p.Drain()
	if ran != 50 {
		t.Fatalf("Drain ran %d callbacks, want 50", ran)
	}
	if p.Pending() != 0 {
		t.Fatalf("Pending after Drain = %d", p.Pending())
	}
}

func TestDrainRunsInPostOrder(t *testing.T) {
	var p Poster
	var got []int
	for i := range 5 {
		p.Queue(func() { got = append(got, i) })
	}
	p.Drain()
	for i, v := range got {
		if v != i {
			t.Fatalf("order %v", got)
		}
	}
}

// A post landing while the loop is draining must wake it again; otherwise the
// result sits until an unrelated event.
func TestPostDuringDrainWakesAgain(t *testing.T) {
	var p Poster
	s := newFake(128)
	p.SetScreen(s)
	p.Post(func() {})
	if interrupts(s.q) != 1 {
		t.Fatal("first post sent no wake")
	}
	posted := make(chan struct{})
	p.Queue(func() {
		// Runs inside Drain on the "UI goroutine": a background goroutine
		// posts now, after the queue has been taken.
		go func() { p.Post(func() {}); close(posted) }()
		<-posted
	})
	p.Drain()
	if n := interrupts(s.q); n != 1 {
		t.Fatalf("post during Drain sent %d wakes, want 1", n)
	}
	if p.Pending() != 1 {
		t.Fatalf("Pending = %d, want the mid-drain post", p.Pending())
	}
}

func TestWakeWithoutScreenIsNoOpAndDoesNotLatch(t *testing.T) {
	var p Poster
	p.Post(func() {}) // no screen: must not panic, must not set the flag
	s := newFake(4)
	p.SetScreen(s)
	p.Wake()
	if n := interrupts(s.q); n != 1 {
		t.Fatalf("first wake after SetScreen sent %d, want 1 (a screenless wake latched the flag)", n)
	}
}

func TestFullQueueDoesNotBlock(t *testing.T) {
	var p Poster
	s := newFake(1)
	s.q <- tcell.NewEventInterrupt(nil) // full
	p.SetScreen(s)
	done := make(chan struct{})
	go func() { p.Post(func() {}); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Post blocked on a full EventQ")
	}
	// Close takes the gate Wake sends under: it must not hang either.
	closed := make(chan struct{})
	go func() { p.Close(nil); close(closed) }()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("Close hung behind a wake on a full EventQ")
	}
}

// After Close has run fini (which closes EventQ), a post must not send on the
// closed channel — that panics.
func TestPostAfterCloseIsDropped(t *testing.T) {
	var p Poster
	s := newFake(4)
	p.SetScreen(s)
	p.Close(func() { close(s.q) })
	if !p.Closed() {
		t.Fatal("Closed() = false after Close")
	}
	p.Post(func() {}) // panics on send to closed channel if not gated
	if p.Pending() != 1 {
		t.Fatalf("Pending = %d; a post after Close still queues", p.Pending())
	}
}

// Wakes racing Close: no send may land on the channel fini closed. Run under
// -race; a send after close panics the test binary.
func TestWakeRacingClose(t *testing.T) {
	for range 200 {
		var p Poster
		s := newFake(128)
		p.SetScreen(s)
		var wg sync.WaitGroup
		for range 8 {
			wg.Go(func() {
				for range 20 {
					p.Post(func() {})
					p.Drain() // clears the flag so later posts send again
				}
			})
		}
		p.Close(func() { close(s.q) })
		wg.Wait()
	}
}
