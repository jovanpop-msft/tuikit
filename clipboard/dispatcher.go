package clipboard

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v3"
)

// Target is the widget a paste lands in. core.ClipboardTarget satisfies it;
// the package names only the one method it calls so it stays a leaf.
//
// Two Targets are compared with ==, so a host's Resolve must hand back the
// same value for the same widget each time (a pointer, as every tuikit widget
// is).
type Target interface{ Paste(text string) }

// Dispatcher moves text between a host's widgets and the system clipboard:
// Copy writes the native clipboard, falling back to OSC 52; Paste reads it,
// falling back to the terminal's OSC 52 reply; HandleEvent takes the
// terminal's bracketed paste and that reply. The host keeps which widget is
// active (Resolve), the wording of its status line, and its goroutine policy
// (Go).
//
// Every method is called on the host's UI goroutine. A Dispatcher must not be
// copied after first use; set its fields before then and leave them.
type Dispatcher struct {
	// Screen is the screen the OSC 52 fallback reads and writes. Required. A
	// func rather than a value so a host that swaps its screen (a test stub,
	// a re-init) needs no second wiring.
	Screen func() tcell.Screen

	// Resolve names the widget Copy/Paste acts on now, or nil. Required. Paste
	// calls it when Ctrl+V is pressed, and again when the text arrives.
	Resolve func() Target

	// Token, optional, names which field of the resolved Target has focus,
	// for a host whose Target is itself rather than the field (a property
	// sheet: the Target is the sheet whichever row has focus, so the identity
	// check alone can't see a move from one row to the next, and a paste aimed
	// at Name would arrive in Description). Compared with ==; nil means "the
	// Target is the field".
	Token func() any

	// Post queues fn to run on the UI goroutine and wakes the event loop
	// (loop.Poster.Post). Required: every result of a background read or
	// write comes back through it.
	Post func(fn func())

	// Go runs fn on a new goroutine; what names it for a panic report. nil is
	// a plain go statement. A host that recovers background panics (a crash
	// there leaves the terminal in raw mode) passes its own.
	Go func(what string, fn func())

	// Pasted, optional, runs after text landed in t through Paste, Deliver or
	// a bracketed paste, so a dialog can follow up as after a keystroke.
	Pasted func(t Target)

	// Unconfirmed, optional, runs on the UI goroutine after Copy fell back to
	// an OSC 52 write that TerminalLikely says the terminal may have ignored
	// (VTE does), so a host can hedge the "Copied" it already showed.
	Unconfirmed func()

	// NativeWrite and NativeRead are the native clipboard; nil is the
	// package's Write and Read. Tests swap them.
	NativeWrite func(text string) bool
	NativeRead  func() (string, bool)

	// writeMu runs Copy's goroutines one at a time, and writeSeq numbers them,
	// so of two quick copies the later one is what the clipboard ends up
	// holding. See Copy.
	writeMu  sync.Mutex
	writeSeq atomic.Uint64

	// doneSeq is the highest Copy number whose goroutine has finished (written
	// or skipped); doneCh is closed, and replaced, on each finish. Both are
	// guarded by doneMu, never by writeMu: a Copy holds writeMu across its
	// write, and a Win32 write can block indefinitely (EmptyClipboard waits on
	// a hung previous owner), so a Paste that took writeMu to check doneSeq
	// would hang with it. Paste waits on doneCh so a read never overtakes a
	// Copy issued before it, but at most pasteWait.
	doneMu  sync.Mutex
	doneSeq uint64
	doneCh  chan struct{}

	// pasteWait bounds Paste's wait for earlier Copies; zero is
	// defaultPasteWait. Tests shorten it.
	pasteWait time.Duration

	// bracket is a terminal bracketed paste in progress (see HandleEvent).
	bracket PasteBuffer

	// pending and pendingToken are what a Ctrl+V was aimed at while the
	// terminal's OSC 52 reply is outstanding. See Paste and Deliver.
	pending      Target
	pendingToken any
}

func (d *Dispatcher) goroutine(what string, fn func()) {
	if d.Go != nil {
		d.Go(what, fn)
		return
	}
	go fn()
}

// defaultPasteWait bounds how long Paste waits for earlier Copies to land: a
// tool write gives up after toolTimeout, so past this the writer is stuck (a
// Win32 write blocked on a hung clipboard owner), and reading a possibly stale
// clipboard beats a Ctrl+V that never does anything.
const defaultPasteWait = toolTimeout + time.Second

// finished records that Copy number seq's goroutine is done and wakes every
// Paste waiting on it.
func (d *Dispatcher) finished(seq uint64) {
	d.doneMu.Lock()
	defer d.doneMu.Unlock()
	d.doneSeq = max(d.doneSeq, seq)
	if d.doneCh != nil {
		close(d.doneCh)
		d.doneCh = nil
	}
}

// waitCopied blocks until every Copy up to seq has finished, or until the
// deadline passes; it reports whether they all finished.
func (d *Dispatcher) waitCopied(seq uint64, deadline <-chan time.Time) bool {
	for {
		d.doneMu.Lock()
		if d.doneSeq >= seq {
			d.doneMu.Unlock()
			return true
		}
		if d.doneCh == nil {
			d.doneCh = make(chan struct{})
		}
		ch := d.doneCh
		d.doneMu.Unlock()
		select {
		case <-ch:
		case <-deadline:
			return false
		}
	}
}

func (d *Dispatcher) token() any {
	if d.Token == nil {
		return nil
	}
	return d.Token()
}

