package core

import "github.com/gdamore/tcell/v3"

// Viewport is the scroll state of a view onto a canvas larger than it in both
// axes: the offset and its clamp, wheel steps, the two scrollbars and their
// drag latches. It is a value the host embeds, not a control: the host draws
// the content (canvas cell X,Y at View's top-left), routes the keys (ScrollKey
// binds the navigation ones), and decides what else a scroll does (drop a
// tooltip, ...) from the moved results.
//
// The geometry is fields the host sets before each use (View, ContentW,
// ContentH, BarsInside), since only the host knows when its canvas or rect
// changed; Clamp afterwards keeps an older offset inside a smaller canvas.
//
// Bars outside (the default) sit in the column right of View and the row below
// it, which the host reserves: both tracks are always drawn, so a bar
// appearing on overflow never changes View (that oscillated at certain sizes
// in Activity Monitor). Bars inside overlay View's last column and row, and
// are drawn only on an overflowing axis, so content that fits is not covered;
// a drawn inside bar takes its line out of the scrollable extent (Visible), so
// the canvas's last column/row can be scrolled out from under it.
type Viewport struct {
	X, Y int // the canvas cell at View's top-left

	View               Rect // where the content is drawn
	ContentW, ContentH int  // the canvas size
	BarsInside         bool // bars over View's last column/row, not beside it

	// vDrag/hDrag latch a scrollbar for the whole Button1 gesture, so a drag
	// wandering off the bar keeps scrolling; Release clears them.
	vDrag, hDrag bool
}

