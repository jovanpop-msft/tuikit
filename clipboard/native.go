package clipboard

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// The OS-native clipboard, shelled out to the platform's clipboard tool (on
// Windows, the Win32 API: native_windows.go). It is meant as a host's primary
// clipboard path, with tcell's OSC 52 terminal clipboard as the fallback when
// no native tool is available (a bare SSH session) or an invocation fails.

// method holds the platform-specific copy/paste implementations chosen by
// detect. nil means no native tool was found.
type method struct {
	copy  func(text string) bool
	paste func() (string, bool)
}

var (
	methodOnce sync.Once
	resolved   *method
)

// win32Method is the Windows method, set by native_windows.go's init and nil
// in every other build.
var win32Method *method

// resolve detects and caches which native clipboard tool to use, once per
// process.
func resolve() *method {
	methodOnce.Do(func() {
		resolved = detect(exec.LookPath)
	})
	return resolved
}

// detect picks a method for runtime.GOOS using lookPath to probe for tools.
// lookPath is a parameter so it is unit-testable without real clipboard
// binaries.
func detect(lookPath func(string) (string, error)) *method {
	found := func(name string) bool {
		_, err := lookPath(name)
		return err == nil
	}
	switch runtime.GOOS {
	case "darwin":
		if found("pbcopy") && found("pbpaste") {
			return &method{
				copy:  func(text string) bool { return runIn(text, "pbcopy") },
				paste: func() (string, bool) { return runOut("pbpaste") },
			}
		}
	case "windows":
		// The Win32 API, not a tool: see native_windows.go.
		return win32Method
	default: // linux and other Unix-likes
		if os.Getenv("WAYLAND_DISPLAY") != "" && found("wl-copy") && found("wl-paste") {
			return &method{
				copy:  func(text string) bool { return runIn(text, "wl-copy") },
				paste: func() (string, bool) { return runOut("wl-paste", "--no-newline") },
			}
		}
		if found("xclip") {
			return &method{
				copy:  func(text string) bool { return runIn(text, "xclip", "-selection", "clipboard") },
				paste: func() (string, bool) { return runOut("xclip", "-selection", "clipboard", "-o") },
			}
		}
		if found("xsel") {
			return &method{
				copy:  func(text string) bool { return runIn(text, "xsel", "--clipboard", "--input") },
				paste: func() (string, bool) { return runOut("xsel", "--clipboard", "--output") },
			}
		}
	}
	return nil
}

// toolTimeout bounds a clipboard tool's run.
const toolTimeout = 2 * time.Second

// runIn runs name(args...) with text piped to stdin, for a copy command.
// Returns false if the tool isn't available or the invocation fails.
func runIn(text string, name string, args ...string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), toolTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run() == nil
}

// runOut runs name(args...) and returns its stdout verbatim, for a paste
// command. ok=false means the tool isn't available or failed, not a
// legitimately empty clipboard (ok=true, text==""). Output is byte-for-byte (no
// newline trimming) since xclip/xsel/pbpaste/wl-paste (--no-newline) emit the
// clipboard's exact bytes.
func runOut(name string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), toolTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

// Write puts text on the native OS clipboard through the first available
// platform tool (the Win32 API on Windows). false means none is available or
// this invocation failed: fall back to OSC 52 (tcell's Screen.SetClipboard).
//
// It blocks for as long as the tool runs (bounded, about two seconds), so call
// it off the UI goroutine.
func Write(text string) bool {
	m := resolve()
	if m == nil {
		return false
	}
	return m.copy(text)
}

// Read returns the native OS clipboard's text. ok=false means no tool is
// available or this invocation failed — fall back to tcell's
// Screen.GetClipboard and its later EventClipboard — not an empty clipboard,
// which is "", true.
//
// It blocks like Write; call it off the UI goroutine.
func Read() (string, bool) {
	m := resolve()
	if m == nil {
		return "", false
	}
	return m.paste()
}
