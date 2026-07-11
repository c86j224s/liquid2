import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/features/document/document_detail_page.dart';

import 'fake_library_repository.dart';

void main() {
  testWidgets('ignores stale translation job refresh results', (tester) async {
    final repository = _StaleRefreshRepository();
    await tester.pumpWidget(_app(repository));
    await tester.pumpAndSettle();

    final targetField = await _showTargetField(tester);
    await tester.enterText(targetField, 'ko');
    await tester.tap(find.widgetWithText(FilledButton, 'Translate'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    await tester.pump(const Duration(seconds: 2));
    expect(repository.requestedJobIds, ['job_old']);

    await tester.enterText(targetField, 'ja');
    await tester.pump();
    await tester.tap(find.widgetWithText(FilledButton, 'Translate'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));
    expect(repository.enqueuedJobIds, ['job_old', 'job_new']);

    repository.oldRefresh.complete(_job(id: 'job_old', status: 'completed'));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 100));

    expect(find.text('translate document · completed'), findsNothing);
    expect(find.text('translate document · queued'), findsOneWidget);
  });
}

Widget _app(FakeLibraryRepository repository) {
  return ProviderScope(
    overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
    child: const MaterialApp(home: DocumentDetailPage(id: 'doc_1')),
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

class _StaleRefreshRepository extends FakeLibraryRepository {
  final oldRefresh = Completer<Job>();
  final enqueuedJobIds = <String>[];

  @override
  Future<Job> translateDocument({
    required String documentId,
    required String sourceContentId,
    required String targetLanguage,
  }) async {
    final id = enqueuedJobIds.isEmpty ? 'job_old' : 'job_new';
    enqueuedJobIds.add(id);
    return _job(id: id, status: 'queued');
  }

  @override
  Future<Job> getJob(String id) {
    requestedJobIds.add(id);
    return oldRefresh.future;
  }
}

Job _job({required String id, required String status}) {
  return Job(
    (b) => b
      ..id = id
      ..kind = 'translate_document'
      ..status = status
      ..error = null
      ..attempts = 0
      ..createdAt = _now
      ..updatedAt = _now,
  );
}

const _now = 1760000000000;
