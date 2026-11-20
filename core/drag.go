package core

import "github.com/gdamore/tcell/v3"

// DefaultDragThreshold is Drag's start distance when its Threshold is zero.
const DefaultDragThreshold = 2

// Drag is a drag-and-drop gesture carrying a payload P from where a Button1
// press armed it to where the button is released (goSSMS drags an Object
// Explorer node into a query editor).
//
// Arming is not starting: the drag starts only once the held pointer has
// moved Threshold cells or more from the press along either axis. Until then
// there is no ghost and Release hands back nothing, so a press held still is a
// click, not a drop.
//
// The zero value is disarmed. A host keeps one and, in its mouse handler:
//
//   - Release on ButtonNone, before Route sees it, and drops the payload where
//     the pointer is only if Release reports the drag started — and only if
//     the drop is still valid (goSSMS drops nothing when a modal opened
//     mid-drag);
//   - Route ahead of its other routing, so an armed drag takes every event
//     until the release and a drag crossing a menu row or a splitter doesn't
//     act on it;
//   - Arm on a fresh press over something draggable, from what the press hit,
//     never from the host's current selection (a press on a scrollbar leaves
//     the selection, and arming from it drags the wrong object).
//
// The drop target, what is draggable and the ghost's text are the host's.
type Drag[P any] struct {
	// Threshold is how far, in cells along either axis, the pointer must move
	// from the press before the drag starts; zero means DefaultDragThreshold.
	Threshold int

	payload        P
	x, y           int
	startX, startY int
	armed, active  bool
}

// Arm readies a drag of p from a press at (x, y). The drag has not started.
func (d *Drag[P]) Arm(p P, x, y int) {
	*d = Drag[P]{Threshold: d.Threshold, payload: p, x: x, y: y, startX: x, startY: y, armed: true}
}

// Motion tracks the held pointer to (x, y), starting an armed drag once it is
// Threshold cells or more from the press. A started drag stays started if the
// pointer comes back. Motion on a disarmed drag does nothing.
func (d *Drag[P]) Motion(x, y int) {
	if !d.armed {
		return
	}
	d.x, d.y = x, y
	th := d.Threshold
	if th <= 0 {
		th = DefaultDragThreshold
	}
	dx, dy := x-d.startX, y-d.startY
	if max(dx, -dx, dy, -dy) >= th {
		d.active = true
	}
}

// Route delivers an event to an armed drag: a held Button1 is Motion, and
// anything else (a wheel tick, another button) cancels the drag. It reports
// whether the drag was armed, in which case the event is consumed; false
// means the caller routes it as usual. The release (ButtonNone) must be
// handled with Release before Route sees it, or the drop is lost.
func (d *Drag[P]) Route(ev *tcell.EventMouse) bool {
	if !d.armed {
		return false
	}
	if ev.Buttons() == tcell.Button1 {
		d.Motion(ev.Position())
	} else {
		d.Cancel()
	}
	return true
}

// Release ends the gesture and disarms the drag. It returns the payload and
// true only if the drag had started; an armed drag released before moving
// Threshold cells was a click and returns the zero P and false.
func (d *Drag[P]) Release() (P, bool) {
	p, started := d.payload, d.active
	d.Cancel()
	if !started {
		var zero P
		return zero, false
	}
	return p, true
}

// Cancel disarms the drag without a drop, keeping Threshold.
func (d *Drag[P]) Cancel() { *d = Drag[P]{Threshold: d.Threshold} }

// Armed reports whether a press armed the drag and it has not been released
// or cancelled, started or not.
func (d *Drag[P]) Armed() bool { return d.armed }

// Active reports whether the armed drag has started.
func (d *Drag[P]) Active() bool { return d.active }

// Payload returns the armed drag's payload, and whether one is armed.
func (d *Drag[P]) Payload() (P, bool) { return d.payload, d.armed }

// Pos returns the pointer's last position: the press, or the latest Motion.
func (d *Drag[P]) Pos() (x, y int) { return d.x, d.y }

// DrawGhost draws text beside the pointer while the drag is active, so the
// payload looks picked up and carried: one cell right of the pointer, or
// ending left of it when the pointer is in the last column of a screen
// screenW wide. Text running off the right edge is clipped. A drag that is
// disarmed or not yet started draws nothing.
func (d *Drag[P]) DrawGhost(s tcell.Screen, text string, style tcell.Style, screenW int) {
	if !d.active {
		return
	}
	x := d.x + 1
	if x >= screenW {
		x = max(0, d.x-DisplayWidth(text))
	}
	if maxW := screenW - x; maxW > 0 {
		DrawTextClipped(s, x, d.y, maxW, style, text)
	}
}
