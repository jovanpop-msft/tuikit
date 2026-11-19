package dialogs

import (
	"fmt"
	"strings"

	"testing"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/tuikittest"
)

func newTestTypedConfirmDialog(t *testing.T) *TypedConfirmDialog {
	t.Helper()
	return NewTypedConfirmDialog()
}

func TestTypedConfirmDialogMismatchRefusesAndStaysOpen(t *testing.T) {
	d := newTestTypedConfirmDialog(t)
	var got *bool
	d.ShowTypedConfirm("Confirm Overwrite", `Database "HealthClinic" already exists.`, "Heal",
		func(confirmed bool) { got = &confirmed })

	for _, r := range "nope" {
		d.HandleKey(tuikittest.Rune(r))
	}
	d.HandleKey(tuikittest.Key(tcell.KeyEnter))

	if !d.Visible() {
		t.Fatal("dialog closed on a mismatched confirm — should stay open")
	}
	if got != nil {
		t.Fatalf("OnConfirm called with %v on a mismatch — should not fire yet", *got)
	}
	if d.status == "" {
		t.Error("expected a status message explaining the mismatch")
	}
}

func TestTypedConfirmDialogMatchConfirms(t *testing.T) {
	d := newTestTypedConfirmDialog(t)
	var got *bool
	d.ShowTypedConfirm("Confirm Overwrite", `Database "HealthClinic" already exists.`, "Heal",
		func(confirmed bool) { got = &confirmed })

	for _, r := range "Heal" {
		d.HandleKey(tuikittest.Rune(r))
	}
	d.HandleKey(tuikittest.Key(tcell.KeyEnter))

	if d.Visible() {
		t.Fatal("dialog stayed open after a matching confirm")
	}
	if got == nil || !*got {
		t.Fatal("OnConfirm(true) was not called after a matching confirm")
	}
}

func TestTypedConfirmDialogMatchIsCaseInsensitiveAndTrimsSpace(t *testing.T) {
	d := newTestTypedConfirmDialog(t)
	d.ShowTypedConfirm("Confirm", "msg", "Heal", func(bool) {})

	for _, r := range "  HEAL  " {
		d.HandleKey(tuikittest.Rune(r))
	}
	if !d.matched() {
		t.Errorf("expected %q to match required %q case-insensitively, trimmed", d.input.Value(), d.required)
	}
}

func TestTypedConfirmDialogEscapeCancelsWithoutRequiringMatch(t *testing.T) {
	d := newTestTypedConfirmDialog(t)
	var got *bool
	d.ShowTypedConfirm("Confirm", "msg", "Heal", func(confirmed bool) { got = &confirmed })

	d.HandleKey(tuikittest.Key(tcell.KeyEscape))

	if d.Visible() {
		t.Fatal("dialog stayed open after Escape")
	}
	if got == nil || *got {
		t.Fatal("OnConfirm(false) was not called after Escape")
	}
}

// longName is a required text of n columns with no break opportunity, as a
// generated SQL Server name (GUID-suffixed, SharePoint content database) is.
func longName(n int) string {
	return strings.Repeat("Sales_Archive_", n/14+1)[:n]
}

