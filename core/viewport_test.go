package core

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

func newTestViewport(inside bool) *Viewport {
	return &Viewport{View: Rect{X: 2, Y: 1, W: 10, H: 5}, ContentW: 40, ContentH: 20, BarsInside: inside}
}

func TestViewportClampAndScroll(t *testing.T) {
	v := newTestViewport(false)
	if v.MaxX() != 30 || v.MaxY() != 15 {
		t.Fatalf("Max = %d,%d, want 30,15", v.MaxX(), v.MaxY())
	}
	if !v.ScrollTo(100, -3) || v.X != 30 || v.Y != 0 {
		t.Fatalf("ScrollTo(100,-3) -> %d,%d, want 30,0", v.X, v.Y)
	}
	if v.ScrollBy(5, 0) {
		t.Error("ScrollBy past the end reported a move")
	}
	if !v.ScrollBy(-4, 2) || v.X != 26 || v.Y != 2 {
		t.Errorf("ScrollBy(-4,2) -> %d,%d, want 26,2", v.X, v.Y)
	}
	// A smaller canvas under an older offset.
	v.ContentW, v.ContentH = 12, 3
	if !v.Clamp() || v.X != 2 || v.Y != 0 {
		t.Errorf("Clamp -> %d,%d, want 2,0", v.X, v.Y)
	}
	if v.Clamp() {
		t.Error("second Clamp reported a change")
	}
}

func TestViewportEnsureVisibleAndConvert(t *testing.T) {
	v := newTestViewport(false)
	v.EnsureVisible(Rect{X: 20, Y: 8, W: 3, H: 2})
	if v.X != 13 || v.Y != 5 {
		t.Fatalf("EnsureVisible past the end -> %d,%d, want 13,5", v.X, v.Y)
	}
	v.EnsureVisible(Rect{X: 4, Y: 1, W: 3, H: 2})
	if v.X != 4 || v.Y != 1 {
		t.Fatalf("EnsureVisible before the start -> %d,%d, want 4,1", v.X, v.Y)
	}
	// Larger than View: the top-left edge wins.
	v.EnsureVisible(Rect{X: 6, Y: 3, W: 30, H: 10})
	if v.X != 6 || v.Y != 3 {
		t.Fatalf("EnsureVisible oversized -> %d,%d, want 6,3", v.X, v.Y)
	}
	cx, cy := v.ToCanvas(2, 1)
	if cx != 6 || cy != 3 {
		t.Errorf("ToCanvas(View top-left) = %d,%d, want 6,3", cx, cy)
	}
	if sx, sy := v.ToScreen(cx, cy); sx != 2 || sy != 1 {
		t.Errorf("ToScreen round trip = %d,%d, want 2,1", sx, sy)
	}
}

func TestViewportBarsPlacementAndDraw(t *testing.T) {
	track, thumb := tcell.StyleDefault.Bold(true), tcell.StyleDefault.Reverse(true)

	out := newTestViewport(false)
	if got := out.VBar(); got != (Rect{X: 12, Y: 1, W: 1, H: 5}) {
		t.Errorf("outside VBar = %+v", got)
	}
	if got := out.HBar(); got != (Rect{X: 2, Y: 6, W: 10, H: 1}) {
		t.Errorf("outside HBar = %+v", got)
	}
	// Outside tracks are drawn even when the canvas fits, and the corner filled.
	out.ContentW, out.ContentH = 5, 5
	b := NewBuffer(20, 10)
	b.Fill('x', tcell.StyleDefault)
	out.DrawScrollbars(b, track, thumb)
	if b.RuneAt(12, 1) != '│' || b.RuneAt(2, 6) != '─' {
		t.Errorf("outside tracks not drawn on a fitting canvas: %q %q", b.RuneAt(12, 1), b.RuneAt(2, 6))
	}
	if b.RuneAt(12, 6) != ' ' {
		t.Errorf("corner = %q, want blank", b.RuneAt(12, 6))
	}

	in := newTestViewport(true)
	if got := in.VBar(); got != (Rect{X: 11, Y: 1, W: 1, H: 5}) {
		t.Errorf("inside VBar = %+v", got)
	}
	if got := in.HBar(); got != (Rect{X: 2, Y: 5, W: 10, H: 1}) {
		t.Errorf("inside HBar = %+v", got)
	}
	// Inside, only the overflowing axis is drawn.
	in.ContentH = 5
	b.Fill('x', tcell.StyleDefault)
	in.DrawScrollbars(b, track, thumb)
	if b.RuneAt(11, 1) != 'x' {
		t.Errorf("inside VBar drawn on a fitting axis: %q", b.RuneAt(11, 1))
	}
	if r := b.RuneAt(2, 5); r != '─' && r != '█' {
		t.Errorf("inside HBar not drawn on overflow: %q", r)
	}
	if b.RuneAt(12, 6) != 'x' {
		t.Error("inside bars filled an outside corner")
	}
}

