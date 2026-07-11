import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/api_library_repository.dart';

void main() {
  test('moveDocumentToFolder patches document folderId', () async {
    final adapter = _RecordingAdapter();
    final repository = ApiLibraryRepository(_api(adapter));

    final detail = await repository.moveDocumentToFolder('doc_1', 'folder_2');

    final request = adapter.requests.single;
    expect(request.path, '/api/v1/documents/doc_1');
    expect(request.method, 'PATCH');
    expect(adapter.bodies[request], '{"folderId":"folder_2"}');
    expect(detail.document.folderId, 'folder_2');
  });
}

Liquid2Api _api(_RecordingAdapter adapter) {
  final dio = Dio(BaseOptions(baseUrl: 'http://api.test'));
  dio.httpClientAdapter = adapter;
  return Liquid2Api(dio: dio, interceptors: const []);
}

class _RecordingAdapter implements HttpClientAdapter {
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
      'id': 'doc_1',
      'title': 'SQLite notes',
      'kind': 'bookmark',
      'status': 'unread',
      'folderId': 'folder_2',
      'createdAt': _now,
      'updatedAt': _now,
    },
    'contents': [],
    'tags': [],
  };
}

const _now = 1760000000000;
