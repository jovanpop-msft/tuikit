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
	// Both drawn: the HBar stops short of the corner, which is the VBar's.
	if got := in.HBar(); got != (Rect{X: 2, Y: 5, W: 9, H: 1}) {
		t.Errorf("inside HBar = %+v", got)
	}
	// Inside, only the overflowing axis is drawn, and a lone HBar spans View.
	in.ContentH = 4
	if got := in.HBar(); got != (Rect{X: 2, Y: 5, W: 10, H: 1}) {
		t.Errorf("lone inside HBar = %+v", got)
	}
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
	if v.HandleScrollbars(viewportPress(5, 3), true) {
		t.Fatal("a press in the view was taken by the bars")
	}
	if v.HandleScrollbars(tcell.NewEventMouse(12, 5, tcell.Button3, 0), true) {
		t.Fatal("a right press on the bar was taken")
	}
	// Last row of the vertical track: the last offset.
	if !v.HandleScrollbars(viewportPress(12, 5), true) || v.Y != 15 || !v.Dragging() {
		t.Fatalf("press on VBar end -> Y=%d dragging=%v, want 15 true", v.Y, v.Dragging())
	}
	// The latch holds off the bar, and a drag crossing the HBar stays vertical.
	if !v.HandleScrollbars(viewportPress(4, 6), false) || v.Y != 15 || v.X != 0 {
		t.Fatalf("drag onto HBar -> X=%d Y=%d, want 0,15", v.X, v.Y)
	}
	if !v.HandleScrollbars(viewportPress(30, 1), false) || v.Y != 0 {
		t.Fatalf("drag above the track -> Y=%d, want 0", v.Y)
	}
	if !v.Release() || v.Dragging() || v.Release() {
		t.Fatal("Release did not clear the latch exactly once")
	}
	if !v.HandleScrollbars(viewportPress(11, 6), true) || v.X != 30 {
		t.Fatalf("press on HBar end -> X=%d, want 30", v.X)
	}
	v.Release()

	// Inside: a bar on a fitting axis is not drawn, so its press falls through.
	in := newTestViewport(true)
	in.ContentH = 4
	if in.HandleScrollbars(viewportPress(11, 2), true) {
		t.Error("press on an undrawn inside VBar was taken")
	}
	// The corner cell is the vertical bar's (the HBar stops short of it), and
	// its end is the last offset with the bar's line out of the extent.
	in.ContentH = 20
	if !in.HandleScrollbars(viewportPress(11, 5), true) || in.X != 0 || in.Y != 16 {
		t.Errorf("press on the inside corner -> X=%d Y=%d, want 0,16", in.X, in.Y)
	}
	in.Release()
	// The HBar's last cell (one short of the corner) is the last X offset.
	if !in.HandleScrollbars(viewportPress(10, 5), true) || in.X != 31 {
		t.Errorf("press on the inside HBar end -> X=%d, want 31", in.X)
	}
}

// A press that began elsewhere, held and dragged onto either bar, latches
// neither; a latched bar keeps every resend, wherever it lands.
func TestViewportScrollbarNeedsFreshPress(t *testing.T) {
	v := newTestViewport(false)
	if v.HandleScrollbars(viewportPress(12, 5), false) || v.Dragging() || v.Y != 0 {
		t.Fatalf("resend on VBar: Y=%d dragging=%v, want 0 false", v.Y, v.Dragging())
	}
	if v.HandleScrollbars(viewportPress(11, 6), false) || v.Dragging() || v.X != 0 {
		t.Fatalf("resend on HBar: X=%d dragging=%v, want 0 false", v.X, v.Dragging())
	}
	if !v.HandleScrollbars(viewportPress(12, 1), true) || !v.Dragging() {
		t.Fatal("fresh press on VBar did not latch")
	}
	if !v.HandleScrollbars(viewportPress(3, 5), false) || v.Y != 15 {
		t.Fatalf("resend off the bar mid-drag: Y=%d, want 15", v.Y)
	}
	v.Release()
	if !v.HandleScrollbars(viewportPress(2, 6), true) || !v.Dragging() {
		t.Fatal("fresh press on HBar did not latch")
	}
	if !v.HandleScrollbars(viewportPress(30, 1), false) || v.X != 30 || v.Y != 15 {
		t.Fatalf("resend off the H bar mid-drag: X=%d Y=%d, want 30 15", v.X, v.Y)
	}
}

