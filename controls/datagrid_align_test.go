package controls

import (
	"strings"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

// screenRow is row y of s, x0 through x1 exclusive.
func screenRow(s tcell.Screen, y, x0, x1 int) string {
	var b strings.Builder
	for x := x0; x < x1; x++ {
		str, _, _ := s.Get(x, y)
		if str == "" {
			str = " "
		}
		b.WriteString(str)
	}
	return b.String()
}

// TestGridRightAlignsOnlyMarkedColumns: an AlignRight column ends its cells at
// its right padding, its header stays left, an unmarked column stays left, a
// cell too wide for the column is cut at the right as a left-aligned one is,
// and SetSource forgets the alignments while SetSourcePreservingView keeps them.
func TestGridRightAlignsOnlyMarkedColumns(t *testing.T) {
	rows := [][]string{{"ab", "7"}, {"c", "12345"}, {"d", "1234567890123"}}
	g := newTestDataGrid()
	g.SetMaxCellWidth(9) // column b is 9 wide: 7 columns of text
	g.SetData([]string{"name", "b"}, rows)
	g.SetColumnAlignments([]ColumnAlign{AlignLeft, AlignRight})

	draw := func() tcell.Screen {
		s := tuikittest.NewMockScreen(t, 40, 10)
		g.Draw(s)
		return s
	}
	x0 := g.colWidths[0] // column b's first cell
	x1 := x0 + g.colWidths[1]
	s := draw()
	for y, want := range map[int]string{
		0: " b      |",
		2: "       7|",
		3: "   12345|",
		4: " 123456…|",
	} {
		if got := screenRow(s, y, x0, x1); got != want {
			t.Errorf("row %d of column b = %q, want %q", y, got, want)
		}
	}
	if got := screenRow(s, 2, 0, x0); !strings.HasPrefix(got, " ab ") {
		t.Errorf("left column = %q, want it left-aligned", got)
	}

	g.SetSourcePreservingView([]string{"name", "b"}, SliceRowSource(rows))
	if got := screenRow(draw(), 2, x0, x1); got != "       7|" {
		t.Errorf("after SetSourcePreservingView, row = %q, want the alignment kept", got)
	}
	g.SetData([]string{"name", "b"}, rows)
	if got := screenRow(draw(), 2, x0, x1); got != " 7      |" {
		t.Errorf("after SetData, row = %q, want the alignment cleared", got)
	}
}

// digitsOnly is a test classifier: digits are numbers, "…" and "" blank.
func digitsOnly(cell string) CellClass {
	switch {
	case cell == "" || cell == "…":
		return CellBlank
	case strings.Trim(cell, "0123456789") == "":
		return CellNumber
	}
	return CellText
}

// TestGridAutoAlignFollowsContent: SetAutoAlign right-aligns a column of
// numbers (placeholders aside), not one with any text or only placeholders;
// a cell written in place is picked up by RefreshColumnWidths; a nil
// classifier for a column set leaves it left; explicit alignments override it.
func TestGridAutoAlignFollowsContent(t *testing.T) {
	rows := [][]string{{"a", "7", "…", "1"}, {"b", "…", "…", "x"}}
	g := newTestDataGrid()
	g.SetAutoAlign(func(columns []string) func(string) CellClass {
		if columns[0] == "record" {
			return nil
		}
		return digitsOnly
	})
	g.SetData([]string{"name", "n", "later", "mixed"}, rows)
	want := []ColumnAlign{AlignLeft, AlignRight, AlignLeft, AlignLeft}
	for i, w := range want {
		if got := g.columnAlign(i); got != w {
			t.Errorf("column %d = %v, want %v", i, got, w)
		}
	}

	rows[0][2] = "12"
	g.RefreshColumnWidths()
	g.Draw(tuikittest.NewMockScreen(t, 40, 10))
	if g.columnAlign(2) != AlignRight {
		t.Error("a backfilled column should right-align after RefreshColumnWidths")
	}

	g.SetColumnAlignments([]ColumnAlign{AlignLeft})
	if g.columnAlign(1) != AlignLeft {
		t.Error("explicit alignments should override the auto ones")
	}

	g.SetData([]string{"record", "n"}, [][]string{{"x", "5"}})
	if g.columnAlign(1) != AlignLeft {
		t.Error("a column set the classifier exempts should stay left")
	}
}
