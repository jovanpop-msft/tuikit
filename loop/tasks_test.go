package loop

import (
	"context"
	"errors"
	"testing"
	"time"
)

// newTasks is a registry whose Post queues on p, drained by hand.
func newTasks(p *Poster, keep int) *Tasks {
	return &Tasks{Post: p.Post, Keep: keep}
}

// Each Start registers the task, in order, under a strictly increasing ID,
// running and with no percentage yet.
func TestTasksStartAssignsIncrementingIDs(t *testing.T) {
	r := newTasks(new(Poster), 0)
	t1, ctx1 := r.Start(context.Background(), "Backup")
	t2, _ := r.Start(context.Background(), "Rebuild")

	if t2.ID <= t1.ID {
		t.Fatalf("IDs %d then %d, want strictly increasing", t1.ID, t2.ID)
	}
	if all := r.All(); len(all) != 2 || all[0] != t1 || all[1] != t2 {
		t.Fatalf("All() = %v, want [t1, t2]", all)
	}
	if t1.Done || t1.Progress != -1 {
		t.Errorf("fresh task: Done=%v Progress=%d, want running and indeterminate", t1.Done, t1.Progress)
	}
	if ctx1.Err() != nil {
		t.Errorf("fresh task's context already done: %v", ctx1.Err())
	}
}

// Cancel cancels the context Start handed back — what the worker checks to
// stop early — and is safe again, and after the task finished.
func TestTaskCancelStopsItsContext(t *testing.T) {
	r := newTasks(new(Poster), 0)
	task, ctx := r.Start(context.Background(), "Backup")

	task.Cancel()
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Errorf("ctx.Err() after Cancel = %v, want context.Canceled", ctx.Err())
	}
	task.Cancel()
	r.Finish(task, nil)
	task.Cancel()
}

// A Task built by hand, with no context, cancels as a no-op.
func TestTaskCancelWithoutContextIsNoop(t *testing.T) {
	(&Task{Label: "no context"}).Cancel()
}

// Finish releases the context even when nobody cancelled.
func TestTasksFinishReleasesTheContext(t *testing.T) {
	r := newTasks(new(Poster), 0)
	task, ctx := r.Start(context.Background(), "Backup")
	r.Finish(task, nil)
	if ctx.Err() == nil {
		t.Error("a finished task's context is still live")
	}
}

// Progress and Done change nothing until the UI goroutine drains them, and
// Done ends in Finish and OnFinish.
func TestTasksProgressAndDoneArePosted(t *testing.T) {
	p := new(Poster)
	r := newTasks(p, 0)
	var finished []*Task
	r.OnFinish = func(t *Task) { finished = append(finished, t) }
	task, _ := r.Start(context.Background(), "Backup")

	r.Progress(task, 40, "40 percent processed.")
	if task.Progress != -1 || task.Message != "" {
		t.Fatalf("Progress applied before the drain: %d %q", task.Progress, task.Message)
	}
	p.Drain()
	if task.Progress != 40 || task.Message != "40 percent processed." {
		t.Errorf("after drain: %d %q", task.Progress, task.Message)
	}

	r.Done(task, nil)
	if task.Done {
		t.Fatal("Done applied before the drain")
	}
	p.Drain()
	if !task.Done || task.Err != nil || task.Finished.IsZero() {
		t.Errorf("after drain: Done=%v Err=%v Finished=%v", task.Done, task.Err, task.Finished)
	}
	if len(finished) != 1 || finished[0] != task {
		t.Errorf("OnFinish saw %v, want the task once", finished)
	}
}

