import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';

import 'fake_library_repository.dart';

void main() {
  testWidgets('creates a bookmark from the ingest screen', (tester) async {
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
    expect(find.text('Bookmark'), findsOneWidget);

    await tester.enterText(
      find.byType(TextField).first,
      'https://example.com/a',
    );
    await tester.tap(find.text('Create'));
    await tester.pumpAndSettle();

    expect(
      repository.createdDocuments,
      contains('bookmark:https://example.com/a'),
    );
    expect(find.text('Document'), findsOneWidget);
  });
}
