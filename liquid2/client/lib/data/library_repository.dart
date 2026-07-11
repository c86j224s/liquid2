import 'dart:typed_data';

import 'package:liquid2_api/liquid2_api.dart';

import 'library_filters.dart';
import 'library_snapshot.dart';

abstract class LibraryRepository {
  Future<LibrarySnapshot> loadLibrary(LibraryFilters filters, {String? cursor});

  Future<DocumentDetail> getDocument(String id);

  Future<List<DocumentNote>> listNotes(String documentId);

  Future<DocumentNote> createNote({
    required String documentId,
    required String body,
    required String format,
  });

  Future<DocumentDetail> markRead(String documentId);

  Future<DocumentDetail> markUnread(String documentId);

  Future<DocumentDetail> moveDocumentToTrash(String documentId);

  Future<DocumentDetail> moveDocumentToFolder(
    String documentId,
    String folderId,
  );

  Future<DocumentDetail> setRating(String documentId, int? rating);

  Future<DocumentDetail> replaceTags(String documentId, List<String> tagIds);

  Future<DocumentDetail> rescrapeDocument(String documentId);

  Future<Job> translateDocument({
    required String documentId,
    required String sourceContentId,
    required String targetLanguage,
  });

  Future<Job> getJob(String id);

  Future<DocumentDetail> bookmarkUrl({
    required String url,
    String? title,
    String? folderId,
    List<String> tagIds = const [],
  });

  Future<DocumentDetail> scrapeUrl({
    required String url,
    String? folderId,
    List<String> tagIds = const [],
  });

  Future<DocumentDetail> uploadFile(UploadFileInput input);
}

class TranslationAlreadyRunningException implements Exception {
  const TranslationAlreadyRunningException();

  @override
  String toString() {
    return 'A translation for this content and language is already queued or running.';
  }
}

class UploadFileInput {
  const UploadFileInput({
    required this.filename,
    required this.bytes,
    this.title,
    this.folderId,
    this.tagIds = const [],
  });

  final String filename;
  final Uint8List bytes;
  final String? title;
  final String? folderId;
  final List<String> tagIds;
}
