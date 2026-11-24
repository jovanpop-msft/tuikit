package controls

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

// A host's own ValueViewer works from the zero value, and passes every event
// on while hidden, so holding one costs a host nothing until it is shown.
func TestValueViewerStandalone(t *testing.T) {
	var v ValueViewer
	if v.OverlayActive() || v.HandleKey(tuikittest.Key(tcell.KeyDown)) ||
		v.HandleMouse(tcell.NewEventMouse(1, 1, tcell.Button1, tcell.ModNone)) {
		t.Fatal("a hidden viewer claimed input")
	}

	v.Show("Definition", "line one\nline two")
	scr := tuikittest.NewScreen(80, 24)
	v.DrawOverlay(scr)
	if !v.OverlayActive() || v.Title() != "Definition" || v.Text() != "line one\nline two" {
		t.Fatalf("after Show: active %v, title %q, text %q", v.OverlayActive(), v.Title(), v.Text())
	}
	if _, _, ok := scr.Find("Definition"); !ok {
		t.Error("the title was not drawn")
	}
	if _, _, ok := scr.Find("line two"); !ok {
		t.Error("the value was not drawn")
	}

	v.SelectAll()
	if got := v.SelectedText(); got != "line one\nline two" {
		t.Errorf("SelectedText() = %q after SelectAll", got)
	}
	if !v.HandleKey(tuikittest.Key(tcell.KeyEscape)) || v.OverlayActive() {
		t.Error("Escape did not close the viewer")
	}
	if v.HasSelection() || v.SelectedText() != "" {
		t.Error("a hidden viewer still reports its selection")
	}
}

// showDrawnViewer opens a viewer on a value long enough to fill its text area
// and returns it with the "[ Close ]" button's screen position.
func showDrawnViewer(t *testing.T) (v *ValueViewer, cx, cy int) {
	t.Helper()
	v = new(ValueViewer)
	v.Show("Value", strings.Repeat("lorem ipsum dolor sit amet ", 40))
	s := tuikittest.NewScreen(80, 24)
	v.DrawOverlay(s)
	cx, cy, ok := s.Find("[ Close ]")
	if !ok {
		t.Fatal("setup: [ Close ] was not drawn")
	}
	return v, cx, cy
}

// A text selection dragged onto Close with the button still held must not
// close the popup: tcell resends the held Button1 on every motion, and the
// first resend over Close used to read as a fresh press there.
func TestValueViewerSelectionDragAcrossCloseStaysOpen(t *testing.T) {
	v, cx, cy := showDrawnViewer(t)
	v.HandleMouse(tuikittest.Press(cx-20, cy-6)) // press in the text
	v.HandleMouse(tuikittest.Press(cx-10, cy-1)) // drag down
	v.HandleMouse(tuikittest.Press(cx+2, cy))    // still held, over Close
	if !v.OverlayActive() {
		t.Fatal("a selection drag crossing [ Close ] closed the popup")
	}
	if !v.HasSelection() {
		t.Error("the drag selected nothing")
	}
	if !v.HandleMouse(tuikittest.Release(cx+2, cy)) || !v.OverlayActive() {
		t.Error("the release over Close was not consumed, or closed the popup")
	}
	if !v.HasSelection() {
		t.Error("the release over Close lost the selection")
	}
}

// A press on Close closes the popup, and the rest of its gesture (resends
// while held, then the release) is swallowed so nothing underneath sees it.
func TestValueViewerClosePressSwallowsItsGesture(t *testing.T) {
	v, cx, cy := showDrawnViewer(t)
	if !v.HandleMouse(tuikittest.Press(cx+1, cy)) || v.OverlayActive() {
		t.Fatal("a press on [ Close ] did not close the popup")
	}
	if !v.HandleMouse(tuikittest.Press(cx-30, cy-5)) {
		t.Error("a held resend after the closing press reached the host")
	}
	if !v.HandleMouse(tuikittest.Release(cx-30, cy-5)) {
		t.Error("the closing press's release reached the host")
	}
	if v.HandleMouse(tuikittest.Press(cx-30, cy-5)) {
		t.Error("a hidden viewer claimed a new click after the gesture ended")
	}
}

// A press on the hint row (not the text, not Close) that drifts onto Close
// keeps the popup open.
func TestValueViewerHintPressDraggedOntoCloseStaysOpen(t *testing.T) {
	v, cx, cy := showDrawnViewer(t)
	v.HandleMouse(tuikittest.Press(cx-30, cy)) // the hint text
	v.HandleMouse(tuikittest.Press(cx+2, cy))
	if !v.OverlayActive() {
		t.Fatal("a press on the hint dragged onto [ Close ] closed the popup")
	}
	v.HandleMouse(tuikittest.Release(cx+2, cy))
	if !v.OverlayActive() {
		t.Error("the release over Close closed the popup")
	}
}

// A re-Show after a press closed the popup, its release never delivered,
// takes its first click: the closing gesture does not survive the showing.
func TestValueViewerReshowAfterPressCloseTakesFirstClick(t *testing.T) {
	v, cx, cy := showDrawnViewer(t)
	v.HandleMouse(tuikittest.Press(cx+1, cy)) // closes; no release
	if v.OverlayActive() {
		t.Fatal("setup: the press did not close the popup")
	}
	v.Show("Value", "short")
	s := tuikittest.NewScreen(80, 24)
	v.DrawOverlay(s)
	cx, cy, _ = s.Find("[ Close ]")
	v.HandleMouse(tuikittest.Press(cx+1, cy))
	if v.OverlayActive() {
		t.Error("the first click on [ Close ] after a re-Show was swallowed")
	}
}
