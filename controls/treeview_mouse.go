package controls

import (
	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
)

// treeZone is the part of the tree a Button1 press began in, which owns every
// event of the gesture until the release (TreeView.gesture).
type treeZone int

const (
	// treeActed is a press on a node or the title button, acted on by the press:
	// the rest of the gesture is swallowed (or, for a node, is the host's drag
	// toward a drop target, which the host routes itself).
	treeActed treeZone = iota
	// treeVBar and treeHBar drag the scrollbar thumbs.
	treeVBar
	treeHBar
)

// HandleMouse handles mouse events, by zone: a held gesture first, then a
// press on a scrollbar or a title-row button, the wheel anywhere over the tree,
// and last a press on a row.
func (tv *TreeView) HandleMouse(ev *tcell.EventMouse) bool {
	mx, my := ev.Position()
	if ev.Buttons() == tcell.ButtonNone {
		tv.gesture.Release()
	}
	l := tv.titleLayout()
	tv.titleButton.hover = l.hasButton && l.button.Contains(mx, my)
	tv.closeButton.hover = l.hasClose && l.close.Contains(mx, my)
	// Ahead of the bounds test: a thumb drag keeps control once the mouse leaves
	// the tree, so a fast drag still reaches the end, and a press that acted
	// (a row, the title button) swallows its resends wherever they land.
	if tv.gesture.Route(ev, tv.routeDrag) {
		return true
	}
	if !tv.rect.Contains(mx, my) {
		return false
	}
	if ev.Buttons() == tcell.Button1 && (tv.pressScrollbar(ev) || tv.pressTitleButton(ev)) {
		return true
	}
	if tv.handleWheel(ev) {
		return true
	}
	return tv.handleRowMouse(ev)
}

// pressScrollbar starts a thumb drag for a Button1 press on either scrollbar,
// reporting whether the press was on one. It runs before the row hit-testing:
// the vertical bar is drawn over the right border column and the horizontal
// bar over the bottom border row (see Draw), the latter outside the row range
// the hit-test checks.
func (tv *TreeView) pressScrollbar(ev *tcell.EventMouse) bool {
	mx, my := ev.Position()
	inner := tv.rect.Inner(1)
	if core.OnScrollbar(mx, my, tv.rect.Right()-1, inner.Y, inner.H, len(tv.nodes)) {
		tv.dragVScrollbar(my)
		tv.gesture.Arm(ev, treeVBar)
		return true
	}
	if core.OnScrollbarH(mx, my, inner.X, tv.rect.Bottom()-1, inner.W, tv.contentW) {
		tv.dragHScrollbar(mx)
		tv.gesture.Arm(ev, treeHBar)
		return true
	}
	return false
}

// pressTitleButton runs the action of the title-row button a Button1 press is
// on (HandleMouse set its hover from the same titleLayout Draw used), reporting
// whether it was on one. The buttons sit on the top border, outside every row,
// so a press there never selects, expands or arms a drag (hosts arm from
// NodeIDAt, which rejects the border). A press runs the action once; its
// resends are the gesture's, and a hold that began on a row and drifted here is
// that row's.
func (tv *TreeView) pressTitleButton(ev *tcell.EventMouse) bool {
	var action func()
	switch {
	case tv.closeButton.hover:
		action = tv.closeButton.action
	case tv.titleButton.hover:
		action = tv.titleButton.action
	default:
		return false
	}
	if action != nil {
		action()
	}
	tv.gesture.Arm(ev, treeActed)
	return true
}

// handleWheel scrolls for a wheel event, reporting whether ev was one. The
// wheel scrolls from anywhere over the tree, the border and the blank space
// below the last node included, and is consumed even at a limit
// (core.MouseHandler): falling through scrolled whatever lay beneath.
func (tv *TreeView) handleWheel(ev *tcell.EventMouse) bool {
	switch ev.Buttons() {
	case tcell.WheelUp:
		// Shift+wheel is the desktop convention for horizontal scroll, and some
		// terminals report it that way rather than as WheelLeft/WheelRight below, so
		// honour both (as DataGrid, Editor and PlanView do).
		if ev.Modifiers()&tcell.ModShift != 0 {
			tv.scrollLeft()
		} else {
			tv.scroll = max(0, tv.scroll-core.WheelRows)
		}
		return true
	case tcell.WheelDown:
		if ev.Modifiers()&tcell.ModShift != 0 {
			tv.scrollRight()
		} else {
			tv.scroll = max(tv.scroll, min(len(tv.nodes)-tv.rect.Inner(1).H, tv.scroll+core.WheelRows))
		}
		return true
	case tcell.WheelLeft:
		tv.scrollLeft()
		return true
	case tcell.WheelRight:
		tv.scrollRight()
		return true
	}
	return false
}

