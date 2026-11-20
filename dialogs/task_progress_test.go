package dialogs

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"

	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/loop"
	"github.com/radix29/tuikit/theme"
	"github.com/radix29/tuikit/tuikittest"
	"github.com/radix29/tuikit/widgets"
)

// taskRows renders rows [from, to) of s as text from column x, trailing blanks
// trimmed.
func taskRows(s *tuikittest.Screen, x, from, to int) []string {
	var out []string
	for y := from; y < to; y++ {
		out = append(out, strings.TrimRight(s.Row(y)[x:], " "))
	}
	return out
}

// finishedTask has a fixed 1h 2m 3s run, so the elapsed row is deterministic.
func finishedTask(err error) *loop.Task {
	start := time.Date(2026, 9, 24, 10, 0, 0, 0, time.UTC)
	return &loop.Task{Progress: 40, Done: true, Err: err, Started: start,
		Finished: start.Add(time.Hour + 2*time.Minute + 3*time.Second)}
}

func barText(w, pct int) string {
	s := tuikittest.NewScreen(w, 1)
	widgets.DrawProgressBar(s, 0, 0, w, pct, tcell.StyleDefault)
	return strings.TrimRight(s.Row(0), " ")
}

func TestDrawTaskProgressLaysOutHeaderBarTimesAndMessage(t *testing.T) {
	inner := core.Rect{X: 0, Y: 0, W: 42, H: 20}
	for _, tc := range []struct {
		name  string
		label bool
		err   error
		want  []string
	}{
		{"succeeded, labelled", true, nil, []string{
			"Database : Sales",
			"",
			"Progress:",
			"",
			barText(40, 100), // a success fills the bar whatever was last reported
			"",
			"Elapsed  : 01:02:03",
			"Remaining: --:--:--",
			"",
			"Completed successfully.",
		}},
		{"failed, unlabelled", false, errors.New("disk full"), []string{
			"Database : Sales",
			"",
			barText(40, 40),
			"",
			"Elapsed  : 01:02:03",
			"Remaining: --:--:--",
			"",
			"Failed: disk full",
			"",
			"",
		}},
	} {
		s := tuikittest.NewScreen(42, 20)
		DrawTaskProgress(s, inner, 19, finishedTask(tc.err),
			TaskProgressView{Header: []string{"Database : Sales"}, ProgressLabel: tc.label})
		got := taskRows(s, 1, 1, 11)
		if strings.Join(got, "\n") != strings.Join(tc.want, "\n") {
			t.Errorf("%s:\n%s\nwant:\n%s", tc.name, strings.Join(got, "\n"), strings.Join(tc.want, "\n"))
		}
	}
}

func TestDrawTaskProgressWithoutATaskDrawsOnlyTheHeader(t *testing.T) {
	s := tuikittest.NewScreen(30, 10)
	DrawTaskProgress(s, core.Rect{W: 30, H: 10}, 9, nil, TaskProgressView{Header: []string{"A", "B"}})
	if got, want := strings.TrimRight(s.Snapshot(), "\n"), "\n A\n B"; got != want {
		t.Errorf("snapshot = %q, want %q", got, want)
	}
}

// A long failure gets every row down to bottom, in the error style, and stops
// there.
func TestDrawTaskProgressWrapsTheMessageDownToBottom(t *testing.T) {
	long := strings.Repeat("word ", 40)
	s := tuikittest.NewScreen(22, 20)
	DrawTaskProgress(s, core.Rect{W: 22, H: 20}, 12, finishedTask(errors.New(long)), TaskProgressView{})
	// No header: bar on row 2, times on 4-5, message from row 7 to row 11.
	for y := 7; y < 12; y++ {
		if strings.TrimSpace(s.Row(y)) == "" {
			t.Errorf("row %d is empty; the message must use every row above bottom", y)
		}
		if st := s.StyleAt(1, y); st != theme.StyleDialogError() {
			t.Errorf("row %d style = %v, want StyleDialogError()", y, st)
		}
	}
	if strings.TrimSpace(s.Row(12)) != "" {
		t.Errorf("row 12 (bottom) = %q, want it left alone", s.Row(12))
	}
}

func TestDrawTaskProgressUsesTheHostsMessage(t *testing.T) {
	s := tuikittest.NewScreen(30, 12)
	DrawTaskProgress(s, core.Rect{W: 30, H: 12}, 11, finishedTask(nil), TaskProgressView{
		Message: func(*loop.Task) (string, bool) { return "Backup completed.", false },
	})
	if s.RowOf("Backup completed.") != 7 {
		t.Errorf("host message not on row 7:\n%s", s.Snapshot())
	}
}

func TestTaskMessage(t *testing.T) {
	cancelled := finishedTask(errors.New("context canceled"))
	cancelled.Cancelled = true
	for _, tc := range []struct {
		t     *loop.Task
		want  string
		isErr bool
	}{
		{&loop.Task{}, "Starting...", false},
		{&loop.Task{Message: "50 percent processed."}, "50 percent processed.", false},
		{finishedTask(nil), "Completed successfully.", false},
		{finishedTask(errors.New("boom")), "Failed: boom", true},
		{cancelled, "Cancelled.", true},
	} {
		msg, isErr := TaskMessage(tc.t)
		if msg != tc.want || isErr != tc.isErr {
			t.Errorf("TaskMessage = %q, %v; want %q, %v", msg, isErr, tc.want, tc.isErr)
		}
	}
}
