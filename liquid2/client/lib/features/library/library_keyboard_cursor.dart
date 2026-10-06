import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../app/providers.dart';
import 'library_hint_mode.dart';
import 'library_pane_focus.dart';

final libraryCursorProvider = NotifierProvider<LibraryCursorController, int>(
  LibraryCursorController.new,
);

class LibraryCursorController extends Notifier<int> {
  @override
  int build() => -1;

  void moveDown(int length) => _select(state < 0 ? 0 : state + 1, length);

  void moveUp(int length) => _select(state < 0 ? 0 : state - 1, length);

  void jumpToFirst(int length) => _select(0, length);

  void jumpToLast(int length) => _select(length - 1, length);

  void jumpTo(int index, int length) => _select(index, length);

  void clampTo(int length) {
    if (state >= length) _select(state, length);
  }

  void _select(int index, int length) {
    state = length == 0 ? -1 : index.clamp(0, length - 1);
  }
}

sealed class LibraryPendingMode {
  const LibraryPendingMode();
}

class LibraryPendingNone extends LibraryPendingMode {
  const LibraryPendingNone();
}

class LibraryPendingG extends LibraryPendingMode {
  const LibraryPendingG();
}

class LibraryPendingHint extends LibraryPendingMode {
  const LibraryPendingHint({
    required this.targets,
    required this.labels,
    this.input = '',
  });

  final List<LibraryHintTarget> targets;
  final List<String> labels;
  final String input;

  LibraryPendingHint extend(String character) => LibraryPendingHint(
    targets: targets,
    labels: labels,
    input: '$input$character',
  );

  // A label keeps the row it was handed at 'f' time: scrolling only moves it
  // or drops it, never re-points it at another document.
  LibraryPendingHint? realign(List<LibraryHintTarget> visible) {
    final rects = {for (final target in visible) target.index: target.rect};
    final nextTargets = <LibraryHintTarget>[];
    final nextLabels = <String>[];
    for (var i = 0; i < targets.length; i++) {
      final rect = rects[targets[i].index];
      if (rect == null) continue;
      nextTargets.add(LibraryHintTarget(index: targets[i].index, rect: rect));
      nextLabels.add(labels[i]);
    }
    if (nextTargets.isEmpty) return null;
    return LibraryPendingHint(
      targets: nextTargets,
      labels: nextLabels,
      input: input,
    );
  }

  int? indexOf(String label) {
    final slot = labels.indexOf(label);
    return slot < 0 ? null : targets[slot].index;
  }

  bool hasPrefix(String prefix) =>
      labels.any((label) => label.startsWith(prefix));
}

class LibraryCursorDownIntent extends Intent {
  const LibraryCursorDownIntent();
}

class LibraryCursorUpIntent extends Intent {
  const LibraryCursorUpIntent();
}

class LibraryCursorFirstIntent extends Intent {
  const LibraryCursorFirstIntent();
}

class LibraryCursorLastIntent extends Intent {
  const LibraryCursorLastIntent();
}

class LibraryCursorOpenIntent extends Intent {
  const LibraryCursorOpenIntent();
}

class LibraryHintIntent extends Intent {
  const LibraryHintIntent();
}

class LibraryFolderPaneIntent extends Intent {
  const LibraryFolderPaneIntent();
}

class LibraryDocumentPaneIntent extends Intent {
  const LibraryDocumentPaneIntent();
}

class LibraryCursorDownAction
    extends _LibraryCursorAction<LibraryCursorDownIntent> {
  LibraryCursorDownAction(super.onTrigger);
}

class LibraryCursorUpAction
    extends _LibraryCursorAction<LibraryCursorUpIntent> {
  LibraryCursorUpAction(super.onTrigger);
}

class LibraryCursorFirstAction
    extends _LibraryCursorAction<LibraryCursorFirstIntent> {
  LibraryCursorFirstAction(super.onTrigger);
}

class LibraryCursorLastAction
    extends _LibraryCursorAction<LibraryCursorLastIntent> {
  LibraryCursorLastAction(super.onTrigger);
}

class LibraryCursorOpenAction
    extends _LibraryCursorAction<LibraryCursorOpenIntent> {
  LibraryCursorOpenAction(super.onTrigger, {super.canTrigger});
}

class LibraryHintAction extends _LibraryCursorAction<LibraryHintIntent> {
  LibraryHintAction(super.onTrigger);
}

class LibraryFolderPaneAction
    extends _LibraryCursorAction<LibraryFolderPaneIntent> {
  LibraryFolderPaneAction(super.onTrigger, {super.canTrigger});
}

class LibraryDocumentPaneAction
    extends _LibraryCursorAction<LibraryDocumentPaneIntent> {
  LibraryDocumentPaneAction(super.onTrigger, {super.canTrigger});
}

class _LibraryCursorAction<T extends Intent> extends Action<T> {
  _LibraryCursorAction(this.onTrigger, {this.canTrigger});

  final VoidCallback onTrigger;
  final ValueGetter<bool>? canTrigger;

  @override
  bool isEnabled(T intent) =>
      !editableHasFocus() && (canTrigger?.call() ?? true);