// insideBars reports which inside bars are drawn: those on an overflowing
// axis, where the extent left by the other drawn bar is what overflows. It is
// a fixed point, since one bar taking a line can make the other axis overflow
// (a canvas exactly View.W wide that overflows vertically shows both bars);
// the second pass settles it, as a bar once on only shrinks the other extent.
// Both false outside, where the bars take no line of View.
func (v *Viewport) insideBars() (vb, hb bool) {
	if !v.BarsInside {
		return false, false
	}
	vb = v.ContentH > v.View.H
	hb = v.ContentW > v.View.W-b2i(vb)
	vb = v.ContentH > v.View.H-b2i(hb)
	hb = v.ContentW > v.View.W-b2i(vb)
	return vb, hb
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Visible is the canvas extent shown at once: View's size less the line each
// drawn inside bar covers. Outside, View's size.
func (v *Viewport) Visible() (w, h int) {
	vb, hb := v.insideBars()
	return max(v.View.W-b2i(vb), 0), max(v.View.H-b2i(hb), 0)
}

// MaxX is the largest horizontal offset: zero when the canvas fits.
func (v *Viewport) MaxX() int {
	w, _ := v.Visible()
	return max(v.ContentW-w, 0)
}

// MaxY is the largest vertical offset: zero when the canvas fits.
func (v *Viewport) MaxY() int {
	_, h := v.Visible()
	return max(v.ContentH-h, 0)
}

// Clamp keeps the offset within [0, Max] on each axis and reports whether it
// changed.
func (v *Viewport) Clamp() bool {
	return v.ScrollTo(v.X, v.Y)
}

// ScrollTo moves the offset, clamped, and reports whether it moved: a host
// returns that from HandleKey so a key at a boundary falls through.
func (v *Viewport) ScrollTo(x, y int) bool {
	x, y = Clamp(x, 0, v.MaxX()), Clamp(y, 0, v.MaxY())
	if x == v.X && y == v.Y {
		return false
	}
	v.X, v.Y = x, y
	return true
}

// ScrollBy is ScrollTo relative to the current offset.
func (v *Viewport) ScrollBy(dx, dy int) bool {
	return v.ScrollTo(v.X+dx, v.Y+dy)
}

// ScrollKey scrolls by a navigation key: Left/Right by stepX columns, Up/Down
// by one row, PgUp/PgDn by the visible height less one row of overlap (at least
// one), Home to the origin, End to the last row with the column kept (as a
// text view's End does; a host wanting the far corner handles End itself
// first). bound reports whether the key is one of these, moved whether the
// offset changed: a host returns moved so a key at a boundary falls through,
// or bound to claim the key either way. Modifiers are not looked at; a host
// that gives Shift+arrow or Ctrl+End its own meaning tests it first.
func (v *Viewport) ScrollKey(ev *tcell.EventKey, stepX int) (bound, moved bool) {
	_, h := v.Visible()
	page := max(h-1, 1)
	switch ev.Key() {
	case tcell.KeyUp:
		return true, v.ScrollBy(0, -1)
	case tcell.KeyDown:
		return true, v.ScrollBy(0, 1)
	case tcell.KeyLeft:
		return true, v.ScrollBy(-stepX, 0)
	case tcell.KeyRight:
		return true, v.ScrollBy(stepX, 0)
	case tcell.KeyPgUp:
		return true, v.ScrollBy(0, -page)
	case tcell.KeyPgDn:
		return true, v.ScrollBy(0, page)
	case tcell.KeyHome:
		return true, v.ScrollTo(0, 0)
	case tcell.KeyEnd:
		return true, v.ScrollTo(v.X, v.MaxY())
	}
	return false, false
}

// EnsureVisible scrolls the least distance that brings canvas rect r into
// the visible extent, clear of inside bars (its top-left edge winning when r
// is larger), and reports whether the offset moved.
func (v *Viewport) EnsureVisible(r Rect) bool {
	w, h := v.Visible()
	x, y := v.X, v.Y
	if r.Right() > x+w {
		x = r.Right() - w
	}
	if r.X < x {
		x = r.X
	}
	if r.Bottom() > y+h {
		y = r.Bottom() - h
	}
	if r.Y < y {
		y = r.Y
	}
	return v.ScrollTo(x, y)
}

// ToCanvas converts a screen cell to the canvas cell drawn there.
func (v *Viewport) ToCanvas(sx, sy int) (cx, cy int) {
	return sx - v.View.X + v.X, sy - v.View.Y + v.Y
}

// ToScreen converts a canvas cell to the screen cell it is drawn at (which may
// lie outside View).
func (v *Viewport) ToScreen(cx, cy int) (sx, sy int) {
	return cx - v.X + v.View.X, cy - v.Y + v.View.Y
}

// VBar is the vertical scrollbar's rect: the column right of View, or View's
// last column with BarsInside.
func (v *Viewport) VBar() Rect {
	x := v.View.Right()
	if v.BarsInside {
		x--
	}
	return Rect{X: x, Y: v.View.Y, W: 1, H: v.View.H}
}

// HBar is the horizontal scrollbar's rect: the row below View, or View's last
// row with BarsInside. Inside with both bars drawn it stops one column short,
// so the corner cell is the vertical bar's alone.
func (v *Viewport) HBar() Rect {
	y, w := v.View.Bottom(), v.View.W
	if v.BarsInside {
		y--
		if vb, _ := v.insideBars(); vb {
			w = max(w-1, 0)
		}
	}
	return Rect{X: v.View.X, Y: y, W: w, H: 1}
}

// showsV/showsH report whether a bar is drawn, and so whether a press on it is
// the bar's: outside always, inside only on overflow (insideBars, not
// MaxY/MaxX, which are derived from it).
func (v *Viewport) showsV() bool {
	vb, _ := v.insideBars()
	return v.View.H > 0 && (!v.BarsInside || vb)
}

func (v *Viewport) showsH() bool {
	_, hb := v.insideBars()
	return v.View.W > 0 && (!v.BarsInside || hb)
}

// DrawScrollbars draws the bars at the current offset. Outside, the corner
// cell where the two meet is filled with track, so it isn't left stale.
func (v *Viewport) DrawScrollbars(s tcell.Screen, track, thumb tcell.Style) {
	w, h := v.Visible()
	if v.showsV() {
		b := v.VBar()
		DrawScrollbar(s, b.X, b.Y, b.H, v.ContentH, h, v.Y, track, thumb)
	}
	if v.showsH() {
		b := v.HBar()
		DrawScrollbarH(s, b.X, b.Y, b.W, v.ContentW, w, v.X, track, thumb)
	}
	if !v.BarsInside && v.View.W > 0 && v.View.H > 0 {
		PutRune(s, v.View.Right(), v.View.Bottom(), ' ', track)
	}
}

// HandleScrollbars handles a Button1 press or drag on either bar: a fresh
// press on a drawn bar latches it and jumps the offset to the pointer, and
// every Button1 after it, until Release, drags that bar wherever the pointer
// is. fresh is the caller's verdict that ev is the first Button1 of its
// gesture, not a resend of a held press (see HandleScrollbarDrag): a press
// that began elsewhere, held and dragged onto a bar, does not latch it.
// Reports whether the event was the bars'; anything else (another button, a
// press off the bars) returns false with nothing changed, so a host can chain
// it before its own hit-testing. A press on a drawn bar is the bar's even when
// that axis fits (nothing to scroll), so it does not fall through to the
// content under an inside bar.
//
// A latched bar alone takes the drag: a vertical drag crossing the horizontal
// bar stays vertical.
func (v *Viewport) HandleScrollbars(ev *tcell.EventMouse, fresh bool) bool {
	if ev.Buttons() != tcell.Button1 {
		return false
	}
	mx, my := ev.Position()
	if !v.vDrag && !v.hDrag {
		switch {
		case !fresh:
			return false
		case v.showsH() && v.HBar().Contains(mx, my):
			v.hDrag = true
		case v.showsV() && v.VBar().Contains(mx, my):
			v.vDrag = true
		default:
			return false
		}
	}
	w, h := v.Visible()
	if v.vDrag {
		b := v.VBar()
		v.ScrollTo(v.X, ScrollOffsetForDrag(my-b.Y, b.H, v.ContentH, h))
	} else {
		b := v.HBar()
		v.ScrollTo(ScrollOffsetForDrag(mx-b.X, b.W, v.ContentW, w), v.Y)
	}
	return true
}

// Dragging reports whether a scrollbar drag is latched.
func (v *Viewport) Dragging() bool { return v.vDrag || v.hDrag }

// Release ends a scrollbar drag (call it on the ButtonNone release) and
// reports whether one was latched.
func (v *Viewport) Release() bool {
	held := v.Dragging()
	v.vDrag, v.hDrag = false, false
	return held
}

// Wheel scrolls by one wheel tick (WheelDelta). wheel reports whether ev was
// a wheel tick at all (a host claims it either way, or lets it fall through
// at a boundary), moved whether the offset changed.
func (v *Viewport) Wheel(ev *tcell.EventMouse, stepX, stepY int) (wheel, moved bool) {
	dx, dy, ok := WheelDelta(ev, stepX, stepY)
	if !ok {
		return false, false
	}
	return true, v.ScrollBy(dx, dy)
}

// WheelDelta is the scroll one wheel tick asks for: WheelUp/WheelDown move
// stepY rows, WheelLeft/WheelRight stepX columns, and Shift turns
// WheelUp/WheelDown sideways (the desktop convention, which some terminals
// report instead of WheelLeft/WheelRight). ok is false for any other event.
func WheelDelta(ev *tcell.EventMouse, stepX, stepY int) (dx, dy int, ok bool) {
	shift := ev.Modifiers()&tcell.ModShift != 0
	switch ev.Buttons() {
	case tcell.WheelUp:
		if shift {
			return -stepX, 0, true
		}
		return 0, -stepY, true
	case tcell.WheelDown:
		if shift {
			return stepX, 0, true
		}
		return 0, stepY, true
	case tcell.WheelLeft:
		return -stepX, 0, true
	case tcell.WheelRight:
		return stepX, 0, true
	}
	return 0, 0, false
}
