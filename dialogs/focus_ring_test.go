package dialogs

import (
	"slices"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
)

// fakeFocusable records what Focus was last told, so a test can assert which
// entry of a list actually ended up focused rather than only which index was
// returned. The two can disagree — that is the bug worth catching.
type fakeFocusable struct{ focused bool }

func (f *fakeFocusable) Focus(on bool) { f.focused = on }
func (f *fakeFocusable) Focused() bool { return f.focused }

func focusList(n int) ([]core.Focusable, []*fakeFocusable) {
	real := make([]*fakeFocusable, n)
	list := make([]core.Focusable, n)
	for i := range real {
		real[i] = &fakeFocusable{}
		list[i] = real[i]
	}
	return list, real
}

// focusedIn reports which entries claim focus. More than one is the failure
// SetFocusIn's blur-everything-first loop exists to prevent.
func focusedIn(real []*fakeFocusable) []int {
	var on []int
	for i, f := range real {
		if f.focused {
			on = append(on, i)
		}
	}
	return on
}

func TestSetFocusInFocusesExactlyOne(t *testing.T) {
	list, real := focusList(4)

	if got := SetFocusIn(list, 2, 0); got != 2 {
		t.Errorf("SetFocusIn(_, 2, 0) = %d, want 2", got)
	}
	if on := focusedIn(real); len(on) != 1 || on[0] != 2 {
		t.Errorf("focused = %v, want exactly [2]", on)
	}

	// Moving focus must blur the previous holder, not add a second one.
	SetFocusIn(list, 0, 2)
	if on := focusedIn(real); len(on) != 1 || on[0] != 0 {
		t.Errorf("after moving to 0, focused = %v, want exactly [0]", on)
	}
}

// An out-of-range index blurs everything and leaves the caller's index alone.
// A dialog that rebuilds a shorter ring on a mode change and refocuses its old
// index reaches this path in normal use.
func TestSetFocusInOutOfRangeKeepsCallersIndex(t *testing.T) {
	list, real := focusList(3)
	SetFocusIn(list, 1, 0)

	for _, i := range []int{-1, 3, 99} {
		if got := SetFocusIn(list, i, 7); got != 7 {
			t.Errorf("SetFocusIn(_, %d, 7) = %d, want the caller's 7 back", i, got)
		}
		if on := focusedIn(real); on != nil {
			t.Errorf("SetFocusIn(_, %d, 7) left %v focused, want nothing", i, on)
		}
	}

	// An empty list is the same answer, without indexing anything.
	if got := SetFocusIn(nil, 0, 4); got != 4 {
		t.Errorf("SetFocusIn(nil, 0, 4) = %d, want 4", got)
	}
}

func TestIndexOfFocusableIsByIdentity(t *testing.T) {
	list, real := focusList(3)
	if got := IndexOfFocusable(list, real[2]); got != 2 {
		t.Errorf("IndexOfFocusable(_, real[2]) = %d, want 2", got)
	}
	// A distinct widget that compares equal field-for-field is still absent:
	// the callers hold the very pointers they put in the list.
	if got := IndexOfFocusable(list, &fakeFocusable{}); got != -1 {
		t.Errorf("IndexOfFocusable of a widget not in the list = %d, want -1", got)
	}
}

// Tab and Backtab wrap at both ends.
func TestFocusCyclingWrapsBothWays(t *testing.T) {
	const n = 4
	for idx, want := range map[int]int{0: 1, 2: 3, 3: 0} {
		if got := NextFocus(idx, n); got != want {
			t.Errorf("NextFocus(%d, %d) = %d, want %d", idx, n, got, want)
		}
	}
	for idx, want := range map[int]int{3: 2, 1: 0, 0: 3} {
		if got := PrevFocus(idx, n); got != want {
			t.Errorf("PrevFocus(%d, %d) = %d, want %d", idx, n, got, want)
		}
	}
	// A single-entry cycle stays put in both directions rather than moving
	// to an index it doesn't have.
	if got := NextFocus(0, 1); got != 0 {
		t.Errorf("NextFocus(0, 1) = %d, want 0", got)
	}
	if got := PrevFocus(0, 1); got != 0 {
		t.Errorf("PrevFocus(0, 1) = %d, want 0", got)
	}
}

