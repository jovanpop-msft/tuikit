package controls

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/theme"
	"github.com/radix29/tuikit/tuikittest"
)

// drawContextMenu shows items at the top-left of a fresh screen and returns
// what was drawn.
func drawContextMenu(items []MenuItem) *tuikittest.Screen {
	s := tuikittest.NewScreen(60, 10)
	cm := &ContextMenu{}
	cm.Show(0, 0, items)
	cm.DrawOverlay(s)
	return s
}

// boxWidth returns the width of the menu box drawn at the top-left: the
// columns from the ┌ to its ┐.
func boxWidth(t *testing.T, s *tuikittest.Screen) int {
	t.Helper()
	top := strings.Split(s.Snapshot(), "\n")[0]
	if !strings.HasPrefix(top, "┌") || !strings.HasSuffix(top, "┐") {
		t.Fatalf("top row %q is not a box border", top)
	}
	return len([]rune(top))
}

// TestMenuItemCheckedDrawsTickAtConstantWidth: a toggle draws MenuTick before
// its label while on and a blank while off, the label does not move, and the
// box keeps its width — so a host need not rebuild its menus on a flip.
func TestMenuItemCheckedDrawsTickAtConstantWidth(t *testing.T) {
	on := false
	items := []MenuItem{
		{Label: "Word Wrap", Shortcut: "Alt+Z", Checked: func() bool { return on }},
		{Label: "Find...", Shortcut: "Ctrl+F"},
	}

	off := drawContextMenu(items)
	on = true
	ticked := drawContextMenu(items)

	if strings.Contains(off.Snapshot(), MenuTick) {
		t.Errorf("off: tick drawn:\n%s", off.Snapshot())
	}
	if x, y, ok := ticked.Find(MenuTick + " Word Wrap"); !ok || y != 1 || x != 2 {
		t.Errorf("on: %q at (%d,%d) found=%v, want (2,1):\n%s", MenuTick+" Word Wrap", x, y, ok, ticked.Snapshot())
	}
	for _, s := range []*tuikittest.Screen{off, ticked} {
		if x, _, _ := s.Find("Word Wrap"); x != 4 {
			t.Errorf("Word Wrap label at column %d, want 4 (after the mark column):\n%s", x, s.Snapshot())
		}
		// A non-toggle row in the same box is indented by the column too.
		if x, _, _ := s.Find("Find..."); x != 4 {
			t.Errorf("Find... label at column %d, want 4 (aligned with the toggle):\n%s", x, s.Snapshot())
		}
	}
	if wOff, wOn := boxWidth(t, off), boxWidth(t, ticked); wOff != wOn {
		t.Errorf("box width off=%d on=%d, want equal", wOff, wOn)
	}
}

// TestMenuWithoutToggleHasNoMarkColumn: a box with no Checked item draws its
// labels where it always did.
func TestMenuWithoutToggleHasNoMarkColumn(t *testing.T) {
	s := drawContextMenu([]MenuItem{{Label: "Open"}, {Divider: true}, {Label: "Close"}})
	if x, _, _ := s.Find("Open"); x != 2 {
		t.Errorf("Open at column %d, want 2:\n%s", x, s.Snapshot())
	}
}

// TestMenuMarkColumnWidensBox: the mark column counts in the box width, so a
// toggle's longest label is not clipped against the shortcut.
func TestMenuMarkColumnWidensBox(t *testing.T) {
	label := strings.Repeat("x", 30)
	plain := drawContextMenu([]MenuItem{{Label: label, Shortcut: "Alt+Z"}})
	toggle := drawContextMenu([]MenuItem{{Label: label, Shortcut: "Alt+Z", Checked: func() bool { return true }}})
	if got, want := boxWidth(t, toggle), boxWidth(t, plain)+menuMarkWidth; got != want {
		t.Errorf("toggle box width %d, want %d", got, want)
	}
	if _, _, ok := toggle.Find(label + "  Alt+Z"); !ok {
		t.Errorf("label and shortcut not separated by the 2-column gap:\n%s", toggle.Snapshot())
	}
}

// TestToolbarOnPaintsSelectedAtConstantLayout: an On button paints in the
// selected style while on, in the bar style while off, and flipping it changes
// no button's position.
func TestToolbarOnPaintsSelectedAtConstantLayout(t *testing.T) {
	on := false
	tb := NewToolbar()
	tb.SetButtons([]ToolbarButton{
		{Icon: "Act.Plan", On: func() bool { return on }},
		{Icon: "Meta"},
	})
	tb.SetBounds(0, 0, 40)
	starts := append([]int(nil), tb.starts...)

	styleAt := func() (toggle, plain tcell.Style) {
		s := tuikittest.NewScreen(40, 2)
		tb.Draw(s)
		x, _, _ := s.Find("Act.Plan")
		mx, _, _ := s.Find("Meta")
		return s.StyleAt(x, 0), s.StyleAt(mx, 0)
	}

	offTog, offPlain := styleAt()
	if offTog != theme.StyleMenuBar() {
		t.Errorf("off toggle style %v, want the bar style", offTog)
	}
	on = true
	onTog, onPlain := styleAt()
	if onTog != theme.StyleMenuSelected().Bold(true) {
		t.Errorf("on toggle style %v, want bold StyleMenuSelected", onTog)
	}
	if offPlain != onPlain {
		t.Errorf("plain button's style changed with the toggle: %v -> %v", offPlain, onPlain)
	}
	tb.SetBounds(0, 0, 40)
	for i := range starts {
		if tb.starts[i] != starts[i] {
			t.Errorf("button %d moved from %d to %d on toggle", i, starts[i], tb.starts[i])
		}
	}
}

// TestToolbarDisabledBeatsOn: a disabled toggle draws disabled even while on.
func TestToolbarDisabledBeatsOn(t *testing.T) {
	tb := NewToolbar()
	tb.SetButtons([]ToolbarButton{{Icon: "Live", On: func() bool { return true }, Enabled: func() bool { return false }}})
	tb.SetBounds(0, 0, 10)
	s := tuikittest.NewScreen(10, 2)
	tb.Draw(s)
	x, _, _ := s.Find("Live")
	if st := s.StyleAt(x, 0); st != theme.StyleDisabled() {
		t.Errorf("disabled on toggle style %v, want StyleDisabled", st)
	}
}
