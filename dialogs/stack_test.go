package dialogs

import (
	"slices"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/tuikittest"
	"github.com/radix29/tuikit/widgets"
)

// stackDialog is a minimal Dialog over ModalDialog that counts the events it
// is handed.
type stackDialog struct {
	ModalDialog
	keys, mice, releases int
}

func newStackDialog(s tcell.Screen, w, h int) *stackDialog {
	d := &stackDialog{}
	d.InitModal("t", w, h)
	d.SetScreen(s)
	return d
}

func (d *stackDialog) Draw(s tcell.Screen) {
	if d.Visible() {
		d.DrawBase(s)
	}
}
func (d *stackDialog) HandleKey(*tcell.EventKey) bool { d.keys++; return true }
func (d *stackDialog) HandleMouse(ev *tcell.EventMouse) bool {
	d.mice++
	if ev.Buttons() == tcell.ButtonNone {
		d.releases++
	}
	return true
}

// clipHostDialog is a stackDialog that is a core.ClipboardHost and
// core.ClipboardEditHandler.
type clipHostDialog struct {
	*stackDialog
	field  *widgets.InputField
	edited core.ClipboardTarget
}

func (d *clipHostDialog) FocusedClipboardTarget() core.ClipboardTarget { return d.field }
func (d *clipHostDialog) ClipboardEdited(t core.ClipboardTarget)       { d.edited = t }

func TestStackSyncKeepsOpenDialogsInZOrder(t *testing.T) {
	scr := tuikittest.NewScreen(80, 24)
	a, b := newStackDialog(scr, 20, 6), newStackDialog(scr, 20, 6)
	var st Stack
	st.Register(a)
	st.Register(b)

	st.Sync()
	if st.Top() != nil {
		t.Fatalf("Top with nothing shown = %v, want nil", st.Top())
	}
	b.Show()
	st.Sync()
	a.Show() // registered first, opened second: z-order follows opening
	st.Sync()
	if got := st.Open(); !slices.Equal(got, []Dialog{b, a}) {
		t.Fatalf("Open = %v, want [b a]", got)
	}
	b.Show() // re-showing an open dialog keeps its place
	st.Sync()
	if len(st.Open()) != 2 || st.Top() != Dialog(a) {
		t.Errorf("after re-show: Open = %v, want [b a]", st.Open())
	}
	a.Hide()
	st.Sync()
	if st.Top() != Dialog(b) {
		t.Errorf("Top after closing a = %v, want b uncovered", st.Top())
	}
	if !slices.Equal(st.Registered(), []Dialog{a, b}) {
		t.Errorf("Registered = %v, want [a b]", st.Registered())
	}
}

// Presses go to the top dialog only; a release reaches every open one, or a
// dialog that opened a child on a button press keeps its press latch and
// refuses its next click.
func TestStackRoutesPressesToTopAndReleasesToAll(t *testing.T) {
	scr := tuikittest.NewScreen(80, 24)
	bottom, top := newStackDialog(scr, 20, 6), newStackDialog(scr, 20, 6)
	var st Stack
	for _, d := range []*stackDialog{bottom, top} {
		st.Register(d)
		d.Show()
	}
	st.Sync()

	if !st.HandleKey(tuikittest.Rune('x')) || !st.HandleMouse(tuikittest.Press(1, 1)) {
		t.Fatal("an open stack must report every event as consumed")
	}
	if bottom.keys != 0 || bottom.mice != 0 || top.keys != 1 || top.mice != 1 {
		t.Errorf("press routing: bottom keys/mice %d/%d, top %d/%d; want 0/0, 1/1",
			bottom.keys, bottom.mice, top.keys, top.mice)
	}
	st.HandleMouse(tuikittest.Release(1, 1))
	if bottom.releases != 1 || top.releases != 1 {
		t.Errorf("release reached bottom %d, top %d times; want 1 each", bottom.releases, top.releases)
	}

	var empty Stack
	if empty.HandleKey(tuikittest.Rune('x')) || empty.HandleMouse(tuikittest.Press(1, 1)) {
		t.Error("an empty stack must leave events to the host")
	}
}