func TestScrollToShow(t *testing.T) {
	for _, tt := range []struct {
		name               string
		sel, scroll, dataH int
		want               int
	}{
		{"already visible scrolls nothing", 5, 3, 10, 3},
		{"at the top edge stays", 3, 3, 10, 3},
		{"at the bottom edge stays", 12, 3, 10, 3},
		{"above the viewport scrolls up to sel", 1, 3, 10, 1},
		{"below the viewport scrolls the minimum", 13, 3, 10, 4},
		{"far below jumps so sel is the last row", 40, 3, 10, 31},
		// A viewport with no rows has nothing to reveal. Unguarded, the
		// "past the bottom" arm is always true here and answers sel+1 —
		// scrolled past the selection, which draws as an empty pane.
		{"zero-height viewport leaves the offset alone", 5, 3, 0, 3},
		{"zero-height viewport at offset zero stays there", 0, 0, 0, 0},
		{"negative height leaves the offset alone", 5, 3, -1, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ScrollToShow(tt.sel, tt.scroll, tt.dataH); got != tt.want {
				t.Errorf("ScrollToShow(%d, %d, %d) = %d, want %d",
					tt.sel, tt.scroll, tt.dataH, got, tt.want)
			}
		})
	}
}

func TestStepButtonSkipsDisabledAndStopsAtTheEnds(t *testing.T) {
	disabled := []bool{false, true, false}
	for _, tt := range []struct {
		name      string
		i, dir, n int
		disabled  []bool
		want      int
	}{
		{"right past a disabled button", 0, +1, 3, disabled, 2},
		{"left past a disabled button", 2, -1, 3, disabled, 0},
		{"right at the end stays", 2, +1, 3, disabled, 2},
		{"left at the start stays", 0, -1, 3, disabled, 0},
		{"nil disabled is all enabled", 0, +1, 3, nil, 1},
		{"short disabled enables the rest", 1, +1, 4, []bool{false, false}, 2},
		{"only disabled buttons ahead stays", 0, +1, 3, []bool{false, true, true}, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := StepButton(tt.i, tt.dir, tt.n, tt.disabled); got != tt.want {
				t.Errorf("StepButton(%d, %d, %d, %v) = %d, want %d",
					tt.i, tt.dir, tt.n, tt.disabled, got, tt.want)
			}
		})
	}
}

func key(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, "", tcell.ModNone) }

func TestButtonRowKey(t *testing.T) {
	var fired, hidden int
	var left []int
	btn := 0
	press := func(ev *tcell.EventKey) {
		t.Helper()
		if !ButtonRowKey(ev, &btn, 3, []bool{false, true, false},
			func() { fired++ }, func() { hidden++ }, func(dir int) { left = append(left, dir) }) {
			t.Errorf("key %v was not swallowed", ev.Key())
		}
	}

	press(key(tcell.KeyRight))
	if btn != 2 {
		t.Errorf("Right from 0 past disabled 1 landed on %d, want 2", btn)
	}
	press(key(tcell.KeyLeft))
	if btn != 0 {
		t.Errorf("Left from 2 past disabled 1 landed on %d, want 0", btn)
	}
	press(key(tcell.KeyEnter))
	press(key(tcell.KeyEscape))
	press(key(tcell.KeyTab))
	press(key(tcell.KeyBacktab))
	press(key(tcell.KeyDown))
	press(key(tcell.KeyUp))
	if fired != 1 || hidden != 1 {
		t.Errorf("fired=%d hidden=%d, want 1 each", fired, hidden)
	}
	if want := []int{+1, -1, +1, -1}; !slices.Equal(left, want) {
		t.Errorf("leave got %v, want %v", left, want)
	}

	// A letter typed on the row is swallowed and changes nothing.
	press(tcell.NewEventKey(tcell.KeyRune, "a", tcell.ModNone))
	if btn != 0 || fired != 1 || hidden != 1 || len(left) != 4 {
		t.Error("a typed letter did something on the button row")
	}
}

// fakeField is a text field for the clipboard and release helpers: a
// core.ClipboardTarget and a core.MouseHandler, with an enabled state, an
// optional overlay and a HitTest narrower than its Bounds (an InputField's box
// beside its label).
type fakeField struct {
	fakeFocusable
	bounds, box core.Rect
	disabled    bool
	overlay     bool
	mouse       []tcell.ButtonMask
}

