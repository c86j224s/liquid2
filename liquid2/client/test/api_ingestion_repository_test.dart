import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/api_library_repository.dart';
import 'package:liquid2_client/data/library_repository.dart';

void main() {
  test('bookmarkUrl forwards body through generated ingestion API', () async {
    final adapter = IngestionRecordingAdapter();
    final repository = ApiLibraryRepository(_api(adapter));

    final detail = await repository.bookmarkUrl(
      url: 'https://example.com/a',
      title: 'Example',
      folderId: 'folder_1',
      tagIds: ['tag_go'],
    );

    final request = adapter.requests.single;
    expect(request.path, '/api/v1/documents/bookmark');
    expect(adapter.bodies[request], contains('"url":"https://example.com/a"'));
    expect(adapter.bodies[request], contains('"folderId":"folder_1"'));
    expect(detail.document.kind, 'bookmark');
  });

  test('uploadFile sends multipart bytes', () async {
    final adapter = IngestionRecordingAdapter();
    final repository = ApiLibraryRepository(_api(adapter));

    await repository.uploadFile(
      UploadFileInput(
        filename: 'note.txt',
        title: 'Note',
        bytes: Uint8List.fromList(utf8.encode('Stored body')),
      ),
    );

    final request = adapter.requests.single;
    expect(request.path, '/api/v1/documents/upload');
    expect(request.contentType, contains('multipart/form-data'));
    expect(adapter.bodies[request], contains('filename="note.txt"'));
    expect(adapter.bodies[request], contains('Stored body'));
  });
}

Liquid2Api _api(IngestionRecordingAdapter adapter) {
  final dio = Dio(BaseOptions(baseUrl: 'http://api.test'));
  dio.httpClientAdapter = adapter;
  return Liquid2Api(dio: dio, interceptors: const []);
}

class IngestionRecordingAdapter implements HttpClientAdapter {
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
      jsonEncode(_documentDetail()),
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
}

Map<String, Object?> _documentDetail() {
  return {
    'document': {
      'id': 'doc_ingested',
      'title': 'Ingested',
      'kind': 'bookmark',
      'status': 'unread',
      'createdAt': _now,
      'updatedAt': _now,
    },
    'contents': [],
    'tags': [],
    'blobs': [],
  };
}

const _now = 1760000000000;
