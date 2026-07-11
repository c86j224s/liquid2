import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/data/library_filters.dart';
import 'package:liquid2_client/data/library_snapshot.dart';

import 'fake_library_repository.dart';
import 'test_viewports.dart';

void main() {
  testWidgets('shows feed item published time in list and detail', (
    tester,
  ) async {
    await setDesktopViewport(tester);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          libraryRepositoryProvider.overrideWithValue(_PublishedAtRepository()),
        ],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Published'), findsOneWidget);
    await tester.tap(find.text('Feed article'));
    await tester.pumpAndSettle();
    expect(find.textContaining('Published'), findsOneWidget);
  });
}

class _PublishedAtRepository extends FakeLibraryRepository {
  @override
  Future<LibrarySnapshot> loadLibrary(
    LibraryFilters filters, {
    String? cursor,
  }) async {
    return LibrarySnapshot(
      documents: [_summary()],
      folders: const [],
      tags: const [],
      totalCount: 1,
    );
  }

  @override
  Future<DocumentDetail> getDocument(String id) async {
    return DocumentDetail(
      (b) => b
        ..document.replace(_metadata())
        ..contents.add(
          DocumentContent(
            (c) => c
              ..id = 'content_1'
              ..role = 'original'
              ..format = 'markdown'
              ..content = 'Feed body',
          ),
        ),
    );
  }

  DocumentSummary _summary() {
    return DocumentSummary(
      (b) => b
        ..id = 'doc_1'
        ..title = 'Feed article'
        ..kind = 'rss_item'
        ..status = 'unread'
        ..createdAt = _now
        ..updatedAt = _now
        ..publishedAt = _publishedAt,
    );
  }

  DocumentMetadata _metadata() {
    return DocumentMetadata(
      (b) => b
        ..id = 'doc_1'
        ..title = 'Feed article'
        ..kind = 'rss_item'
        ..status = 'unread'
        ..createdAt = _now
        ..updatedAt = _now
        ..publishedAt = _publishedAt,
    );
  }
}

const _now = 1760000000000;
const _publishedAt = 1759990000000;