func (f *fakeField) HasSelection() bool       { return false }
func (f *fakeField) SelectedText() string     { return "" }
func (f *fakeField) Cut() string              { return "" }
func (f *fakeField) Paste(string)             {}
func (f *fakeField) SelectAll()               {}
func (f *fakeField) CanCopy() bool            { return true }
func (f *fakeField) CanCut() bool             { return true }
func (f *fakeField) CanPaste() bool           { return true }
func (f *fakeField) CanSelectAll() bool       { return true }
func (f *fakeField) Enabled() bool            { return !f.disabled }
func (f *fakeField) SetEnabled(v bool)        { f.disabled = !v }
func (f *fakeField) Bounds() core.Rect        { return f.bounds }
func (f *fakeField) OverlayActive() bool      { return f.overlay }
func (f *fakeField) DrawOverlay(tcell.Screen) {}
func (f *fakeField) HandleMouse(ev *tcell.EventMouse) bool {
	f.mouse = append(f.mouse, ev.Buttons())
	return true
}

// fakeToggle is a mouse-driven non-text control (a checkbox): never a
// clipboard target, never handed a second release.
type fakeToggle struct {
	fakeFocusable
	bounds core.Rect
	mouse  int
}

func (f *fakeToggle) Bounds() core.Rect                  { return f.bounds }
func (f *fakeToggle) HandleMouse(*tcell.EventMouse) bool { f.mouse++; return true }

type boxedField struct{ *fakeField }

func (b boxedField) HitTest(x, y int) bool { return b.box.Contains(x, y) }

func TestFocusedClipboardTargetIsNilOffATextField(t *testing.T) {
	field := &fakeField{}
	list := []core.Focusable{&fakeToggle{}, field}
	if got := FocusedClipboardTarget(list, 1); got != field {
		t.Errorf("FocusedClipboardTarget on the field = %v, want the field", got)
	}
	for _, i := range []int{0, -1, 2} {
		if got := FocusedClipboardTarget(list, i); got != nil {
			t.Errorf("FocusedClipboardTarget(_, %d) = %v, want a nil interface", i, got)
		}
	}
}

func TestClipboardFieldAt(t *testing.T) {
	toggle := &fakeToggle{bounds: core.Rect{X: 0, Y: 0, W: 10, H: 1}}
	plain := &fakeField{bounds: core.Rect{X: 0, Y: 1, W: 10, H: 1}}
	boxed := boxedField{&fakeField{
		bounds: core.Rect{X: 0, Y: 2, W: 20, H: 1},
		box:    core.Rect{X: 10, Y: 2, W: 10, H: 1},
	}}
	off := &fakeField{bounds: core.Rect{X: 0, Y: 3, W: 10, H: 1}, disabled: true}
	list := []core.Focusable{toggle, plain, boxed, off}

	for _, tt := range []struct {
		name string
		x, y int
		want int
	}{
		{"a checkbox is never a target", 1, 0, -1},
		{"a field without HitTest is hit on its Bounds", 1, 1, 1},
		{"a field with HitTest is hit on its box", 12, 2, 2},
		{"a field with HitTest is missed on its label", 1, 2, -1},
		{"a disabled field is skipped", 1, 3, -1},
		{"empty space", 50, 50, -1},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClipboardFieldAt(list, tt.x, tt.y); got != tt.want {
				t.Errorf("ClipboardFieldAt(_, %d, %d) = %d, want %d", tt.x, tt.y, got, tt.want)
			}
		})
	}

	// An open overlay is drawn over the fields: nothing beneath it is hit.
	toggleOverlay := &fakeField{overlay: true}
	if got := ClipboardFieldAt(append(list, toggleOverlay), 1, 1); got != -1 {
		t.Errorf("ClipboardFieldAt with an overlay open = %d, want -1", got)
	}
}

