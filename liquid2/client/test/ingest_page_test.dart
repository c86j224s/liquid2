import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';

import 'fake_library_repository.dart';

void main() {
  testWidgets('saves a link from the ingest screen', (tester) async {
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Ingest'));
    await tester.pumpAndSettle();

    expect(find.text('Ingest'), findsOneWidget);
    expect(find.text('URL'), findsWidgets);
    expect(find.text('File'), findsOneWidget);
    expect(find.text('Link only'), findsOneWidget);
    expect(find.text('Save page'), findsOneWidget);
    expect(find.text('Title override (optional)'), findsOneWidget);

    await tester.enterText(
      find.byType(TextField).first,
      'https://example.com/a',
    );
    await tester.tap(find.text('Save link'));
    await tester.pumpAndSettle();

    expect(
      repository.createdDocuments,
      contains('bookmark:https://example.com/a'),
    );
    expect(find.text('Document'), findsOneWidget);
  });

  testWidgets('routes URL save page through scrape repository call', (
    tester,
  ) async {
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Ingest'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(TextField).first,
      'https://example.com/article',
    );

    await tester.tap(find.text('Save page'));
    await tester.pumpAndSettle();
    expect(find.text('Title override (optional)'), findsNothing);

    await tester.tap(find.widgetWithText(FilledButton, 'Save page'));
    await tester.pumpAndSettle();

    expect(
      repository.createdDocuments,
      contains('scrape:https://example.com/article'),
    );
    expect(
      repository.createdDocuments,
      isNot(contains('bookmark:https://example.com/article')),
    );
  });
}
