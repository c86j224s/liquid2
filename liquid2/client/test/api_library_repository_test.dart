import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/api_library_repository.dart';
import 'package:liquid2_client/data/library_filters.dart';

void main() {
  test(
    'loadLibrary forwards filters and cursor through generated API',
    () async {
      final adapter = RecordingAdapter();
      final repository = ApiLibraryRepository(_api(adapter));

      final snapshot = await repository.loadLibrary(
        const LibraryFilters(
          query: 'sqlite',
          read: DocumentReadFilter.unread,
          folderId: 'folder_1',
          tagSlug: 'go',
          ratingMin: 4,
          sort: DocumentSort.ratingDesc,
        ),
        cursor: 'next_1',
      );

      final documentsRequest = adapter.requests.firstWhere(
        (request) => request.path == '/api/v1/documents',
      );
      expect(documentsRequest.queryParameters, {
        'q': 'sqlite',
        'status': 'unread',
        'folderId': 'folder_1',
        'includeFolderDescendants': true,
        'tag': 'go',
        'ratingMin': 4,
        'sort': 'rating_desc',
        'cursor': 'next_1',
      });
      expect(snapshot.nextCursor, 'next_2');
      expect(snapshot.totalCount, 42);
      expect(snapshot.documents.single.id, 'doc_1');
    },
  );

  test('setRating accepts null for clear rating flow', () async {
    final adapter = RecordingAdapter();
    final repository = ApiLibraryRepository(_api(adapter));

    final detail = await repository.setRating('doc_1', null);

    final request = adapter.requests.singleWhere(
      (request) => request.path == '/api/v1/documents/doc_1/rating',
    );
    expect(request.method, 'PUT');
    expect(adapter.bodies[request], '{}');
    expect(detail.document.rating, isNull);
  });

  test('moveDocumentToTrash posts to document trash endpoint', () async {
    final adapter = RecordingAdapter();
    final repository = ApiLibraryRepository(_api(adapter));

    final detail = await repository.moveDocumentToTrash('doc_1');

    final request = adapter.requests.singleWhere(
      (request) => request.path == '/api/v1/documents/doc_1/move-to-trash',
    );
    expect(request.method, 'POST');
    expect(detail.document.id, 'doc_1');
  });

  test('rescrapeDocument posts to document re-scrape endpoint', () async {
    final adapter = RecordingAdapter();
    final repository = ApiLibraryRepository(_api(adapter));

    final detail = await repository.rescrapeDocument('doc_1');

    final request = adapter.requests.singleWhere(
      (request) => request.path == '/api/v1/documents/doc_1/rescrape',
    );
    expect(request.method, 'POST');
    expect(detail.document.id, 'doc_1');
  });
}

Liquid2Api _api(RecordingAdapter adapter) {
  final dio = Dio(BaseOptions(baseUrl: 'http://api.test'));
  dio.httpClientAdapter = adapter;
  return Liquid2Api(dio: dio, interceptors: const []);
}

class RecordingAdapter implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  final bodies = <RequestOptions, String>{};

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    bodies[options] = await _body(requestStream);
    return ResponseBody.fromString(
      jsonEncode(_responseFor(options)),
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}

  Future<String> _body(Stream<Uint8List>? stream) async {
    final bytes = <int>[];
    if (stream != null) {
      await for (final chunk in stream) {
        bytes.addAll(chunk);
      }
    }
    return utf8.decode(bytes);
  }

  Object _responseFor(RequestOptions options) {
    return switch (options.path) {
      '/api/v1/documents' => {
        'items': [_documentSummary()],
        'nextCursor': 'next_2',
        'totalCount': 42,
      },
      '/api/v1/folders' => {
        'items': [_folder()],
      },
      '/api/v1/tags' => {
        'items': [_tag()],
      },
      '/api/v1/documents/doc_1/rating' => _documentDetail(rating: null),
      '/api/v1/documents/doc_1/move-to-trash' => _documentDetail(rating: 4),
      '/api/v1/documents/doc_1/rescrape' => _documentDetail(rating: 4),
      _ => throw StateError('Unexpected request: ${options.path}'),
    };
  }
}

Map<String, Object?> _documentDetail({int? rating}) => {
  'document': _documentMetadata(rating: rating),
  'contents': [],
  'tags': [],
};

Map<String, Object?> _documentSummary() => {
  'id': 'doc_1',
  'title': 'SQLite notes',
  'kind': 'bookmark',
  'status': 'unread',
  'rating': 4,
  'folderId': 'folder_1',
  'tags': ['go'],
  'createdAt': _now,
  'updatedAt': _now,
  'publishedAt': null,
};

Map<String, Object?> _documentMetadata({int? rating}) => {
  'id': 'doc_1',
  'title': 'SQLite notes',
  'kind': 'bookmark',
  'status': 'unread',
  'rating': rating,
  'createdAt': _now,
  'updatedAt': _now,
  'publishedAt': null,
};

Map<String, Object?> _folder() => {
  'id': 'folder_1',
  'name': 'Inbox',
  'sortOrder': 0,
  'createdAt': _now,
  'updatedAt': _now,
  'children': [],
};

Map<String, Object?> _tag() => {
  'id': 'tag_go',
  'name': 'go',
  'slug': 'go',
  'createdAt': _now,
};

const _now = 1760000000000;
