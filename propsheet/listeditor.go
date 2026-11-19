package propsheet

import (
	"strings"

	"github.com/radix29/tuikit/controls"
	"github.com/radix29/tuikit/widgets"
)

// ListEditorSpec describes a list-editor page: a grid of items, the selected
// one's settings in detail rows below it, and a name field with Add and Remove.
// The items and their pending state stay the caller's; the editor owns the grid,
// the selection, the Add/Remove/Revert flow and the hint that reports on it.
//
// E is the item type, normally a pointer; its zero value means "nothing
// selected", so Show receives it when the grid is empty.
type ListEditorSpec[E comparable] struct {
	// Noun names one item in the hints ("profile", "resource pool"), and Article
	// is its article; "" picks "an" before a vowel, else "a".
	Noun, Article string

	// NameLabel and NameWidth are the Add name field's label and width; the label
	// defaults to "New <Noun>".
	NameLabel string
	NameWidth int

	Headers    []string
	GridHeight int

	// Items is what the grid lists, in order, and Cells one item's cells.
	Items func() []E
	Cells func(E) []string

	// Show fills the detail rows from an item, or blanks them for the zero E.
	// Commit folds the detail rows back into an item (never the zero E); it runs
	// before the selection moves and before every Add, Remove and form commit.
	// During NewListEditor's first Show the editor is not yet returned, so Show
	// works from its argument, not from Current.
	Show   func(E)
	Commit func(E)

	// Initial picks the item selected first; nil, or no match, is the first row.
	Initial func(E) bool

	// Add makes an item named name (never ""): hint is what the hint row says
	// afterwards ("" clears it), and ok false refuses, leaving the name typed.
	// The new item must be last in Items.
	Add func(name string) (hint string, ok bool)
	// Remove takes e off the list, or with ok false refuses to; hint as for Add.
	Remove func(e E) (hint string, ok bool)

	// Dirty, Revert and Validate are the grid row's (GridRow.DirtyFn, ...).
	// Revert puts the caller's items back as loaded; the editor then reloads the
	// grid and the detail rows.
	Dirty    func() bool
	Revert   func()
	Validate func() error

	// Changed, if set, runs after the selection moves and after an Add, Remove or
	// Revert: where a page publishes its items to another.
	Changed func()
}

// ListEditor is the running editor of a ListEditorSpec. Build it after the
// detail rows Show and Commit use, place GridRow, the detail rows and
// AddRemoveRows in a form made with NewForm, and call Redraw or Reselect after
// changing the items some other way.
//
// Moving the grid cursor commits the detail rows into the item left, shows the
// item reached, and redraws the grid with the cursor kept: BindGridDetail, which
// documents why the redraw must not be SetData.
type ListEditor[E comparable] struct {
	spec    ListEditorSpec[E]
	grid    *controls.DataGrid
	gridRow *GridRow
	name    *TextRow
	hint    *HintRow
	buttons *ButtonsRow
	current E
}

// NewListEditor builds the editor for spec and shows its first item.
func NewListEditor[E comparable](spec ListEditorSpec[E]) *ListEditor[E] {
	l := &ListEditor[E]{spec: spec, grid: controls.NewDataGrid(), hint: Hint()}
	l.grid.SetData(spec.Headers, l.rows())
	l.grid.SetCellCursor(true)
	if spec.Initial != nil {
		for i, e := range spec.Items() {
			if spec.Initial(e) {
				l.grid.SetSelectedRow(i)
				break
			}
		}
	}
	reload := BindGridDetail(l.grid, spec.Headers, l.rows, l.Commit, l.sync)
	moved := l.grid.OnSelectRow
	l.grid.OnSelectRow = func(row int) {
		moved(row)
		l.changed()
	}

	l.gridRow = NewGridRow(l.grid, spec.GridHeight)
	l.gridRow.DirtyFn = spec.Dirty
	l.gridRow.ValidateFn = spec.Validate
	l.gridRow.RevertFn = func() {
		var zero E
		l.current = zero
		if spec.Revert != nil {
			spec.Revert()
		}
		reload()
		l.changed()
	}

	label := spec.NameLabel
	if label == "" {
		label = "New " + spec.Noun
	}
	l.name = Text(label, "", spec.NameWidth)
	l.buttons = Buttons(widgets.NewButton("Add", l.add), widgets.NewButton("Remove", l.remove))
	return l
}

