import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/data/library_filters.dart';
import 'package:liquid2_client/data/library_snapshot.dart';
import 'package:liquid2_client/data/tag_repository.dart';
import 'package:liquid2_client/features/document/document_tag_editor.dart';
import 'package:liquid2_client/features/document/document_tag_providers.dart';

import 'fake_library_repository.dart';

void main() {
  testWidgets('creates and assigns a new tag from the document detail', (
    tester,
  ) async {
    final library = _TaggableLibraryRepository();
    final tags = _FakeTagRepository(library);

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          libraryRepositoryProvider.overrideWithValue(library),
          tagRepositoryProvider.overrideWithValue(tags),
        ],
        child: const MaterialApp(
          home: Scaffold(
            body: DocumentTagEditor(documentId: _documentID, assigned: []),
          ),
        ),
      ),
    );
    await tester.pumpAndSettle();

    await tester.enterText(find.byKey(const Key('document-tag-input')), ' Go ');
    await tester.tap(find.byTooltip('Create tag'));
    await tester.pumpAndSettle();

    expect(tags.createdNames, ['Go']);
    expect(library.assignedIds, contains('tag_new_1'));
    expect(find.text('Go'), findsOneWidget);
  });
}

class _FakeTagRepository implements TagRepository {
  _FakeTagRepository(this.library);

  final _TaggableLibraryRepository library;
  final createdNames = <String>[];

  @override
  Future<Tag> createTag(String name) async {
    final trimmed = name.trim();
    createdNames.add(trimmed);
    final tag = _tag('tag_new_${createdNames.length}', trimmed);
    library.tags.add(tag);
    return tag;
  }
}

class _TaggableLibraryRepository extends FakeLibraryRepository {
  final tags = <Tag>[];
  var assignedIds = <String>{};

  @override
  Future<LibrarySnapshot> loadLibrary(
    LibraryFilters filters, {
    String? cursor,
  }) async {
    return LibrarySnapshot(
      documents: const [],
      folders: const [],
      tags: tags,
      totalCount: 0,
    );
  }

  @override
  Future<DocumentDetail> getDocument(String id) async => _detail();

  @override
  Future<DocumentDetail> replaceTags(String documentId, List<String> tagIds) {
    assignedIds = tagIds.toSet();
    return Future.value(_detail());
  }

  DocumentDetail _detail() {
    return DocumentDetail(
      (b) => b
        ..document.replace(_metadata())
        ..tags.addAll(tags.where((tag) => assignedIds.contains(tag.id))),
    );
  }
}

DocumentMetadata _metadata() {
  return DocumentMetadata(
    (b) => b
      ..id = _documentID
      ..title = 'SQLite notes'
      ..kind = 'bookmark'
      ..status = 'unread'
      ..createdAt = _now
      ..updatedAt = _now,
  );
}

Tag _tag(String id, String name) {
  return Tag(
    (b) => b
      ..id = id
      ..name = name
      ..slug = name.toLowerCase()
      ..createdAt = _now,
  );
}

const _documentID = 'doc_1';
const _now = 1760000000000;
