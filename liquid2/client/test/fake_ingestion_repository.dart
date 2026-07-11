import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/library_repository.dart';

mixin FakeIngestionRepository {
  final createdDocuments = <String>[];

  Future<DocumentDetail> bookmarkUrl({
    required String url,
    String? title,
    String? folderId,
    List<String> tagIds = const [],
  }) async {
    createdDocuments.add('bookmark:$url');
    return _ingestedDetail(
      id: 'doc_bookmark',
      title: title?.trim().isNotEmpty == true ? title!.trim() : url,
      kind: 'bookmark',
      folderId: folderId,
    );
  }

  Future<DocumentDetail> scrapeUrl({
    required String url,
    String? folderId,
    List<String> tagIds = const [],
  }) async {
    createdDocuments.add('scrape:$url');
    return _ingestedDetail(
      id: 'doc_scrape',
      title: 'Scraped document',
      kind: 'scraped_article',
      folderId: folderId,
      content: 'Scraped body',
    );
  }

  Future<DocumentDetail> uploadFile(UploadFileInput input) async {
    createdDocuments.add('upload:${input.filename}');
    return _ingestedDetail(
      id: 'doc_upload',
      title: input.title?.trim().isNotEmpty == true
          ? input.title!.trim()
          : input.filename,
      kind: 'uploaded_file',
      folderId: input.folderId,
      content: 'Uploaded body',
    );
  }
}

DocumentDetail _ingestedDetail({
  required String id,
  required String title,
  required String kind,
  String? folderId,
  String? content,
}) {
  return DocumentDetail(
    (b) => b
      ..document.replace(
        DocumentMetadata(
          (m) => m
            ..id = id
            ..title = title
            ..kind = kind
            ..folderId = folderId
            ..status = 'unread'
            ..createdAt = _now
            ..updatedAt = _now,
        ),
      )
      ..contents.addAll(
        content == null
            ? const []
            : [
                DocumentContent(
                  (c) => c
                    ..id = 'content_$id'
                    ..role = 'extracted'
                    ..format = 'text'
                    ..content = content,
                ),
              ],
      ),
  );
}

const _now = 1760000000000;
