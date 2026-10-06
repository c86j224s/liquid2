import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../app/providers.dart';
import '../../data/folder_tree.dart';

enum LibraryPane { documents, folders }

final libraryPaneProvider =
    NotifierProvider<LibraryPaneController, LibraryPane>(
      LibraryPaneController.new,
    );

class LibraryPaneController extends Notifier<LibraryPane> {
  @override
  LibraryPane build() => LibraryPane.documents;

  void activate(LibraryPane pane) => state = pane;
}

final libraryFolderCursorProvider =
    NotifierProvider<LibraryFolderCursorController, int>(
      LibraryFolderCursorController.new,
    );

class LibraryFolderCursorController extends Notifier<int> {
  @override
  int build() => 0;

  void moveDown(int length) => _select(state + 1, length);

  void moveUp(int length) => _select(state - 1, length);

  void jumpToFirst(int length) => _select(0, length);

  void jumpToLast(int length) => _select(length - 1, length);

  void jumpTo(int index, int length) => _select(index, length);

  void _select(int index, int length) {
    state = length == 0 ? 0 : index.clamp(0, length - 1);
  }
}

// The tree rows the folder cursor walks: row 0 is 'All documents', row i + 1
// is the flattened folder at i.
final libraryFolderRowsProvider = Provider<List<FolderTreeItem>>((ref) {
  final snapshot = ref.watch(librarySnapshotProvider).value;
  return snapshot == null ? const [] : flattenFolderTree(snapshot.folders);
});

int libraryFolderRowCount(List<FolderTreeItem> rows) => rows.length + 1;

String? libraryFolderIdAtRow(List<FolderTreeItem> rows, int row) =>
    row <= 0 || row > rows.length ? null : rows[row - 1].folder.id;

int libraryFolderRowOf(List<FolderTreeItem> rows, String? folderId) {
  if (folderId == null) return 0;
  final index = rows.indexWhere((item) => item.folder.id == folderId);
  return index < 0 ? 0 : index + 1;
}

// Only the desktop layout keeps the folder tree on screen beside the list, so
// the pane bindings stand down wherever this scope reports false.
class LibraryFolderPaneScope extends InheritedWidget {
  const LibraryFolderPaneScope({
    required this.enabled,
    required super.child,
    super.key,
  });

  final bool enabled;

  static bool of(BuildContext context) =>
      context
          .dependOnInheritedWidgetOfExactType<LibraryFolderPaneScope>()
          ?.enabled ??
      false;

  @override
  bool updateShouldNotify(LibraryFolderPaneScope oldWidget) =>
      enabled != oldWidget.enabled;
}
