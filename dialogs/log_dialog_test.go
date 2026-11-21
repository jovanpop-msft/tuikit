package dialogs

import (
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

func newTestLog(t *testing.T, spec LogDialogSpec) (*LogDialog, *tuikittest.Screen) {
	t.Helper()
	if spec.W == 0 {
		spec.Title, spec.W, spec.H = "Log", 60, 20
	}
	scr := tuikittest.NewScreen(100, 40)
	d := NewLogDialog(spec)
	d.SetScreen(scr)
	return d, scr
}

func editorLines(d *LogDialog) int { return strings.Count(d.Editor().Text(), "\n") + 1 }

func TestLogDialogRecordsNewestFirstAndCaps(t *testing.T) {
	d, _ := newTestLog(t, LogDialogSpec{Max: 5})
	for i := range 8 {
		d.Record("line " + strconv.Itoa(i))
	}
	got := d.Lines()
	if len(got) != 5 || got[0] != "line 7" || got[4] != "line 3" {
		t.Errorf("Lines() = %q, want line 7 … line 3", got)
	}
}

// Record must not touch the editor (it may run on a background goroutine);
// Show catches it up, and a line recorded while open reaches it on the next
// Draw.
func TestLogDialogSyncsOnShowAndDraw(t *testing.T) {
	d, scr := newTestLog(t, LogDialogSpec{})
	d.Record("hidden")
	if !d.Dirty() || d.Editor().Text() != "" {
		t.Fatal("Record updated the editor itself")
	}
	d.Show()
	if d.Dirty() || d.Editor().Text() != "hidden" {
		t.Fatalf("after Show: dirty %v, text %q", d.Dirty(), d.Editor().Text())
	}
	d.Record("visible")
	d.Draw(scr)
	if !strings.HasPrefix(d.Editor().Text(), "visible\n") {
		t.Errorf("after Draw, text = %q, want the new line first", d.Editor().Text())
	}
	if scr.RowOf("visible") < 0 {
		t.Error("the new line was not drawn")
	}
}

func TestLogDialogShowKeepsOrResets(t *testing.T) {
	keep, _ := newTestLog(t, LogDialogSpec{})
	keep.Record("a")
	keep.Show()
	keep.Hide()
	keep.Show()
	if keep.Len() != 1 {
		t.Errorf("without ResetOnShow, Len() = %d after reopening, want 1", keep.Len())
	}

	reset, _ := newTestLog(t, LogDialogSpec{ResetOnShow: true})
	reset.Show()
	reset.Record("a")
	reset.Sync()
	reset.Editor().SelectAll()
	reset.Show()
	if reset.Len() != 0 || reset.Editor().Text() != "" || reset.Editor().HasSelection() {
		t.Errorf("with ResetOnShow: Len %d, text %q, selection %v; want an empty log",
			reset.Len(), reset.Editor().Text(), reset.Editor().HasSelection())
	}
}

func TestLogDialogDrawsTheEmptyHint(t *testing.T) {
	d, scr := newTestLog(t, LogDialogSpec{Empty: "Nothing yet."})
	d.Show()
	d.Draw(scr)
	if scr.RowOf("Nothing yet.") < 0 {
		t.Error("empty hint not drawn")
	}
	d.Record("x")
	d.Draw(scr)
	if scr.RowOf("Nothing yet.") >= 0 {
		t.Error("empty hint drawn over a non-empty log")
	}
}

// A line arriving while text is selected keeps the selection on its text: a
// rebuild with SetText dropped it, so a second Copy copied nothing.
func TestLogDialogKeepsTheSelectionWhenALineArrives(t *testing.T) {
	d, scr := newTestLog(t, LogDialogSpec{})
	for i := range 5 {
		d.Record("line " + strconv.Itoa(i))
	}
	d.Show()
	d.Draw(scr)
	d.Editor().SelectRange(1, 0, 3, 0)
	before := d.Editor().SelectedText()

	d.Record("Copied to clipboard")
	d.Draw(scr)

	if got := d.Editor().SelectedText(); got != before || got == "" {
		t.Errorf("selection %q became %q", before, got)
	}
}

// A line arriving keeps a scrolled view on the same text.
func TestLogDialogKeepsAScrolledView(t *testing.T) {
	d, scr := newTestLog(t, LogDialogSpec{})
	for i := range 100 {
		d.Record("line " + strconv.Itoa(i))
	}
	d.Show()
	d.Editor().GoToLine(60)
	d.Draw(scr)
	top, _ := d.Editor().ScrollPos()
	if top == 0 {
		t.Fatal("the view did not scroll to begin with")
	}
	d.Record("new")
	d.Draw(scr)
	if got, _ := d.Editor().ScrollPos(); got != top+1 {
		t.Errorf("scroll row = %d, want %d", got, top+1)
	}
}

func TestLogDialogTrimsTheEditorAtMax(t *testing.T) {
	d, scr := newTestLog(t, LogDialogSpec{Max: 10})
	for i := range 10 {
		d.Record(strconv.Itoa(i))
	}
	d.Show()
	d.Record("one more")
	d.Draw(scr)
	if got := editorLines(d); got != 10 {
		t.Errorf("editor holds %d lines, want 10", got)
	}
}

// Multi-line records arriving while the dialog is open must leave the editor
// as a fresh Show would: the trim follows the records kept, not Max lines.
// Recorded from a goroutine; run with -race.
func TestLogDialogSyncKeepsWholeMultiLineRecords(t *testing.T) {
	d, scr := newTestLog(t, LogDialogSpec{Max: 4})
	d.Show()
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := range 40 {
			line := "msg " + strconv.Itoa(i)
			if i%3 == 0 {
				line += "\n  detail A\r\n  detail B"
			}
			d.Record(line)
		}
	})
	for range 20 {
		d.Draw(scr)
	}
	wg.Wait()
	d.Draw(scr)
	if got, want := d.Editor().Text(), strings.Join(d.Lines(), "\n"); got != strings.ReplaceAll(want, "\r\n", "\n") {
		t.Errorf("synced editor = %q, want the retained records %q", got, want)
	}
}