// An open dialog owns the clipboard even when it has no text: falling past it
// let Ctrl+X in a dialog cut the text of the editor behind it.
func TestStackClipboardBelongsToTheTopDialog(t *testing.T) {
	scr := tuikittest.NewScreen(80, 24)
	var st Stack
	if _, owned := st.ClipboardTarget(); owned {
		t.Error("an empty stack claimed the clipboard")
	}

	host := &clipHostDialog{stackDialog: newStackDialog(scr, 30, 8), field: widgets.NewInputField("", 10, false)}
	plain := newStackDialog(scr, 20, 6)
	st.Register(host)
	st.Register(plain)
	host.Show()
	st.Sync()
	if target, owned := st.ClipboardTarget(); !owned || target != core.ClipboardTarget(host.field) {
		t.Errorf("host on top: ClipboardTarget = %v, %v; want its field, true", target, owned)
	}
	st.ClipboardEdited(host.field)
	if host.edited != core.ClipboardTarget(host.field) {
		t.Error("ClipboardEdited did not reach the top host")
	}

	plain.Show()
	st.Sync()
	if target, owned := st.ClipboardTarget(); !owned || target != nil {
		t.Errorf("non-host on top: ClipboardTarget = %v, %v; want nil, true", target, owned)
	}
}

// A clamped dialog narrows the ClipScreen to its interior. A second dialog
// drawn on the same ClipScreen must start from the whole screen: it once
// inherited that clip, so its dim and border outside the first one's interior
// were dropped. Covered both through DrawBase on a shared ClipScreen (a host
// without Stack) and through Stack.Draw.
func TestADialogDoesNotInheritAnotherDialogsClip(t *testing.T) {
	for _, viaStack := range []bool{false, true} {
		scr := tuikittest.NewScreen(40, 12)
		parent := newStackDialog(scr, 78, 5) // clamped in width: rows 3–7
		child := newStackDialog(scr, 10, 8)  // fits: rows 2–9, outside parent's interior
		parent.Show()
		child.Show()

		if viaStack {
			var st Stack
			st.Register(parent)
			st.Register(child)
			st.Sync()
			st.Draw(scr)
		} else {
			cs := core.NewClipScreen(scr)
			parent.Draw(cs)
			child.Draw(cs)
		}

		r := child.Rect()
		if got := scr.RuneAt(r.X, r.Y); got == ' ' || got == 0 {
			t.Errorf("viaStack=%v: child's top-left corner at (%d,%d) = %q, clipped by the parent", viaStack, r.X, r.Y, got)
		}
		if got := scr.RuneAt(r.X, r.Bottom()-1); got == ' ' || got == 0 {
			t.Errorf("viaStack=%v: child's bottom-left corner = %q, clipped by the parent", viaStack, got)
		}
	}
}

// Dialogs that open in the same tick stack by when they were shown, not by
// registration order: a host that builds dialogs on first use registers them
// in an order that says nothing about how they nest, so a confirmation shown
// over a dialog registered after it would open underneath.
func TestStackSyncStacksSameTickOpeningsInShowOrder(t *testing.T) {
	scr := tuikittest.NewScreen(80, 24)
	a, b, c := newStackDialog(scr, 20, 6), newStackDialog(scr, 20, 6), newStackDialog(scr, 20, 6)
	var st Stack
	st.Register(a)
	st.Register(b)
	st.Register(c)

	c.Show()
	a.Show()
	b.Show()
	st.Sync()
	if got := st.Open(); !slices.Equal(got, []Dialog{c, a, b}) {
		t.Fatalf("Open = %v, want [c a b] (show order)", got)
	}

	// Newcomers sort among themselves only: dialogs already open keep their
	// places below them, however recently they were re-shown.
	b.Hide()
	st.Sync()
	b.Show()
	c.Show() // already open: a re-show must not lift it over b
	st.Sync()
	if got := st.Open(); !slices.Equal(got, []Dialog{c, a, b}) {
		t.Errorf("Open = %v, want [c a b]", got)
	}

	// A dialog registered after it was shown still stacks by show order.
	d, e := newStackDialog(scr, 20, 6), newStackDialog(scr, 20, 6)
	e.Show()
	d.Show()
	st.Register(d)
	st.Register(e)
	st.Sync()
	if got := st.Open(); !slices.Equal(got, []Dialog{c, a, b, e, d}) {
		t.Errorf("Open = %v, want [c a b e d]", got)
	}
}

