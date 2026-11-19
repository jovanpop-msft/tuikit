package controls

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

func TestCheckItemsMarksEachValueAndToggles(t *testing.T) {
	on := map[string]bool{"b": true}
	items := CheckItems([]string{"a", "b"}, func(s string) bool { return on[s] },
		func(s string) string { return "x" + s },
		func(s string) { on[s] = !on[s] })
	want := []string{CheckOff + "xa", CheckOn + "xb"}
	for i, it := range items {
		if it.Label != want[i] || !it.KeepOpen || !it.enabled() {
			t.Errorf("item %d = %q keepOpen=%v, want %q, KeepOpen, enabled", i, it.Label, it.KeepOpen, want[i])
		}
	}
	items[0].Action()
	if !on["a"] {
		t.Error("toggling a did not turn it on")
	}
}

// newCheckMenu shows a two-row checklist with a trailing Clear row, rebuilt
// from the state on every refresh.
func newCheckMenu(on map[string]bool, toggles *int) *ContextMenu {
	cm := &ContextMenu{}
	build := func() []MenuItem {
		items := CheckItems([]string{"a", "b"}, func(s string) bool { return on[s] },
			func(s string) string { return s },
			func(s string) { on[s] = !on[s]; *toggles++ })
		return append(items, MenuItem{Divider: true}, MenuItem{Label: "Clear", Action: func() { clear(on) }})
	}
	cm.Show(0, 0, build())
	cm.SetRefresh(build)
	return cm
}

func TestKeepOpenItemTicksByKeyWithHoverKept(t *testing.T) {
	on, n := map[string]bool{}, 0
	cm := newCheckMenu(on, &n)
	cm.HandleKey(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone))
	cm.HandleKey(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone))
	cm.HandleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if !cm.OverlayActive() || !on["b"] || n != 1 {
		t.Fatalf("after Enter on b: open=%v on=%v toggles=%d, want open, b on, 1", cm.OverlayActive(), on, n)
	}
	if got := cm.Items()[1].Label; got != CheckOn+"b" {
		t.Errorf("row b after the tick = %q, want it refreshed to %q", got, CheckOn+"b")
	}
	if h := cm.menu.root().hover; h != 1 {
		t.Errorf("hover = %d after the tick, want 1 (kept)", h)
	}
	// A plain item still closes the menu.
	cm.HandleKey(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone))
	cm.HandleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if cm.OverlayActive() || on["b"] {
		t.Errorf("Clear: open=%v on=%v, want closed and cleared", cm.OverlayActive(), on)
	}
}

// A press ticks once: the motion resends of the held button and the release
// that ends the click must not tick the row again (ARCHITECTURE.md § Mouse
// gestures), and the next press ticks it back.
func TestKeepOpenItemTicksOncePerClick(t *testing.T) {
	on, n := map[string]bool{}, 0
	cm := newCheckMenu(on, &n)
	cm.DrawOverlay(tuikittest.NewScreen(40, 20))
	click := func() {
		cm.HandleMouse(tcell.NewEventMouse(3, 1, tcell.Button1, tcell.ModNone))
		cm.HandleMouse(tcell.NewEventMouse(4, 1, tcell.Button1, tcell.ModNone))
		cm.HandleMouse(tcell.NewEventMouse(4, 1, tcell.ButtonNone, tcell.ModNone))
	}
	click()
	if n != 1 || !on["a"] || !cm.OverlayActive() {
		t.Fatalf("one click: toggles=%d on=%v open=%v, want 1, a on, open", n, on, cm.OverlayActive())
	}
	s := tuikittest.NewScreen(40, 20)
	cm.DrawOverlay(s)
	if row := s.RowOf(CheckOn + "a"); row != 1 {
		t.Errorf("ticked a drawn on row %d, want 1\n%s", row, s.Snapshot())
	}
	click()
	if n != 2 || on["a"] {
		t.Errorf("second click: toggles=%d on=%v, want 2, a off", n, on)
	}
}

// An action that shows another menu wins over the refresh of the one it
// replaced; one that hides the menu leaves it hidden.
func TestKeepOpenActionThatReshowsOrHidesIsNotOverwritten(t *testing.T) {
	cm := &ContextMenu{}
	other := []MenuItem{{Label: "other"}}
	cm.Show(0, 0, []MenuItem{
		{Label: "reshow", KeepOpen: true, Action: func() { cm.Show(0, 0, other) }},
		{Label: "hide", KeepOpen: true, Action: cm.Hide},
	})
	cm.SetRefresh(func() []MenuItem { return []MenuItem{{Label: "stale"}} })
	cm.HandleKey(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone))
	cm.HandleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if got := cm.Items(); len(got) != 1 || got[0].Label != "other" {
		t.Fatalf("items = %v, want the re-shown menu", got)
	}

	cm.Show(0, 0, []MenuItem{{Label: "hide", KeepOpen: true, Action: cm.Hide}})
	cm.SetRefresh(func() []MenuItem { return []MenuItem{{Label: "stale"}} })
	cm.HandleKey(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone))
	cm.HandleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if cm.OverlayActive() {
		t.Error("menu open after a KeepOpen action hid it")
	}
}

// A refresh with fewer rows than the hover drops the highlight rather than
// leaving it past the end.
func TestKeepOpenRefreshShorterClampsHover(t *testing.T) {
	cm := &ContextMenu{}
	cm.Show(0, 0, []MenuItem{{Label: "a"}, {Label: "b", KeepOpen: true}})
	cm.SetRefresh(func() []MenuItem { return []MenuItem{{Label: "a"}} })
	cm.HandleKey(tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone))
	cm.HandleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if h := cm.menu.root().hover; h != -1 {
		t.Errorf("hover = %d, want -1", h)
	}
	cm.DrawOverlay(tuikittest.NewScreen(40, 20)) // must not index past the items
}

func TestMenuBarKeepOpenItemRunsWithTheDropdownOpen(t *testing.T) {
	n := 0
	mb := NewMenuBar()
	mb.SetBounds(0, 0, 80)
	mb.SetMenus([]Menu{{Label: "View", Items: []MenuItem{
		{Label: "Wrap", KeepOpen: true, Action: func() { n++ }},
	}}})
	mb.Open()
	mb.HandleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
	if n != 1 || !mb.OverlayActive() {
		t.Fatalf("Enter: runs=%d open=%v, want 1 and open", n, mb.OverlayActive())
	}
	mb.DrawOverlay(tuikittest.NewScreen(80, 10))
	mb.HandleMouse(tcell.NewEventMouse(3, itemRowY(0), tcell.Button1, tcell.ModNone))
	mb.HandleMouse(tcell.NewEventMouse(4, itemRowY(0), tcell.Button1, tcell.ModNone))
	mb.HandleMouse(tcell.NewEventMouse(4, itemRowY(0), tcell.ButtonNone, tcell.ModNone))
	if n != 2 || !mb.OverlayActive() {
		t.Errorf("click: runs=%d open=%v, want 2 and open", n, mb.OverlayActive())
	}
}