// Record from several goroutines while the UI goroutine draws. Run with -race.
func TestLogDialogRecordIsSafeFromBackgroundGoroutines(t *testing.T) {
	d, scr := newTestLog(t, LogDialogSpec{Max: 256})
	d.Show()
	var wg sync.WaitGroup
	for w := range 4 {
		wg.Go(func() {
			for i := range 100 {
				d.Record("worker " + strconv.Itoa(w) + " line " + strconv.Itoa(i))
			}
		})
	}
	for range 50 {
		d.Draw(scr)
	}
	wg.Wait()
	d.Draw(scr)
	if d.Len() != 256 || editorLines(d) != 256 {
		t.Errorf("Len %d, editor lines %d; want 256 each", d.Len(), editorLines(d))
	}
}

// The editor gets first refusal of keys; a read-only one refuses Escape and
// Enter, which close.
func TestLogDialogEscapeAndEnterClose(t *testing.T) {
	for _, k := range []tcell.Key{tcell.KeyEscape, tcell.KeyEnter} {
		d, _ := newTestLog(t, LogDialogSpec{})
		d.Record("x")
		d.Show()
		d.HandleKey(key(k))
		if d.Visible() {
			t.Errorf("key %v did not close the dialog", k)
		}
	}
}

// The last button closes; any other runs OnButton and leaves the dialog open.
func TestLogDialogButtons(t *testing.T) {
	var pressed []int
	d, scr := newTestLog(t, LogDialogSpec{
		Buttons:  []string{"Copy", "Close"},
		OnButton: func(b int) { pressed = append(pressed, b) },
	})
	d.Record("x")
	d.Show()
	d.Draw(scr)

	click := func(label string) {
		x, y, ok := scr.Find("[ " + label + " ]")
		if !ok {
			t.Fatalf("button %q not drawn", label)
		}
		d.HandleMouse(tuikittest.Press(x+2, y))
		d.HandleMouse(tuikittest.Release(x+2, y))
	}
	click("Copy")
	if len(pressed) != 1 || pressed[0] != 0 || !d.Visible() {
		t.Fatalf("Copy: pressed %v, visible %v; want [0] and still open", pressed, d.Visible())
	}
	click("Close")
	if len(pressed) != 1 || d.Visible() {
		t.Errorf("Close: pressed %v, visible %v; want no OnButton and closed", pressed, d.Visible())
	}
}

// The default button row is Close alone.
func TestLogDialogDefaultsToClose(t *testing.T) {
	d, scr := newTestLog(t, LogDialogSpec{})
	d.Show()
	d.Draw(scr)
	if scr.RowOf("[ Close ]") < 0 {
		t.Error("no Close button drawn")
	}
}

func TestLogDialogIsTheClipboardTarget(t *testing.T) {
	d, scr := newTestLog(t, LogDialogSpec{})
	d.Record("copy me")
	d.Show()
	d.Draw(scr)
	if d.FocusedClipboardTarget() == nil {
		t.Fatal("no clipboard target")
	}
	x, y, _ := scr.Find("copy me")
	if !d.FocusClipboardTargetAt(x, y) {
		t.Error("the log text is not a clipboard focus point")
	}
}
