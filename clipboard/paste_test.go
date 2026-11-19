package clipboard

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

func pasteRunes(p *PasteBuffer, s string) {
	for _, r := range s {
		p.Key(tcell.NewEventKey(tcell.KeyRune, string(r), tcell.ModNone))
	}
}

// Only characters, newlines and tabs are pasted; any other key a terminal
// sends mid-paste is dropped, not acted on.
func TestPasteBufferKeepsTextAndDropsOtherKeys(t *testing.T) {
	var p PasteBuffer
	if p.Active() {
		t.Fatal("zero PasteBuffer is Active")
	}
	p.Begin()
	if !p.Active() {
		t.Fatal("not Active after Begin")
	}
	for _, ev := range []*tcell.EventKey{
		tcell.NewEventKey(tcell.KeyRune, "a", tcell.ModNone),
		tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone),
		tcell.NewEventKey(tcell.KeyTab, "", tcell.ModNone),
		tcell.NewEventKey(tcell.KeyRune, "b", tcell.ModNone),
		tcell.NewEventKey(tcell.KeyF5, "", tcell.ModNone),
	} {
		p.Key(ev)
	}
	if got, want := p.End(), "a\n\tb"; got != want {
		t.Fatalf("End() = %q, want %q", got, want)
	}
	if p.Active() {
		t.Fatal("still Active after End")
	}
	if got := p.End(); got != "" {
		t.Fatalf("second End() = %q, want the buffer emptied", got)
	}
}

// A pasted line break survives whichever byte the terminal sends for it.
// tcell's legacy key mode decodes a raw LF as KeyCtrlJ, which a paste buffer
// that dropped it ran `SELECT a⏎FROM t` together as `SELECT aFROM t` from
// Alacritty or tmux `paste-buffer -r`. The events are built from the raw bytes
// so they go through tcell's own normalization instead of naming KeyCtrlJ.
func TestPasteBufferKeepsEveryLineBreakShape(t *testing.T) {
	for _, tc := range []struct {
		name, in, want string
	}{
		{"LF", "a\nb", "a\nb"},
		{"CR", "a\rb", "a\nb"},
		{"CRLF", "a\r\nb", "a\nb"},
		{"CR CR", "a\r\rb", "a\n\nb"},
		{"LF LF", "a\n\nb", "a\n\nb"},
		{"LF CR", "a\n\rb", "a\n\nb"},
		{"CRLF CRLF", "a\r\n\r\nb", "a\n\nb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var p PasteBuffer
			p.Begin()
			pasteRunes(&p, tc.in)
			if got := p.End(); got != tc.want {
				t.Fatalf("pasting %q gave %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// A CR ending one paste must not swallow an LF opening the next.
func TestPasteBufferCRDoesNotCarryOver(t *testing.T) {
	var p PasteBuffer
	var got string
	for _, in := range []string{"a\r", "\nb"} {
		p.Begin()
		pasteRunes(&p, in)
		got += p.End()
	}
	if want := "a\n\nb"; got != want {
		t.Fatalf("two pastes gave %q, want %q", got, want)
	}
}

// Begin discards a paste whose end marker never came.
func TestPasteBufferBeginDiscardsAnUnfinishedPaste(t *testing.T) {
	var p PasteBuffer
	p.Begin()
	pasteRunes(&p, "lost")
	p.Begin()
	pasteRunes(&p, "kept")
	if got := p.End(); got != "kept" {
		t.Fatalf("End() = %q, want %q", got, "kept")
	}
}

// Every U+FEFF goes, wherever it sits, on End and through StripBOMs.
func TestPasteDropsBOMs(t *testing.T) {
	const in, want = "\uFEFFSELECT 1\uFEFF;", "SELECT 1;"
	if got := StripBOMs(in); got != want {
		t.Fatalf("StripBOMs(%q) = %q, want %q", in, got, want)
	}
	var p PasteBuffer
	p.Begin()
	pasteRunes(&p, in)
	if got := p.End(); got != want {
		t.Fatalf("End() = %q, want %q", got, want)
	}
}
