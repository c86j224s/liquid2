import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/api_tag_repository.dart';

void main() {
  test('createTag forwards a trimmed tag name', () async {
    final adapter = _RecordingAdapter();
    final repository = ApiTagRepository(_api(adapter));

    final tag = await repository.createTag('  Research  ');

    expect(tag.id, 'tag_research');
    expect(adapter.requests.single.path, '/api/v1/tags');
    expect(adapter.bodies.single, '{"name":"Research"}');
  });
}

Liquid2Api _api(_RecordingAdapter adapter) {
  final dio = Dio(BaseOptions(baseUrl: 'http://api.test'));
  dio.httpClientAdapter = adapter;
  return Liquid2Api(dio: dio, interceptors: const []);
}

class _RecordingAdapter implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  final bodies = <String>[];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    bodies.add(await _body(requestStream));
    return ResponseBody.fromString(
      jsonEncode({
        'id': 'tag_research',
        'name': 'Research',
        'slug': 'research',
        'createdAt': _now,
      }),
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

const _now = 1760000000000;
