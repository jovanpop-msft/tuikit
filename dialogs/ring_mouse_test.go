package dialogs

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/tuikittest"
	"github.com/radix29/tuikit/widgets"
)

// ringRig is the smallest dialog RingMouse drives: a text field, a checkbox
// and a drop-down whose list overhangs the box, over an OK/Cancel row. It
// stands in for goSSMS's Log Search, Filter, Options and Profile dialogs and
// tuikit's Find/Replace, which hand-rolled this routing in five orders.
type ringRig struct {
	ModalDialog
	drag    FieldGesture
	field   *widgets.InputField
	cb      *widgets.CheckBox
	dd      *widgets.DropDown
	extra   []core.Focusable
	focus   int
	pressed []int
}

var ringButtons = []string{"OK", "Cancel"}

func newRingRig(t *testing.T) *ringRig {
	t.Helper()
	r := &ringRig{
		field: widgets.NewInputField("Name:", 10, false),
		cb:    widgets.NewCheckBox("Case"),
		dd:    widgets.NewDropDown("Op:", []string{"a", "b", "c", "d", "e", "f", "g", "h"}, 6),
	}
	r.InitModal("Ring", 40, 8)
	r.SetScreen(tuikittest.NewScreen(80, 24)) // box: x 20..59, y 8..15
	r.Show()
	in := r.InnerRect()
	r.field.SetBounds(in.X+1, in.Y)
	r.cb.SetBounds(in.X+1, in.Y+1)
	r.dd.SetBounds(in.X+1, in.Y+2) // the list runs y 12..19, past the box
	r.setFocus(0)
	return r
}

func (r *ringRig) ring() []core.Focusable {
	return append([]core.Focusable{r.field, r.cb, r.dd}, r.extra...)
}

func (r *ringRig) setFocus(i int) {
	r.focus = SetFocusIn(r.ring(), i, r.focus)
	if i == len(r.ring()) {
		r.focus = i
	}
}

func (r *ringRig) mouse(ev *tcell.EventMouse) bool {
	return RingMouse(ev, &r.ModalDialog, &r.drag, r.ring(), r.focus, ringButtons,
		r.setFocus, func(btn int) { r.pressed = append(r.pressed, btn) })
}

func (r *ringRig) okAt() (int, int) { return r.ButtonRowStartX(ringButtons) + 2, r.ButtonRowY() }

// fieldAt is a point inside the text field's box.
func (r *ringRig) fieldAt() (int, int) { return r.field.Bounds().Right() - 2, r.field.Bounds().Y }

// ddHeaderAt is the closed drop-down's value box, which opens and closes it.
func (r *ringRig) ddHeaderAt() (int, int) { return r.dd.Bounds().Right() - 2, r.dd.Bounds().Y }

// listItemAt is item i of the open list, while nothing has scrolled it.
func (r *ringRig) listItemAt(i int) (int, int) {
	return r.dd.Bounds().Right() - 3, r.dd.Bounds().Y + 1 + i
}

// fakeEditor is a focused text field outside FieldGesture's reach that still
// needs its release: what an editor.Editor is to Profile's ForwardRelease.
type fakeEditor struct{ *fakeField }

func (fakeEditor) CancelMouseDrag() {}

