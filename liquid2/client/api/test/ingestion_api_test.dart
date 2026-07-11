import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for IngestionApi
void main() {
  final instance = Liquid2Api().getIngestionApi();

  group(IngestionApi, () {
    // Bookmark URL
    //
    //Future<DocumentDetail> bookmarkDocument(BookmarkDocumentInputBody bookmarkDocumentInputBody) async
    test('test bookmarkDocument', () async {
      // TODO
    });

    // Re-scrape document
    //
    //Future<DocumentDetail> rescrapeDocument(String id) async
    test('test rescrapeDocument', () async {
      // TODO
    });

    // Scrape URL
    //
    //Future<DocumentDetail> scrapeDocument(ScrapeDocumentInputBody scrapeDocumentInputBody) async
    test('test scrapeDocument', () async {
      // TODO
    });

    // Scrape URL and translate
    //
    //Future<ScrapeTranslateDocumentOutputBody> scrapeTranslateDocument(ScrapeTranslateDocumentInputBody scrapeTranslateDocumentInputBody) async
    test('test scrapeTranslateDocument', () async {
      // TODO
    });

    // Upload document
    //
    //Future<DocumentDetail> uploadDocument(MultipartFile file, { String folderId, BuiltList<String> tagIds, String title }) async
    test('test uploadDocument', () async {
      // TODO
    });

  });
}
