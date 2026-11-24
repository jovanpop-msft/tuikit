package dialogs

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/tuikittest"
)

// scrollRig is one scrollable dialog for TestRowDragOntoScrollbarDoesNotScroll:
// a content point inside a row, the last cell of the scrollbar track, and the
// dialog's current scroll offset.
type scrollRig struct {
	handle   func(*tcell.EventMouse) bool
	row, bar [2]int
	scroll   func() int
	dragging func() bool
}

// With all-motion tracking a held press resends Button1 on every motion. A
// press on a row, held and dragged onto the scrollbar column, must not grab
// the thumb (ModalDialog.ScrollbarDrag starts a drag on a fresh press only);
// it once jumped the view to the end. A fresh press on the bar still scrolls.
func TestRowDragOntoScrollbarDoesNotScroll(t *testing.T) {
	rigs := map[string]func(t *testing.T) scrollRig{
		"list": func(t *testing.T) scrollRig {
			rows := numbered(60)
			var pressed []string
			d, scr := newTestList(t, &rows, false, &pressed)
			d.Draw(scr)
			in := d.InnerRect()
			return scrollRig{d.HandleMouse, [2]int{in.X + 2, in.Y + 2},
				[2]int{d.Rect().Right() - 1, in.Y + d.dataH()},
				d.Scroll, func() bool { return d.sbDragging }}
		},
		"text": func(t *testing.T) scrollRig {
			d, scr := newTestText(80, 24)
			d.Draw(scr)
			in := d.InnerRect()
			if len(d.Rows()) <= d.DataH() {
				t.Fatalf("rows (%d) must exceed DataH (%d)", len(d.Rows()), d.DataH())
			}
			return scrollRig{d.HandleMouse, [2]int{in.X + 2, in.Y + 2},
				[2]int{d.Rect().Right() - 1, in.Y + d.DataH()},
				d.Scroll, func() bool { return d.sbDragging }}
		},
		"properties": func(t *testing.T) scrollRig {
			d := newTestPropertiesDialog(40)
			in := d.InnerRect()
			return scrollRig{d.HandleMouse, [2]int{in.X + 2, in.Y + 3},
				[2]int{d.Rect().Right() - 1, in.Y + 2 + propsDataH(in) - 1},
				func() int { return d.scroll }, func() bool { return d.sbDragging }}
		},
		"file": func(t *testing.T) scrollRig {
			dir := t.TempDir()
			for i := range 20 {
				name := filepath.Join(dir, "file"+string(rune('a'+i))+".txt")
				if err := os.WriteFile(name, []byte("x"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			d := NewFileDialog()
			d.ShowOpen("Open File", filepath.Join(dir, "filea.txt"), func(string) {})
			d.rect = core.Rect{X: 0, Y: 0, W: 60, H: 20}
			if len(d.entries) <= fileListRows {
				t.Fatalf("need more entries than fileListRows (%d), got %d", fileListRows, len(d.entries))
			}
			lr := d.listRect()
			return scrollRig{d.HandleMouse, [2]int{lr.X + 2, lr.Y + 1},
				[2]int{lr.Right() - 1, lr.Y + lr.H - 1},
				func() int { return d.scroll }, func() bool { return d.sbDragging }}
		},
	}
	for name, mk := range rigs {
		t.Run(name, func(t *testing.T) {
			r := mk(t)
			r.handle(tuikittest.Press(r.row[0], r.row[1]))
			r.handle(tuikittest.Press(r.bar[0], r.bar[1])) // held, onto the bar
			if r.scroll() != 0 || r.dragging() {
				t.Fatalf("row drag onto the bar: scroll %d dragging %v, want 0 false", r.scroll(), r.dragging())
			}
			r.handle(tuikittest.Release(r.bar[0], r.bar[1]))

			r.handle(tuikittest.Press(r.bar[0], r.bar[1]))
			if r.scroll() == 0 || !r.dragging() {
				t.Fatalf("fresh press on the bar: scroll %d dragging %v, want >0 true", r.scroll(), r.dragging())
			}
		})
	}
}
