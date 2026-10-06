import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/data/library_filters.dart';
import 'package:liquid2_client/features/library/folder_tree_view.dart';
import 'package:liquid2_client/features/library/library_keyboard_cursor.dart';
import 'package:liquid2_client/features/library/library_page.dart';
import 'package:liquid2_client/features/library/library_pane_focus.dart';

import 'fake_library_repository.dart';
import 'test_viewports.dart';

void main() {
  testWidgets('h activates the folder pane and j moves only the folder cursor',
      (tester) async {
    await _pumpLibrary(tester);
    expect(_pane(tester), LibraryPane.documents);

    await _sendKey(tester, LogicalKeyboardKey.keyH);
    expect(_pane(tester), LibraryPane.folders);
    expect(_folderCursor(tester), 0);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_folderCursor(tester), 1);
    expect(_documentCursor(tester), -1);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_folderCursor(tester), 2);
    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_folderCursor(tester), 2);

    await _sendKey(tester, LogicalKeyboardKey.keyK);
    expect(_folderCursor(tester), 1);
    expect(_documentCursor(tester), -1);
  });

  testWidgets('the folder cursor is drawn apart from the selected folder',
      (tester) async {
    await _pumpLibrary(tester);
    await _sendKey(tester, LogicalKeyboardKey.keyH);
    await _sendKey(tester, LogicalKeyboardKey.keyJ);

    expect(_row(tester, 0).selected, isTrue);
    expect(_row(tester, 0).isCursor, isFalse);
    expect(_row(tester, 1).selected, isFalse);
    expect(_row(tester, 1).isCursor, isTrue);
  });

  testWidgets('enter applies the folder and hands the pane back to documents',
      (tester) async {
    await _pumpLibrary(tester);
    await _sendKey(tester, LogicalKeyboardKey.keyH);
    await _sendKey(tester, LogicalKeyboardKey.keyJ);

    await _sendKey(tester, LogicalKeyboardKey.enter);
    expect(_filters(tester).folderId, 'folder_1');
    expect(_pane(tester), LibraryPane.documents);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_documentCursor(tester), 0);
    expect(_folderCursor(tester), 1);
  });

  testWidgets('l returns to the document pane without changing the folder',
      (tester) async {
    await _pumpLibrary(tester);
    await _sendKey(tester, LogicalKeyboardKey.keyH);
    await _sendKey(tester, LogicalKeyboardKey.keyJ);

    await _sendKey(tester, LogicalKeyboardKey.keyL);
    expect(_pane(tester), LibraryPane.documents);
    expect(_filters(tester).folderId, isNull);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_documentCursor(tester), 0);
    expect(_folderCursor(tester), 1);
  });

  testWidgets('G jumps to the last folder row and gg returns to the first',
      (tester) async {
    await _pumpLibrary(tester);
    await _sendKey(tester, LogicalKeyboardKey.keyH);

    await tester.sendKeyDownEvent(LogicalKeyboardKey.shiftLeft);
    await _sendKey(tester, LogicalKeyboardKey.keyG);
    await tester.sendKeyUpEvent(LogicalKeyboardKey.shiftLeft);
    expect(_folderCursor(tester), 2);

    await _sendKey(tester, LogicalKeyboardKey.keyG);
    expect(_folderCursor(tester), 2);

    await _sendKey(tester, LogicalKeyboardKey.keyG);
    expect(_folderCursor(tester), 0);
    expect(_documentCursor(tester), -1);
  });

  testWidgets('the folder pane stays reachable when no documents match',
      (tester) async {
    await _pumpLibrary(tester);
    _container(tester).read(libraryFiltersProvider.notifier).setQuery('zzzz');
    await tester.pumpAndSettle();
    expect(find.text('No documents match the current filters.'), findsOneWidget);

    await _sendKey(tester, LogicalKeyboardKey.keyH);
    expect(_pane(tester), LibraryPane.folders);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_folderCursor(tester), 1);
  });

  testWidgets('a focused text field suppresses the folder pane bindings',
      (tester) async {
    await _pumpLibrary(tester);

    await tester.tap(find.byKey(const Key('library-search-field')));
    await tester.pumpAndSettle();
    expect(
      LibraryFolderPaneAction(() {}).isEnabled(const LibraryFolderPaneIntent()),
      isFalse,
    );

    await _sendKey(tester, LogicalKeyboardKey.keyH);
    expect(_pane(tester), LibraryPane.documents);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_folderCursor(tester), 0);
    expect(_documentCursor(tester), -1);
  });
}

Future<void> _pumpLibrary(WidgetTester tester) async {
  await setDesktopViewport(tester);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        libraryRepositoryProvider.overrideWithValue(
          FakeLibraryRepository(hasSecondPage: true),
        ),
      ],
      child: const Liquid2App(),
    ),
  );
  await tester.pumpAndSettle();
}

Future<void> _sendKey(WidgetTester tester, LogicalKeyboardKey key) async {
  await tester.sendKeyEvent(key);
  await tester.pumpAndSettle();
}

ProviderContainer _container(WidgetTester tester) {
  return ProviderScope.containerOf(
    tester.element(find.byType(LibraryPage)),
    listen: false,
  );
}

LibraryPane _pane(WidgetTester tester) =>
    _container(tester).read(libraryPaneProvider);

int _folderCursor(WidgetTester tester) =>
    _container(tester).read(libraryFolderCursorProvider);

int _documentCursor(WidgetTester tester) =>
    _container(tester).read(libraryCursorProvider);

LibraryFilters _filters(WidgetTester tester) =>
    _container(tester).read(libraryFiltersProvider);

FolderTreeRow _row(WidgetTester tester, int index) =>
    tester.widget<FolderTreeRow>(find.byKey(Key('folder-tree-row-$index')));
