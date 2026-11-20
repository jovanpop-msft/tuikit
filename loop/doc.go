// Package loop is the UI-goroutine side of a tcell v3 host's async work: the
// queue background goroutines hand results to the event loop through, and the
// token that drops a superseded load's result.
//
//   - poster.go — Poster: Post a callback from any goroutine and wake the
//     loop; the loop Drains the queue on every iteration; Close orders the
//     host's screen.Fini() against a wake in flight
//   - latest.go — Latest: the latest-only load lifecycle; Begin supersedes
//     and cancels the run in flight, Done drops a stale result and releases
//     the current run's context, Cancel/Abandon stop without/with superseding
//
// It holds the protocol ARCHITECTURE.md § Threading states, written once: the
// wakeup is sent from the posting goroutine, never from inside the posted
// callback; a burst of posts sends one wakeup; the send never blocks; and no
// send reaches EventQ() after Fini() has closed it. A host keeps its own
// widgets single-threaded as before — nothing here touches one.
//
// It imports only the standard library and tcell, nothing of tuikit.
package loop
