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
  testWidgets('shows document folder path in list and detail', (tester) async {
    await setDesktopViewport(tester);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          libraryRepositoryProvider.overrideWithValue(_FolderPathRepository()),
        ],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('Inbox / Research'), findsOneWidget);
    await tester.tap(find.text('Foldered document'));
    await tester.pumpAndSettle();
    expect(find.textContaining('Inbox / Research'), findsOneWidget);
  });
}

class _FolderPathRepository extends FakeLibraryRepository {
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
        ..folderPath.addAll(_path()),
    );
  }

  DocumentSummary _summary() {
    return DocumentSummary(
      (b) => b
        ..id = 'doc_1'
        ..title = 'Foldered document'
        ..kind = 'bookmark'
        ..status = 'unread'
        ..createdAt = _now
        ..updatedAt = _now
        ..folderPath.addAll(_path()),
    );
  }

  DocumentMetadata _metadata() {
    return DocumentMetadata(
      (b) => b
        ..id = 'doc_1'
        ..title = 'Foldered document'
        ..kind = 'bookmark'
        ..status = 'unread'
        ..createdAt = _now
        ..updatedAt = _now,
    );
  }

  List<FolderBreadcrumb> _path() {
    return [
      FolderBreadcrumb(
        (b) => b
          ..id = 'folder_1'
          ..name = 'Inbox',
      ),
      FolderBreadcrumb(
        (b) => b
          ..id = 'folder_2'
          ..name = 'Research',
      ),
    ];
  }
}

const _now = 1760000000000;
