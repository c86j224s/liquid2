part of 'api_library_repository.dart';

mixin ApiLibraryDocumentActions {
  Liquid2Api get api;

  Future<Job> getJob(String id) async {
    final response = await api.getJobsApi().getJob(id: id);
    return _required(response.data, 'Job response was empty.');
  }

  Future<Job> translateDocument({
    required String documentId,
    required String sourceContentId,
    required String targetLanguage,
  }) async {
    try {
      final response = await api.getDocumentsApi().translateDocument(
        id: documentId,
        translateDocumentInputBody: TranslateDocumentInputBody(
          (b) => b
            ..sourceContentId = sourceContentId
            ..targetLanguage = targetLanguage,
        ),
      );
      return _required(response.data?.job, 'Translation response was empty.');
    } on DioException catch (error) {
      if (error.response?.statusCode == 409) {
        throw const TranslationAlreadyRunningException();
      }
      rethrow;
    }
  }

  Future<DocumentDetail> moveDocumentToTrash(String documentId) async {
    final response = await api.getDocumentsApi().moveDocumentToTrash(
      id: documentId,
    );
    return _required(response.data, 'Document detail response was empty.');
  }

  Future<DocumentDetail> moveDocumentToFolder(
    String documentId,
    String folderId,
  ) async {
    final response = await api.getDocumentsApi().updateDocument(
      id: documentId,
      updateDocumentInputBody: UpdateDocumentInputBody(
        (b) => b.folderId = _optionalText(folderId),
      ),
    );
    return _required(response.data, 'Document detail response was empty.');
  }

  Future<DocumentDetail> rescrapeDocument(String documentId) async {
    final response = await api.getIngestionApi().rescrapeDocument(
      id: documentId,
    );
    return _required(response.data, 'Document detail response was empty.');
  }
}
