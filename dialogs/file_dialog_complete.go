package dialogs

import (
	"strings"
	"unicode"

	"github.com/radix29/tuikit/widgets"
)

// ---------------------------------------------------------------------------
// Tab completion
// ---------------------------------------------------------------------------

// completeField extends f's value with the longest prefix shared by every
// directory entry (dirsOnly, for the path field) or every entry (the name field)
// matching what's typed after the last separator: shell-style Tab completion.
// Returns false, leaving f untouched, when there's nothing to complete, so the
// caller falls through to Tab focus-cycling.
//
// On a FileSystem that FoldsCase the match ignores case and the completion takes
// the on-disk casing, so "health" becomes "HealthClinic_full.bak" and "backup"
// becomes "Backup\". An exact byte match there found no candidate for "health",
// and the caller's fall-through then threw focus out of the field mid-word.
func (d *FileDialog) completeField(f *widgets.InputField, dirsOnly bool) bool {
	fs := d.FileSystem()
	fold := fs.FoldsCase()
	val := f.Value()
	dirPart, base := fs.Split(val)
	searchDir := d.dir
	if dirPart != "" {
		if fs.IsAbs(dirPart) {
			searchDir = fs.Clean(dirPart)
		} else {
			searchDir = fs.Join(d.dir, dirPart)
		}
	}
	// Complete against the directory already on screen, not a second listing: this
	// runs on every Tab, a network round trip on a remote filesystem. d.entries
	// carries the dialog's own ".." row, which fs.List never reports and which would
	// poison the common prefix. On a folding host "c:\backup" is that directory too.
	infos := d.entries
	if searchDir != d.dir && !(fold && strings.EqualFold(searchDir, d.dir)) {
		var err error
		if infos, err = fs.List(searchDir); err != nil {
			return false
		}
	}
	var candidates []FileEntry
	for _, e := range infos {
		if e.Name == ".." {
			continue
		}
		if dirsOnly && !e.IsDir {
			continue
		}
		if hasNamePrefix(e.Name, base, fold) {
			candidates = append(candidates, e)
		}
	}
	if len(candidates) == 0 {
		return false
	}
	names := make([]string, len(candidates))
	for i, e := range candidates {
		names[i] = e.Name
	}
	common := commonPrefix(names, fold)
	if common == "" {
		return false
	}
	completed := dirPart + common
	// The listing already says whether the one match is a directory; an Exists
	// probe would be another round trip that blocks the event loop on a remote host.
	if len(candidates) == 1 && candidates[0].IsDir {
		completed += fs.Separator()
	}
	// Compared with the whole value, not common with base: on a folding host
	// "backup" -> "Backup\" changes only case and the separator, and is still a
	// completion.
	if completed == val {
		return false
	}
	f.SetValue(completed)
	return true
}

// hasNamePrefix reports whether name starts with prefix, ignoring case when
// fold. It compares the first len([]rune(prefix)) runes rather than lowering
// both strings: case folding can change a string's byte length.
func hasNamePrefix(name, prefix string, fold bool) bool {
	if !fold {
		return strings.HasPrefix(name, prefix)
	}
	n, p := []rune(name), []rune(prefix)
	return len(n) >= len(p) && strings.EqualFold(string(n[:len(p)]), prefix)
}

// commonPrefix returns the longest string every element of strs starts with
// (strs is never empty), ignoring case when fold, in strs[0]'s casing. It
// compares rune by rune: a byte prefix could end mid-rune on multi-byte
// filenames, and InputField.SetValue's []rune(v) turns that invalid tail into a
// stray replacement character.
func commonPrefix(strs []string, fold bool) string {
	prefix := []rune(strs[0])
	for _, s := range strs[1:] {
		r := []rune(s)
		n := min(len(r), len(prefix))
		i := 0
		for i < n && (prefix[i] == r[i] || fold && runesEqualFold(prefix[i], r[i])) {
			i++
		}
		prefix = prefix[:i]
	}
	return string(prefix)
}

// runesEqualFold reports whether a and b are in the same unicode.SimpleFold
// orbit, the equivalence strings.EqualFold uses.
func runesEqualFold(a, b rune) bool {
	for r := unicode.SimpleFold(a); r != a; r = unicode.SimpleFold(r) {
		if r == b {
			return true
		}
	}
	return a == b
}
