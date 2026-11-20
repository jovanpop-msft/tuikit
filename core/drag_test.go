package core

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

func mouseAt(x, y int, b tcell.ButtonMask) *tcell.EventMouse {
	return tcell.NewEventMouse(x, y, b, tcell.ModNone)
}

// A press held still, or moved less than the threshold, is a click: the drag
// is armed but not started, and Release hands back nothing.
func TestDragBelowThresholdIsAClick(t *testing.T) {
	var d Drag[string]
	d.Arm("obj", 10, 5)
	if !d.Armed() || d.Active() {
		t.Fatalf("after Arm: Armed=%v Active=%v, want true, false", d.Armed(), d.Active())
	}
	d.Motion(10+DefaultDragThreshold-1, 5-(DefaultDragThreshold-1))
	if d.Active() {
		t.Fatal("drag started before the pointer moved DefaultDragThreshold cells")
	}
	if p, ok := d.Release(); ok || p != "" {
		t.Errorf("Release of an unstarted drag = %q, %v, want \"\", false", p, ok)
	}
	if d.Armed() {
		t.Error("Release left the drag armed")
	}
}

// The threshold counts along either axis, in either direction, and once
// started a drag stays started when the pointer comes back.
func TestDragStartsPastThresholdOnEitherAxis(t *testing.T) {
	for _, mv := range [][2]int{{2, 0}, {-2, 0}, {0, 2}, {0, -2}, {1, -2}} {
		var d Drag[int]
		d.Arm(7, 10, 10)
		d.Motion(10+mv[0], 10+mv[1])
		if !d.Active() {
			t.Errorf("move %v did not start the drag", mv)
			continue
		}
		d.Motion(10, 10)
		if x, y := d.Pos(); x != 10 || y != 10 {
			t.Errorf("Pos = %d,%d, want 10,10", x, y)
		}
		if p, ok := d.Release(); !ok || p != 7 {
			t.Errorf("move %v: Release = %d, %v, want 7, true", mv, p, ok)
		}
	}
}

func TestDragThresholdField(t *testing.T) {
	d := Drag[int]{Threshold: 4}
	d.Arm(1, 0, 0)
	d.Motion(3, 0)
	if d.Active() {
		t.Fatal("started at 3 cells with Threshold 4")
	}
	d.Motion(4, 0)
	if !d.Active() {
		t.Fatal("did not start at 4 cells with Threshold 4")
	}
	d.Release()
	if d.Threshold != 4 {
		t.Errorf("Release reset Threshold to %d", d.Threshold)
	}
	d.Arm(1, 0, 0)
	d.Cancel()
	if d.Threshold != 4 {
		t.Errorf("Arm/Cancel reset Threshold to %d", d.Threshold)
	}
}

// Route: a disarmed drag passes events on; an armed one consumes every event,
// tracks a held Button1, and is cancelled by anything else.
func TestDragRoute(t *testing.T) {
	var d Drag[string]
	if d.Route(mouseAt(3, 3, tcell.Button1)) {
		t.Fatal("a disarmed drag consumed an event")
	}
	d.Arm("obj", 0, 0)
	if !d.Route(mouseAt(5, 0, tcell.Button1)) || !d.Active() {
		t.Fatal("a held Button1 past the threshold was not consumed as motion")
	}
	if !d.Route(mouseAt(5, 0, tcell.WheelDown)) {
		t.Fatal("an armed drag did not consume a wheel tick")
	}
	if d.Armed() || d.Active() {
		t.Error("a wheel tick did not cancel the drag")
	}
	if _, ok := d.Release(); ok {
		t.Error("a cancelled drag still dropped")
	}
	// Motion after a cancel must not re-start anything.
	d.Motion(20, 20)
	if d.Active() {
		t.Error("Motion on a disarmed drag started it")
	}
}

func TestDragPayload(t *testing.T) {
	var d Drag[*int]
	if p, ok := d.Payload(); ok || p != nil {
		t.Fatal("zero Drag reports a payload")
	}
	v := 3
	d.Arm(&v, 0, 0)
	if p, ok := d.Payload(); !ok || p != &v {
		t.Fatal("armed Drag lost its payload")
	}
}

// The ghost draws only once the drag started, right of the pointer, and
// flips to end left of it when the pointer sits in the last column.
func TestDragGhost(t *testing.T) {
	b := NewBuffer(10, 3)
	var d Drag[int]
	d.Arm(0, 2, 1)
	d.DrawGhost(b, "abc", tcell.StyleDefault, 10)
	if b.RuneAt(3, 1) != ' ' {
		t.Fatal("ghost drawn before the drag started")
	}
	d.Motion(4, 1)
	d.DrawGhost(b, "abc", tcell.StyleDefault, 10)
	if got := string([]rune{b.RuneAt(5, 1), b.RuneAt(6, 1), b.RuneAt(7, 1)}); got != "abc" {
		t.Errorf("ghost at x=5 reads %q, want abc", got)
	}

	b.Clear()
	d.Motion(9, 2)
	d.DrawGhost(b, "abc", tcell.StyleDefault, 10)
	if got := string([]rune{b.RuneAt(6, 2), b.RuneAt(7, 2), b.RuneAt(8, 2)}); got != "abc" {
		t.Errorf("flipped ghost reads %q at x=6..8, want abc", got)
	}
	if b.RuneAt(9, 2) != ' ' {
		t.Error("flipped ghost covers the pointer's cell")
	}

	// Near (not at) the edge the text is clipped, not flipped.
	b.Clear()
	d.Motion(7, 0)
	d.DrawGhost(b, "abc", tcell.StyleDefault, 10)
	if b.RuneAt(8, 0) != 'a' || b.RuneAt(9, 0) != 'b' {
		t.Errorf("clipped ghost = %q%q, want ab", b.RuneAt(8, 0), b.RuneAt(9, 0))
	}
}
