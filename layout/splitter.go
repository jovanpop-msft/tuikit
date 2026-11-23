package layout

import (
	"math"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// ---------------------------------------------------------------------------
// Splitter
// ---------------------------------------------------------------------------

// SplitterDir describes the orientation of a Splitter.
type SplitterDir int

const (
	SplitterVertical   SplitterDir = iota // column divider (left/right panes)
	SplitterHorizontal                    // row divider (top/bottom panes)
)

// SplitterCollapse names the pane a Splitter has collapsed, if any.
type SplitterCollapse int

const (
	CollapseNone   SplitterCollapse = iota // both panes and the bar show
	CollapseFirst                          // the first (top or left) pane is hidden
	CollapseSecond                         // the second (bottom or right) pane is hidden
)

// Splitter is a draggable divider between two panel regions.
// It owns the geometry calculation; the caller owns the two child panels.
type Splitter struct {
	rect     core.Rect
	dir      SplitterDir
	ratio    float64 // fraction of the rect dimension given to the first pane
	minRatio float64
	maxRatio float64
	dragging bool
	dragBase int    // screen coordinate where drag started
	posBase  int    // SplitPos when the drag started
	label    string // optional label drawn on the splitter bar
	active   bool   // see Focus
	collapse SplitterCollapse

	// press is held from the first Button1 of a gesture until its release,
	// wherever the press landed: HandleMouse calls Fresh on every event, not only
	// on a hit. Without it, tcell's all-motion tracking resends Button1 on every
	// motion, so an editor text-selection drag crossing the bar would grab it and
	// resize the panes.
	press core.PressLatch
}

// Default ratio limits of a new Splitter: neither pane can be dragged below a
// tenth of the rect. SetLimits changes them.
const (
	defaultMinRatio = 0.1
	defaultMaxRatio = 0.9
)

// NewSplitter creates a splitter giving ratio of the rect to the first pane
// (top or left), within the default limits until SetLimits.
func NewSplitter(dir SplitterDir, ratio float64) *Splitter {
	sp := new(Splitter{dir: dir, ratio: 0.5, minRatio: defaultMinRatio, maxRatio: defaultMaxRatio})
	sp.SetRatio(ratio)
	return sp
}

// ResizeHint is the suffix a stacked pane's label carries to say how its bar
// moves: a drag, or the Ctrl+Up/Down HandleKey answers to.
const ResizeHint = "(drag or Ctrl+Up/Down to resize)"

// NewStackSplitter is the stacked-panes preset: a horizontal bar halving the
// rect, within the default limits, with label drawn on it as the title strip
// of the second pane (e.g. "─── Results ─── " + ResizeHint); "" draws none.
// SetRatio and SetLimits still apply.
func NewStackSplitter(label string) *Splitter {
	sp := NewSplitter(SplitterHorizontal, 0.5)
	sp.SetLabel(label)
	return sp
}

// SetLabel sets the text drawn on a horizontal splitter's bar; "" draws none.
// A vertical bar has no room for one and ignores it.
func (sp *Splitter) SetLabel(label string) { sp.label = label }

// SetLimits bounds the ratio a drag, a key or SetRatio can reach, and clamps
// the current ratio into the new range. Limits outside 0–1, NaN, or min > max
// are ignored.
func (sp *Splitter) SetLimits(minRatio, maxRatio float64) {
	if !(minRatio >= 0 && maxRatio <= 1 && minRatio <= maxRatio) {
		return
	}
	sp.minRatio, sp.maxRatio = minRatio, maxRatio
	sp.ratio = core.Clamp(sp.ratio, minRatio, maxRatio)
}

// SetBounds positions the splitter in the available rect.
func (sp *Splitter) SetBounds(x, y, w, h int) {
	sp.rect = core.Rect{X: x, Y: y, W: w, H: h}
}

// Bounds returns the rect SetBounds last gave the splitter: both panes and the
// bar between them.
func (sp *Splitter) Bounds() core.Rect { return sp.rect }

// Focus marks the bar (and label) as belonging to the focused region,
// drawing it in the focused-border style.
func (sp *Splitter) Focus(v bool) { sp.active = v }

func (sp *Splitter) Focused() bool { return sp.active }

// SetCollapsed hides one pane, giving the other the whole rect, or restores
// both with CollapseNone. While collapsed there is no bar: Draw draws nothing,
// HandleKey and HandleMouse consume nothing, the hidden pane's rect is empty
// (zero width or height, at the edge it sat on), so a host hit-testing
// through FirstRect/SecondRect needs no flag of its own. The ratio is kept,
// so a restore puts the bar back where it was. A drag in progress ends.
func (sp *Splitter) SetCollapsed(c SplitterCollapse) {
	if c < CollapseNone || c > CollapseSecond {
		return
	}
	sp.collapse = c
	if c != CollapseNone {
		sp.dragging = false
	}
}

// Collapsed returns the pane SetCollapsed hid, or CollapseNone.
func (sp *Splitter) Collapsed() SplitterCollapse { return sp.collapse }

// Ratio returns the current split ratio.
func (sp *Splitter) Ratio() float64 { return sp.ratio }

// SetRatio explicitly sets the split ratio. NaN and ±Inf (a corrupt saved
// layout) are ignored: core.Clamp passes NaN through, since both its
// comparisons are false, and a NaN ratio makes every pane position garbage.
func (sp *Splitter) SetRatio(r float64) {
	if math.IsNaN(r) || math.IsInf(r, 0) {
		return
	}
	sp.ratio = core.Clamp(r, sp.minRatio, sp.maxRatio)
}

// axis returns the start and length of the rect along the split direction.
func (sp *Splitter) axis() (origin, size int) {
	if sp.dir == SplitterHorizontal {
		return sp.rect.Y, sp.rect.H
	}
	return sp.rect.X, sp.rect.W
}

// SplitPos returns the absolute screen coordinate of the divider line —
// while collapsed, where the bar will be once restored. It
// stays inside the rect whatever the ratio: a ratio of 1 (SetLimits allows
// it) would otherwise put the bar one past the end and give SecondRect a
// negative size.
func (sp *Splitter) SplitPos() int {
	origin, size := sp.axis()
	if size <= 0 {
		return origin
	}
	return core.Clamp(origin+int(float64(size)*sp.ratio), origin, origin+size-1)
}

// setPos moves the bar to the cell target, within the limits. The ratio is
// stored at the cell's centre so SplitPos's floor returns target exactly: a
// ratio built as ratioBase+delta/size lands just below the integer often
// enough (a tenth of the columns from 0.3 on 80 wide) that the bar trailed
// the pointer by one cell. A zero size has nothing to measure against, so
// the ratio holds (0/0 is NaN, which core.Clamp passes through).
func (sp *Splitter) setPos(target int) {
	origin, size := sp.axis()
	if size <= 0 {
		return
	}
	r := (float64(target-origin) + 0.5) / float64(size)
	sp.ratio = core.Clamp(r, sp.minRatio, sp.maxRatio)
}

// FirstRect returns the rect for the first (top or left) pane: the whole rect
// when the second is collapsed, an empty one at the start when it is.
func (sp *Splitter) FirstRect() core.Rect {
	switch sp.collapse {
	case CollapseFirst:
		if sp.dir == SplitterHorizontal {
			return core.Rect{X: sp.rect.X, Y: sp.rect.Y, W: sp.rect.W}
		}
		return core.Rect{X: sp.rect.X, Y: sp.rect.Y, H: sp.rect.H}
	case CollapseSecond:
		return sp.rect
	}
	pos := sp.SplitPos()
	if sp.dir == SplitterHorizontal {
		return core.Rect{X: sp.rect.X, Y: sp.rect.Y, W: sp.rect.W, H: pos - sp.rect.Y}
	}
	return core.Rect{X: sp.rect.X, Y: sp.rect.Y, W: pos - sp.rect.X, H: sp.rect.H}
}

// SecondRect returns the rect for the second (bottom or right) pane: the
// whole rect when the first is collapsed, an empty one at the end when it is.
func (sp *Splitter) SecondRect() core.Rect {
	switch sp.collapse {
	case CollapseFirst:
		return sp.rect
	case CollapseSecond:
		if sp.dir == SplitterHorizontal {
			return core.Rect{X: sp.rect.X, Y: sp.rect.Bottom(), W: sp.rect.W}
		}
		return core.Rect{X: sp.rect.Right(), Y: sp.rect.Y, H: sp.rect.H}
	}
	pos := sp.SplitPos()
	if sp.dir == SplitterHorizontal {
		afterBar := pos + 1
		return core.Rect{X: sp.rect.X, Y: afterBar, W: sp.rect.W, H: sp.rect.Y + sp.rect.H - afterBar}
	}
	afterBar := pos + 1
	return core.Rect{X: afterBar, Y: sp.rect.Y, W: sp.rect.X + sp.rect.W - afterBar, H: sp.rect.H}
}

// Draw renders the divider line, or nothing while collapsed.
func (sp *Splitter) Draw(s tcell.Screen) {
	if sp.collapse != CollapseNone {
		return
	}
	p := theme.Active()
	barColor := p.Splitter
	if sp.dragging {
		barColor = p.SplitterHover
	}
	style := theme.On(barColor, p.TextDim)
	if sp.active {
		style = theme.StyleActiveBlock().Bold(true)
	}
	pos := sp.SplitPos()

	if sp.dir == SplitterHorizontal {
		core.FillRect(s, core.Rect{X: sp.rect.X, Y: pos, W: sp.rect.W, H: 1}, '─', style)
		if sp.label != "" {
			core.DrawTextClipped(s, sp.rect.X+2, pos, sp.rect.W-4, style, sp.label)
		}
	} else {
		core.FillRect(s, core.Rect{X: pos, Y: sp.rect.Y, W: 1, H: sp.rect.H}, '│', style)
	}
}

// HandleKey moves the bar with Ctrl+Arrow, a twentieth of the rect per press
// but never less than one cell: a fixed 0.05 ratio step is half a cell on a
// 10-wide split, so some presses didn't move it. Shift must be unheld so it
// doesn't swallow Ctrl+Shift combos (e.g. the editor's "move line"). A
// collapsed splitter has no bar to move and consumes nothing.
func (sp *Splitter) HandleKey(ev *tcell.EventKey) bool {
	if sp.collapse != CollapseNone {
		return false
	}
	isResizeMod := ev.Modifiers()&tcell.ModCtrl != 0 && ev.Modifiers()&tcell.ModShift == 0
	if !isResizeMod {
		return false
	}
	back, fwd := tcell.KeyLeft, tcell.KeyRight
	if sp.dir == SplitterHorizontal {
		back, fwd = tcell.KeyUp, tcell.KeyDown
	}
	_, size := sp.axis()
	step := max(1, int(math.Round(0.05*float64(size))))
	switch ev.Key() {
	case back:
		sp.setPos(sp.SplitPos() - step)
	case fwd:
		sp.setPos(sp.SplitPos() + step)
	default:
		return false
	}
	return true
}

// HandleMouse handles drag events on the bar and reports whether it consumed
// them. A resize begins only on a press on the bar (see press), so the
// host must pass every Button1 event through here. A collapsed splitter
// consumes nothing, but still tracks the press latch, so a gesture begun
// while collapsed cannot grab the bar after a restore.
func (sp *Splitter) HandleMouse(ev *tcell.EventMouse) bool {
	mx, my := ev.Position()
	pos := sp.SplitPos()

	sp.press.Release(ev)
	fresh := sp.press.Fresh(ev)
	if sp.collapse != CollapseNone {
		return false
	}

	onBar := false
	if sp.dir == SplitterHorizontal {
		onBar = my == pos && mx >= sp.rect.X && mx < sp.rect.X+sp.rect.W
	} else {
		onBar = mx == pos && my >= sp.rect.Y && my < sp.rect.Y+sp.rect.H
	}

	if !sp.dragging && fresh && onBar {
		sp.dragging = true
		if sp.dir == SplitterHorizontal {
			sp.dragBase = my
		} else {
			sp.dragBase = mx
		}
		sp.posBase = pos
		return true
	}
	if sp.dragging {
		if ev.Buttons() == tcell.ButtonNone {
			sp.dragging = false
			return true
		}
		// Tracked in cells, not ratio, so the bar sits under the pointer
		// (see setPos), which also holds the ratio through a resize to zero.
		at := mx
		if sp.dir == SplitterHorizontal {
			at = my
		}
		sp.setPos(sp.posBase + at - sp.dragBase)
		return true
	}
	return false
}
