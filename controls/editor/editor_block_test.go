package editor

import (
	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/tuikittest"
)

// blockSelect arms a block (column) selection over rows [topRow, botRow] at
// columns [anchorCol, cursorCol), the state Alt+Shift+Arrow or an Alt+drag
// leaves behind.
func blockSelect(e *Editor, topRow, anchorCol, botRow, cursorCol int) {
	e.selecting, e.selBlock = true, true
	e.selAnchorRow, e.selAnchorCol = topRow, anchorCol
	e.cursorRow, e.cursorCol = botRow, cursorCol
}

func TestEditorBlockTypingInsertsOnEveryRow(t *testing.T) {
	e := newTestEditor("aaaa\nbbbb\ncccc")
	blockSelect(e, 0, 2, 2, 2) // zero-width column caret at column 2

	e.HandleKey(tuikittest.Rune('X', tcell.ModNone))
	e.HandleKey(tuikittest.Rune('Y', tcell.ModNone))

	if got, want := e.Text(), "aaXYaa\nbbXYbb\nccXYcc"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
	if !e.blockEditing() {
		t.Fatal("block should stay armed after typing, so the next key keeps going into every row")
	}
	if e.selAnchorCol != 4 || e.cursorCol != 4 {
		t.Fatalf("block column after typing = (%d,%d), want (4,4)", e.selAnchorCol, e.cursorCol)
	}
}

func TestEditorBlockTypingReplacesSelectedColumns(t *testing.T) {
	e := newTestEditor("abcdef\nghijkl\nmnopqr")
	blockSelect(e, 0, 1, 2, 4)

	e.HandleKey(tuikittest.Rune('_', tcell.ModNone))

	if got, want := e.Text(), "a_ef\ng_kl\nm_qr"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
}

// A row shorter than the block's column is padded with spaces so the inserted
// text stays in the column the user aimed at, matching SSMS/Notepad++.
func TestEditorBlockTypingPadsShortRows(t *testing.T) {
	e := newTestEditor("aaaa\nb\ncccc")
	blockSelect(e, 0, 3, 2, 3)

	e.HandleKey(tuikittest.Rune('X', tcell.ModNone))

	if got, want := e.Text(), "aaaXa\nb  X\ncccXc"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
}

func TestEditorBlockBackspaceAndDelete(t *testing.T) {
	e := newTestEditor("aaaa\nbb\ncccc")
	blockSelect(e, 0, 3, 2, 3)

	e.HandleKey(tuikittest.Key(tcell.KeyBackspace, tcell.ModNone))
	// Row 1 has no rune at column 2, so it is left alone rather than padded.
	if got, want := e.Text(), "aaa\nbb\nccc"; got != want {
		t.Fatalf("after Backspace: Text() = %q, want %q", got, want)
	}
	if e.selAnchorCol != 2 || e.cursorCol != 2 {
		t.Fatalf("block column after Backspace = (%d,%d), want (2,2)", e.selAnchorCol, e.cursorCol)
	}

	e.HandleKey(tuikittest.Key(tcell.KeyDelete, tcell.ModNone))
	if got, want := e.Text(), "aa\nbb\ncc"; got != want {
		t.Fatalf("after Delete: Text() = %q, want %q", got, want)
	}
}

func TestEditorBlockBackspaceDeletesSelectedColumns(t *testing.T) {
	e := newTestEditor("abcdef\nghijkl")
	blockSelect(e, 0, 1, 1, 3)

	e.HandleKey(tuikittest.Key(tcell.KeyBackspace, tcell.ModNone))

	if got, want := e.Text(), "adef\ngjkl"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
}

func TestEditorBlockTabIndentsEveryRow(t *testing.T) {
	e := newTestEditor("aa\nbb")
	blockSelect(e, 0, 1, 1, 1)

	e.HandleKey(tuikittest.Key(tcell.KeyTab, tcell.ModNone))

	if got, want := e.Text(), "a    a\nb    b"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
}

// A zero-width block is a column caret, not a selection: copying it must not
// produce a run of newlines (which would then execute as a blank query).
func TestEditorZeroWidthBlockCopiesNothing(t *testing.T) {
	e := newTestEditor("aaaa\nbbbb")
	blockSelect(e, 0, 2, 1, 2)

	if got := e.SelectedText(); got != "" {
		t.Fatalf("zero-width block SelectedText() = %q, want %q", got, "")
	}
}

