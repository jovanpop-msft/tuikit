// Package clipboard is a terminal application's system-clipboard back end:
// the native OS clipboard, the hint for whether tcell's OSC 52 fallback
// reached anything, the buffer for a bracketed paste, and the Dispatcher that
// wires them to a host's widgets and event loop.
//
//   - native.go         — Write/Read through the platform's clipboard tool
//     (pbcopy/pbpaste, wl-copy/wl-paste, xclip, xsel); Windows goes through
//     the Win32 API instead (native_windows.go)
//   - native_windows.go — CF_UNICODETEXT via user32/kernel32
//   - terminal.go       — TerminalLikely: did an OSC 52 write probably land
//   - paste.go          — PasteBuffer (a bracketed paste's keys as text) and
//     StripBOMs
//   - dispatcher.go     — Dispatcher: Copy (newest wins, OSC 52 fallback),
//     Paste (aimed once, dropped if its target changed), and HandleEvent for
//     a bracketed paste and the OSC 52 reply
//
// Write and Read are synchronous and spawn no goroutine (ARCHITECTURE.md §
// Threading): they shell out or wait on the Win32 clipboard lock, up to a
// couple of seconds. A host either calls them off its UI goroutine and posts
// the result back itself, or uses Dispatcher, which does exactly that through
// the Post and Go it is given. Which widget a paste lands in stays with the
// host (Dispatcher.Resolve). It imports only the standard library and tcell,
// nothing of tuikit.
package clipboard
