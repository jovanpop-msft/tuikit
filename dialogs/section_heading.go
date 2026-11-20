package dialogs

import (
	"github.com/gdamore/tcell/v3"

	"github.com/radix29/tuikit/core"
	"github.com/radix29/tuikit/theme"
)

// DrawSectionHeading draws a section heading inside a dialog: title in bold
// dialog style on row y, clipped to w, and a dialog-line rule w wide on row
// y+1. It returns y+2, the first row free for the section's content.
//
// Every "title over a rule" heading goes through here, so a dialog's own
// headings can't drift from propsheet's Section rows and page titles (they
// once did: Restore's were drawn not bold). A heading set into the rule on one
// row is a different shape and is not this.
func DrawSectionHeading(s tcell.Screen, x, y, w int, title string) int {
	core.DrawTextClipped(s, x, y, w, theme.StyleDialog().Bold(true), title)
	core.DrawHLine(s, x, y+1, w, theme.StyleDialogLine())
	return y + 2
}