// Copy sends text to the clipboard off the UI goroutine: the native clipboard
// first, falling back to OSC 52 (tcell's SetClipboard) when no native tool
// handled it. The shell-out runs on a background goroutine so a stalled tool
// can't freeze the event loop; the fallback returns to the UI goroutine
// through Post, since SetClipboard writes to the terminal.
//
// One write at a time, and only the newest: racing goroutines would let the
// earlier copy finish last and win. Under writeMu a write a later Copy has
// superseded is skipped, whichever goroutine gets the lock first, as is its
// OSC 52 fallback if a newer Copy came in meanwhile.
func (d *Dispatcher) Copy(text string) {
	seq := d.writeSeq.Add(1)
	write := d.NativeWrite
	if write == nil {
		write = Write
	}
	d.goroutine("writing to the clipboard", func() {
		d.writeMu.Lock()
		defer func() {
			d.writeMu.Unlock()
			// Superseded copies count too: a Paste issued after them must
			// not wait on a write that will never happen.
			d.finished(seq)
		}()
		if d.writeSeq.Load() != seq {
			return
		}
		if write(text) {
			return
		}
		d.Post(func() {
			if d.writeSeq.Load() != seq {
				return
			}
			s := d.Screen()
			s.SetClipboard([]byte(text))
			// OSC 52 has no acknowledgement; see TerminalLikely.
			if d.Unconfirmed != nil && !TerminalLikely(s) {
				d.Unconfirmed()
			}
		})
	})
}

// Copies reports how many times Copy has been called, so a host (or its test)
// can tell whether an action wrote the clipboard at all.
func (d *Dispatcher) Copies() uint64 { return d.writeSeq.Load() }

// Paste runs a Ctrl+V: it resolves the target and token now, reads the native
// clipboard on a background goroutine, and applies the text on the UI
// goroutine through Deliver. The read waits for every earlier Copy to finish
// writing, but no longer than pasteWait: past that a writer is stuck, and the
// read goes ahead on what the clipboard holds. With no native tool it asks the terminal
// (GetClipboard), whose reply arrives as an *tcell.EventClipboard for
// HandleEvent. Nothing happens when Resolve names no target.
func (d *Dispatcher) Paste() {
	target := d.Resolve()
	if target == nil {
		return
	}
	token := d.token()
	read := d.NativeRead
	if read == nil {
		read = Read
	}
	// Every Copy issued before this Ctrl+V must land first, or a slow
	// native writer (xclip, wl-copy, clip.exe over ssh) lets us read the
	// old contents. The wait is bounded (see defaultPasteWait), and never
	// takes writeMu, which a stuck write holds.
	seq := d.writeSeq.Load()
	wait := d.pasteWait
	if wait <= 0 {
		wait = defaultPasteWait
	}
	d.goroutine("reading the clipboard", func() {
		timer := time.NewTimer(wait)
		defer timer.Stop()
		d.waitCopied(seq, timer.C)
		text, ok := read()
		d.Post(func() {
			if ok {
				d.Deliver(target, token, text)
				return
			}
			d.pending, d.pendingToken = target, token
			d.Screen().GetClipboard()
		})
	})
}

// Deliver applies text read asynchronously to target, and only to target:
// it is dropped unless Resolve and Token still name what they did when the
// read started (target and token).
//
// Between Ctrl+V and the text arriving the user can have closed the dialog it
// was meant for. Re-resolving would land a paste aimed at a dialog's field in
// the editor behind it. Dropping it is right: the text is still on the
// clipboard. StripBOMs is applied.
func (d *Dispatcher) Deliver(target Target, token any, text string) {
	if target == nil || d.Resolve() != target || d.token() != token {
		return
	}
	target.Paste(StripBOMs(text))
	if d.Pasted != nil {
		d.Pasted(target)
	}
}

// HandleEvent takes the clipboard's share of the host's event stream, and
// reports whether ev was consumed and whether it needs no frame:
//
//   - *tcell.EventPaste start/end: a terminal bracketed paste (the terminal's
//     own Paste, or a middle-click). Between the markers every *tcell.EventKey
//     is pasted content, buffered in a PasteBuffer, and the end applies it to
//     the current target as one Paste (see PasteBuffer for why not replayed
//     keys). A buffered key reports skipFrame: a frame per key made a
//     10,001-line paste take minutes. Only keys skip one, so a lost end marker
//     never freezes the screen.
//   - *tcell.EventClipboard: the reply to Paste's GetClipboard, delivered to
//     what that Paste was aimed at. The pending target is cleared first, so an
//     unsolicited reply (the terminal answering a request never made) pastes
//     nothing anywhere.
//
// Anything else is not consumed. A host calls this before its own key
// handling.
func (d *Dispatcher) HandleEvent(ev tcell.Event) (consumed, skipFrame bool) {
	switch e := ev.(type) {
	case *tcell.EventKey:
		if d.bracket.Active() {
			d.bracket.Key(e)
			return true, true
		}
	case *tcell.EventPaste:
		if e.Start() {
			d.bracket.Begin()
			return true, false
		}
		// End strips BOMs already.
		if text := d.bracket.End(); text != "" {
			if t := d.Resolve(); t != nil {
				t.Paste(text)
				if d.Pasted != nil {
					d.Pasted(t)
				}
			}
		}
		return true, false
	case *tcell.EventClipboard:
		target, token := d.pending, d.pendingToken
		d.pending, d.pendingToken = nil, nil
		d.Deliver(target, token, string(e.Data()))
		return true, false
	}
	return false, false
}
