import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/library_repository.dart';

import 'fake_library_repository.dart';
import 'fake_translation_repository.dart';

class CompletingTranslationRepository extends FakeLibraryRepository {
  var completed = false;

  @override
  Future<Job> translateDocument({
    required String documentId,
    required String sourceContentId,
    required String targetLanguage,
  }) async {
    translateRequests.add(
      FakeTranslationRequest(
        documentId: documentId,
        sourceContentId: sourceContentId,
        targetLanguage: targetLanguage,
      ),
    );
    return fakeTranslationJob(status: 'queued');
  }

  @override
  Future<Job> getJob(String id) async {
    requestedJobIds.add(id);
    completed = true;
    return fakeTranslationJob(status: 'completed');
  }

  @override
  Future<DocumentDetail> getDocument(String id) async {
    return DocumentDetail(
      (b) => b
        ..document.replace(_metadata())
        ..contents.add(
          _content('content_1', 'original', 'Stored document body'),
        )
        ..contents.addAll(
          completed
              ? [_content('content_2', 'translation', 'Translated body', 'ko')]
              : const [],
        ),
    );
  }

  DocumentMetadata _metadata() {
    return DocumentMetadata(
      (b) => b
        ..id = 'doc_1'
        ..title = 'SQLite notes'
        ..kind = 'bookmark'
        ..status = 'unread'
        ..createdAt = _now
        ..updatedAt = _now,
    );
  }

  DocumentContent _content(
    String id,
    String role,
    String content, [
    String? language,
  ]) {
    return DocumentContent(
      (b) => b
        ..id = id
        ..role = role
        ..format = 'markdown'
        ..language = language
        ..content = content,
    );
  }
}

class ConflictTranslationRepository extends FakeLibraryRepository {
  @override
  Future<Job> translateDocument({
    required String documentId,
    required String sourceContentId,
    required String targetLanguage,
  }) {
    return Future.error(const TranslationAlreadyRunningException());
  }
}

const _now = 1760000000000;