  @override
  void invoke(T intent) => onTrigger();
}

// Typing in the search field must stay typing, so the bindings stand down
// whenever a text field owns the primary focus.
bool editableHasFocus() {
  final context = FocusManager.instance.primaryFocus?.context;
  return context?.findAncestorWidgetOfExactType<EditableText>() != null;
}

const _shortcuts = <ShortcutActivator, Intent>{
  SingleActivator(LogicalKeyboardKey.keyJ): LibraryCursorDownIntent(),
  SingleActivator(LogicalKeyboardKey.keyK): LibraryCursorUpIntent(),
  SingleActivator(LogicalKeyboardKey.keyG): LibraryCursorFirstIntent(),
  SingleActivator(LogicalKeyboardKey.keyG, shift: true):
      LibraryCursorLastIntent(),
  SingleActivator(LogicalKeyboardKey.keyF): LibraryHintIntent(),
  SingleActivator(LogicalKeyboardKey.keyH): LibraryFolderPaneIntent(),
  SingleActivator(LogicalKeyboardKey.keyL): LibraryDocumentPaneIntent(),
  SingleActivator(LogicalKeyboardKey.enter): LibraryCursorOpenIntent(),
};

class LibraryKeyboardCursor extends ConsumerStatefulWidget {
  const LibraryKeyboardCursor({
    required this.length,
    required this.onOpen,
    required this.hintTargets,
    required this.child,
    super.key,
  });

  final int length;
  final ValueChanged<int> onOpen;
  final ValueGetter<List<LibraryHintTarget>> hintTargets;
  final Widget child;

  @override
  ConsumerState<LibraryKeyboardCursor> createState() =>
      _LibraryKeyboardCursorState();
}

class _LibraryKeyboardCursorState extends ConsumerState<LibraryKeyboardCursor> {
  static const _pendingWindow = Duration(milliseconds: 700);

  final _focusNode = FocusNode(debugLabel: 'LibraryKeyboardCursor');
  LibraryPendingMode _pending = const LibraryPendingNone();
  Timer? _pendingTimer;
  var _hintSyncScheduled = false;
  var _folderPaneEnabled = false;

