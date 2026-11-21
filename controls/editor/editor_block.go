package editor

import (
	"strings"

	"github.com/radix29/tuikit/core"
)

// ---------------------------------------------------------------------------
// Block (column) editing for Editor — typing, deleting and pasting across
// every row of a rectangular selection at once, Notepad++ style
// ---------------------------------------------------------------------------

// blockEditing reports whether an edit should apply to every row of a block
// (column) selection rather than the cursor's row alone. It stays true after a
// zero-width block (equal columns, what typing in column mode leaves behind), so
// typed characters keep going into every row instead of the first collapsing the
// block and the rest landing on one line. HasSelection alone is not enough: it
// reports false once anchor and cursor coincide.
func (e *Editor) blockEditing() bool { return e.selecting && e.selBlock }

// blockRows returns the block selection's row range, ordered.
func (e *Editor) blockRows() (top, bot int) {
	return min(e.selAnchorRow, e.cursorRow), max(e.selAnchorRow, e.cursorRow)
}

// blockCaretState is Editor.blockCaret: the column x of the zero-width block
// setBlockCaret armed, and the corners and document version it armed it at.
type blockCaretState struct {
	ok             bool
	version        uint64
	ar, ac, cr, cc int
	x              int
}

// setBlockCaret re-arms the block over rows top..bot as a zero-width column at
// display column x: the state every block edit ends in, so the next typed
// character continues down the same column. The corners are stored as the rune
// index at x on their own rows, which map back to x unless a wide glyph
// straddles it; then that corner snaps to the glyph's start, as vertical motion
// does. x itself is kept in blockCaret, so the block stays zero-width either way.
func (e *Editor) setBlockCaret(top, bot, x int) {
	x = max(x, 0)
	e.selecting, e.selBlock = true, true
	e.selAnchorRow, e.selAnchorCol = top, core.RuneIndexAtColumn(e.doc.Line(top), x)
	e.cursorRow, e.cursorCol = bot, core.RuneIndexAtColumn(e.doc.Line(bot), x)
	e.blockCaret = blockCaretState{ok: true, version: e.doc.Version(),
		ar: top, ac: e.selAnchorCol, cr: bot, cc: e.cursorCol, x: x}
}

// blockCaretX returns the zero-width block's column setBlockCaret stored, if
// nothing has moved a corner or edited the document since.
func (e *Editor) blockCaretX() (int, bool) {
	b := e.blockCaret
	if !b.ok || !e.selecting || !e.selBlock || b.version != e.doc.Version() ||
		b.ar != e.selAnchorRow || b.ac != e.selAnchorCol || b.cr != e.cursorRow || b.cc != e.cursorCol {
		return 0, false
	}
	return b.x, true
}

// blockPadLine pads row's line with spaces to at least x display columns,
// writing it back to the document, and returns the line and the rune index at x,
// where a block edit inserts on that row. A block edit at a column past a short
// row's end must insert at that column to stay rectangular, so the gap is filled
// with spaces (SSMS's and Notepad++'s column-mode behaviour). Width, not rune
// count: a row holding a wide or combining rune reaches x at a different index.
func (e *Editor) blockPadLine(row, x int) ([]rune, int) {
	line := e.doc.Line(row)
	if w := core.RunesWidth(line); w < x {
		nl := make([]rune, len(line), len(line)+x-w)
		copy(nl, line)
		for range x - w {
			nl = append(nl, ' ')
		}
		e.doc.setLine(row, nl)
		line = nl
	}
	return line, min(core.RuneIndexAtColumn(line, x), len(line))
}

// blockCollapse removes the block selection's contents (no-op for a zero-width
// block), leaving a zero-width block at its left display column, and returns that
// column.
func (e *Editor) blockCollapse() int {
	top, bot := e.blockRows()
	loX, hiX := e.blockXBounds()
	if hiX > loX {
		for row := top; row <= bot; row++ {
			lo, hi := e.blockRuneRange(row, loX, hiX)
			line := e.doc.Line(row)
			e.doc.setLine(row, append(line[:lo], line[hi:]...))
		}
	}
	e.setBlockCaret(top, bot, loX)
	return loX
}

// blockInsertRune replaces the block's contents with r on every row it spans,
// then leaves the block armed just right of it: r's width further on (none for
// a combining mark, which joins the cluster before it), measured on the top row
// so every row keeps one screen column.
func (e *Editor) blockInsertRune(r rune) {
	top, bot := e.blockRows()
	x := e.blockCollapse()
	dx := 0
	for row := top; row <= bot; row++ {
		line, i := e.blockPadLine(row, x)
		nl := make([]rune, len(line)+1)
		copy(nl, line[:i])
		nl[i] = r
		copy(nl[i+1:], line[i:])
		e.doc.setLine(row, nl)
		if row == top {
			dx = core.RunesWidth(nl[:i+1]) - core.RunesWidth(line[:i])
		}
	}
	e.setBlockCaret(top, bot, x+dx)
}

