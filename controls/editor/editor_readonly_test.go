package editor

import "testing"

// TestExportedLineMutatorsRespectReadOnly: the line/block actions are called
// directly by hosts (a Menu action closure), bypassing HandleKey's read-only
// filter, so each must refuse a read-only editor itself.
func TestExportedLineMutatorsRespectReadOnly(t *testing.T) {
	const text = "alpha\nbeta\ngamma"
	cases := []struct {
		name string
		fn   func(*Editor)
	}{
		{"DuplicateLines", (*Editor).DuplicateLines},
		{"DeleteLines", (*Editor).DeleteLines},
		{"MoveLinesUp", (*Editor).MoveLinesUp},
		{"MoveLinesDown", (*Editor).MoveLinesDown},
		{"IndentLines", (*Editor).IndentLines},
		{"DedentLines", (*Editor).DedentLines},
		{"ToggleLineComments", (*Editor).ToggleLineComments},
		{"UppercaseSelection", (*Editor).UppercaseSelection},
		{"LowercaseSelection", (*Editor).LowercaseSelection},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, ro := range []bool{true, false} {
				e := NewEditor(nil)
				e.SetText(text)
				e.SetLineComment("--")
				e.SetReadOnly(ro)
				// Middle line, partly selected: every action has something to do
				// (Dedent needs indentation, so indent it first in the text).
				e.doc.setLine(1, []rune("    beta"))
				e.SelectRange(1, 4, 1, 6)
				before := e.Text()
				ver := e.doc.Version()
				c.fn(e)
				changed := e.Text() != before || e.doc.Version() != ver
				if ro && changed {
					t.Errorf("read-only: text %q -> %q, version %d -> %d", before, e.Text(), ver, e.doc.Version())
				}
				if !ro && !changed {
					t.Errorf("writable: %s changed nothing (test setup no longer exercises it)", c.name)
				}
				if ro && len(e.undoStack) > 0 {
					t.Errorf("read-only: an undo step was pushed")
				}
			}
		})
	}
}

// TestSelectedLineRangeMatchesSelectedLinesText: the range is exactly the rows
// SelectedLinesText joins, including the column-0 end exclusion, and !ok in the
// same cases.
func TestSelectedLineRangeMatchesSelectedLinesText(t *testing.T) {
	e := NewEditor(nil)
	e.SetText("l0\nl1\nl2\nl3\nl4")

	if _, _, ok := e.SelectedLineRange(); ok {
		t.Fatal("no selection: ok = true")
	}

	cases := []struct {
		ar, ac, r, c int
		top, bottom  int
		wantText     string
	}{
		{1, 1, 3, 1, 1, 3, "l1\nl2\nl3"},
		{3, 1, 1, 0, 1, 3, "l1\nl2\nl3"}, // backwards
		{1, 0, 3, 0, 1, 2, "l1\nl2"},     // ends at column 0 of row 3: excluded
		{2, 0, 2, 1, 2, 2, "l2"},
	}
	for _, c := range cases {
		e.SelectRange(c.ar, c.ac, c.r, c.c)
		top, bottom, ok := e.SelectedLineRange()
		text, tok := e.SelectedLinesText()
		if !ok || !tok || top != c.top || bottom != c.bottom || text != c.wantText {
			t.Errorf("select (%d,%d)-(%d,%d): range (%d,%d,%v) text %q,%v; want (%d,%d) %q",
				c.ar, c.ac, c.r, c.c, top, bottom, ok, text, tok, c.top, c.bottom, c.wantText)
		}
	}

	e.SelectRange(1, 0, 3, 1)
	e.selBlock = true
	if _, _, ok := e.SelectedLineRange(); ok {
		t.Error("block selection: ok = true")
	}
}
