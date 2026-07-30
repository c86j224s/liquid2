import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/features/document/document_detail_page.dart';

import 'fake_library_repository.dart';

void main() {
  testWidgets('edits a document title from the detail screen', (tester) async {
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const MaterialApp(home: DocumentDetailPage(id: 'doc_1')),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Edit title'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.widgetWithText(TextFormField, 'Title'),
      'Renamed note',
    );
    await tester.tap(find.widgetWithText(FilledButton, 'Save'));
    await tester.pumpAndSettle();

    expect(repository.documentTitle, 'Renamed note');
    expect(find.text('Renamed note'), findsOneWidget);
    expect(find.text('Updated title.'), findsOneWidget);
  });

  testWidgets('does not save an empty title', (tester) async {
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const MaterialApp(home: DocumentDetailPage(id: 'doc_1')),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Edit title'));
    await tester.pumpAndSettle();
    await tester.enterText(find.widgetWithText(TextFormField, 'Title'), ' ');
    await tester.tap(find.widgetWithText(FilledButton, 'Save'));
    await tester.pumpAndSettle();

    expect(repository.documentTitle, 'SQLite notes');
    expect(find.text('Title is required.'), findsOneWidget);
  });
}
