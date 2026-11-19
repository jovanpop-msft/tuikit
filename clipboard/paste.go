package clipboard

import (
	"strings"

	"github.com/gdamore/tcell/v3"
)

// PasteBuffer collects a terminal bracketed paste. tcell's EnablePaste only
// brackets the paste with *tcell.EventPaste start/end markers; the content
// still arrives as ordinary EventKeys. A host calls Begin on the start marker,
// Key for every EventKey while Active, and End on the end marker, then applies
// the returned text to its target as one edit.
//
// Applying the whole text once is the point, rather than replaying the keys as
// typing: a replayed newline arrives as KeyEnter, which an open completion
// popup takes as "commit the candidate", silently rewriting the pasted text.
// One edit is also one undo step. A host should also skip drawing a frame per
// buffered key: a frame each made a 10,001-line, 170 KB paste take minutes.
//
// The zero value is ready to use and not Active.
type PasteBuffer struct {
	active bool
	buf    strings.Builder
	// afterCR is true when the last key was KeyEnter (a CR), so an LF right
	// behind it is the second half of a CRLF and adds no second break; see Key.
	afterCR bool
}

// Begin starts a paste, discarding anything left from an earlier one.
func (p *PasteBuffer) Begin() {
	p.active = true
	p.buf.Reset()
	p.afterCR = false
}

// Active reports whether a paste is in progress: between Begin and End, every
// EventKey belongs to Key rather than to the host's key handling.
func (p *PasteBuffer) Active() bool { return p.active }

// Key appends one key of the paste. Anything that isn't a character, newline
// or tab is dropped rather than acted on (a stray escape sequence can decode
// as a function key), so a paste can never trigger a command.
//
// A newline arrives as either key. Most terminals rewrite LF to CR inside a
// paste, which tcell reports as KeyEnter; Alacritty and tmux `paste-buffer -r`
// send LF unchanged, which tcell's legacy key mode reports as KeyCtrlJ.
// Dropping KeyCtrlJ would run every pasted line into the next. A CRLF
// clipboard sends both, so an LF straight after a CR is skipped rather than
// doubling the break.
func (p *PasteBuffer) Key(ev *tcell.EventKey) {
	afterCR := p.afterCR
	p.afterCR = false
	switch ev.Key() {
	case tcell.KeyRune:
		p.buf.WriteString(ev.Str())
	case tcell.KeyEnter:
		p.buf.WriteByte('\n')
		p.afterCR = true
	case tcell.KeyCtrlJ:
		if !afterCR {
			p.buf.WriteByte('\n')
		}
	case tcell.KeyTab:
		p.buf.WriteByte('\t')
	}
}

// End finishes the paste and returns its text with StripBOMs applied ("" for
// an empty paste). The buffer is left empty and not Active.
func (p *PasteBuffer) End() string {
	p.active = false
	p.afterCR = false
	text := StripBOMs(p.buf.String())
	p.buf.Reset()
	return text
}

// StripBOMs removes every U+FEFF from pasted text. It is invisible in an
// editor, but a SQL parser (SQL Server's, for one) rejects it as a syntax
// error near a seemingly empty token, and clipboards carry it: clip.exe, and
// other Windows tools, leave the BOM their Unicode input needs on the clipboard
// as a leading U+FEFF, which a paste into the middle of a line buries
// mid-text. As a zero-width no-break space it is deprecated in favour of
// U+2060, so dropping it everywhere loses nothing meant.
//
// End applies it; a host applies it to text from Read or an EventClipboard.
func StripBOMs(text string) string {
	return strings.ReplaceAll(text, "\uFEFF", "")
}
