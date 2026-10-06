import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/features/document/document_detail_page.dart';
import 'package:liquid2_client/features/library/document_list_panel.dart';
import 'package:liquid2_client/features/library/document_list_tile_body.dart';
import 'package:liquid2_client/features/library/library_keyboard_cursor.dart';

import 'fake_library_repository.dart';
import 'test_viewports.dart';

void main() {
  testWidgets('j moves the cursor down and k moves it back up', (tester) async {
    await _pumpLibrary(tester);
    expect(_cursor(tester), -1);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_cursor(tester), 0);
    expect(_isCursorTile(tester, 'doc_1'), isTrue);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_cursor(tester), 1);
    expect(_isCursorTile(tester, 'doc_1'), isFalse);
    expect(_isCursorTile(tester, 'doc_2'), isTrue);

    await _sendKey(tester, LogicalKeyboardKey.keyK);
    expect(_cursor(tester), 0);

    await _sendKey(tester, LogicalKeyboardKey.keyK);
    expect(_cursor(tester), 0);
  });

  testWidgets('j stops at the last document', (tester) async {
    await _pumpLibrary(tester);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_cursor(tester), 1);
  });

  testWidgets('G jumps to the last document and gg returns to the first', (
    tester,
  ) async {
    await _pumpLibrary(tester);

    await tester.sendKeyDownEvent(LogicalKeyboardKey.shiftLeft);
    await _sendKey(tester, LogicalKeyboardKey.keyG);
    await tester.sendKeyUpEvent(LogicalKeyboardKey.shiftLeft);
    expect(_cursor(tester), 1);

    await _sendKey(tester, LogicalKeyboardKey.keyG);
    expect(_cursor(tester), 1);

    await _sendKey(tester, LogicalKeyboardKey.keyG);
    expect(_cursor(tester), 0);
  });

  testWidgets('an unbound key cancels a pending g', (tester) async {
    await _pumpLibrary(tester);
    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    await _sendKey(tester, LogicalKeyboardKey.keyJ);

    await _sendKey(tester, LogicalKeyboardKey.keyG);
    await _sendKey(tester, LogicalKeyboardKey.keyX);
    await _sendKey(tester, LogicalKeyboardKey.keyG);
    expect(_cursor(tester), 1);
  });

  testWidgets('enter opens the document under the cursor', (tester) async {
    await _pumpLibrary(tester);
    await _sendKey(tester, LogicalKeyboardKey.keyJ);

    await _sendKey(tester, LogicalKeyboardKey.enter);
    expect(find.byType(DocumentDetailPage), findsOneWidget);
  });

  testWidgets('j does not move the cursor while the search field has focus', (
    tester,
  ) async {
    await _pumpLibrary(tester);

    await tester.tap(find.byKey(const Key('library-search-field')));
    await tester.pumpAndSettle();
    expect(
      LibraryCursorDownAction(() {}).isEnabled(const LibraryCursorDownIntent()),
      isFalse,
    );

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_cursor(tester), -1);
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
  await tester.tap(find.text('Load more'));
  await tester.pumpAndSettle();
}

Future<void> _sendKey(WidgetTester tester, LogicalKeyboardKey key) async {
  await tester.sendKeyEvent(key);
  await tester.pumpAndSettle();
}

bool _isCursorTile(WidgetTester tester, String documentId) {
  return tester
      .widget<DocumentTileSurface>(find.byKey(Key('document-tile-$documentId')))
      .isCursor;
}

int _cursor(WidgetTester tester) {
  return ProviderScope.containerOf(
    tester.element(find.byType(DocumentListPanel)),
    listen: false,
  ).read(libraryCursorProvider);
}
