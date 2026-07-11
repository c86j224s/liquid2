import 'package:liquid2_api/liquid2_api.dart';

mixin FakeTranslationRepository {
  final translateRequests = <FakeTranslationRequest>[];
  final requestedJobIds = <String>[];
  Job? translationJob;

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
    translationJob = fakeTranslationJob(status: 'queued');
    return translationJob!;
  }

  Future<Job> getJob(String id) async {
    requestedJobIds.add(id);
    return translationJob ?? fakeTranslationJob(status: 'queued');
  }
}

class FakeTranslationRequest {
  const FakeTranslationRequest({
    required this.documentId,
    required this.sourceContentId,
    required this.targetLanguage,
  });

  final String documentId;
  final String sourceContentId;
  final String targetLanguage;
}

Job fakeTranslationJob({required String status, String? error}) {
  return Job(
    (b) => b
      ..id = 'job_translate_1'
      ..kind = 'translate_document'
      ..status = status
      ..error = error
      ..attempts = 0
      ..createdAt = _now
      ..updatedAt = _now,
  );
}

const _now = 1760000000000;