// A required name wider than the default dialog widens it up to the screen cap;
// past the cap the input narrows inside the border and the prompt wraps, so no
// cell is drawn outside the box and every character of the name is on screen.
func TestTypedConfirmLongNameFitsAndShowsWhole(t *testing.T) {
	for _, sz := range [][2]int{{80, 24}, {200, 50}} {
		for _, n := range []int{20, 60, 128} {
			scr := tuikittest.NewScreen(sz[0], sz[1])
			d := withScreen(NewTypedConfirmDialog(), scr)
			name := longName(n)
			d.ShowTypedConfirm("Delete Database", "Delete the database? This cannot be undone.", name, func(bool) {})
			d.Draw(scr)

			r := d.Rect()
			if ib := d.input.Bounds(); ib.X+ib.W-1 > r.X+r.W-2 {
				t.Errorf("%dx%d n=%d: input ends at x=%d, past the right border at x=%d",
					sz[0], sz[1], n, ib.X+ib.W-1, r.X+r.W-1)
			}
			var prompt strings.Builder
			for _, line := range d.promptLines {
				prompt.WriteString(line)
			}
			if !strings.Contains(prompt.String(), name) {
				t.Errorf("%dx%d n=%d: prompt lines %q don't hold the whole name", sz[0], sz[1], n, d.promptLines)
			}
			// What reached the screen, not just promptLines: the rows the prompt
			// occupies, inside the border, concatenated.
			var drawn strings.Builder
			py := scr.RowOf("Type")
			for y := py; y >= 0 && y < py+len(d.promptLines); y++ {
				drawn.WriteString(strings.TrimSpace(string([]rune(scr.Row(y))[r.X+1 : r.X+r.W-1])))
			}
			if !strings.Contains(drawn.String(), name) {
				t.Errorf("%dx%d n=%d: screen prompt %q doesn't hold the whole name", sz[0], sz[1], n, drawn.String())
			}
			// Nothing right of the box: DrawBase dims the backdrop but keeps its runes,
			// so a written cell there shows as a non-space.
			for y := range sz[1] {
				row := []rune(scr.Row(y))
				if tail := strings.TrimRight(string(row[r.X+r.W:]), " "); tail != "" {
					t.Errorf("%dx%d n=%d: row %d drawn right of the box: %q", sz[0], sz[1], n, y, tail)
				}
			}
			// The input row sits below the last prompt row, above the status row.
			if iy := d.input.Bounds().Y; iy != py+len(d.promptLines) || iy >= d.ButtonRowY()-2 {
				t.Errorf("%dx%d n=%d: input at y=%d, prompt ends y=%d, status row y=%d",
					sz[0], sz[1], n, iy, py+len(d.promptLines)-1, d.ButtonRowY()-2)
			}
		}
	}
}

// The whole 128-character name is typeable into the narrowed, scrolling input
// and confirms.
func TestTypedConfirmLongNameTypesAndConfirms(t *testing.T) {
	scr := tuikittest.NewScreen(80, 24)
	d := withScreen(NewTypedConfirmDialog(), scr)
	name := longName(128)
	var got *bool
	d.ShowTypedConfirm("Delete Database", "Delete?", name, func(c bool) { got = &c })
	for _, r := range name {
		d.HandleKey(tuikittest.Rune(r))
	}
	d.Draw(scr)
	d.HandleKey(tuikittest.Key(tcell.KeyEnter))
	if got == nil || !*got {
		t.Fatalf("OnConfirm = %v, want true after typing the whole name (status %q)", got, d.status)
	}
}

// A doubled space in the required text survives wrapping: WrapText would fold it
// into one, showing a name that never matches.
func TestTypedConfirmPromptKeepsDoubledSpace(t *testing.T) {
	name := longName(50) + "  " + longName(50)
	lines := wrapPrompt(fmt.Sprintf(Strings.TypeToConfirm, name), name, 40)
	if got := strings.Join(lines, ""); !strings.Contains(got, name) {
		t.Errorf("wrapped prompt %q lost the name's doubled space", lines)
	}
	for _, l := range lines {
		if w := core.DisplayWidth(l); w > 40 {
			t.Errorf("line %q is %d columns, want <= 40", l, w)
		}
	}
}

// A required name with surrounding spaces confirms whether or not the user types
// them; a whitespace-only one is never matched by an empty input.
func TestTypedConfirmRequiredWithSurroundingSpace(t *testing.T) {
	for _, tc := range []struct {
		required, typed string
		want            bool
	}{
		{" Sales ", "sales", true},
		{" Sales ", " Sales ", true},
		{"Sales ", "  SALES", true},
		{" Sales ", "Sale", false},
		{"   ", "", false},
		{"   ", "   ", true},
	} {
		d := newTestTypedConfirmDialog(t)
		d.ShowTypedConfirm("Confirm", "msg", tc.required, func(bool) {})
		for _, r := range tc.typed {
			d.HandleKey(tuikittest.Rune(r))
		}
		if got := d.matched(); got != tc.want {
			t.Errorf("required %q, typed %q: matched = %v, want %v", tc.required, tc.typed, got, tc.want)
		}
	}
}