func TestForwardReleaseToFocusedField(t *testing.T) {
	release := tcell.NewEventMouse(1, 1, tcell.ButtonNone, tcell.ModNone)
	press := tcell.NewEventMouse(1, 1, tcell.Button1, tcell.ModNone)

	field := &fakeField{}
	if ForwardReleaseToFocusedField(press, field) {
		t.Error("a press was reported as the release")
	}
	if len(field.mouse) != 0 {
		t.Error("a press was forwarded")
	}
	if !ForwardReleaseToFocusedField(release, field) {
		t.Error("the release was not reported")
	}
	if len(field.mouse) != 1 || field.mouse[0] != tcell.ButtonNone {
		t.Errorf("the text field got %v, want one release", field.mouse)
	}

	// A checkbox already had its release; a second would toggle it.
	toggle := &fakeToggle{}
	if !ForwardReleaseToFocusedField(release, toggle) {
		t.Error("the release on a checkbox was not reported")
	}
	if toggle.mouse != 0 {
		t.Error("the release was forwarded to a non-text control")
	}

	if !ForwardReleaseToFocusedField(release, nil) {
		t.Error("the release with nothing focused was not reported")
	}
}

// fakeEnabler is a field that can be greyed out, for RingStep's skip.
type fakeEnabler struct {
	fakeFocusable
	off bool
}

func (f *fakeEnabler) SetEnabled(on bool) { f.off = !on }
func (f *fakeEnabler) Enabled() bool      { return !f.off }

func TestRingStepWrapsThroughTheButtonRowAndSkipsDisabled(t *testing.T) {
	a, b, c := &fakeEnabler{}, &fakeEnabler{off: true}, &fakeEnabler{}
	ring := []core.Focusable{a, b, c} // the button row is stop 3
	tests := []struct {
		name           string
		cur, dir, want int
	}{
		{"forward past a disabled field", 0, +1, 2},
		{"last field onto the row", 2, +1, 3},
		{"row wraps to the first field", 3, +1, 0},
		{"first field back onto the row", 0, -1, 3},
		{"row back to the last field", 3, -1, 2},
		{"back past a disabled field", 2, -1, 0},
		{"out of range counts as the row", 9, +1, 0},
		{"negative counts as the row", -1, -1, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RingStep(ring, tt.cur, tt.dir); got != tt.want {
				t.Errorf("RingStep(_, %d, %d) = %d, want %d", tt.cur, tt.dir, got, tt.want)
			}
		})
	}
	if got := RingStep([]core.Focusable{b}, 1, +1); got != 1 {
		t.Errorf("RingStep over an all-disabled ring = %d, want the row (1)", got)
	}
	if got := RingStep(nil, 0, +1); got != 0 {
		t.Errorf("RingStep(nil) = %d, want the row (0)", got)
	}
}

// keyField consumes only the keys it is told to, as a widget declines the
// ring's keys.
type keyField struct {
	fakeFocusable
	takes tcell.Key
	got   int
}

func (f *keyField) HandleKey(ev *tcell.EventKey) bool {
	if ev.Key() == f.takes {
		f.got++
		return true
	}
	return false
}

func TestFieldRingKeyOffersTheWidgetFirst(t *testing.T) {
	key := func(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, "", tcell.ModNone) }
	var moved []int
	fired, hid := 0, 0
	run := func(f core.Focusable, k tcell.Key) {
		if !FieldRingKey(key(k), f, func() { fired++ }, func() { hid++ }, func(d int) { moved = append(moved, d) }) {
			t.Fatalf("FieldRingKey(%v) = false, want every key swallowed", k)
		}
	}

	// A widget that declines: the ring gets every key.
	plain := &keyField{}
	for _, k := range []tcell.Key{tcell.KeyTab, tcell.KeyDown, tcell.KeyBacktab, tcell.KeyUp} {
		run(plain, k)
	}
	if want := []int{+1, +1, -1, -1}; !slices.Equal(moved, want) {
		t.Fatalf("moves = %v, want %v", moved, want)
	}
	run(plain, tcell.KeyEnter)
	run(plain, tcell.KeyEscape)
	run(nil, tcell.KeyEnter)
	if fired != 2 || hid != 1 {
		t.Fatalf("fired=%d hid=%d, want 2 and 1", fired, hid)
	}

	// A widget that takes the key keeps it from the ring.
	moved, fired, hid = nil, 0, 0
	for _, k := range []tcell.Key{tcell.KeyDown, tcell.KeyEnter, tcell.KeyEscape} {
		f := &keyField{takes: k}
		run(f, k)
		if f.got != 1 {
			t.Errorf("%v: widget saw it %d times, want 1", k, f.got)
		}
	}
	if len(moved) != 0 || fired != 0 || hid != 0 {
		t.Fatalf("ring acted on consumed keys: moves=%v fired=%d hid=%d", moved, fired, hid)
	}
}
