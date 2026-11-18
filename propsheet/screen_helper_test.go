package propsheet

import "github.com/gdamore/tcell/v3"

// withScreen gives p the screen a dialogs.Stack would (ModalDialog.SetScreen),
// for a test that shows a sheet without registering it.
func withScreen(p *PropertySheet, s tcell.Screen) *PropertySheet {
	p.SetScreen(s)
	return p
}
