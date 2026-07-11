import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/api_feed_repository.dart';
import 'package:liquid2_client/data/feed_repository.dart';

void main() {
  test('loadFeeds requests feeds folders and poll feed jobs', () async {
    final adapter = FeedRecordingAdapter();
    final repository = ApiFeedRepository(_api(adapter));

    final snapshot = await repository.loadFeeds();

    expect(adapter.requests.map((request) => request.path), [
      '/api/v1/feeds',
      '/api/v1/folders',
      '/api/v1/jobs',
      '/api/v1/settings',
    ]);
    expect(adapter.requests[2].queryParameters, {
      'kind': 'poll_feed',
      'limit': 10,
    });
    expect(snapshot.feeds.single.id, 'feed_1');
    expect(snapshot.jobs.single.status, 'failed');
    expect(snapshot.settings.feedPollIntervalSeconds, 7200);
  });

  test(
    'create update delete and refresh forward through generated API',
    () async {
      final adapter = FeedRecordingAdapter();
      final repository = ApiFeedRepository(_api(adapter));

      await repository.createFeed(
        const FeedInput(
          url: ' https://example.com/feed.xml ',
          title: 'Example',
        ),
      );
      await repository.updateFeed(
        'feed_1',
        const FeedInput(url: 'https://example.com/next.xml'),
      );
      await repository.deleteFeed('feed_1');
      final job = await repository.refreshFeed('feed_1');
      final settings = await repository.updateSettings(
        const FeedSettingsInput(
          feedSchedulerEnabled: true,
          feedPollIntervalSeconds: 900,
        ),
      );

      expect(adapter.requests.map((request) => request.method), [
        'POST',
        'PATCH',
        'DELETE',
        'POST',
        'PATCH',
      ]);
      expect(adapter.bodies[adapter.requests[0]], contains('"enabled":true'));
      expect(adapter.bodies[adapter.requests[1]], contains('"title":""'));
      expect(adapter.bodies[adapter.requests[1]], contains('"folderId":""'));
      expect(adapter.requests[2].path, '/api/v1/feeds/feed_1');
      expect(adapter.requests[3].path, '/api/v1/feeds/feed_1/refresh');
      expect(adapter.requests[4].path, '/api/v1/settings');
      expect(
        adapter.bodies[adapter.requests[4]],
        contains('"feedSchedulerEnabled":true'),
      );
      expect(job.id, 'job_1');
      expect(settings.feedPollIntervalSeconds, 900);
      expect(settings.feedNextPollAt, _now + 900000);
    },
  );
}

Liquid2Api _api(FeedRecordingAdapter adapter) {
  final dio = Dio(BaseOptions(baseUrl: 'http://api.test'));
  dio.httpClientAdapter = adapter;
  return Liquid2Api(dio: dio, interceptors: const []);
}

class FeedRecordingAdapter implements HttpClientAdapter {
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
    if (options.method == 'DELETE') {
      return ResponseBody.fromString('', 204);
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
    return utf8.decode(bytes);
  }

  Object _responseFor(RequestOptions options) {
    return switch (options.path) {
      '/api/v1/feeds' when options.method == 'GET' => {
        'items': [_feed()],
      },
      '/api/v1/feeds' => _feed(),
      '/api/v1/folders' => {
        'items': [_folder()],
      },
      '/api/v1/jobs' => {
        'items': [_job(status: 'failed', error: 'job failed')],
      },
      '/api/v1/settings' => _settings(
        enabled: options.method == 'PATCH',
        interval: options.method == 'PATCH' ? 900 : 7200,
      ),
      '/api/v1/feeds/feed_1' => _feed(),
      '/api/v1/feeds/feed_1/refresh' => {'job': _job(status: 'queued')},
      _ => throw StateError('Unexpected request: ${options.path}'),
    };
  }
}

Map<String, Object?> _settings({bool enabled = false, int interval = 7200}) {
  return {
    'feedSchedulerEnabled': enabled,
    'feedPollIntervalSeconds': interval,
    'feedNextPollAt': enabled ? _now + interval * 1000 : null,
    'updatedAt': _now,
  };
}

Map<String, Object?> _feed() {
  return {
    'id': 'feed_1',
    'url': 'https://example.com/feed.xml',
    'title': 'Example Feed',
    'folderId': 'folder_1',
    'enabled': true,
    'createdAt': _now,
    'updatedAt': _now,
  };
}

Map<String, Object?> _folder() {
  return {
    'id': 'folder_1',
    'name': 'Inbox',
    'sortOrder': 0,
    'createdAt': _now,
    'updatedAt': _now,
    'children': [],
  };
}

Map<String, Object?> _job({required String status, String? error}) {
  return {
    'id': 'job_1',
    'kind': 'poll_feed',
    'status': status,
    'error': error,
    'attempts': 1,
    'createdAt': _now,
    'updatedAt': _now,
  };
}

const _now = 1760000000000;
