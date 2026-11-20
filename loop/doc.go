// Package loop is the plumbing of a tcell v3 host's async work: the queue
// background goroutines hand results to the event loop through, the token that
// drops a superseded load's result, and the starter that keeps a panic on a
// background goroutine from taking the process down.
//
//   - poster.go — Poster: Post a callback from any goroutine and wake the
//     loop; the loop Drains the queue on every iteration; Close orders the
//     host's screen.Fini() against a wake in flight
//   - latest.go — Latest: the latest-only load lifecycle; Begin supersedes
//     and cancels the run in flight, Done drops a stale result and releases
//     the current run's context, Cancel/Abandon stop without/with superseding
//   - runner.go — Runner: the background side; Go/GoRepair start a labelled,
//     recovered goroutine (GoRepair posts a repair before the report), GoOnce
//     is a single-flight run whose busy latch a panic releases too, Recover
//     is the bare deferred form, FanOut a bounded pool recovering per item,
//     Tick a spinner's redraw clock that stops when done closes;
//     Label sets the pprof op label every traceback header then prints
//   - tasks.go — Tasks/Task: the registry of non-modal background tasks;
//     Start hands out a task and its context, Progress/Done post from the
//     worker, Finish settles it on the UI goroutine (Cancelled only for an
//     error after a Cancel asked for while it ran, never ErrTaskPanicked),
//     Keep caps the history without evicting a running task
//
// It holds the protocol ARCHITECTURE.md § Threading states, written once: the
// wakeup is sent from the posting goroutine, never from inside the posted
// callback; a burst of posts sends one wakeup; the send never blocks; and no
// send reaches EventQ() after Fini() has closed it; and no background
// goroutine's panic goes unrecovered, since the UI goroutine cannot recover it
// and the host's Fini() would never run. A host keeps its own widgets
// single-threaded as before — nothing here touches one.
//
// It imports only the standard library and tcell, nothing of tuikit.
package loop