// Text copied out of a block selection goes back in as a block — one line per
// row at the cursor's column — even though the OS clipboard carries no mark
// saying so.
func TestEditorBlockCopyPastesBackAsBlock(t *testing.T) {
	e := newTestEditor("12ab\n34cd\n56ef")
	blockSelect(e, 0, 2, 2, 4)

	copied := e.SelectedText()
	if copied != "ab\ncd\nef" {
		t.Fatalf("block SelectedText() = %q, want %q", copied, "ab\ncd\nef")
	}

	e.selecting, e.selBlock = false, false
	e.cursorRow, e.cursorCol = 0, 0
	e.Paste(copied)

	if got, want := e.Text(), "ab12ab\ncd34cd\nef56ef"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
	if e.cursorRow != 2 || e.cursorCol != 2 {
		t.Fatalf("cursor after block paste = (%d,%d), want (2,2)", e.cursorRow, e.cursorCol)
	}
}

// A block paste taller than the remaining buffer appends the rows it needs,
// padded to the paste column so the block stays rectangular.
func TestEditorBlockPasteExtendsBuffer(t *testing.T) {
	e := newTestEditor("12ab\n34cd\n56ef")
	blockSelect(e, 0, 2, 2, 4)
	copied := e.SelectedText()

	e.selecting, e.selBlock = false, false
	e.cursorRow, e.cursorCol = 2, 4
	e.Paste(copied)

	if got, want := e.Text(), "12ab\n34cd\n56efab\n    cd\n    ef"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
}

// Text that didn't come from a block selection still pastes as a plain
// stream, even while a block selection is showing.
func TestEditorNonBlockPasteStaysLinear(t *testing.T) {
	e := newTestEditor("12ab\n34cd")
	e.cursorRow, e.cursorCol = 0, 0
	e.Paste("xy\nz")

	if got, want := e.Text(), "xy\nz12ab\n34cd"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
}

func TestEditorBlockPasteReplacesBlockSelection(t *testing.T) {
	e := newTestEditor("12ab\n34cd\n56ef")
	blockSelect(e, 0, 2, 2, 4)
	copied := e.SelectedText()

	blockSelect(e, 0, 0, 2, 2)
	e.Paste(copied)

	if got, want := e.Text(), "abab\ncdcd\nefef"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
}

func TestEditorBlockEditIsOneUndoStep(t *testing.T) {
	e := newTestEditor("aaaa\nbbbb")
	blockSelect(e, 0, 2, 1, 2)
	e.HandleKey(tuikittest.Rune('X', tcell.ModNone))

	e.HandleKey(tuikittest.Key(tcell.KeyCtrlZ, tcell.ModNone))
	if got, want := e.Text(), "aaaa\nbbbb"; got != want {
		t.Fatalf("after undo: Text() = %q, want %q", got, want)
	}
}

// A block is a rectangle on screen, so its columns are display columns: a CJK
// row reaches column 4 at rune 2, an ASCII row at rune 4. Slicing every row at
// the corners' rune indices selected (and typing replaced) text outside the
// highlighted column.
func TestEditorBlockSelectionIsRectangularOverWideRunes(t *testing.T) {
	e := newTestEditor("名前|x\nabcd|y")
	e.cursorRow, e.cursorCol, e.desiredCol = 0, 2, 4 // before the first '|', display column 4
	e.HandleKey(tuikittest.Key(tcell.KeyDown, tcell.ModAlt|tcell.ModShift))
	e.HandleKey(tuikittest.Key(tcell.KeyRight, tcell.ModAlt|tcell.ModShift))

	if got, want := e.SelectedText(), "|\n|"; got != want {
		t.Fatalf("SelectedText() = %q, want %q", got, want)
	}
	for row, want := range [][2]int{{2, 3}, {4, 5}} {
		lo, hi, ok := e.selectionRangeForLine(row)
		if !ok || lo != want[0] || hi != want[1] {
			t.Fatalf("row %d selection range = [%d,%d) ok=%v, want [%d,%d)", row, lo, hi, ok, want[0], want[1])
		}
	}
	e.HandleKey(tuikittest.Rune('#', tcell.ModNone))
	if got, want := e.Text(), "名前#x\nabcd#y"; got != want {
		t.Fatalf("after typing: Text() = %q, want %q", got, want)
	}
	e.HandleKey(tuikittest.Rune('!', tcell.ModNone))
	if got, want := e.Text(), "名前#!x\nabcd#!y"; got != want {
		t.Fatalf("after typing again: Text() = %q, want %q", got, want)
	}
}