// One subtest per step of RingMouse's order. Each fails when its step is
// moved, dropped or answered differently: the rows of the divergence table
// docs/tuikit-reuse-plan-2.md (goSSMS) recorded across the five dialogs.
func TestRingMouse(t *testing.T) {
	t.Run("TrackPress first: a list pick dragged onto OK does not press it", func(t *testing.T) {
		r := newRingRig(t)
		r.mouse(tuikittest.Press(r.ddHeaderAt()))
		r.mouse(tuikittest.Release(r.ddHeaderAt()))
		if !r.dd.OverlayActive() {
			t.Fatal("the drop-down did not open")
		}
		r.mouse(tuikittest.Press(r.listItemAt(2)))
		r.mouse(tuikittest.Press(r.okAt())) // the held button's resend
		if len(r.pressed) != 0 {
			t.Fatalf("a drag from a list pick pressed button %v", r.pressed)
		}
		r.mouse(tuikittest.Release(r.okAt()))
		r.mouse(tuikittest.Press(r.okAt()))
		if len(r.pressed) != 1 || r.pressed[0] != 0 {
			t.Errorf("a fresh click on OK pressed %v, want [0]", r.pressed)
		}
	})

	t.Run("a release outside the box reaches a checkbox's latch", func(t *testing.T) {
		r := newRingRig(t)
		at := r.cb.Bounds()
		r.mouse(tuikittest.Press(at.X, at.Y))
		if !r.cb.Checked() {
			t.Fatal("the press did not toggle the checkbox")
		}
		r.mouse(tuikittest.Release(0, 0))
		r.mouse(tuikittest.Press(at.X, at.Y))
		if r.cb.Checked() {
			t.Error("the next press was swallowed as the stale gesture's resend")
		}
	})

	t.Run("a release outside the box ends a field's drag", func(t *testing.T) {
		r := newRingRig(t)
		r.mouse(tuikittest.Press(r.fieldAt()))
		if r.drag.Field() != r.field {
			t.Fatal("a press in the field did not claim the gesture")
		}
		if !r.mouse(tuikittest.Release(0, 0)) {
			t.Error("a release outside the box was not spoken for")
		}
		if r.drag.Field() != nil {
			t.Error("the field still holds the gesture after the release")
		}
	})

	t.Run("an open list overhanging the box gets the click first", func(t *testing.T) {
		r := newRingRig(t)
		r.mouse(tuikittest.Press(r.ddHeaderAt()))
		r.mouse(tuikittest.Release(r.ddHeaderAt()))
		x, y := r.listItemAt(6)
		if r.ContainsMouse(x, y) {
			t.Fatalf("item 6 at y=%d is inside the box; the rig must overhang it", y)
		}
		if !r.mouse(tuikittest.Press(x, y)) {
			t.Fatal("the pick was not spoken for")
		}
		if r.dd.Selected() != 6 {
			t.Errorf("the pick outside the box selected %d, want 6", r.dd.Selected())
		}
		if r.focus != 2 {
			t.Errorf("focus = %d after picking from the list, want the drop-down (2)", r.focus)
		}
	})

	t.Run("an open list covers the field below it", func(t *testing.T) {
		r := newRingRig(t)
		// Move the field under the list's first item.
		r.field.SetBounds(r.dd.Bounds().X, r.dd.Bounds().Y+1)
		r.mouse(tuikittest.Press(r.ddHeaderAt()))
		r.mouse(tuikittest.Release(r.ddHeaderAt()))
		r.mouse(tuikittest.Press(r.listItemAt(0)))
		if r.drag.Field() != nil {
			t.Error("the press on the list went to the field under it")
		}
		if r.dd.Selected() != 0 || r.dd.OverlayActive() {
			t.Error("the list did not take the pick")
		}
	})

	t.Run("a press outside the box with no list open does nothing", func(t *testing.T) {
		r := newRingRig(t)
		if !r.mouse(tuikittest.Press(1, 1)) {
			t.Error("a press outside the modal box leaked to the host")
		}
		if r.focus != 0 || len(r.pressed) != 0 || r.cb.Checked() {
			t.Error("a press outside the box acted on the dialog")
		}
	})

	t.Run("a focused text field outside the gesture gets its release", func(t *testing.T) {
		r := newRingRig(t)
		ed := fakeEditor{&fakeField{bounds: core.Rect{X: 22, Y: 13, W: 10, H: 1}}}
		r.extra = []core.Focusable{ed}
		r.setFocus(3)
		r.mouse(tuikittest.Release(23, 13))
		if len(ed.mouse) != 1 || ed.mouse[0] != tcell.ButtonNone {
			t.Errorf("the focused editor got %v, want exactly one release", ed.mouse)
		}
	})

	t.Run("the wheel inside the box is left to the host", func(t *testing.T) {
		r := newRingRig(t)
		x, y := r.fieldAt()
		if r.mouse(tuikittest.Mouse(x, y, tcell.WheelDown)) {
			t.Error("a wheel tick inside the box was spoken for; RingMouse answers false")
		}
		if !r.mouse(tuikittest.Mouse(1, 1, tcell.WheelDown)) {
			t.Error("a wheel tick outside the box leaked to the host")
		}
	})

	t.Run("Replay: a selection drag onto OK extends and does not press it", func(t *testing.T) {
		r := newRingRig(t)
		r.field.SetValue("hello")
		in := r.field.Bounds()
		r.mouse(tuikittest.Press(in.Right()-1, in.Y)) // past the text: the cursor at its end
		r.mouse(tuikittest.Press(in.X-1, in.Y))       // left of the box: the drag
		if !r.field.HasSelection() {
			t.Error("the selection froze when the drag left the field's box")
		}
		r.mouse(tuikittest.Press(r.okAt()))
		if len(r.pressed) != 0 {
			t.Fatalf("a selection drag pressed button %v", r.pressed)
		}
		if r.drag.Field() != r.field {
			t.Error("the drag left the field's gesture")
		}
	})

	t.Run("a click on a button focuses the row and presses it", func(t *testing.T) {
		r := newRingRig(t)
		r.mouse(tuikittest.Press(r.okAt()))
		if len(r.pressed) != 1 || r.pressed[0] != 0 || r.focus != len(r.ring()) {
			t.Errorf("pressed %v, focus %d; want [0] on the button row (%d)",
				r.pressed, r.focus, len(r.ring()))
		}
	})

	t.Run("a click in a field claims the gesture and focuses it", func(t *testing.T) {
		r := newRingRig(t)
		r.setFocus(1)
		r.mouse(tuikittest.Press(r.fieldAt()))
		if r.focus != 0 || r.drag.Field() != r.field {
			t.Errorf("focus %d, gesture %v; want the field", r.focus, r.drag.Field())
		}
	})

	t.Run("a disabled field is not claimed", func(t *testing.T) {
		r := newRingRig(t)
		// A fake, not the InputField: its HitTest already refuses when
		// disabled, which would hide RingMouse's own check.
		ed := fakeEditor{&fakeField{bounds: core.Rect{X: 22, Y: 13, W: 10, H: 1}, disabled: true}}
		r.extra = []core.Focusable{ed}
		r.setFocus(1)
		r.mouse(tuikittest.Press(23, 13))
		if r.focus != 1 || r.drag.Field() != nil {
			t.Errorf("a disabled field took focus %d / gesture %v", r.focus, r.drag.Field())

		}
	})

	t.Run("a self-routing member takes its press and focus", func(t *testing.T) {
		r := newRingRig(t)
		at := r.cb.Bounds()
		r.mouse(tuikittest.Press(at.X, at.Y))
		if !r.cb.Checked() || r.focus != 1 {
			t.Errorf("checked %v, focus %d; want toggled and focused (1)", r.cb.Checked(), r.focus)
		}
	})
}