// Inside bars take their line out of the scrollable extent, so the canvas's
// last row and column can be scrolled out from under them (they sat under the
// bars at the largest offset, hiding the plan graph's edge tiles).
func TestViewportInsideBarsReachLastCell(t *testing.T) {
	v := &Viewport{View: Rect{W: 10, H: 5}, ContentW: 40, ContentH: 20, BarsInside: true}
	if w, h := v.Visible(); w != 9 || h != 4 {
		t.Fatalf("Visible = %d,%d, want 9,4", w, h)
	}
	v.ScrollTo(1<<30, 1<<30)
	if v.X != 31 || v.Y != 16 {
		t.Fatalf("largest offset = %d,%d, want 31,16", v.X, v.Y)
	}
	if _, cy := v.ToCanvas(v.HBar().X, v.HBar().Y-1); cy != v.ContentH-1 {
		t.Errorf("row above the HBar = canvas %d, want %d", cy, v.ContentH-1)
	}
	if cx, _ := v.ToCanvas(v.VBar().X-1, v.VBar().Y); cx != v.ContentW-1 {
		t.Errorf("column left of the VBar = canvas %d, want %d", cx, v.ContentW-1)
	}

	// EnsureVisible on the bottom-right cell clears both bars.
	v.ScrollTo(0, 0)
	v.EnsureVisible(Rect{X: 39, Y: 19, W: 1, H: 1})
	if v.X != 31 || v.Y != 16 {
		t.Errorf("EnsureVisible(last cell) -> %d,%d, want 31,16", v.X, v.Y)
	}

	// A page is the visible height less one: 3 rows, not 4.
	v.ScrollTo(0, 0)
	if _, moved := v.ScrollKey(tcell.NewEventKey(tcell.KeyPgDn, "", 0), 1); !moved || v.Y != 3 {
		t.Errorf("PgDn inside -> Y=%d, want 3", v.Y)
	}

	// Outside is unchanged: Visible is View.
	out := &Viewport{View: Rect{W: 10, H: 5}, ContentW: 40, ContentH: 20}
	if w, h := out.Visible(); w != 10 || h != 5 || out.MaxX() != 30 || out.MaxY() != 15 {
		t.Errorf("outside Visible %d,%d Max %d,%d, want 10,5 30,15", w, h, out.MaxX(), out.MaxY())
	}
}

