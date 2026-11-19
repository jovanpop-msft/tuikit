package propsheet

import (
	"testing"

	"github.com/gdamore/tcell/v3"
)

// Under tcell.OptAdvancedKeys, Ctrl+Z arrives as KeyRune 'z'+ModCtrl. The
// sheet folds it itself (core.NormalizeKey), so the revert still fires on a
// host that doesn't.
func TestSheetCtrlZInAdvancedKeyFormReverts(t *testing.T) {
	p, row := dirtySheet(t)

	if !p.HandleKey(tcell.NewEventKeyEx(tcell.KeyRune, "z", tcell.ModCtrl, true, 0, 1)) {
		t.Fatal("advanced Ctrl+Z was not consumed by the sheet")
	}
	if got := row.Value(); got != "srv01" {
		t.Errorf("row value after advanced Ctrl+Z = %q, want the loaded %q", got, "srv01")
	}
}
