import 'package:flutter/gestures.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/features/document/document_detail_page.dart';
import 'package:liquid2_client/features/library/document_list_panel.dart';
import 'package:liquid2_client/features/library/library_hint_mode.dart';
import 'package:liquid2_client/features/library/library_keyboard_cursor.dart';

import 'fake_library_repository.dart';
import 'test_viewports.dart';

void main() {
  testWidgets('f labels every visible document row', (tester) async {
    await _pumpLibrary(tester);
    expect(find.byType(LibraryHintLabel), findsNothing);

    await _sendKey(tester, LogicalKeyboardKey.keyF);
    expect(find.byType(LibraryHintLabel), findsNWidgets(2));
    expect(find.text('A'), findsOneWidget);
    expect(find.text('S'), findsOneWidget);
  });

  testWidgets('typing a hint label opens that document', (tester) async {
    await _pumpLibrary(tester);

    await _sendKey(tester, LogicalKeyboardKey.keyF);
    await _sendKey(tester, LogicalKeyboardKey.keyS);
    expect(find.byType(DocumentDetailPage), findsOneWidget);
  });

  testWidgets('escape dismisses the labels and j still moves the cursor', (
    tester,
  ) async {
    await _pumpLibrary(tester);

    await _sendKey(tester, LogicalKeyboardKey.keyF);
    expect(find.byType(LibraryHintLabel), findsNWidgets(2));

    await _sendKey(tester, LogicalKeyboardKey.escape);
    expect(find.byType(LibraryHintLabel), findsNothing);
    expect(find.byType(DocumentDetailPage), findsNothing);
    expect(_cursor(tester), -1);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_cursor(tester), 0);

    await _sendKey(tester, LogicalKeyboardKey.keyJ);
    expect(_cursor(tester), 1);
  });

  testWidgets('a key matching no label dismisses hint mode', (tester) async {
    await _pumpLibrary(tester);

    await _sendKey(tester, LogicalKeyboardKey.keyF);
    await _sendKey(tester, LogicalKeyboardKey.keyZ);
    expect(find.byType(LibraryHintLabel), findsNothing);
    expect(find.byType(DocumentDetailPage), findsNothing);
  });

  testWidgets('scrolling keeps a label on its row', (tester) async {
    await _pumpLibrary(tester, extraDocuments: 10);

    await _sendKey(tester, LogicalKeyboardKey.keyF);
    final before = _rowRect(tester, 'doc_2');
    expect(_labelRect(tester, 'S').center.dy, closeTo(before.center.dy, 0.5));

    await _scrollList(tester, 200);
    final after = _rowRect(tester, 'doc_2');
    expect(after.top, lessThan(before.top));
    expect(_labelRect(tester, 'S').center.dy, closeTo(after.center.dy, 0.5));
  });

  testWidgets('a row scrolled out of the viewport loses its label', (
    tester,
  ) async {
    await _pumpLibrary(tester, extraDocuments: 10);

    await _sendKey(tester, LogicalKeyboardKey.keyF);
    final labelled = find.byType(LibraryHintLabel).evaluate().length;
    expect(_label('A'), findsOneWidget);

    await _scrollList(tester, 200);
    expect(find.byKey(const Key('document-tile-doc_1')), findsNothing);
    expect(_label('A'), findsNothing);
    expect(find.byType(LibraryHintLabel), findsNWidgets(labelled - 1));
    expect(_label('S'), findsOneWidget);

    await _sendKey(tester, LogicalKeyboardKey.keyA);
    expect(find.byType(DocumentDetailPage), findsNothing);
    expect(find.byType(LibraryHintLabel), findsNothing);
  });

  testWidgets('after scrolling a label still opens its own document', (
    tester,
  ) async {
    await _pumpLibrary(tester, extraDocuments: 10);

    await _sendKey(tester, LogicalKeyboardKey.keyF);
    await _scrollList(tester, 200);
    expect(_label('D'), findsOneWidget);

    await _sendKey(tester, LogicalKeyboardKey.keyD);
    expect(
      tester.widget<DocumentDetailPage>(find.byType(DocumentDetailPage)).id,
      'doc_3',
    );
  });

  testWidgets('f does not open hint mode while the search field has focus', (
    tester,
  ) async {
    await _pumpLibrary(tester);

    await tester.tap(find.byKey(const Key('library-search-field')));
    await tester.pumpAndSettle();
    expect(
      LibraryHintAction(() {}).isEnabled(const LibraryHintIntent()),
      isFalse,
    );

    await _sendKey(tester, LogicalKeyboardKey.keyF);
    expect(find.byType(LibraryHintLabel), findsNothing);

    await tester.enterText(find.byKey(const Key('library-search-field')), 'f');
    await tester.pumpAndSettle();
    expect(
      tester
          .widget<TextField>(find.byKey(const Key('library-search-field')))
          .controller
          ?.text,
      'f',
    );
    expect(find.byType(LibraryHintLabel), findsNothing);
  });
}

Future<void> _pumpLibrary(WidgetTester tester, {int extraDocuments = 0}) async {
  await setDesktopViewport(tester);
  await tester.pumpWidget(
    ProviderScope(
      overrides: [
        libraryRepositoryProvider.overrideWithValue(
          FakeLibraryRepository(
            hasSecondPage: true,
            extraDocuments: extraDocuments,
          ),
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

Future<void> _scrollList(WidgetTester tester, double delta) async {
  final list = find.descendant(
    of: find.byType(DocumentListPanel),
    matching: find.byType(ListView),
  );
  final pointer = TestPointer(1, PointerDeviceKind.mouse);
  await tester.sendEventToBinding(pointer.hover(tester.getCenter(list)));
  await tester.sendEventToBinding(pointer.scroll(Offset(0, delta)));
  await tester.pumpAndSettle();
}

Finder _label(String label) => find.ancestor(
  of: find.text(label),
  matching: find.byType(LibraryHintLabel),
);

Rect _labelRect(WidgetTester tester, String label) =>
    tester.getRect(_label(label));

Rect _rowRect(WidgetTester tester, String documentId) =>
    tester.getRect(find.byKey(Key('document-tile-$documentId')));

int _cursor(WidgetTester tester) {
  return ProviderScope.containerOf(
    tester.element(find.byType(DocumentListPanel)),
    listen: false,
  ).read(libraryCursorProvider);
}
