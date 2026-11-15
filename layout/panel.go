package layout

import "github.com/gdamore/tcell/v3"

// ---------------------------------------------------------------------------
// Panel interface
// ---------------------------------------------------------------------------

// Panel is the contract that every right-hand side panel must satisfy.
// SetBounds is called by the layout manager whenever the available space
// changes.  Title is shown in the tab bar.
type Panel interface {
	SetBounds(x, y, w, h int)
	Draw(s tcell.Screen)
	HandleKey(ev *tcell.EventKey) bool
	HandleMouse(ev *tcell.EventMouse) bool
	Title() string
}

// Activatable is an optional interface a Panel may implement to be told when it
// becomes (true) or stops being (false) the active panel. PanelManager calls
// SetActive on every implementer whenever the active panel changes.
type Activatable interface {
	SetActive(active bool)
}

// Dirty is an optional interface a Panel may implement to report unsaved
// changes; PanelManager marks such a panel's tab with a trailing "*".
type Dirty interface {
	Dirty() bool
}

// Closable is an optional interface a Panel may implement to forbid the tab
// bar's [x] button (e.g. a panel the host keeps open for good). Absence means
// closable.
type Closable interface {
	Closable() bool
}

// Disposable is an optional interface a Panel may implement to release what it
// owns (an in-flight read, a collector goroutine, a connection) on close.
// PanelManager never calls it: the host does, just before RemovePanel, so every
// panel type is disposed through one check rather than a per-type list.
type Disposable interface {
	Close()
}

// OverlayPanel is an optional interface a Panel implements when it can show
// something floating outside a child's rect (a grid's cell menu or value
// popup, a completion list) — core.Overlay's OverlayActive, without
// DrawOverlay, since a panel draws its own overlays in Draw. While it reports
// true, PanelManager.OverlayActive does too and PanelManager.HandleMouse hands
// the panel every mouse event wherever it lands, so a press outside the panel
// still reaches the overlay and dismisses it (core.MouseHandler). The panel's
// HandleMouse must offer such an event to the overlay before any positional
// routing of its own.
type OverlayPanel interface {
	OverlayActive() bool
}
