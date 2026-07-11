import 'package:built_collection/built_collection.dart';
import 'package:dio/dio.dart';
import 'package:liquid2_api/liquid2_api.dart';

import 'library_filters.dart';
import 'library_repository.dart';
import 'library_snapshot.dart';

part 'api_library_document_actions.dart';

class ApiLibraryRepository
    with ApiLibraryDocumentActions
    implements LibraryRepository {
  const ApiLibraryRepository(this.api);

  @override
  final Liquid2Api api;

  @override
  Future<LibrarySnapshot> loadLibrary(
    LibraryFilters filters, {
    String? cursor,
  }) async {
    final documents = await api.getDocumentsApi().listDocuments(
      q: filters.query,
      status: filters.read.apiValue,
      folderId: filters.folderId,
      includeFolderDescendants: filters.folderId == null
          ? null
          : filters.includeFolderDescendants,
      tag: filters.tagSlug,
      ratingMin: filters.ratingMin,
      sort: filters.sort?.apiValue,
      cursor: cursor,
    );
    final folders = await api.getFoldersApi().listFolders();
    final tags = await api.getTagsApi().listTags();
    return LibrarySnapshot(
      documents: documents.data?.items?.toList() ?? const [],
      folders: folders.data?.items?.toList() ?? const [],
      tags: tags.data?.items?.toList() ?? const [],
      totalCount:
          documents.data?.totalCount ?? documents.data?.items?.length ?? 0,
      nextCursor: documents.data?.nextCursor,
    );
  }

  @override
  Future<DocumentDetail> getDocument(String id) async {
    final response = await api.getDocumentsApi().getDocument(id: id);
    return _required(response.data, 'Document detail response was empty.');
  }

  @override
  Future<List<DocumentNote>> listNotes(String documentId) async {
    final response = await api.getDocumentNotesApi().listDocumentNotes(
      id: documentId,
    );
    return response.data?.items?.toList() ?? const [];
  }

  @override
  Future<DocumentNote> createNote({
    required String documentId,
    required String body,
    required String format,
  }) async {
    final response = await api.getDocumentNotesApi().createDocumentNote(
      id: documentId,
      noteBodyInputBody: NoteBodyInputBody(
        (b) => b
          ..body = body
          ..format = format == 'markdown'
              ? NoteBodyInputBodyFormatEnum.markdown
              : NoteBodyInputBodyFormatEnum.text,
      ),
    );
    return _required(response.data, 'Document note response was empty.');
  }

  @override
  Future<DocumentDetail> markRead(String documentId) async {
    final response = await api.getDocumentsApi().markDocumentRead(
      id: documentId,
    );
    return _required(response.data, 'Document detail response was empty.');
  }

  @override
  Future<DocumentDetail> markUnread(String documentId) async {
    final response = await api.getDocumentsApi().markDocumentUnread(
      id: documentId,
    );
    return _required(response.data, 'Document detail response was empty.');
  }

  @override
  Future<DocumentDetail> setRating(String documentId, int? rating) async {
    final response = await api.getDocumentsApi().setDocumentRating(
      id: documentId,
      ratingInputBody: RatingInputBody((b) => b.rating = rating),
    );
    return _required(response.data, 'Document detail response was empty.');
  }

  @override
  Future<DocumentDetail> replaceTags(String documentId, List<String> tagIds) {
    return api
        .getDocumentsApi()
        .replaceDocumentTags(
          id: documentId,
          replaceTagsInputBody: ReplaceTagsInputBody(
            (b) => b.tagIds.addAll(tagIds),
          ),
        )
        .then((response) {
          return _required(
            response.data,
            'Document detail response was empty.',
          );
        });
  }

  @override
  Future<DocumentDetail> bookmarkUrl({
    required String url,
    String? title,
    String? folderId,
    List<String> tagIds = const [],
  }) async {
    final response = await api.getIngestionApi().bookmarkDocument(
      bookmarkDocumentInputBody: BookmarkDocumentInputBody(
        (b) => b
          ..url = url
          ..title = _optionalText(title)
          ..folderId = _optionalText(folderId)
          ..tagIds.replace(tagIds),
      ),
    );
    return _required(response.data, 'Document detail response was empty.');
  }

  @override
  Future<DocumentDetail> scrapeUrl({
    required String url,
    String? folderId,
    List<String> tagIds = const [],
  }) async {
    final response = await api.getIngestionApi().scrapeDocument(
      scrapeDocumentInputBody: ScrapeDocumentInputBody(
        (b) => b
          ..url = url
          ..folderId = _optionalText(folderId)
          ..tagIds.replace(tagIds),
      ),
    );
    return _required(response.data, 'Document detail response was empty.');
  }

  @override
  Future<DocumentDetail> uploadFile(UploadFileInput input) async {
    final response = await api.getIngestionApi().uploadDocument(
      file: MultipartFile.fromBytes(input.bytes, filename: input.filename),
      title: _optionalText(input.title),
      folderId: _optionalText(input.folderId),
      tagIds: BuiltList<String>(input.tagIds),
    );
    return _required(response.data, 'Document detail response was empty.');
  }
}

T _required<T>(T? value, String message) {
  if (value == null) {
    throw StateError(message);
  }
  return value;
}

String? _optionalText(String? value) {
  final trimmed = value?.trim();
  return trimmed == null || trimmed.isEmpty ? null : trimmed;
}