// Typing, Backspace and Delete at a zero-width column caret change every row
// at the same screen column, whatever width the runes left of it have.
func TestEditorBlockCaretEditsOverWideRunesStayInOneColumn(t *testing.T) {
	e := newTestEditor("注ab\nxyab\né.ab")
	blockSelect(e, 0, 1, 2, 2) // column 2: after 注, after xy, after é.
	colsAt := func(what string, wantX int) {
		t.Helper()
		if x := core.ColumnOfRune(e.doc.Line(0), e.selAnchorCol); x != wantX {
			t.Fatalf("%s: anchor at column %d, want %d", what, x, wantX)
		}
		if x := core.ColumnOfRune(e.doc.Line(2), e.cursorCol); x != wantX {
			t.Fatalf("%s: cursor at column %d, want %d", what, x, wantX)
		}
	}

	e.HandleKey(tuikittest.Rune('Z', tcell.ModNone))
	if got, want := e.Text(), "注Zab\nxyZab\né.Zab"; got != want {
		t.Fatalf("after typing: Text() = %q, want %q", got, want)
	}
	colsAt("after typing", 3)

	e.HandleKey(tuikittest.Key(tcell.KeyDelete, tcell.ModNone))
	if got, want := e.Text(), "注Zb\nxyZb\né.Zb"; got != want {
		t.Fatalf("after Delete: Text() = %q, want %q", got, want)
	}
	colsAt("after Delete", 3)

	e.HandleKey(tuikittest.Key(tcell.KeyBackspace, tcell.ModNone))
	if got, want := e.Text(), "注b\nxyb\né.b"; got != want {
		t.Fatalf("after Backspace: Text() = %q, want %q", got, want)
	}
	colsAt("after Backspace", 2)

	// Backspace over the wide glyph: rows lose one cluster each, the caret the
	// top row's two columns.
	e.HandleKey(tuikittest.Key(tcell.KeyBackspace, tcell.ModNone))
	if got, want := e.Text(), "b\nxb\néb"; got != want {
		t.Fatalf("after Backspace over 注: Text() = %q, want %q", got, want)
	}
	colsAt("after Backspace over 注", 0)
}

// A block edit leaves a zero-width caret even when a wide glyph straddles its
// column on one corner row (that corner snaps to the glyph's start, the other
// doesn't): the next keystroke must insert, not delete a column on every row.
func TestEditorBlockCaretStaysZeroWidthWhenAGlyphStraddlesIt(t *testing.T) {
	zeroWidth := func(e *Editor, what string) {
		t.Helper()
		if lo, hi := e.blockXBounds(); lo != hi {
			t.Fatalf("%s: block caret spans columns [%d,%d), want zero width", what, lo, hi)
		}
		if got := e.SelectedText(); got != "" && got != "\n" {
			t.Fatalf("%s: SelectedText() = %q, want nothing selected", what, got)
		}
	}

	e := newTestEditor("abcdef\n名名名")
	blockSelect(e, 0, 1, 1, 2)
	e.HandleKey(tuikittest.Rune('#', tcell.ModNone))
	zeroWidth(e, "after typing over a straddled block")
	before := e.Text()
	e.HandleKey(tuikittest.Rune('x', tcell.ModNone))
	if got, want := e.Text(), "a#xef\n#x名"; got != want {
		t.Fatalf("second keystroke: Text() = %q (was %q), want %q", got, before, want)
	}
	zeroWidth(e, "after the second keystroke")

	e = newTestEditor("a名\n名c")
	blockSelect(e, 0, 2, 1, 2) // zero-width at column 3: after 名 on row 0, after c on row 1
	e.HandleKey(tuikittest.Key(tcell.KeyBackspace, tcell.ModNone))
	if got, want := e.Text(), "a\n名"; got != want {
		t.Fatalf("Backspace: Text() = %q, want %q", got, want)
	}
	zeroWidth(e, "after Backspace")
	e.HandleKey(tuikittest.Rune('x', tcell.ModNone))
	// Column 1 is mid-glyph on row 1, so that row inserts at the glyph's start.
	if got, want := e.Text(), "ax\nx名"; got != want {
		t.Fatalf("typing after Backspace: Text() = %q, want %q", got, want)
	}

	// Moving a corner drops the stored column: the block is the corners' again.
	e.HandleKey(tuikittest.Key(tcell.KeyLeft, tcell.ModAlt|tcell.ModShift))
	if lo, hi := e.blockXBounds(); lo == hi {
		t.Fatalf("after Alt+Shift+Left the block is still zero-width at %d", lo)
	}
}

