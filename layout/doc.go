// Package layout provides Panel interface, PanelManager (tab/combo switcher),
// Splitter (draggable/keyboard-resizable divider between two regions) and
// DrawTitleBar (a panel's title row).
// These types are pure layout infrastructure; they have no dependency on any
// application-level code.
//
//   - panel.go         — the Panel interface and its optional companions
//     (Activatable, Dirty, Closable, Disposable)
//   - panel_manager.go — PanelManager
//   - splitter.go      — SplitterDir, Splitter
//   - title_bar.go     — DrawTitleBar, TitleBarRightRect: a panel's title row
package layout
