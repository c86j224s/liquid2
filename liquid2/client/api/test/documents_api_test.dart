import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for DocumentsApi
void main() {
  final instance = Liquid2Api().getDocumentsApi();

  group(DocumentsApi, () {
    // Soft-delete document
    //
    //Future<DeletedOutputBody> deleteDocument(String id) async
    test('test deleteDocument', () async {
      // TODO
    });

    // Get document detail
    //
    //Future<DocumentDetail> getDocument(String id) async
    test('test getDocument', () async {
      // TODO
    });

    // List documents
    //
    //Future<DocumentList> listDocuments({ String q, String status, String folderId, bool includeFolderDescendants, String tag, int ratingMin, String kind, String sort, bool includeDeleted, bool includeTrash, int limit, String cursor }) async
    test('test listDocuments', () async {
      // TODO
    });

    // Mark document read
    //
    //Future<DocumentDetail> markDocumentRead(String id) async
    test('test markDocumentRead', () async {
      // TODO
    });

    // Mark document unread
    //
    //Future<DocumentDetail> markDocumentUnread(String id) async
    test('test markDocumentUnread', () async {
      // TODO
    });

    // Move document to trash
    //
    //Future<DocumentDetail> moveDocumentToTrash(String id) async
    test('test moveDocumentToTrash', () async {
      // TODO
    });

    // Replace document tags
    //
    //Future<DocumentDetail> replaceDocumentTags(String id, ReplaceTagsInputBody replaceTagsInputBody) async
    test('test replaceDocumentTags', () async {
      // TODO
    });

    // Set document rating
    //
    //Future<DocumentDetail> setDocumentRating(String id, RatingInputBody ratingInputBody) async
    test('test setDocumentRating', () async {
      // TODO
    });

    // Translate document content
    //
    //Future<TranslateDocumentOutputBody> translateDocument(String id, TranslateDocumentInputBody translateDocumentInputBody) async
    test('test translateDocument', () async {
      // TODO
    });

    // Update document metadata
    //
    //Future<DocumentDetail> updateDocument(String id, UpdateDocumentInputBody updateDocumentInputBody) async
    test('test updateDocument', () async {
      // TODO
    });

  });
}