// A zero-width Backspace on a row where a wide glyph straddles the column
// deletes left of where that row's caret is drawn (the glyph's start), never the
// glyph, and the caret lands where a row whose caret sat on the column deleted.
func TestEditorBlockBackspaceBesideAStraddlingGlyph(t *testing.T) {
	e := newTestEditor("a#xef\n#x名")
	e.setBlockCaret(0, 1, 3) // after x on row 0; mid-名 on row 1, drawn before it
	e.HandleKey(tuikittest.Key(tcell.KeyBackspace, tcell.ModNone))
	if got, want := e.Text(), "a#ef\n#名"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
	if lo, hi := e.blockXBounds(); lo != 2 || hi != 2 {
		t.Fatalf("caret at columns [%d,%d), want 2", lo, hi)
	}

	// The first edited row's caret is mid-glyph at the line start: nothing left
	// of it to delete; the caret follows the row that did delete.
	e = newTestEditor("\n名z\nab")
	e.setBlockCaret(0, 2, 1)
	e.HandleKey(tuikittest.Key(tcell.KeyBackspace, tcell.ModNone))
	if got, want := e.Text(), "\n名z\nb"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
	if lo, hi := e.blockXBounds(); lo != 0 || hi != 0 {
		t.Fatalf("caret at columns [%d,%d), want 0", lo, hi)
	}
}

// A wide glyph straddling either edge of the block is wholly inside it, never
// half selected.
func TestEditorBlockWideGlyphStraddlingAnEdgeIsInside(t *testing.T) {
	e := newTestEditor("a注b\nwxyz")
	blockSelect(e, 0, 0, 1, 2) // columns [0,2): 注 covers columns 1-2
	if got, want := e.SelectedText(), "a注\nwx"; got != want {
		t.Fatalf("right edge: SelectedText() = %q, want %q", got, want)
	}
	// Columns [2,3), anchored on the ASCII row: 注 covers columns 1-2.
	e.selAnchorRow, e.selAnchorCol = 1, 2
	e.cursorRow, e.cursorCol = 0, 2 // 'b', column 3
	if got, want := e.SelectedText(), "注\ny"; got != want {
		t.Fatalf("left edge: SelectedText() = %q, want %q", got, want)
	}
}

// A combining mark adds no column: é (e + U+0301) is one cell.
func TestEditorBlockCombiningMarkIsOneCell(t *testing.T) {
	e := newTestEditor("éxy\nabc")
	blockSelect(e, 0, 2, 1, 1) // column 1 on both rows: after é, after a
	e.HandleKey(tuikittest.Rune('|', tcell.ModNone))
	if got, want := e.Text(), "é|xy\na|bc"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
	blockSelect(e, 0, 0, 1, 2) // columns [0,2): "é|" and "a|" (rune 2 on row 1)
	if got, want := e.SelectedText(), "é|\na|"; got != want {
		t.Fatalf("SelectedText() = %q, want %q", got, want)
	}
}

// A block copied over rows of mixed width pastes back at one screen column.
func TestEditorBlockCopyPasteRoundTripOverWideRunes(t *testing.T) {
	e := newTestEditor("注ab\nxyab\nzzab")
	blockSelect(e, 0, 1, 2, 4) // columns [2,4): "ab" on every row
	copied := e.SelectedText()
	if copied != "ab\nab\nab" {
		t.Fatalf("SelectedText() = %q, want %q", copied, "ab\nab\nab")
	}
	e.selecting, e.selBlock = false, false
	e.cursorRow, e.cursorCol = 0, 1 // column 2, after 注
	e.Paste(copied)
	if got, want := e.Text(), "注abab\nxyabab\nzzabab"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
	if e.cursorRow != 2 || e.cursorCol != 4 {
		t.Fatalf("cursor after paste = (%d,%d), want (2,4)", e.cursorRow, e.cursorCol)
	}
}

