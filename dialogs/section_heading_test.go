package dialogs

import (
	"testing"

	"github.com/radix29/tuikit/theme"
	"github.com/radix29/tuikit/tuikittest"
)

func TestDrawSectionHeadingDrawsABoldTitleOverARule(t *testing.T) {
	s := tuikittest.NewScreen(20, 4)

	next := DrawSectionHeading(s, 2, 1, 10, "Files")

	if next != 3 {
		t.Errorf("next = %d, want 3 (the row under the rule)", next)
	}
	want := "\n  Files\n  ──────────\n"
	if got := s.Snapshot(); got != want {
		t.Errorf("snapshot =\n%q\nwant\n%q", got, want)
	}
	if st := s.StyleAt(2, 1); st != theme.StyleDialog().Bold(true) {
		t.Errorf("title style = %v, want StyleDialog().Bold(true)", st)
	}
	if st := s.StyleAt(2, 2); st != theme.StyleDialogLine() {
		t.Errorf("rule style = %v, want StyleDialogLine()", st)
	}
}

func TestDrawSectionHeadingClipsTheTitleToTheWidth(t *testing.T) {
	s := tuikittest.NewScreen(20, 3)

	DrawSectionHeading(s, 0, 0, 5, "Files in the backup")

	if got := s.Row(0); got[:6] != "Files " {
		t.Errorf("row 0 = %q, want the title clipped to 5 columns", got)
	}
}
