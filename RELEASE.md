# Release notes

The current release, in brief. Detail and history are in
[CHANGELOG.md](CHANGELOG.md).

## v0.1.0

### New

- `clipboard` package: system clipboard and an async `Dispatcher`.
- `loop` package: `Poster`, `Latest`, `Runner`, `Tasks` for async UI work.
- Dialogs: `MessageDialog`, `TextDialog`, `ListDialog`, `LogDialog`, `FindReplaceDialog`, `KeyDiagnosticsDialog`; `ProgressDialog.Run`.
- Dialog helpers: `Lazy`/`Replace`/`Built`, `FieldRing`, `RingMouse`.
- Controls: `TreeModel`, `TabBar`, `TreeView.SetCloseButton`, `ChoiceItems`/`CheckItems`, `DataGrid` column alignment and Ctrl+click marks.
- Editor: `PrependLines`, `LogFeed`, current-line highlight, in-selection search, `CompletionItem.Soft`.
- Core: `Drag`, `OverlayGuard`, `Viewport`, `LinePane`, `FormatBytes`, `ParseClock`.
- Charts: `Pin`, readouts, axis callouts. Layout: collapsible `Splitter`, `NewStackSplitter`.
- Propsheet: `FilterIndex`, `PendingEdits`, `PreservingSelectRow`, `StaticBlock`.
- `examples/spindemo`.

### Fixes

- Disabled dialog buttons ignore clicks and focus.
- File dialog: Windows drive/UNC paths, symlinks, directory navigation.
- Editor: selection collapse on arrows, replace across line breaks, completion undo.
- Pasting skips leading blank lines and control characters.
- Page keys always move at least one row; empty dropdown does not open.
- `TreeView` keeps buttons over the title; `SetChildren` retires replaced nodes.
- Number parsing, grapheme boundaries, chart value and duration formatting.
- Clipboard `Paste` waits for an earlier `Copy`.

### Changes

- **Breaking:** dialog constructors and `ModalDialog.InitModal` no longer take a `tcell.Screen`; call `Stack.SetScreen` once.
- **Breaking:** `propsheet.NewPropertySheet(title)` loses its screen argument.
- `Stack.Sync` stacks dialogs by show order, not registration order.
