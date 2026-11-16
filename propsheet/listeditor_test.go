package propsheet

import (
	"slices"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

type leItem struct{ name, value string }

// leFixture is a ListEditor over items whose one detail row edits value.
type leFixture struct {
	items   []*leItem
	loaded  []*leItem
	detail  *TextRow
	shown   []*leItem // every item Show received, nil for none
	changed int
	refuse  string // Add and Remove refuse with this hint when set
	l       *ListEditor[*leItem]
}

func newLEFixture(initial string, names ...string) *leFixture {
	fx := &leFixture{detail: Text("Value", "", 10)}
	for _, n := range names {
		fx.items = append(fx.items, &leItem{name: n, value: n + "-v"})
	}
	fx.loaded = slices.Clone(fx.items)
	fx.l = NewListEditor(ListEditorSpec[*leItem]{
		Noun:       "account",
		NameWidth:  10,
		Headers:    []string{"Name", "Value"},
		GridHeight: 8,
		Items:      func() []*leItem { return fx.items },
		Cells:      func(e *leItem) []string { return []string{e.name, e.value} },
		Initial:    func(e *leItem) bool { return e.name == initial },
		Show: func(e *leItem) {
			fx.shown = append(fx.shown, e)
			if e == nil {
				fx.detail.SetValue("")
				return
			}
			fx.detail.SetValue(e.value)
		},
		Commit: func(e *leItem) { e.value = fx.detail.Value() },
		Add: func(name string) (string, bool) {
			if fx.refuse != "" {
				return fx.refuse, false
			}
			fx.items = append(fx.items, &leItem{name: name})
			return "added " + name, true
		},
		Remove: func(e *leItem) (string, bool) {
			if fx.refuse != "" {
				return fx.refuse, false
			}
			fx.items = slices.DeleteFunc(fx.items, func(x *leItem) bool { return x == e })
			return "", true
		},
		Revert:  func() { fx.items = slices.Clone(fx.loaded) },
		Changed: func() { fx.changed++ },
	})
	return fx
}

func (fx *leFixture) current() string {
	if c := fx.l.Current(); c != nil {
		return c.name
	}
	return ""
}

// Moving the grid cursor is the whole of the master-detail contract: the detail
// edit goes into the item left, the item reached is shown, and the cursor stays
// where the key put it (a SetData redraw put it back on row 0, so only the first
// row was reachable).
func TestListEditorMovingTheCursorCommitsAndShows(t *testing.T) {
	fx := newLEFixture("", "a", "b", "c")
	if fx.current() != "a" || fx.detail.Value() != "a-v" {
		t.Fatalf("first shown %q with %q, want a with a-v", fx.current(), fx.detail.Value())
	}
	fx.detail.SetValue("edited")
	fx.l.Grid().SetBounds(0, 0, 40, 8)
	fx.l.Grid().HandleKey(tuikittest.Key(tcell.KeyDown, tcell.ModNone))
	if fx.items[0].value != "edited" {
		t.Errorf("item left has %q, want the detail edit", fx.items[0].value)
	}
	if fx.current() != "b" || fx.detail.Value() != "b-v" || fx.l.Grid().SelectedRow() != 1 {
		t.Errorf("after Down: current %q, detail %q, row %d; want b, b-v, 1",
			fx.current(), fx.detail.Value(), fx.l.Grid().SelectedRow())
	}
	if fx.changed != 1 {
		t.Errorf("Changed ran %d times, want 1", fx.changed)
	}
}

func TestListEditorInitialSelection(t *testing.T) {
	fx := newLEFixture("c", "a", "b", "c")
	if fx.current() != "c" || fx.l.Grid().SelectedRow() != 2 {
		t.Errorf("Initial c selected %q at row %d", fx.current(), fx.l.Grid().SelectedRow())
	}
}

func TestListEditorAdd(t *testing.T) {
	fx := newLEFixture("", "a")
	fx.l.add()
	if len(fx.items) != 1 || fx.l.Hint().Text() != "Type a name for the new account first." {
		t.Errorf("Add with no name: %d items, hint %q", len(fx.items), fx.l.Hint().Text())
	}

	fx.l.name.SetValue("b")
	fx.refuse = "no"
	fx.l.add()
	if len(fx.items) != 1 || fx.l.Hint().Text() != "no" || fx.l.name.Value() != "b" {
		t.Errorf("refused Add: %d items, hint %q, name %q; want 1, no, b kept",
			len(fx.items), fx.l.Hint().Text(), fx.l.name.Value())
	}

	fx.refuse = ""
	fx.detail.SetValue("a-edited")
	fx.l.add()
	switch {
	case fx.items[0].value != "a-edited":
		t.Error("Add did not commit the selected item's detail first")
	case fx.current() != "b" || fx.l.Grid().SelectedRow() != 1:
		t.Errorf("after Add: current %q at row %d, want the new item b at 1", fx.current(), fx.l.Grid().SelectedRow())
	case fx.l.name.Value() != "" || fx.l.Hint().Text() != "added b":
		t.Errorf("after Add: name %q, hint %q", fx.l.name.Value(), fx.l.Hint().Text())
	case fx.changed != 1:
		t.Errorf("Changed ran %d times, want 1", fx.changed)
	}
}

func TestListEditorRemove(t *testing.T) {
	fx := newLEFixture("c", "a", "b", "c")
	fx.refuse = "built in"
	fx.l.remove()
	if len(fx.items) != 3 || fx.l.Hint().Text() != "built in" {
		t.Errorf("refused Remove: %d items, hint %q", len(fx.items), fx.l.Hint().Text())
	}
	fx.refuse = ""
	fx.l.remove() // the last row: the selection moves up to the new last
	if len(fx.items) != 2 || fx.current() != "b" || fx.l.Hint().Text() != "" {
		t.Errorf("after Remove: %d items, current %q, hint %q; want 2, b, cleared",
			len(fx.items), fx.current(), fx.l.Hint().Text())
	}
	fx.l.remove()
	fx.l.remove()
	if len(fx.items) != 0 || fx.current() != "" || fx.shown[len(fx.shown)-1] != nil {
		t.Errorf("emptied: %d items, current %q; want none shown", len(fx.items), fx.current())
	}
	fx.l.remove()
	if want := "Select an account in the grid above to remove it."; fx.l.Hint().Text() != want {
		t.Errorf("Remove with nothing selected: hint %q, want %q", fx.l.Hint().Text(), want)
	}
}

// Revert must forget the selection before the caller's items go back, or the
// detail rows' edit is committed into a reverted item and survives the Revert.
func TestListEditorRevertDoesNotCommitTheDetail(t *testing.T) {
	fx := newLEFixture("", "a")
	fx.l.name.SetValue("b")
	fx.l.add()
	fx.detail.SetValue("edited") // b's detail, b is reverted away
	fx.l.GridRow().RevertFn()
	switch {
	case len(fx.items) != 1 || fx.items[0].value != "a-v":
		t.Errorf("after Revert: %d items, a = %q; want 1, a-v", len(fx.items), fx.items[0].value)
	case fx.current() != "a" || fx.detail.Value() != "a-v":
		t.Errorf("after Revert: current %q, detail %q; want a, a-v", fx.current(), fx.detail.Value())
	case fx.changed != 2:
		t.Errorf("Changed ran %d times over Add and Revert, want 2", fx.changed)
	}
	fx.detail.SetValue("again")
	fx.l.GridRow().RevertFn()
	if fx.items[0].value != "a-v" {
		t.Errorf("Revert committed the detail edit: a = %q", fx.items[0].value)
	}
}

func TestListEditorFormCommitsTheSelectedItem(t *testing.T) {
	fx := newLEFixture("", "a")
	f := fx.l.NewForm(fx.l.GridRow(), fx.detail)
	fx.detail.SetValue("x")
	f.Commit()
	if fx.items[0].value != "x" {
		t.Errorf("Form.Commit left a = %q", fx.items[0].value)
	}
}

func TestListEditorArticle(t *testing.T) {
	for noun, want := range map[string]string{"account": "an", "profile": "a", "external resource pool": "an", "": "a"} {
		l := &ListEditor[*leItem]{spec: ListEditorSpec[*leItem]{Noun: noun}}
		if got := l.article(); got != want {
			t.Errorf("article(%q) = %q, want %q", noun, got, want)
		}
	}
	l := &ListEditor[*leItem]{spec: ListEditorSpec[*leItem]{Noun: "user", Article: "a"}}
	if l.article() != "a" {
		t.Error("Article did not override the vowel rule")
	}
}