// Alt+drag arms the same block the keyboard does over the same text.
func TestEditorBlockAltDragOverWideRunesMatchesKeyboard(t *testing.T) {
	e := newTestEditor("名前|x\nabcd|y")
	e.SetGutterVisible(false)
	e.SetBounds(0, 0, 30, 4)

	e.HandleMouse(tcell.NewEventMouse(4, 0, tcell.Button1, tcell.ModAlt))
	e.HandleMouse(tcell.NewEventMouse(5, 1, tcell.Button1, tcell.ModAlt))
	e.HandleMouse(tcell.NewEventMouse(5, 1, tcell.ButtonNone, tcell.ModNone))

	if !e.selBlock {
		t.Fatal("Alt+drag did not arm a block selection")
	}
	if got, want := e.SelectedText(), "|\n|"; got != want {
		t.Fatalf("SelectedText() = %q, want %q", got, want)
	}
}

// A block edit that removes a wide glyph straddling the block's left edge leaves
// that row narrower than the block's column. The cursor corner used to stay at
// the virtual index past the row's end, so dropping the block (Esc) and typing
// indexed out of range. Row 1 below is 3 runes, 4 columns; the block covers
// column [3,4), the right half of 漢, so the whole glyph goes.
func TestEditorBlockDeleteOfStraddlingGlyphKeepsCursorInRow(t *testing.T) {
	for _, k := range []struct {
		name string
		key  tcell.Key
		mod  tcell.ModMask
	}{
		{"Backspace", tcell.KeyBackspace2, tcell.ModNone},
		{"Delete", tcell.KeyDelete, tcell.ModNone},
		{"Ctrl+Backspace", tcell.KeyBackspace2, tcell.ModCtrl},
		{"Ctrl+Delete", tcell.KeyDelete, tcell.ModCtrl},
	} {
		t.Run(k.name, func(t *testing.T) {
			e := newTestEditor("👍/\n'a漢\n/")
			e.HandleKey(tuikittest.Key(tcell.KeyDown, tcell.ModNone))
			e.HandleKey(tuikittest.Key(tcell.KeyEnd, tcell.ModNone))
			e.HandleKey(tuikittest.Key(tcell.KeyUp, tcell.ModAlt|tcell.ModShift))
			e.HandleKey(tuikittest.Key(k.key, k.mod))

			if got, want := e.Text(), "👍/\n'a\n/"; got != want {
				t.Fatalf("Text() after block %s = %q, want %q", k.name, got, want)
			}
			if l := len(e.doc.Line(e.cursorRow)); e.cursorCol > l {
				t.Fatalf("cursor (%d,%d) past its row's end %d", e.cursorRow, e.cursorCol, l)
			}
			e.HandleKey(tuikittest.Key(tcell.KeyEscape, tcell.ModNone))
			e.HandleKey(tuikittest.Rune('x', tcell.ModNone))
			if got, want := e.Text(), "👍/\n'ax\n/"; got != want {
				t.Fatalf("Text() after Esc, x = %q, want %q", got, want)
			}
		})
	}
}

// Clamping the cursor corner to a short row must not move the block's column:
// extending it down afterwards continues from the block's column, not from the
// short row's end, and the anchor's ragged edge keeps that column too.
func TestEditorBlockExtendAfterShortRowKeepsColumn(t *testing.T) {
	e := newTestEditor("👍/\n'a漢\n'a漢\nabcdef")
	blockSelect(e, 0, 2, 2, 3) // display columns [3,4) on rows 0-2

	e.HandleKey(tuikittest.Key(tcell.KeyDelete, tcell.ModNone))
	e.HandleKey(tuikittest.Key(tcell.KeyDown, tcell.ModAlt|tcell.ModShift))

	if lo, hi := e.blockXBounds(); lo != 3 || hi != 3 {
		t.Fatalf("block columns after Alt+Shift+Down = [%d,%d), want [3,3)", lo, hi)
	}
	e.HandleKey(tuikittest.Rune('X', tcell.ModNone))
	if got, want := e.Text(), "👍/X\n'a X\n'a X\nabcXdef"; got != want {
		t.Fatalf("Text() = %q, want %q", got, want)
	}
}
