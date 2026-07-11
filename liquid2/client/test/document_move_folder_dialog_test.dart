import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/features/document/document_actions_bar.dart';

import 'fake_folder_repository.dart';
import 'fake_library_repository.dart';

void main() {
  testWidgets('moves a document to a selected non-system folder', (
    tester,
  ) async {
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(
      _Host(
        document: _document(folderId: 'folder_1'),
        repository: repository,
        folders: [
          fakeFolder('folder_1', 'Inbox', systemRole: 'inbox'),
          fakeFolder('folder_2', 'Projects'),
        ],
      ),
    );

    await tester.tap(find.widgetWithText(OutlinedButton, 'Move folder'));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownButtonFormField<String>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('Projects').last);
    await tester.pumpAndSettle();
    await tester.tap(find.widgetWithText(FilledButton, 'Move'));
    await tester.pumpAndSettle();

    expect(repository.movedToFolderId, 'folder_2');
    expect(find.text('Moved document.'), findsOneWidget);
  });

  testWidgets('excludes feeds and trash branches from move choices', (
    tester,
  ) async {
    await tester.pumpWidget(
      _Host(
        document: _document(),
        folders: [
          fakeFolder('folder_1', 'Inbox', systemRole: 'inbox'),
          fakeFolder(
            'folder_feeds',
            'Feeds',
            systemRole: 'feeds',
            children: [fakeFolder('folder_feed_child', 'Feed child')],
          ),
          fakeFolder(
            'folder_trash',
            'Trash',
            systemRole: 'trash',
            children: [fakeFolder('folder_trash_child', 'Trash child')],
          ),
        ],
      ),
    );

    await tester.tap(find.widgetWithText(OutlinedButton, 'Move folder'));
    await tester.pumpAndSettle();
    await tester.tap(find.byType(DropdownButtonFormField<String>));
    await tester.pumpAndSettle();

    expect(find.text('Inbox'), findsWidgets);
    expect(find.text('Feeds'), findsNothing);
    expect(find.text('Feed child'), findsNothing);
    expect(find.text('Trash'), findsNothing);
    expect(find.text('Trash child'), findsNothing);
  });
}

class _Host extends StatelessWidget {
  const _Host({required this.document, required this.folders, this.repository});

  final DocumentMetadata document;
  final List<Folder> folders;
  final FakeLibraryRepository? repository;

  @override
  Widget build(BuildContext context) {
    return ProviderScope(
      overrides: [
        libraryRepositoryProvider.overrideWithValue(
          repository ?? FakeLibraryRepository(),
        ),
        folderRepositoryProvider.overrideWithValue(
          FakeFolderRepository(folders: folders),
        ),
      ],
      child: MaterialApp(
        home: Scaffold(body: DocumentActionsBar(document: document)),
      ),
    );
  }
}

DocumentMetadata _document({String? folderId}) {
  return DocumentMetadata(
    (b) => b
      ..id = 'doc_1'
      ..title = 'SQLite notes'
      ..kind = 'scraped_article'
      ..status = 'unread'
      ..folderId = folderId
      ..createdAt = _now
      ..updatedAt = _now,
  );
}

const _now = 1760000000000;