// handleRowMouse handles a press on a node row: Button1 selects, toggles from
// the expander or activates on a double-click; Button2 selects and opens the
// context menu.
func (tv *TreeView) handleRowMouse(ev *tcell.EventMouse) bool {
	mx, my := ev.Position()
	inner := tv.rect.Inner(1)
	row := my - inner.Y
	if row < 0 || row >= inner.H {
		return false
	}

	idx := tv.scroll + row
	if idx < 0 || idx >= len(tv.nodes) {
		return false
	}
	// A press acts on one specific node, so it needs the stricter content hit-test:
	// the row bound above doesn't constrain mx, and the vertical scrollbar is drawn
	// over the right border column. The scrollbar test above already claims a
	// press there whenever the bar shows, so this only changes the no-bar case, but
	// nodeIndexAt is also what app-level drag arming hit-tests with (see NodeIDAt)
	// and the two must agree on what "landed on a node" means.
	if b := ev.Buttons(); b == tcell.Button1 || b == tcell.Button2 {
		if _, ok := tv.nodeIndexAt(mx, my); !ok {
			return false
		}
	}
	switch ev.Buttons() {
	case tcell.Button1:
		// The resends of this press, and a host's drag toward a drop target (the
		// query editor), are the gesture's: they never re-select or re-toggle.
		tv.gesture.Arm(ev, treeActed)
		node := &tv.nodes[idx]
		// A second press on the same node inside the interval is a double-click:
		// activate rather than re-select. Recorded before the expander test so a
		// double-click on the glyph only toggles (activating from it too would fire on
		// the second half of an ordinary expand).
		doubleClick := tv.clicks.Press(node.ID, ev.When())
		// Only the "[+]"/"[-]" expander glyph toggles expand/collapse; a click
		// elsewhere on the row reselects it, which lets a node be click-dragged into
		// the query editor without flipping its expand state (a drag starts on the
		// label). mx is a screen column while the expander's position is virtual
		// (depth*2..depth*2+4, see Draw), so it must be translated through tv.scrollX
		// or this matches only at scrollX==0.
		vcol := mx - inner.X + tv.scrollX
		onExpander := node.HasKids && vcol >= node.Depth*2 && vcol < node.Depth*2+4
		if tv.sel != idx {
			tv.sel = idx
			tv.fireSelect()
		}
		if onExpander {
			tv.toggleExpand()
			return true
		}
		// A double-click on the node's text (icon and label) is Enter: the default
		// action, else expand/collapse. Blank row space past the label only selects.
		if doubleClick && node.onText(vcol) {
			if !tv.activateSelected() {
				tv.toggleExpand()
			}
		}
		return true
	case tcell.Button2: // tcell v3: Button2 is Secondary (right-click); Button3 is Middle.
		// A right-click selects the row it lands on, and a host following the
		// selection (a details pane, a status line) must hear about it before the
		// menu opens, or it keeps describing the previous node while the highlight
		// shows this one. Only on a change: a held right button resends Button2,
		// and re-firing would reload whatever the host hangs off OnSelect.
		if tv.sel != idx {
			tv.sel = idx
			tv.fireSelect()
		}
		if tv.OnRightClick != nil {
			tv.OnRightClick(tv.nodes[idx].ID, mx, my)
		}
		return true
	}
	return false
}

// routeDrag delivers a held Button1 to the zone its press began in.
func (tv *TreeView) routeDrag(ev *tcell.EventMouse, z treeZone) {
	mx, my := ev.Position()
	switch z {
	case treeVBar:
		tv.dragVScrollbar(my)
	case treeHBar:
		tv.dragHScrollbar(mx)
	}
}

// dragVScrollbar points the view at the track position of screen row my.
func (tv *TreeView) dragVScrollbar(my int) {
	if inner := tv.rect.Inner(1); inner.H > 0 && len(tv.nodes) > inner.H {
		tv.scroll = core.ScrollOffsetForDrag(my-inner.Y, inner.H, len(tv.nodes), inner.H)
	}
}

// dragHScrollbar points the view at the track position of screen column mx.
func (tv *TreeView) dragHScrollbar(mx int) {
	if inner := tv.rect.Inner(1); inner.W > 0 && tv.contentW > inner.W {
		tv.scrollX = core.ScrollOffsetForDrag(mx-inner.X, inner.W, tv.contentW, inner.W)
	}
}

// nodeIndexAt returns the index into tv.nodes of the row drawn at screen
// position (mx, my), and whether that position is over a node row. The test is
// against the content area, not the whole rect, so border columns and rows the
// scrollbars are drawn over never resolve to a node.
func (tv *TreeView) nodeIndexAt(mx, my int) (int, bool) {
	inner := tv.rect.Inner(1)
	if !inner.Contains(mx, my) {
		return 0, false
	}
	idx := tv.scroll + (my - inner.Y)
	if idx < 0 || idx >= len(tv.nodes) {
		return 0, false
	}
	return idx, true
}

// NodeIDAt returns the ID of the node drawn at screen position (mx, my), and
// whether there is one: how a host learns what a press landed on rather than
// inferring it from the selection afterwards. A press on the scrollbar, the
// border, or blank space below the last node reports no node. A host's node
// drag-and-drop arms from this; arming from the selection instead lets a
// scrollbar drag arm as a node drag, and the thumb stops following the mouse.
func (tv *TreeView) NodeIDAt(mx, my int) (int, bool) {
	idx, ok := tv.nodeIndexAt(mx, my)
	if !ok {
		return 0, false
	}
	return tv.nodes[idx].ID, true
}

// scrollLeft/scrollRight nudge the horizontal scroll offset by 4 columns,
// clamped to [0, contentW-inner.W] so the thumb never runs past either end.
func (tv *TreeView) scrollLeft() {
	tv.scrollX = max(0, tv.scrollX-core.WheelCells)
}

func (tv *TreeView) scrollRight() {
	maxScroll := max(0, tv.contentW-tv.rect.Inner(1).W)
	tv.scrollX = min(maxScroll, tv.scrollX+core.WheelCells)
}
