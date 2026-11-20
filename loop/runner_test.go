package loop

import (
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
)

// recorder is a Runner whose Post and Report append to one ordered log, so a
// test can check what happened and in which order.
type recorder struct {
	mu     sync.Mutex
	events []string
	posted []func()
	stacks []string
}

func (rec *recorder) runner(workers int) *Runner {
	return &Runner{
		Post: func(fn func()) {
			rec.mu.Lock()
			defer rec.mu.Unlock()
			rec.events = append(rec.events, "post")
			rec.posted = append(rec.posted, fn)
		},
		Report: func(what string, r any, stack []byte) {
			rec.mu.Lock()
			defer rec.mu.Unlock()
			rec.events = append(rec.events, "report "+what)
			rec.stacks = append(rec.stacks, string(stack))
		},
		Workers: workers,
	}
}

func (rec *recorder) snapshot() []string {
	rec.mu.Lock()
	defer rec.mu.Unlock()
	return append([]string(nil), rec.events...)
}

// A repair posted after the report would overwrite the panic's status message
// with whatever the repair sets, and the status bar would no longer say what
// died. The repair goes first.
func TestGoRepairPostsTheRepairBeforeReporting(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := &recorder{}
		r := rec.runner(0)
		repaired := false

		r.GoRepair("reading the thing", func() { repaired = true }, func() { panic("boom") })
		synctest.Wait()

		got := rec.snapshot()
		if len(got) != 2 || got[0] != "post" || got[1] != "report reading the thing" {
			t.Fatalf("events = %q, want [post, report reading the thing]", got)
		}
		rec.posted[0]()
		if !repaired {
			t.Error("the posted callback is not the repair")
		}
	})
}

// The repair belongs to the panic: after a clean return the operation cleared
// its own latch, and a repair would clear one a newer operation had since
// taken.
func TestGoRepairDoesNothingWhenNothingPanicked(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := &recorder{}
		rec.runner(0).GoRepair("reading the thing", func() {}, func() {})
		synctest.Wait()
		if got := rec.snapshot(); len(got) != 0 {
			t.Errorf("events = %q, want none", got)
		}
	})
}

// The reported stack is the only record of what the goroutine was doing, and
// Go prints its pprof labels in the traceback header — but only if the label
// is still set while the panic unwinds.
func TestGoLabelsTheGoroutineOnTheReportedStack(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		rec := &recorder{}
		rec.runner(0).Go("loading the thing", func() { panic("boom") })
		synctest.Wait()

		if len(rec.stacks) != 1 {
			t.Fatalf("%d reports, want 1", len(rec.stacks))
		}
		header, _, _ := strings.Cut(rec.stacks[0], "\n")
		if !strings.Contains(header, `op: "loading the thing"`) {
			t.Errorf("traceback header = %q, want it to name the operation", header)
		}
	})
}

// A zero Runner recovers and stays quiet rather than panicking on a nil
// callback — the second panic would be the unrecovered one.
func TestZeroRunnerRecoversWithoutCallbacks(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var r Runner
		r.GoRepair("x", func() { t.Error("repair ran without a Post") }, func() { panic("boom") })
		r.FanOut(3, "x", func(int) { panic("boom") }, nil)
		synctest.Wait()
	})
}

func TestRecoverReportsADeferredPanic(t *testing.T) {
	rec := &recorder{}
	r := rec.runner(0)
	func() {
		defer r.Recover("doing the thing")
		panic("boom")
	}()
	if got := rec.snapshot(); len(got) != 1 || got[0] != "report doing the thing" {
		t.Errorf("events = %q, want [report doing the thing]", got)
	}
}

// One panic costs its item only. With more items than workers, a worker whose
// first item panicked must survive to take the next; the first Workers items,
// which the queue hands out first, are the ones that panic.
func TestFanOutRecoversEachItemAndRunsTheRest(t *testing.T) {
	const workers = 4
	rec := &recorder{}
	n := workers * 3
	done := make([]bool, n)
	var repaired sync.Map

	rec.runner(workers).FanOut(n, "reading a log", func(i int) {
		if i < workers {
			panic("boom")
		}
		done[i] = true
	}, func(i int) { repaired.Store(i, true) })

	// onPanic has run for every panicked item by the time FanOut returns.
	for i, ok := range done {
		if want := i >= workers; ok != want {
			t.Errorf("item %d ran = %v, want %v", i, ok, want)
		}
		if _, ok := repaired.Load(i); ok != (i < workers) {
			t.Errorf("item %d onPanic ran = %v, want %v", i, ok, i < workers)
		}
	}
	if got := len(rec.snapshot()); got != workers {
		t.Errorf("%d reports, want %d", got, workers)
	}
}

// FanOut never runs more than Workers items at once.
func TestFanOutRespectsTheWorkerBound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		const workers = 3
		var mu sync.Mutex
		running, peak := 0, 0
		gate := make(chan struct{})
		r := &Runner{Workers: workers}

		go r.FanOut(workers*4, "x", func(int) {
			mu.Lock()
			running++
			peak = max(peak, running)
			mu.Unlock()
			<-gate
			mu.Lock()
			running--
			mu.Unlock()
		}, nil)
		synctest.Wait()
		close(gate)
		synctest.Wait()

		if peak != workers {
			t.Errorf("peak concurrency = %d, want %d", peak, workers)
		}
	})
}

// A panic in onPanic escapes the per-item recovery; the worker's own recovery
// must still catch it, or the process dies. The labelled stack is that
// worker's.
func TestFanOutSurvivesAPanicInOnPanic(t *testing.T) {
	rec := &recorder{}
	rec.runner(2).FanOut(4, "reading a log", func(int) { panic("boom") }, func(int) { panic("again") })

	got := rec.snapshot()
	if len(got) == 0 {
		t.Fatal("no report, want the onPanic panic reported")
	}
	for _, s := range rec.stacks {
		header, _, _ := strings.Cut(s, "\n")
		if !strings.Contains(header, `op: "reading a log"`) {
			t.Errorf("traceback header = %q, want it to name the operation", header)
		}
	}
}

// Label is set directly rather than through pprof.Do, so it is still on the
// goroutine while a deferred function runs during the unwind.
func TestLabelSurvivesTheUnwind(t *testing.T) {
	stacks := make(chan string, 1)
	go func() {
		defer func() { _ = recover() }()
		Label("probing the label")
		defer func() { stacks <- string(debug.Stack()) }()
		panic("boom")
	}()
	header, _, _ := strings.Cut(<-stacks, "\n")
	if !strings.Contains(header, `op: "probing the label"`) {
		t.Errorf("traceback header = %q, want the label", header)
	}
}
