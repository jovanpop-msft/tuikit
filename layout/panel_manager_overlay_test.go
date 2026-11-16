package layout

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

// overlayPanel is a fakePanel with an overlay that any press dismisses and
// falls through (core.MouseHandler), as a DataGrid's cell menu does.
type overlayPanel struct {
	fakePanel
	open bool
}

func (p *overlayPanel) OverlayActive() bool { return p.open }
func (p *overlayPanel) HandleMouse(ev *tcell.EventMouse) bool {
	p.mouse = append(p.mouse, ev.Buttons())
	if p.open && ev.Buttons()&(tcell.Button1|tcell.Button2) != 0 {
		p.open = false
	}
	return false
}

// The active panel's overlay must see a press that lands outside the
// PanelManager — over Object Explorer, beside it — or it stays open next to
// whatever that press opened there. Routing by position never delivered it.
func TestAPressOutsideReachesTheActivePanelsOverlay(t *testing.T) {
	pm := NewPanelManager()
	p := &overlayPanel{fakePanel: fakePanel{title: "Details"}, open: true}
	pm.AddPanel(p)
	pm.SetBounds(30, 1, 50, 20)

	if !pm.OverlayActive() {
		t.Fatal("OverlayActive = false with the panel's overlay open")
	}
	pm.HandleMouse(tcell.NewEventMouse(5, 10, tcell.Button2, tcell.ModNone))
	if len(p.mouse) != 1 {
		t.Fatalf("panel got %d events for a right-click left of the PanelManager, want 1", len(p.mouse))
	}
	if p.open || pm.OverlayActive() {
		t.Error("the overlay is still open after a press outside")
	}

	// Closed, the panel is back to positional routing: nothing outside.
	pm.HandleMouse(tcell.NewEventMouse(5, 10, tcell.Button2, tcell.ModNone))
	if len(p.mouse) != 1 {
		t.Errorf("panel got a press outside the PanelManager with no overlay open (%d events)", len(p.mouse))
	}
}

// The tab drop-down is an overlay too: a press outside the PanelManager's
// columns closes it, as one outside its list inside them always did.
func TestAPressOutsideClosesTheTabDropDown(t *testing.T) {
	pm, _ := openCombo(t, 3, 50, 12)
	pm.SetBounds(30, 1, 50, 12)
	if !pm.OverlayActive() {
		t.Fatal("OverlayActive = false with the drop-down open")
	}
	pm.HandleMouse(tcell.NewEventMouse(0, 5, tcell.ButtonNone, tcell.ModNone))
	if pm.HandleMouse(tcell.NewEventMouse(5, 5, tcell.Button1, tcell.ModNone)) {
		t.Error("a press outside was claimed; it must fall through to what it hit")
	}
	if pm.comboOpen || pm.OverlayActive() {
		t.Error("the drop-down is still open after a press outside")
	}
}
