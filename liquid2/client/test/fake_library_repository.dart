import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/library_filters.dart';
import 'package:liquid2_client/data/library_repository.dart';
import 'package:liquid2_client/data/library_snapshot.dart';

import 'fake_folder_repository.dart';
import 'fake_ingestion_repository.dart';
import 'fake_library_notes_repository.dart';
import 'fake_translation_repository.dart';

class FakeLibraryRepository
    with
        FakeLibraryNotesRepository,
        FakeIngestionRepository,
        FakeTranslationRepository
    implements LibraryRepository {
  FakeLibraryRepository({
    this.hasSecondPage = false,
    this.includeChildFolder = false,
  });

  final bool hasSecondPage;
  final bool includeChildFolder;
  int? rating = 3;
  var read = false;
  var movedToTrash = false;
  String? movedToFolderId;
  var rescraped = false;
  var tagIds = <String>{'tag_go'};
  final requestedCursors = <String?>[];
  final requestedFilters = <LibraryFilters>[];

  @override
  Future<LibrarySnapshot> loadLibrary(
    LibraryFilters filters, {
    String? cursor,
  }) async {
    requestedCursors.add(cursor);
    requestedFilters.add(filters);
    final allDocs = [_summary('doc_1'), if (hasSecondPage) _summary('doc_2')];
    final matches = allDocs.where((doc) => _matches(doc, filters)).toList();
    final docs = cursor == 'page_2' ? matches.skip(1) : matches.take(1);
    return LibrarySnapshot(
      documents: docs.toList(),
      folders: [
        fakeFolder(
          'folder_1',
          'Inbox',
          children: includeChildFolder
              ? [fakeFolder('folder_child', 'Research', parentId: 'folder_1')]
              : const [],
        ),
        fakeFolder('folder_trash', 'Trash', systemRole: 'trash'),
      ],
      tags: _tags,
      totalCount: matches.length,
      nextCursor: cursor == null && matches.length > 1 ? 'page_2' : null,
    );
  }

  @override
  Future<DocumentDetail> getDocument(String id) async => _detail();

  @override
  Future<DocumentDetail> markRead(String documentId) async {
    read = true;
    return _detail();
  }

  @override
  Future<DocumentDetail> markUnread(String documentId) async {
    read = false;
    return _detail();
  }

  @override
  Future<DocumentDetail> moveDocumentToTrash(String documentId) async {
    movedToTrash = true;
    return _detail();
  }

  @override
  Future<DocumentDetail> moveDocumentToFolder(
    String documentId,
    String folderId,
  ) async {
    movedToFolderId = folderId;
    return _detail();
  }

  @override
  Future<DocumentDetail> rescrapeDocument(String documentId) async {
    rescraped = true;
    return _detail();
  }

  @override
  Future<DocumentDetail> setRating(String documentId, int? rating) async {
    this.rating = rating;
    return _detail();
  }

  @override
  Future<DocumentDetail> replaceTags(String documentId, List<String> tagIds) {
    this.tagIds = tagIds.toSet();
    return Future.value(_detail());
  }

  bool _matches(DocumentSummary doc, LibraryFilters filters) {
    final query = filters.query?.toLowerCase();
    return (query == null || doc.title.toLowerCase().contains(query)) &&
        (filters.read.apiValue == null ||
            doc.status == filters.read.apiValue) &&
        (filters.ratingMin == null ||
            (doc.rating ?? 0) >= filters.ratingMin!) &&
        (filters.tagSlug == null ||
            (doc.tags?.contains(filters.tagSlug) ?? false));
  }

  DocumentSummary _summary(String id) {
    return DocumentSummary(
      (b) => b
        ..id = id
        ..title = id == 'doc_1' ? 'SQLite notes' : 'Second document'
        ..kind = 'bookmark'
        ..status = read ? 'read' : 'unread'
        ..rating = rating
        ..folderId = 'folder_1'
        ..createdAt = _now
        ..updatedAt = _now
        ..tags.addAll(_assignedTags().map((tag) => tag.slug)),
    );
  }

  DocumentDetail _detail() {
    return DocumentDetail(
      (b) => b
        ..document.replace(_metadata())
        ..contents.add(_content())
        ..tags.addAll(_assignedTags()),
    );
  }

  DocumentMetadata _metadata() {
    return DocumentMetadata(
      (m) => m
        ..id = 'doc_1'
        ..title = 'SQLite notes'
        ..kind = 'bookmark'
        ..status = read ? 'read' : 'unread'
        ..rating = rating
        ..folderId = movedToFolderId ?? 'folder_1'
        ..createdAt = _now
        ..updatedAt = _now,
    );
  }

  DocumentContent _content() {
    return DocumentContent(
      (c) => c
        ..id = 'content_1'
        ..role = 'original'
        ..format = 'markdown'
        ..content = 'Stored document body',
    );
  }

  List<Tag> _assignedTags() {
    return _tags.where((tag) => tagIds.contains(tag.id)).toList();
  }
}

final _tags = [
  Tag(
    (b) => b
      ..id = 'tag_go'
      ..name = 'go'
      ..slug = 'go'
      ..createdAt = _now,
  ),
  Tag(
    (b) => b
      ..id = 'tag_sqlite'
      ..name = 'sqlite'
      ..slug = 'sqlite'
      ..createdAt = _now,
  ),
];

const _now = 1760000000000;
