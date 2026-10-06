import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/features/document/document_detail_page.dart';
import 'package:liquid2_client/features/document/document_sibling_navigation.dart';
import 'package:liquid2_client/features/library/document_list_panel.dart';
import 'package:liquid2_client/features/library/document_list_tile_body.dart';
import 'package:liquid2_client/features/library/library_keyboard_cursor.dart';

import 'fake_library_repository.dart';
import 'test_viewports.dart';

void main() {
  testWidgets('J opens the next document and K returns to the previous', (
    tester,
  ) async {
    await _pumpDetail(tester);
    expect(_openDocumentId(tester), 'doc_1');

    await _sendShiftKey(tester, LogicalKeyboardKey.keyJ);
    expect(_openDocumentId(tester), 'doc_2');

    await _sendShiftKey(tester, LogicalKeyboardKey.keyK);
    expect(_openDocumentId(tester), 'doc_1');
  });

  testWidgets('J stops on the last document and K stops on the first', (
    tester,
  ) async {
    await _pumpDetail(tester);

    await _sendShiftKey(tester, LogicalKeyboardKey.keyK);
    expect(_openDocumentId(tester), 'doc_1');

    await _sendShiftKey(tester, LogicalKeyboardKey.keyJ);
    await _sendShiftKey(tester, LogicalKeyboardKey.keyJ);
    expect(_openDocumentId(tester), 'doc_2');
  });

  testWidgets('returning to the library keeps the cursor on the last document', (
    tester,
  ) async {
    await _pumpDetail(tester);
    await _sendShiftKey(tester, LogicalKeyboardKey.keyJ);
    expect(_openDocumentId(tester), 'doc_2');

    await tester.binding.handlePopRoute();
    await tester.pumpAndSettle();

    expect(find.byType(DocumentDetailPage), findsNothing);
    expect(_cursor(tester), 1);
    expect(_isCursorTile(tester, 'doc_2'), isTrue);
  });

  testWidgets('J keeps the router uri on the document it opened', (
    tester,
  ) async {
    await _pumpDetail(tester);
    expect(_routerUri(tester), '/documents/doc_1');

    await _sendShiftKey(tester, LogicalKeyboardKey.keyJ);
    expect(_openDocumentId(tester), 'doc_2');
    expect(_routerUri(tester), '/documents/doc_2');

    await _sendShiftKey(tester, LogicalKeyboardKey.keyK);
    expect(_routerUri(tester), '/documents/doc_1');
  });

  testWidgets('J stands down while a text field has focus', (tester) async {
    await _pumpDetail(tester);

    await tester.tap(find.byKey(const Key('document-tag-input')));
    await tester.pumpAndSettle();
    expect(
      DocumentNextAction(() {}).isEnabled(const DocumentNextIntent()),
      isFalse,
    );

    await _sendShiftKey(tester, LogicalKeyboardKey.keyJ);
    expect(_openDocumentId(tester), 'doc_1');
  });
}

Future<void> _pumpDetail(WidgetTester tester) async {
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
  await tester.sendKeyEvent(LogicalKeyboardKey.keyJ);
  await tester.pumpAndSettle();
  await tester.sendKeyEvent(LogicalKeyboardKey.enter);
  await tester.pumpAndSettle();
}

Future<void> _sendShiftKey(WidgetTester tester, LogicalKeyboardKey key) async {
  await tester.sendKeyDownEvent(LogicalKeyboardKey.shiftLeft);
  await tester.sendKeyEvent(key);
  await tester.sendKeyUpEvent(LogicalKeyboardKey.shiftLeft);
  await tester.pumpAndSettle();
}

String _openDocumentId(WidgetTester tester) {
  return tester.widget<DocumentDetailPage>(find.byType(DocumentDetailPage)).id;
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

String _routerUri(WidgetTester tester) {
  return GoRouter.of(
    tester.element(find.byType(DocumentDetailPage)),
  ).routerDelegate.currentConfiguration.uri.toString();
}