  @override
  void didChangeDependencies() {
    super.didChangeDependencies();
    _folderPaneEnabled = LibraryFolderPaneScope.of(context);
    if (_folderPaneEnabled) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      ref.read(libraryPaneProvider.notifier).activate(LibraryPane.documents);
    });
  }

  @override
  void dispose() {
    _pendingTimer?.cancel();
    _focusNode.dispose();
    super.dispose();
  }

  LibraryCursorController get _cursor =>
      ref.read(libraryCursorProvider.notifier);

  LibraryFolderCursorController get _folderCursor =>
      ref.read(libraryFolderCursorProvider.notifier);

  bool get _foldersActive =>
      _folderPaneEnabled &&
      ref.read(libraryPaneProvider) == LibraryPane.folders;

  int get _folderLength =>
      libraryFolderRowCount(ref.read(libraryFolderRowsProvider));

  void _moveDown() {
    _clearPending();
    if (_foldersActive) {
      _folderCursor.moveDown(_folderLength);
      return;
    }
    _cursor.moveDown(widget.length);
  }

  void _moveUp() {
    _clearPending();
    if (_foldersActive) {
      _folderCursor.moveUp(_folderLength);
      return;
    }
    _cursor.moveUp(widget.length);
  }

  void _jumpToLast() {
    _clearPending();
    if (_foldersActive) {
      _folderCursor.jumpToLast(_folderLength);
      return;
    }
    _cursor.jumpToLast(widget.length);
  }

  // Entering the folder pane parks the cursor on whatever folder is filtering
  // right now, so 'h' alone never looks like it moved the selection.
  void _activateFolderPane() {
    _clearPending();
    final rows = ref.read(libraryFolderRowsProvider);
    _folderCursor.jumpTo(
      libraryFolderRowOf(rows, ref.read(libraryFiltersProvider).folderId),
      libraryFolderRowCount(rows),
    );
    ref.read(libraryPaneProvider.notifier).activate(LibraryPane.folders);
  }

  void _activateDocumentPane() {
    _clearPending();
    ref.read(libraryPaneProvider.notifier).activate(LibraryPane.documents);
  }

  void _selectFolder() {
    final rows = ref.read(libraryFolderRowsProvider);
    final row = ref.read(libraryFolderCursorProvider);
    if (row > rows.length) return;
    ref
        .read(libraryFiltersProvider.notifier)
        .setFolder(libraryFolderIdAtRow(rows, row));
    ref.read(libraryPaneProvider.notifier).activate(LibraryPane.documents);
  }

  void _handleG() {
    if (_pending is! LibraryPendingG) {
      _setPending(const LibraryPendingG());
      _pendingTimer = Timer(_pendingWindow, _clearPending);
      return;
    }
    _clearPending();
    if (_foldersActive) {
      _folderCursor.jumpToFirst(_folderLength);
      return;
    }
    _cursor.jumpToFirst(widget.length);
  }

  void _openCursor() {
    _clearPending();
    if (_foldersActive) {
      _selectFolder();
      return;
    }
    final index = ref.read(libraryCursorProvider);
    if (index < 0 || index >= widget.length) return;
    widget.onOpen(index);
  }

  void _enterHint() {
    _clearPending();
    final targets = _visibleTargets();
    if (targets.isEmpty) return;
    _setPending(
      LibraryPendingHint(
        targets: targets,
        labels: buildHintLabels(targets.length),
      ),
    );
  }

  List<LibraryHintTarget> _visibleTargets() {
    final box = context.findRenderObject() as RenderBox?;
    if (box == null || !box.hasSize) return const [];
    final origin = box.localToGlobal(Offset.zero);
    return [
      for (final target in widget.hintTargets()) target.shift(-origin),
    ];
  }

  // Rows move only once the scrolled viewport has laid out again, so the
  // rects are re-read after the frame the notification belongs to.
  bool _handleScrollNotification(ScrollNotification notification) {
    if (_pending is! LibraryPendingHint || _hintSyncScheduled) return false;
    _hintSyncScheduled = true;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      _hintSyncScheduled = false;
      if (!mounted) return;
      _syncHint();
    });
    return false;
  }

  void _syncHint() {
    final pending = _pending;
    if (pending is! LibraryPendingHint) return;
    final next = pending.realign(_visibleTargets());
    if (next == null) {
      _clearPending();
      return;
    }
    _setPending(next);
  }

  void _setPending(LibraryPendingMode pending) {
    _pendingTimer?.cancel();
    _pendingTimer = null;
    if (_pending == pending) return;
    setState(() => _pending = pending);
  }

  void _clearPending() => _setPending(const LibraryPendingNone());

  KeyEventResult _handleHintKey(FocusNode node, KeyEvent event) {
    final pending = _pending;
    if (pending is! LibraryPendingHint) return KeyEventResult.ignored;
    if (event is KeyUpEvent) return KeyEventResult.ignored;
    if (editableHasFocus()) return KeyEventResult.ignored;
    if (event.logicalKey == LogicalKeyboardKey.escape) {
      _clearPending();
      _focusNode.requestFocus();
      return KeyEventResult.handled;
    }
    final character = _characterOf(event);
    if (character == null || !hintAlphabet.contains(character)) {
      _clearPending();
      _focusNode.requestFocus();
      return KeyEventResult.handled;
    }
    final next = pending.extend(character);
    final index = next.indexOf(next.input);
    if (index != null) {
      _clearPending();
      _focusNode.requestFocus();
      widget.onOpen(index);
      return KeyEventResult.handled;
    }
    if (next.hasPrefix(next.input)) {
      _setPending(next);
      return KeyEventResult.handled;
    }
    _clearPending();
    _focusNode.requestFocus();
    return KeyEventResult.handled;
  }

  KeyEventResult _handleUnboundKey(FocusNode node, KeyEvent event) {
    if (_pending is LibraryPendingG &&
        event is KeyDownEvent &&
        event.logicalKey != LogicalKeyboardKey.keyG) {
      _clearPending();
    }
    return KeyEventResult.ignored;
  }

  @override
  Widget build(BuildContext context) {
    final pending = _pending;
    return Focus(
      canRequestFocus: false,
      skipTraversal: true,
      onKeyEvent: _handleUnboundKey,
      child: Shortcuts(
        shortcuts: _shortcuts,
        child: Actions(
          actions: <Type, Action<Intent>>{
            LibraryCursorDownIntent: LibraryCursorDownAction(_moveDown),
            LibraryCursorUpIntent: LibraryCursorUpAction(_moveUp),
            LibraryCursorFirstIntent: LibraryCursorFirstAction(_handleG),
            LibraryCursorLastIntent: LibraryCursorLastAction(_jumpToLast),
            LibraryHintIntent: LibraryHintAction(_enterHint),
            LibraryFolderPaneIntent: LibraryFolderPaneAction(
              _activateFolderPane,
              canTrigger: () => _folderPaneEnabled,
            ),
            LibraryDocumentPaneIntent: LibraryDocumentPaneAction(
              _activateDocumentPane,
              canTrigger: () => _folderPaneEnabled,
            ),
            LibraryCursorOpenIntent: LibraryCursorOpenAction(
              _openCursor,
              canTrigger: () =>
                  _foldersActive || ref.read(libraryCursorProvider) >= 0,
            ),
          },
          child: Focus(
            focusNode: _focusNode,
            autofocus: true,
            onKeyEvent: _handleHintKey,
            child: Stack(
              children: [
                NotificationListener<ScrollNotification>(
                  onNotification: _handleScrollNotification,
                  child: widget.child,
                ),
                if (pending is LibraryPendingHint)
                  Positioned.fill(
                    child: LibraryHintLayer(
                      targets: pending.targets,
                      labels: pending.labels,
                      input: pending.input,
                    ),
                  ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}

String? _characterOf(KeyEvent event) {
  final character = event.character;
  if (character != null && character.length == 1) {
    return character.toLowerCase();
  }
  final label = event.logicalKey.keyLabel;
  return label.length == 1 ? label.toLowerCase() : null;
}