func (l *ListEditor[E]) rows() [][]string {
	items := l.spec.Items()
	out := make([][]string, len(items))
	for i, e := range items {
		out[i] = l.spec.Cells(e)
	}
	return out
}

// sync makes the grid's selected item current and shows it.
func (l *ListEditor[E]) sync() {
	var zero E
	l.current = zero
	if items, i := l.spec.Items(), l.grid.SelectedRow(); i >= 0 && i < len(items) {
		l.current = items[i]
	}
	l.spec.Show(l.current)
}

func (l *ListEditor[E]) changed() {
	if l.spec.Changed != nil {
		l.spec.Changed()
	}
}

func (l *ListEditor[E]) article() string {
	if l.spec.Article != "" {
		return l.spec.Article
	}
	if l.spec.Noun != "" && strings.ContainsRune("aeiouAEIOU", rune(l.spec.Noun[0])) {
		return "an"
	}
	return "a"
}

func (l *ListEditor[E]) setHint(text string) {
	if text == "" {
		l.hint.Clear()
		return
	}
	l.hint.Set(text)
}

func (l *ListEditor[E]) add() {
	l.Commit()
	name := l.name.Value()
	if name == "" {
		l.hint.Set("Type a name for the new " + l.spec.Noun + " first.")
		return
	}
	hint, ok := l.spec.Add(name)
	l.setHint(hint)
	if !ok {
		return
	}
	l.name.SetValue("")
	l.Reselect(len(l.spec.Items()) - 1)
	l.changed()
}

func (l *ListEditor[E]) remove() {
	l.Commit()
	items := l.spec.Items()
	i := l.grid.SelectedRow()
	if i < 0 || i >= len(items) {
		l.hint.Set("Select " + l.article() + " " + l.spec.Noun + " in the grid above to remove it.")
		return
	}
	hint, ok := l.spec.Remove(items[i])
	l.setHint(hint)
	if !ok {
		return
	}
	l.Reselect(min(i, len(l.spec.Items())-1))
	l.changed()
}

// Current is the selected item, the zero E when there is none.
func (l *ListEditor[E]) Current() E { return l.current }

// Commit folds the detail rows into the selected item, if any: the form's commit
// hook (NewForm sets it).
func (l *ListEditor[E]) Commit() {
	var zero E
	if l.current != zero {
		l.spec.Commit(l.current)
	}
}

// Redraw redraws the grid from the items, keeping the cursor, column widths and
// scroll: for a change to an item's cells.
func (l *ListEditor[E]) Redraw() { l.grid.SetDataPreservingView(l.spec.Headers, l.rows()) }

// Reselect is Redraw for a change to the item set: the old cursor means nothing,
// so row is selected, scrolled just far enough to show, and shown in the detail
// rows. The detail rows are not committed first.
func (l *ListEditor[E]) Reselect(row int) {
	var zero E
	l.current = zero
	l.grid.SetDataPreservingView(l.spec.Headers, l.rows())
	l.grid.SetSelectedRow(row)
	l.sync()
}

// Grid is the editor's grid, GridRow its form row, and Hint the row Add and
// Remove report on, which a page's own buttons may use too.
func (l *ListEditor[E]) Grid() *controls.DataGrid { return l.grid }
func (l *ListEditor[E]) GridRow() *GridRow        { return l.gridRow }
func (l *ListEditor[E]) Hint() *HintRow           { return l.hint }

// AddRemoveRows is the "Add or remove" section: its heading, the name field,
// the Add and Remove buttons and the hint.
func (l *ListEditor[E]) AddRemoveRows() []Row {
	return []Row{Section("Add or remove"), l.name, l.buttons, l.hint}
}

// NewForm is NewForm(rows...) with the editor's Commit as its commit hook.
func (l *ListEditor[E]) NewForm(rows ...Row) *Form {
	f := NewForm(rows...)
	f.SetCommit(l.Commit)
	return f
}
