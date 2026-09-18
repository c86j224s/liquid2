import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/app/android_share_shell.dart';
import 'fake_library_repository.dart';

class FailingScrapeRepository extends FakeLibraryRepository {
  bool failBookmark = false;
  @override
  Future<DocumentDetail> scrapeUrl({
    required String url,
    String? folderId,
    List<String> tagIds = const [],
  }) async {
    createdDocuments.add('scrape:$url');
    throw StateError('scrape failed');
  }

  @override
  Future<DocumentDetail> bookmarkUrl({
    required String url,
    String? title,
    String? folderId,
    List<String> tagIds = const [],
  }) async {
    if (failBookmark) throw StateError('offline');
    return super.bookmarkUrl(
      url: url,
      title: title,
      folderId: folderId,
      tagIds: tagIds,
    );
  }
}

void main() {
  test('share extracts distinct HTTP links and rejects credentials', () {
    expect(sharedUrls('제목 https://example.com/a https://example.com/a'), [
      'https://example.com/a',
    ]);
    expect(sharedUrls('https://user:secret@example.com'), isEmpty);
  });
  test('successful scrape does not bookmark', () async {
    final r = FakeLibraryRepository();
    expect(await saveSharedUrl(r, 'https://example.com'), true);
    expect(r.createdDocuments, ['scrape:https://example.com']);
  });
  test('failed scrape falls back to same URL bookmark', () async {
    final r = FailingScrapeRepository();
    expect(await saveSharedUrl(r, 'https://example.com'), false);
    expect(r.createdDocuments, [
      'scrape:https://example.com',
      'bookmark:https://example.com',
    ]);
  });
  test('both failures remain failure', () async {
    final r = FailingScrapeRepository()..failBookmark = true;
    await expectLater(
      saveSharedUrl(r, 'https://example.com'),
      throwsStateError,
    );
  });
}
