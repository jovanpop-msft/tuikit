package widgets

import (
	"strconv"

	"github.com/gdamore/tcell/v3"
	"github.com/radix29/tuikit/core"
)

// ProgressBarLabelWidth is the columns DrawProgressBar keeps right of the bar
// for its percentage: a space and "100%".
const ProgressBarLabelWidth = 5

// DrawProgressBar draws a determinate progress bar w columns wide at (x, y):
// a run of '█' for the done part and '░' for the rest, then " NN%" in the
// last ProgressBarLabelWidth columns. pct is clamped to 100; a negative pct
// means indeterminate — an all-shaded bar and no percentage — for a task that
// has not reported progress yet. Too narrow for a one-cell bar
// (w <= ProgressBarLabelWidth), it draws nothing.
//
// Like Spinner it is stateless and never takes focus, so it is a function, not
// a widget value: the host redraws it from its own progress state.
func DrawProgressBar(s tcell.Screen, x, y, w, pct int, st tcell.Style) {
	barW := w - ProgressBarLabelWidth
	if barW < 1 {
		return
	}
	pct = min(pct, 100)
	filled := 0
	if pct > 0 {
		filled = barW * pct / 100
	}
	for i := range barW {
		ch := '░'
		if i < filled {
			ch = '█'
		}
		core.PutRune(s, x+i, y, ch, st)
	}
	if pct >= 0 {
		core.DrawText(s, x+barW+1, y, st, strconv.Itoa(pct)+"%")
	}
}