func TestViewportInsideBarsFixedPoint(t *testing.T) {
	tests := []struct {
		name       string
		cw, ch     int
		vb, hb     bool
		maxX, maxY int
	}{
		{"fits", 10, 5, false, false, 0, 0},
		// Exactly View.W wide, overflowing vertically: the VBar covers the
		// last column, which then needs a horizontal scroll (and its bar).
		{"exactly wide, tall", 10, 20, true, true, 1, 16},
		{"exactly tall, wide", 40, 5, true, true, 31, 1},
		// One short of View on the other axis: the second bar never appears.
		{"one narrower, tall", 9, 20, true, false, 0, 15},
		{"one shorter, wide", 40, 4, false, true, 30, 0},
	}
	for _, tt := range tests {
		v := &Viewport{View: Rect{W: 10, H: 5}, ContentW: tt.cw, ContentH: tt.ch, BarsInside: true}
		if vb, hb := v.showsV(), v.showsH(); vb != tt.vb || hb != tt.hb {
			t.Errorf("%s: shows V,H = %v,%v, want %v,%v", tt.name, vb, hb, tt.vb, tt.hb)
		}
		if v.MaxX() != tt.maxX || v.MaxY() != tt.maxY {
			t.Errorf("%s: Max = %d,%d, want %d,%d", tt.name, v.MaxX(), v.MaxY(), tt.maxX, tt.maxY)
		}
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

func TestViewportScrollKey(t *testing.T) {
	key := func(k tcell.Key) *tcell.EventKey { return tcell.NewEventKey(k, "", 0) }
	// View 10x5 over 40x20: MaxX 30, MaxY 15, a page is 4 rows.
	tests := []struct {
		name      string
		k         tcell.Key
		x, y      int // offset before
		wantX     int
		wantY     int
		wantMoved bool
	}{
		{"down", tcell.KeyDown, 0, 0, 0, 1, true},
		{"up at the top", tcell.KeyUp, 3, 0, 3, 0, false},
		{"up", tcell.KeyUp, 0, 5, 0, 4, true},
		{"right by stepX", tcell.KeyRight, 0, 0, 3, 0, true},
		{"right clamps", tcell.KeyRight, 29, 0, 30, 0, true},
		{"right at the edge", tcell.KeyRight, 30, 0, 30, 0, false},
		{"left at the edge", tcell.KeyLeft, 0, 2, 0, 2, false},
		{"left", tcell.KeyLeft, 7, 0, 4, 0, true},
		{"page down", tcell.KeyPgDn, 0, 0, 0, 4, true},
		{"page down clamps", tcell.KeyPgDn, 0, 13, 0, 15, true},
		{"page up", tcell.KeyPgUp, 0, 9, 0, 5, true},
		{"page up clamps", tcell.KeyPgUp, 0, 2, 0, 0, true},
		{"home", tcell.KeyHome, 8, 6, 0, 0, true},
		{"home at the origin", tcell.KeyHome, 0, 0, 0, 0, false},
		{"end keeps the column", tcell.KeyEnd, 8, 6, 8, 15, true},
		{"end at the bottom", tcell.KeyEnd, 8, 15, 8, 15, false},
	}
	for _, tt := range tests {
		v := newTestViewport(false)
		v.X, v.Y = tt.x, tt.y
		bound, moved := v.ScrollKey(key(tt.k), 3)
		if !bound || moved != tt.wantMoved || v.X != tt.wantX || v.Y != tt.wantY {
			t.Errorf("%s: = %v,%v at %d,%d, want true,%v at %d,%d",
				tt.name, bound, moved, v.X, v.Y, tt.wantMoved, tt.wantX, tt.wantY)
		}
	}

	v := newTestViewport(false)
	v.X, v.Y = 2, 2
	for _, ev := range []*tcell.EventKey{key(tcell.KeyTab), key(tcell.KeyEnter), tcell.NewEventKey(tcell.KeyRune, "j", 0)} {
		if bound, moved := v.ScrollKey(ev, 3); bound || moved || v.X != 2 || v.Y != 2 {
			t.Errorf("ScrollKey(%v) = %v,%v at %d,%d, want false,false at 2,2", ev.Name(), bound, moved, v.X, v.Y)
		}
	}

	// A zero-size view still pages by one row, and nothing scrolls past the
	// canvas; an empty canvas never moves.
	z := &Viewport{ContentW: 4, ContentH: 3}
	if _, moved := z.ScrollKey(key(tcell.KeyPgDn), 3); !moved || z.Y != 1 {
		t.Errorf("PgDn in a zero-height view: moved=%v Y=%d, want true 1", moved, z.Y)
	}
	if _, moved := z.ScrollKey(key(tcell.KeyEnd), 3); !moved || z.Y != 3 {
		t.Errorf("End in a zero-height view: moved=%v Y=%d, want true 3", moved, z.Y)
	}
	e := &Viewport{View: Rect{W: 10, H: 5}}
	for _, k := range []tcell.Key{tcell.KeyDown, tcell.KeyRight, tcell.KeyPgDn, tcell.KeyEnd} {
		if bound, moved := e.ScrollKey(key(k), 3); !bound || moved || e.X != 0 || e.Y != 0 {
			t.Errorf("key %v on an empty canvas = %v,%v at %d,%d, want true,false at 0,0", k, bound, moved, e.X, e.Y)
		}
	}
}