func viewportPress(x, y int) *tcell.EventMouse { return tcell.NewEventMouse(x, y, tcell.Button1, 0) }

func TestViewportScrollbarDrag(t *testing.T) {
	v := newTestViewport(false)
	if v.HandleScrollbars(viewportPress(5, 3)) {
		t.Fatal("a press in the view was taken by the bars")
	}
	if v.HandleScrollbars(tcell.NewEventMouse(12, 5, tcell.Button3, 0)) {
		t.Fatal("a right press on the bar was taken")
	}
	// Last row of the vertical track: the last offset.
	if !v.HandleScrollbars(viewportPress(12, 5)) || v.Y != 15 || !v.Dragging() {
		t.Fatalf("press on VBar end -> Y=%d dragging=%v, want 15 true", v.Y, v.Dragging())
	}
	// The latch holds off the bar, and a drag crossing the HBar stays vertical.
	if !v.HandleScrollbars(viewportPress(4, 6)) || v.Y != 15 || v.X != 0 {
		t.Fatalf("drag onto HBar -> X=%d Y=%d, want 0,15", v.X, v.Y)
	}
	if !v.HandleScrollbars(viewportPress(30, 1)) || v.Y != 0 {
		t.Fatalf("drag above the track -> Y=%d, want 0", v.Y)
	}
	if !v.Release() || v.Dragging() || v.Release() {
		t.Fatal("Release did not clear the latch exactly once")
	}
	if !v.HandleScrollbars(viewportPress(11, 6)) || v.X != 30 {
		t.Fatalf("press on HBar end -> X=%d, want 30", v.X)
	}
	v.Release()

	// Inside: a bar on a fitting axis is not drawn, so its press falls through.
	in := newTestViewport(true)
	in.ContentH = 5
	if in.HandleScrollbars(viewportPress(11, 2)) {
		t.Error("press on an undrawn inside VBar was taken")
	}
	// The cell both inside bars cover goes to the horizontal one.
	in.ContentH = 20
	if !in.HandleScrollbars(viewportPress(11, 5)) || in.X != 30 || in.Y != 0 {
		t.Errorf("press on the inside corner -> X=%d Y=%d, want 30,0", in.X, in.Y)
	}
}

func TestViewportWheel(t *testing.T) {
	tests := []struct {
		btn    tcell.ButtonMask
		mod    tcell.ModMask
		dx, dy int
	}{
		{tcell.WheelDown, 0, 0, 3},
		{tcell.WheelUp, 0, 0, -3},
		{tcell.WheelDown, tcell.ModShift, 4, 0},
		{tcell.WheelUp, tcell.ModShift, -4, 0},
		{tcell.WheelRight, 0, 4, 0},
		{tcell.WheelLeft, 0, -4, 0},
	}
	for _, tt := range tests {
		dx, dy, ok := WheelDelta(tcell.NewEventMouse(0, 0, tt.btn, tt.mod), 4, 3)
		if !ok || dx != tt.dx || dy != tt.dy {
			t.Errorf("WheelDelta(%v,%v) = %d,%d,%v, want %d,%d", tt.btn, tt.mod, dx, dy, ok, tt.dx, tt.dy)
		}
	}
	if _, _, ok := WheelDelta(viewportPress(0, 0), 4, 3); ok {
		t.Error("WheelDelta took a Button1 press")
	}
	v := newTestViewport(false)
	if wheel, moved := v.Wheel(tcell.NewEventMouse(0, 0, tcell.WheelUp, 0), 4, 1); !wheel || moved {
		t.Errorf("Wheel up at the top = %v,%v, want true,false", wheel, moved)
	}
	if wheel, moved := v.Wheel(tcell.NewEventMouse(0, 0, tcell.WheelDown, 0), 4, 1); !wheel || !moved || v.Y != 1 {
		t.Errorf("Wheel down = %v,%v Y=%d, want true,true 1", wheel, moved, v.Y)
	}
	if wheel, _ := v.Wheel(viewportPress(0, 0), 4, 1); wheel {
		t.Error("Wheel took a Button1 press")
	}
}