// The cancelled rule: an error after a Cancel asked for while running is a
// cancel; a success despite one stays a success; an error with no Cancel is a
// failure; and a panic is a failure even after a Cancel.
func TestTasksFinishCancelledRule(t *testing.T) {
	r := newTasks(new(Poster), 0)
	start := func() *Task { task, _ := r.Start(context.Background(), "x"); return task }

	stopped := start()
	stopped.Cancel()
	r.Finish(stopped, context.Canceled)
	if !stopped.Cancelled || !errors.Is(stopped.Err, context.Canceled) {
		t.Errorf("cancelled run: Cancelled=%v Err=%v", stopped.Cancelled, stopped.Err)
	}

	raced := start()
	raced.Cancel()
	r.Finish(raced, nil)
	if raced.Cancelled {
		t.Error("a run that finished despite the cancel reads as cancelled")
	}

	failed := start()
	r.Finish(failed, errors.New("disk full"))
	if failed.Cancelled {
		t.Error("an uncancelled failure reads as cancelled")
	}

	// Finish's own Cancel (releasing the context) must not count as asked for.
	failed.Cancel()
	if failed.Cancelled {
		t.Error("cancelling a finished task changed its outcome")
	}

	panicked := start()
	panicked.Cancel()
	r.Finish(panicked, ErrTaskPanicked)
	if panicked.Cancelled {
		t.Error("a panicked task reads as cancelled")
	}
}

// Running counts only unfinished tasks.
func TestTasksRunning(t *testing.T) {
	r := newTasks(new(Poster), 0)
	t1, _ := r.Start(context.Background(), "Backup")
	_, _ = r.Start(context.Background(), "Rebuild")
	if got := r.Running(); got != 2 {
		t.Fatalf("Running = %d, want 2", got)
	}
	r.Finish(t1, nil)
	if got := r.Running(); got != 1 {
		t.Errorf("Running after finishing one = %d, want 1", got)
	}
}

// The history cap evicts the oldest finished tasks and never a running one,
// even the oldest.
func TestTasksKeepNeverEvictsARunningTask(t *testing.T) {
	const keep = 5
	r := newTasks(new(Poster), keep)
	oldestRunning, _ := r.Start(context.Background(), "still running")
	var last *Task
	for range keep + 10 {
		task, _ := r.Start(context.Background(), "finished")
		r.Finish(task, nil)
		last = task
	}
	_, _ = r.Start(context.Background(), "trigger prune")

	all := r.All()
	if len(all) != keep {
		t.Fatalf("len(All()) = %d, want %d", len(all), keep)
	}
	if all[0] != oldestRunning {
		t.Error("the oldest running task was evicted")
	}
	found := false
	for _, task := range all {
		found = found || task == last
	}
	if !found {
		t.Error("the newest finished task was evicted before older ones")
	}
}

// Keep <= 0 keeps every task.
func TestTasksZeroKeepKeepsEverything(t *testing.T) {
	r := newTasks(new(Poster), 0)
	for range 100 {
		task, _ := r.Start(context.Background(), "x")
		r.Finish(task, nil)
	}
	if n := len(r.All()); n != 100 {
		t.Errorf("len(All()) = %d, want 100", n)
	}
}

// Times: no estimate at 0% or without a percentage; a linear one at 50%; and
// a finished task's elapsed time stops at Finished.
func TestTaskTimes(t *testing.T) {
	start := time.Now().Add(-10 * time.Second)

	if _, _, ok := (&Task{Started: start, Progress: 0}).Times(); ok {
		t.Error("an estimate at 0%")
	}
	if _, _, ok := (&Task{Started: start, Progress: -1}).Times(); ok {
		t.Error("an estimate with no percentage")
	}
	elapsed, remaining, ok := (&Task{Started: start, Progress: 50}).Times()
	if !ok || elapsed < 10*time.Second || remaining < 9*time.Second || remaining > 11*time.Second {
		t.Errorf("at 50%%: elapsed %v remaining %v ok %v, want ~10s remaining", elapsed, remaining, ok)
	}
	fin := start.Add(3 * time.Second)
	elapsed, _, ok = (&Task{Started: start, Finished: fin, Done: true, Progress: 100}).Times()
	if ok || elapsed != 3*time.Second {
		t.Errorf("finished: elapsed %v ok %v, want 3s and no estimate", elapsed, ok)
	}
}
