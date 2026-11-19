// Package clipboard is a terminal application's system-clipboard back end:
// the native OS clipboard, the hint for whether tcell's OSC 52 fallback
// reached anything, and the buffer for a bracketed paste.
//
//   - native.go         — Write/Read through the platform's clipboard tool
//     (pbcopy/pbpaste, wl-copy/wl-paste, xclip, xsel); Windows goes through
//     the Win32 API instead (native_windows.go)
//   - native_windows.go — CF_UNICODETEXT via user32/kernel32
//   - terminal.go       — TerminalLikely: did an OSC 52 write probably land
//   - paste.go          — PasteBuffer (a bracketed paste's keys as text) and
//     StripBOMs
//
// Everything here is synchronous and spawns no goroutine (ARCHITECTURE.md §
// Threading). Write and Read shell out or wait on the Win32 clipboard lock, up
// to a couple of seconds, so a host calls them off its UI goroutine and posts
// the result back; the OSC 52 fallback itself (tcell's SetClipboard and
// GetClipboard) and the choice of which widget a paste lands in stay with the
// host. It imports only the standard library and tcell, nothing of tuikit.
package clipboard
