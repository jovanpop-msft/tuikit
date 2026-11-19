// Package controls provides higher-level, reusable TUI controls:
//
//   - MenuBar / ContextMenu — application menu bar (menu_bar.go) and floating
//     right-click popup menu (context_menu.go); their shared MenuItem/Menu types
//     live in menu_item.go, and every menu box (root and submenus: draw, scroll,
//     hit-test, keys) in menu_box.go; ClipboardMenuItems builds the standard
//     Cut/Copy/Paste/Select All menu for a text field (clipboard_menu.go);
//     ChoiceItems a pick-one selector's menu (choice_items.go) and CheckItems a
//     pick-many checklist whose KeepOpen rows tick in place (check_items.go)
//   - Toolbar — row of icon-only buttons with hover tooltips (toolbar.go)
//   - ToolRow — a panel's one-row text toolbar: layout, hit-testing, the
//     "More ▾" overflow and the gated Run, drawn by the host (tool_row.go); it hides cells by the
//     same policy as Toolbar (hideToFit, common.go)
//   - TabBar — one-row strip of plain tabs with an optional right-aligned
//     cell; the host owns the labels and the active index (tab_bar.go)
//   - TreeView — collapsible/expandable tree with generic node data (treeview.go)
//   - ListBox — scrollable single-column list of strings (listbox.go)
//   - DataGrid — scrollable, column-aligned tabular data display, split across
//     datagrid.go, datagrid_draw.go, datagrid_input.go and datagrid_overlay.go;
//     a host's column sort shows its order through PermutedSource, swaps it in
//     with SetSourceReordered and offers SortMenuItems (datagrid_sort.go)
//   - ValueViewer — DataGrid's "Show Value" popup (value_viewer.go), a read-only
//     editor.Editor in a box
//
// The text editor is its own package, controls/editor.
//
// Controls depend on core, theme and controls/editor, not on any application types.
// The application layer passes data in and reads state out; controls never call
// back into the application directly — they fire callbacks (func values) that the
// caller wires up.
package controls
