package dialogs

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// "Not there" is an answer, including a path through a file (ENOTDIR).
func TestLocalFileSystemExistsMissingIsNotAnError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{filepath.Join(dir, "missing"), filepath.Join(file, "child")} {
		exists, isDir, err := LocalFileSystem{}.Exists(p)
		if exists || isDir || err != nil {
			t.Errorf("Exists(%q) = (%v, %v, %v), want (false, false, nil)", p, exists, isDir, err)
		}
	}
}

// Any other Stat failure is "couldn't ask" and must reach the caller: the
// save-overwrite guard prompts on it, and swallowing it skipped the prompt.
func TestLocalFileSystemExistsReportsPermissionError(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores directory permissions")
	}
	dir := t.TempDir()
	locked := filepath.Join(dir, "locked")
	if err := os.Mkdir(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o700) })

	exists, _, err := LocalFileSystem{}.Exists(filepath.Join(locked, "x"))
	if err == nil {
		t.Fatalf("Exists under an unreadable directory = (%v, nil), want an error", exists)
	}
}

// On Windows the local filesystem has a drive-list level above the drive roots,
// as WindowsPathRules does for a remote host; elsewhere it is unchanged.
func TestLocalFileSystemDriveList(t *testing.T) {
	var lfs LocalFileSystem
	if runtime.GOOS != "windows" {
		if got := lfs.Parent("/"); got != "/" {
			t.Errorf("Parent(/) = %q, want /", got)
		}
		if got := lfs.Clean(""); got == "" {
			t.Error(`Clean("") must stay the working directory off Windows`)
		}
		return
	}
	if got := lfs.Parent(`C:\`); got != "" {
		t.Errorf(`Parent(C:\) = %q, want ""`, got)
	}
	if got := lfs.Clean(""); got != "" {
		t.Errorf(`Clean("") = %q, want ""`, got)
	}
	drives, err := lfs.List("")
	if err != nil || len(drives) == 0 {
		t.Fatalf(`List("") = %v, %v; want at least one drive`, drives, err)
	}
	for _, d := range drives {
		if !d.IsDir || len(d.Name) != 2 || d.Name[1] != ':' {
			t.Errorf("drive entry %+v is not a bare drive like C:", d)
		}
		if got := lfs.Clean(lfs.Join("", d.Name)); got != d.Name+`\` {
			t.Errorf("entering %q gives %q, want its root", d.Name, got)
		}
	}
}

// symlinkFixture holds real/, link -> real, dangling -> missing and
// flink -> file.txt; it skips when the platform refuses symlinks.
func symlinkFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "real"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{"zlink": "real", "dangling": "missing", "flink": "file.txt"} {
		if err := os.Symlink(target, filepath.Join(dir, name)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	return dir
}

func TestLocalFileSystemListSymlinkToDirIsDir(t *testing.T) {
	dir := symlinkFixture(t)
	entries, err := LocalFileSystem{}.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]FileEntry{}
	for _, e := range entries {
		got[e.Name] = e
	}
	if !got["zlink"].IsDir {
		t.Error("link to a directory listed as a file")
	}
	if got["dangling"].IsDir || got["flink"].IsDir {
		t.Error("dangling link or link to a file listed as a directory")
	}
	if got["flink"].Size != 1 {
		t.Errorf("flink size = %d, want the target's 1", got["flink"].Size)
	}
}

func TestFileDialogSymlinkedDirBehavesAsDir(t *testing.T) {
	dir := symlinkFixture(t)
	d := NewFileDialog()
	d.loadDir(dir)
	// "..", real, zlink, then files: the link sorts among the dirs.
	if d.entries[1].Name != "real" || d.entries[2].Name != "zlink" || !d.entries[2].IsDir {
		t.Fatalf("entries = %v, want zlink sorted among the dirs", d.entries)
	}
	d.nameField.SetValue("")
	d.pathField.SetValue("z")
	if !d.completeField(d.pathField, true) {
		t.Error("dirsOnly completion does not offer the linked directory")
	}
	d.sel = 2
	d.activateSelected()
	if d.dir != filepath.Join(dir, "zlink") {
		t.Errorf("dir = %q, want navigation into the link", d.dir)
	}
}