// blockBackspace deletes the block's contents or, when zero-width, the grapheme
// left of the caret on every row it spans. On a row where a wide glyph straddles
// the column the caret is drawn at the glyph's start (RuneIndexAtColumn, where
// typing inserts too), so the grapheme before that glyph goes, never the glyph.
// Rows too narrow to reach the column are left alone, not padded: nothing to
// delete. The caret moves to where the deletion began on a row whose caret sat
// exactly at the column, else on the first edited row.
func (e *Editor) blockBackspace() {
	top, bot := e.blockRows()
	loX, hiX := e.blockXBounds()
	if hiX > loX {
		e.blockCollapse()
		return
	}
	if loX == 0 {
		return
	}
	newX, exact := -1, false
	for row := top; row <= bot; row++ {
		line := e.doc.Line(row)
		if core.RunesWidth(line) < loX {
			continue
		}
		idx := core.RuneIndexAtColumn(line, loX)
		if idx == 0 {
			continue
		}
		lo := core.PrevGrapheme(line, idx)
		if onCol := core.ColumnOfRune(line, idx) == loX; newX < 0 || (onCol && !exact) {
			newX, exact = core.RunesWidth(line[:lo]), onCol
		}
		e.doc.setLine(row, append(line[:lo], line[idx:]...))
	}
	if newX < 0 {
		newX = loX - 1
	}
	e.setBlockCaret(top, bot, newX)
}

// blockDelete deletes the block's contents, or — when the block is
// zero-width — the grapheme at its column on every row it spans (a wide one
// straddling the column counts as at it).
func (e *Editor) blockDelete() {
	top, bot := e.blockRows()
	loX, hiX := e.blockXBounds()
	if hiX > loX {
		e.blockCollapse()
		return
	}
	for row := top; row <= bot; row++ {
		line := e.doc.Line(row)
		idx, _ := core.ClusterAtColumn(line, loX)
		if idx >= len(line) {
			continue
		}
		e.doc.setLine(row, append(line[:idx], line[core.NextGrapheme(line, idx):]...))
	}
	e.setBlockCaret(top, bot, loX)
}

// blockPasteSpan is the row range blockPaste(text) touches, for its undo step:
// from the first row of the selection or cursor (where the paste starts) down to
// the last pasted row, or to the end of the buffer when the paste appends rows
// there, so the growth happens inside it. The pasted rows are counted from the
// selection's last row: deleting a linear selection first joins its rows into
// one, so the paste reaches that many rows past it in the pre-edit numbering.
func (e *Editor) blockPasteSpan(text string) (lo, hi int) {
	lo, bot := e.cursorRow, e.cursorRow
	if e.selecting {
		lo, bot = e.blockRows()
	}
	return lo, min(bot+strings.Count(text, "\n")+1, e.doc.Len())
}

// blockPaste inserts text rectangularly: line i of the pasted text goes into the
// i'th row below the cursor, all at the same column, appending rows at the end of
// the buffer if the paste runs past it. This is what a block copy pasted back
// expects (see Paste for how one is recognised). The column is a display
// column, mapped onto each row's own runes, so rows holding wide characters
// stay aligned on screen. The caller pushes the undo step.
func (e *Editor) blockPaste(text string) {
	parts := strings.Split(normalizeEOL(e.expandTabs(text)), "\n")
	var x int
	switch {
	case e.blockEditing():
		x = e.blockCollapse()
		e.cursorRow, _ = e.blockRows()
	case e.HasSelection():
		e.deleteSelection()
		x = e.cursorDisplayCol()
	default:
		x = e.cursorDisplayCol()
	}
	e.selecting, e.selBlock = false, false
	row := e.cursorRow
	for i, part := range parts {
		r := row + i
		if r >= e.doc.Len() {
			e.doc.edit(func(lines [][]rune) [][]rune { return append(lines, []rune{}) })
		}
		seg := []rune(part)
		line, i := e.blockPadLine(r, x)
		nl := make([]rune, 0, len(line)+len(seg))
		nl = append(nl, line[:i]...)
		nl = append(nl, seg...)
		nl = append(nl, line[i:]...)
		e.doc.setLine(r, nl)
		e.cursorCol = i + len(seg)
	}
	e.cursorRow = row + len(parts) - 1
}
