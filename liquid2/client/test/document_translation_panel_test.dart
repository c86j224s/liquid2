import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/features/document/document_detail_page.dart';

import 'document_translation_test_support.dart';
import 'fake_library_repository.dart';

void main() {
  testWidgets('enqueues translation and refreshes completed job', (
    tester,
  ) async {
    final repository = CompletingTranslationRepository();
    await tester.pumpWidget(_app(repository));
    await tester.pumpAndSettle();

    expect(find.text('Stored document body'), findsOneWidget);
    expect(find.text('Translated body'), findsNothing);

    final targetField = await _showTargetField(tester);
    await tester.enterText(targetField, 'KO');
    await tester.tap(find.widgetWithText(FilledButton, 'Translate'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    expect(repository.translateRequests.single.targetLanguage, 'ko');
    expect(find.text('translate document · queued'), findsOneWidget);

    await tester.pump(const Duration(seconds: 2));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    expect(repository.requestedJobIds, ['job_translate_1']);
    expect(find.text('translate document · completed'), findsOneWidget);
    expect(find.text('Translated body'), findsOneWidget);
  });

  testWidgets('validates target language before enqueue', (tester) async {
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(_app(repository));
    await tester.pumpAndSettle();

    final targetField = await _showTargetField(tester);
    await tester.enterText(targetField, '??');
    await tester.tap(find.widgetWithText(FilledButton, 'Translate'));
    await tester.pump();

    expect(find.text('Invalid language'), findsOneWidget);
    expect(repository.translateRequests, isEmpty);
  });

  testWidgets('keeps queued translation visible after reopening detail', (
    tester,
  ) async {
    final repository = FakeLibraryRepository();
    final container = ProviderContainer(
      overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
    );
    addTearDown(container.dispose);

    await _pumpWithContainer(
      tester,
      container,
      const DocumentDetailPage(id: 'doc_1'),
    );
    await tester.pumpAndSettle();

    final targetField = await _showTargetField(tester);
    await tester.enterText(targetField, 'ko');
    await tester.tap(find.widgetWithText(FilledButton, 'Translate'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    expect(find.text('translate document · queued'), findsOneWidget);

    await _pumpWithContainer(tester, container, const SizedBox.shrink());
    await tester.pump();
    await _pumpWithContainer(
      tester,
      container,
      const DocumentDetailPage(id: 'doc_1'),
    );
    await tester.pumpAndSettle();
    await _showTargetField(tester);

    expect(find.text('translate document · queued'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Translating'), findsOneWidget);

    await tester.pump(const Duration(seconds: 2));
    await tester.pump();

    expect(repository.requestedJobIds, ['job_translate_1']);
  });

  testWidgets('shows a friendly message when translation is already running', (
    tester,
  ) async {
    final repository = ConflictTranslationRepository();
    await tester.pumpWidget(_app(repository));
    await tester.pumpAndSettle();

    final targetField = await _showTargetField(tester);
    await tester.enterText(targetField, 'ko');
    await tester.tap(find.widgetWithText(FilledButton, 'Translate'));
    await tester.pump();

    expect(
      find.text(
        'Translation is already queued or running. The result will appear here when the current job completes.',
      ),
      findsWidgets,
    );
    expect(find.textContaining('Exception'), findsNothing);
  });
}

Widget _app(FakeLibraryRepository repository) {
  return ProviderScope(
    overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
    child: const MaterialApp(home: DocumentDetailPage(id: 'doc_1')),
  );
}

Future<void> _pumpWithContainer(
  WidgetTester tester,
  ProviderContainer container,
  Widget child,
) {
  return tester.pumpWidget(
    UncontrolledProviderScope(
      container: container,
      child: MaterialApp(home: child),
    ),
  );
}

Future<Finder> _showTargetField(WidgetTester tester) async {
  final field = find.byKey(const Key('translation-target-language'));
  for (var attempt = 0; attempt < 6 && field.evaluate().isEmpty; attempt++) {
    await tester.drag(find.byType(ListView), const Offset(0, -260));
    await tester.pumpAndSettle();
  }
  expect(field, findsOneWidget);
  await tester.ensureVisible(field);
  await tester.pumpAndSettle();
  return field;
}
