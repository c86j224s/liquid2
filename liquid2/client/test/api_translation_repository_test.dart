import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/api_library_repository.dart';
import 'package:liquid2_client/data/library_repository.dart';

void main() {
  test('translateDocument forwards generated body and returns job', () async {
    final adapter = _RecordingAdapter();
    final repository = ApiLibraryRepository(_api(adapter));

    final job = await repository.translateDocument(
      documentId: 'doc_1',
      sourceContentId: 'content_1',
      targetLanguage: 'ko',
    );

    final request = adapter.requests.single;
    expect(request.method, 'POST');
    expect(request.path, '/api/v1/documents/doc_1/translate');
    expect(adapter.bodies[request], {
      'sourceContentId': 'content_1',
      'targetLanguage': 'ko',
    });
    expect(job.id, 'job_translate_1');
  });

  test('getJob forwards job id', () async {
    final adapter = _RecordingAdapter();
    final repository = ApiLibraryRepository(_api(adapter));

    final job = await repository.getJob('job_translate_1');

    final request = adapter.requests.single;
    expect(request.method, 'GET');
    expect(request.path, '/api/v1/jobs/job_translate_1');
    expect(job.status, 'completed');
  });

  test('translateDocument maps conflict to friendly domain error', () async {
    final adapter = _RecordingAdapter()..translateStatus = 409;
    final repository = ApiLibraryRepository(_api(adapter));

    await expectLater(
      repository.translateDocument(
        documentId: 'doc_1',
        sourceContentId: 'content_1',
        targetLanguage: 'ko',
      ),
      throwsA(isA<TranslationAlreadyRunningException>()),
    );
  });
}

Liquid2Api _api(_RecordingAdapter adapter) {
  final dio = Dio(BaseOptions(baseUrl: 'http://api.test'));
  dio.httpClientAdapter = adapter;
  return Liquid2Api(dio: dio, interceptors: const []);
}

class _RecordingAdapter implements HttpClientAdapter {
  final requests = <RequestOptions>[];
  final bodies = <RequestOptions, Object?>{};
  var translateStatus = 200;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    requests.add(options);
    bodies[options] = jsonDecode(await _body(requestStream));
    if (options.path == '/api/v1/documents/doc_1/translate' &&
        translateStatus != 200) {
      return ResponseBody.fromString(
        jsonEncode({
          'title': 'Conflict',
          'status': translateStatus,
          'detail': 'translation already queued',
        }),
        translateStatus,
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      );
    }
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
    return bytes.isEmpty ? 'null' : utf8.decode(bytes);
  }

  Object _responseFor(RequestOptions options) {
    return switch (options.path) {
      '/api/v1/documents/doc_1/translate' => {'job': _job('queued')},
      '/api/v1/jobs/job_translate_1' => _job('completed'),
      _ => throw StateError('Unexpected request: ${options.path}'),
    };
  }
}

Map<String, Object?> _job(String status) {
  return {
    'id': 'job_translate_1',
    'kind': 'translate_document',
    'status': status,
    'error': null,
    'attempts': 0,
    'createdAt': _now,
    'updatedAt': _now,
    'startedAt': null,
    'finishedAt': status == 'completed' ? _now : null,
  };
}

const _now = 1760000000000;
