# Changelog

All notable changes to tuikit are documented here, newest first, from
`v0.1.0` onward. `RELEASE.md` carries the current release only, as a short
summary of the entry below it.

## Unreleased

### Migrating

- `DataGrid.SetSourcePreservingView`, `SetDataPreservingView`,
  `SetSourceKeepingSelection` and `SetSourceReordered` carry column state by
  header text, not by index: dragged widths, alignments, the sort indicator,
  the cursor's column and a block's columns stay on the column they were on
  when one is inserted or removed ahead of it (the n-th occurrence of a
  repeated header pairs with the n-th). A renamed column starts at its
  defaults, and a column that is gone takes its state with it: the cursor
  moves to the nearest surviving column to its left, and a block spanning a
  removed or newly inserted column collapses. A host that guarded a changed
  column set with `SetSource` can drop the guard. Row state is unchanged.

## v0.1.0

The first release with a stable shape for the dialog stack. Dialogs no longer
take a screen at construction: `dialogs.Stack` hands every registered dialog
its screen, so a host builds dialogs lazily and registers them on demand.
The release also adds the `clipboard` and `loop` packages, a `TreeModel`, a
drag-and-drop carrier, chart pins and readouts, a dozen ready-made dialogs and
the property-sheet building blocks goSSMS had been keeping to itself.
**Breaking** (one change, below).

### Migrating

- Dialog constructors lose their `tcell.Screen` argument:
  `NewAlertDialog`, `NewConfirmDialog`, `NewFileDialog`, `NewProgressDialog`,
  `NewPromptDialog`, `NewPropertiesDialog`, `NewTypedConfirmDialog` and
  `propsheet.NewPropertySheet(title)`. Call `Stack.SetScreen(screen)` once;
  it passes the screen to every dialog registered, including those registered
  later.
- `ModalDialog.InitModal(title, w, h)` loses its screen too. A custom dialog
  embedding `ModalDialog` drops the argument; one that read the screen after
  construction gets it from `ModalDialog.SetScreen`/`Screen()` once the stack
  has set it (nil before then, for a dialog built without a stack).
- `Stack.Sync` now stacks newly opened dialogs by the order they were shown
  (`ModalDialog.ShowOrder`), not the order they were registered.

### New

- **`clipboard` package.** `Write`/`Read` through pbcopy, wl-copy, xclip or
  xsel, and the Win32 `CF_UNICODETEXT` path on Windows (the library's one
  GOOS-suffixed file). `Dispatcher` runs Copy/Paste off the UI goroutine:
  newest copy wins, a paste waits for an earlier copy, bracketed paste arrives
  as one edit, the OSC 52 reply is handled. `PasteBuffer`, `StripBOMs`,
  `TerminalLikely`.
- **`loop` package.** `Poster` (post from any goroutine, one coalesced wake
  per burst, `Drain`, `Close`), `Latest` (latest-only loads), `Runner`
  (`Go`/`GoRepair`/`GoOnce`/`FanOut`/`Tick`, panics reported not fatal) and
  `Tasks`/`Task` for non-modal background work.
- **Dialogs:** `MessageDialog`, `TextDialog`, `ListDialog`, `LogDialog`,
  `FindReplaceDialog`, `KeyDiagnosticsDialog`; `ProgressDialog.Run` owns a
  background job, its cancel and its report; `DrawTaskProgress`,
  `DrawSectionHeading`; `Lazy`, `Replace` and `Built` build dialogs on demand;
  `FieldRing`, `RingMouse` and the focus-ring helpers for flat-ring dialogs;
  `PropertiesDialog` gains cursor navigation, selection and clipboard.
- **Editor:** `PrependLines`, `LogFeed` (a read-only log's buffer),
  current-line highlight, `FindAtCursor`, `SetReplacement`,
  `SelectedLineRange`, search that works inside a selection and keeps its
  match across edits, hanging-indent wrap, `CompletionItem.Soft`
  (suggestion mode: Tab commits, Enter is a newline) and extended completion
  edits.
- **Controls:** `TreeModel`/`TreeItem`/`TreePlaceholder` (a lazily loaded
  tree with stable node IDs), `TreeView.SetCloseButton`, `TabBar` (a one-row
  strip with a right-aligned cell), `ToolRow` gate functions and overflow
  menu, `ChoiceItems`/`CheckItems`/`SortMenuItems`, toggle menu items and
  toolbar buttons, `ContextMenu.SetRefresh`, `DataGrid` column alignment and
  auto-align, Ctrl+click row marks that survive a sort, and
  `SetSourceReordered`/`PermutedSource` for a re-sort that keeps selection.
- **Core:** `Drag[P]` (a payload carried between regions, with a ghost),
  `OverlayGuard`, `OpenOverlay`, `RouteOverlay`, `Viewport` (scroll offset,
  bars, wheel, `ScrollKey`), `LinePane`, `WrapHanging`, `SplitColumns`,
  `NormalizeKey`/`IsCtrlEnter`, `MouseEventName`, `FormatBytes`,
  `FormatClock`/`ParseClock`, `ParseFormattedNumber`, `MatchesAny`,
  `TruncateLeft`.
- **Charts:** `Pin`, `PlotHit`, `DrawReadout`/`PlaceReadout`,
  `DrawAxisCallout`, `DrawSectionBar`, `BarLegendItems`.
- **Layout:** `Splitter` collapse (`SetCollapsed`), `NewStackSplitter`,
  `DrawTitleBar`/`TitleBarRightRect`.
- **Propsheet:** `FilterIndex`, `PendingEdits`, `PreservingSelectRow`,
  `StaticBlock`, `BindGridDetail`, `BindCellToggle`/`NewCellToggleGrid`,
  `GridHolder`, `Form.Grids`.
- **Widgets/theme:** `DrawProgressBar`, `Spinner.Interval`; title and dialog
  highlight/warning styles. `examples/spindemo` animates every spinner.

### Fixes

- Dialogs: disabled buttons no longer take clicks or focus; a file dialog
  handles Windows drive and UNC paths, symlinks and directory navigation.
- Editor: arrow keys collapse a selection; replace handles line breaks and
  keeps match state; completion text normalises EOLs and tabs and undoes in
  one step.
- Input field: pasting skips leading blank lines and no longer carries
  invisible control characters into the value.
- DataGrid, ListBox, TreeView: page keys move at least one row;
  `SetSourceReordered` reads a nil permutation as the identity and is faster
  on large marked sets.
- TreeView: a left key on a child selects its parent; `SetChildren` retires
  replaced nodes; the title and its buttons share one layout, so a narrow
  tree truncates the title, never a button.
- Dropdown: an empty list does not open.
- PanelManager: tab scrolling keeps the active tab visible; press gestures
  reach the right panel.
- Charts: `FormatValue` handles zero and small values; sub-second durations
  format correctly on the axis.
- Strings: grapheme boundaries, rune-column bounds and number parsing
  (rejects malformed grouping, accepts commas correctly).
- Clipboard: `Paste` waits for an earlier `Copy`; a stuck copy no longer
  blocks paste.

### Changes

- Dialog constructors and `InitModal` are screen-free (see Migrating).
- `Stack.Sync` orders dialogs by show order.
- `TreeView` title row: buttons always win the top border over the title.
- `README.md` and `ARCHITECTURE.md` document the `clipboard` and `loop`
  packages, the threading protocol and the new dialogs.
