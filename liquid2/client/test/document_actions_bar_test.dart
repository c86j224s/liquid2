import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/features/document/document_actions_bar.dart';

import 'fake_library_repository.dart';

void main() {
  testWidgets('shows source action when a document has a source URL', (
    tester,
  ) async {
    await tester.pumpWidget(_TestHost(document: _document(sourceUrl: _source)));

    expect(find.text('Open source'), findsOneWidget);
    expect(find.byIcon(Icons.open_in_new), findsOneWidget);
  });

  testWidgets('uses canonical URL as source action fallback', (tester) async {
    await tester.pumpWidget(
      _TestHost(document: _document(canonicalUrl: _canonical)),
    );

    expect(find.text('Open source'), findsOneWidget);
  });

  testWidgets('hides source action when a document has no URL', (tester) async {
    await tester.pumpWidget(_TestHost(document: _document()));

    expect(find.text('Open source'), findsNothing);
  });

  testWidgets('shows move action for non-RSS documents', (tester) async {
    await tester.pumpWidget(_TestHost(document: _document(kind: 'bookmark')));

    expect(find.widgetWithText(OutlinedButton, 'Move folder'), findsOneWidget);
  });

  testWidgets('hides move action for RSS items', (tester) async {
    await tester.pumpWidget(_TestHost(document: _document(kind: 'rss_item')));

    expect(find.widgetWithText(OutlinedButton, 'Move folder'), findsNothing);
  });

  testWidgets('re-scrapes URL-backed scraped documents', (tester) async {
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(
      _TestHost(
        document: _document(sourceUrl: _source),
        repository: repository,
      ),
    );

    await tester.tap(find.text('Re-scrape'));
    await tester.pumpAndSettle();

    expect(repository.rescraped, isTrue);
    expect(find.text('Re-scraped document.'), findsOneWidget);
  });

  testWidgets('hides re-scrape action for bookmarks', (tester) async {
    await tester.pumpWidget(
      _TestHost(
        document: _document(kind: 'bookmark', sourceUrl: _source),
      ),
    );

    expect(find.text('Re-scrape'), findsNothing);
  });

  testWidgets('shows unread documents with an outlined read action', (
    tester,
  ) async {
    await tester.pumpWidget(_TestHost(document: _document(status: 'unread')));

    expect(find.widgetWithText(OutlinedButton, 'Mark read'), findsOneWidget);
    expect(find.widgetWithText(FilledButton, 'Mark read'), findsNothing);
  });

  testWidgets('shows read documents with a filled read-state action', (
    tester,
  ) async {
    await tester.pumpWidget(_TestHost(document: _document(status: 'read')));

    expect(find.widgetWithText(FilledButton, 'Mark unread'), findsOneWidget);
    expect(find.widgetWithText(OutlinedButton, 'Mark unread'), findsNothing);
  });

  testWidgets('moves a document to trash through repository', (tester) async {
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(
      _TestHost(document: _document(), repository: repository),
    );

    await tester.tap(find.byTooltip('Move to trash'));
    await tester.pumpAndSettle();

    expect(repository.movedToTrash, isTrue);
  });
}

class _TestHost extends StatelessWidget {
  const _TestHost({required this.document, this.repository});

  final DocumentMetadata document;
  final FakeLibraryRepository? repository;

  @override
  Widget build(BuildContext context) {
    final router = GoRouter(
      routes: [
        GoRoute(
          path: '/',
          builder: (context, state) {
            return Scaffold(body: DocumentActionsBar(document: document));
          },
        ),
      ],
    );

    return ProviderScope(
      overrides: [
        if (repository != null)
          libraryRepositoryProvider.overrideWithValue(repository!),
      ],
      child: MaterialApp.router(routerConfig: router),
    );
  }
}

DocumentMetadata _document({
  String? sourceUrl,
  String? canonicalUrl,
  String kind = 'scraped_article',
  String status = 'unread',
}) {
  return DocumentMetadata(
    (b) => b
      ..id = 'doc_1'
      ..title = 'SQLite notes'
      ..kind = kind
      ..sourceUrl = sourceUrl
      ..canonicalUrl = canonicalUrl
      ..status = status
      ..createdAt = _now
      ..updatedAt = _now,
  );
}

const _source = 'https://source.example/article';
const _canonical = 'https://canonical.example/article';
const _now = 1760000000000;
