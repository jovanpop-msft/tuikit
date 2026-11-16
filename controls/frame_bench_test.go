package controls

import (
	"strconv"
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/tuikittest"
)

// The frame benchmarks draw into a real tcell screen on a mock terminal, not
// discardScreen: what they measure is the cost per cell of reaching tcell's
// cell buffer, which a fake that swallows SetContent hides. That cost was two
// allocations per cell per frame before core's drawing went through Put —
// one in putGrapheme's []rune, one in tcell's SetContent re-packing the rune
// into a string — on every visible cell, every frame.

const frameW, frameH = 200, 50

func frameScreen(b *testing.B) tcell.Screen { return tuikittest.NewMockScreen(b, frameW, frameH) }

// BenchmarkFrameDataGrid is a full-screen results grid: header, separator,
// 46 rows of six columns, the status bar and both scrollbars.
func BenchmarkFrameDataGrid(b *testing.B) {
	rows := make([][]string, 1000)
	for i := range rows {
		n := strconv.Itoa(i)
		rows[i] = []string{n, "dbo.SomeTable_" + n, "2026-09-10 12:34:56.789", "ONLINE", "NULL", "a longer value that the column clamp truncates " + n}
	}
	g := NewDataGrid()
	g.SetRowNumbers(true)
	g.SetData([]string{"id", "name", "created", "state", "note", "detail"}, rows)
	g.SetBounds(0, 0, frameW, frameH)
	s := frameScreen(b)

	b.ReportAllocs()
	for b.Loop() {
		g.Draw(s)
	}
}

// BenchmarkFrameDataGridWideChars is review step D3 (2026-11-16): a large
// result of wide-character data, 100k rows by 30 columns of CJK, emoji and
// combining marks, drawn scrolled to the middle in both directions (the
// display-width math of the columns left of the view must not be redone per
// cell). The bar is a frame under 5 ms.
func BenchmarkFrameDataGridWideChars(b *testing.B) {
	const nRows, nCols = 100_000, 30
	cols := make([]string, nCols)
	for c := range cols {
		cols[c] = "列_" + strconv.Itoa(c)
	}
	cells := []string{"你好世界", "日本語のテキスト", "été café", "❤️ ok", "🇷🇴 flag", "한국어 텍스트", "plain ascii text value", "NULL"}
	rows := make([][]string, nRows)
	for i := range rows {
		r := make([]string, nCols)
		for c := range r {
			r[c] = cells[(i+c)%len(cells)] + " " + strconv.Itoa(i)
		}
		rows[i] = r
	}
	g := NewDataGrid()
	g.SetRowNumbers(true)
	g.SetData(cols, rows)
	g.SetBounds(0, 0, frameW, frameH)
	g.SetScroll(nRows/2, nCols/2)
	s := frameScreen(b)
	g.Draw(s)
	if g.ScrollRow() != nRows/2 || g.ScrollCol() != nCols/2 {
		b.Fatalf("scrolled to row %d col %d, want %d, %d", g.ScrollRow(), g.ScrollCol(), nRows/2, nCols/2)
	}

	b.ReportAllocs()
	for b.Loop() {
		g.Draw(s)
	}
}