// withScreen gives d the screen a Stack would (ModalDialog.SetScreen), for a
// test that shows a dialog without registering it.
func withScreen[T interface{ SetScreen(tcell.Screen) }](d T, s tcell.Screen) T {
	d.SetScreen(s)
	return d
}

// A dialog takes its screen from the Stack: registered before SetScreen or
// after, it is centred on that screen without its constructor seeing one.
func TestStackGivesItsScreenToEveryRegisteredDialog(t *testing.T) {
	scr := tuikittest.NewScreen(80, 24)
	early, late := newStackDialog(nil, 20, 6), newStackDialog(nil, 20, 6)
	if early.Screen() != nil {
		t.Fatal("a dialog built without a screen has one")
	}
	var st Stack
	st.Register(early)
	st.SetScreen(scr)
	st.Register(late)
	want := core.Rect{X: 30, Y: 9, W: 20, H: 6}
	for name, d := range map[string]*stackDialog{"registered before SetScreen": early, "registered after": late} {
		if d.Screen() != scr {
			t.Errorf("%s: Screen() = %v, want the Stack's", name, d.Screen())
		}
		if got := d.Rect(); got != want {
			t.Errorf("%s: Rect() = %+v, want %+v (centred on 80x24)", name, got, want)
		}
	}
}

// Re-showing an open dialog refreshes it in place: its ShowOrder is kept, so
// it stays under a dialog opened after it, even across a Sync between the two.
func TestStackReshowOfOpenDialogKeepsPlace(t *testing.T) {
	scr := tuikittest.NewScreen(80, 24)
	a, b := newStackDialog(scr, 20, 6), newStackDialog(scr, 20, 6)
	var st Stack
	st.Register(a)
	st.Register(b)

	a.Show()
	st.Sync()
	b.Show()
	st.Sync()
	order := a.ShowOrder()
	a.Show() // refresh while open, no Sync in between
	if a.ShowOrder() != order {
		t.Errorf("ShowOrder after re-showing an open dialog = %d, want %d unchanged", a.ShowOrder(), order)
	}
	st.Sync()
	if got := st.Open(); !slices.Equal(got, []Dialog{a, b}) {
		t.Errorf("Open = %v, want [a b]: a re-show must not lift a", got)
	}
}

// A dialog hidden and shown again between two Syncs was visible at both, so it
// kept its old slot under a dialog opened in the meantime, which then took the
// keys: a message box whose OK handler hides it, opens a confirmation and
// reshows the message box with new text left the confirmation on top.
func TestStackHideShowBetweenSyncsGoesOnTop(t *testing.T) {
	scr := tuikittest.NewScreen(80, 24)
	a, b, c := newStackDialog(scr, 20, 6), newStackDialog(scr, 20, 6), newStackDialog(scr, 20, 6)
	var st Stack
	for _, d := range []*stackDialog{a, b, c} {
		st.Register(d)
	}
	a.Show()
	st.Sync()
	b.Show()
	st.Sync()

	b.Hide() // one handler: hide b, open c, reshow b
	c.Show()
	b.Show()
	st.Sync()
	if got := st.Open(); !slices.Equal(got, []Dialog{a, c, b}) {
		t.Fatalf("Open = %v, want [a c b]: b was shown last", got)
	}
	st.HandleKey(tuikittest.Rune('x'))
	if b.keys != 1 || c.keys != 0 {
		t.Errorf("keys: b %d, c %d; want the reshown b to take them", b.keys, c.keys)
	}

	// The same with no dialog opened in between: b goes back on top of a.
	b.Hide()
	b.Show()
	st.Sync()
	if got := st.Open(); !slices.Equal(got, []Dialog{a, c, b}) {
		t.Errorf("Open after hide/show alone = %v, want [a c b]", got)
	}

	// A dialog lower in the stack reopened between Syncs is raised over those
	// that stayed open.
	a.Hide()
	a.Show()
	st.Sync()
	if got := st.Open(); !slices.Equal(got, []Dialog{c, b, a}) {
		t.Errorf("Open after reopening a = %v, want [c b a]", got)
	}
}
