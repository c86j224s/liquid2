import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/api_folder_repository.dart';
import 'package:liquid2_client/data/folder_repository.dart';

void main() {
  test('folder repository forwards CRUD requests', () async {
    final adapter = RecordingFolderAdapter();
    final repository = ApiFolderRepository(_api(adapter));

    final folders = await repository.listFolders();
    final created = await repository.createFolder(
      const FolderMutationInput(name: 'Research', sortOrder: 10),
    );
    final updated = await repository.updateFolder(
      'folder_1',
      const FolderMutationInput(name: 'Archive', parentId: 'root'),
    );
    await repository.deleteFolder('folder_1');

    expect(folders.single.systemRole, 'inbox');
    expect(created.name, 'Research');
    expect(updated.name, 'Archive');
    expect(adapter.requests.map((request) => request.path), [
      '/api/v1/folders',
      '/api/v1/folders',
      '/api/v1/folders/folder_1',
      '/api/v1/folders/folder_1',
    ]);
    expect(adapter.requests.last.queryParameters, {
      'documentAction': 'reject_if_not_empty',
    });
    expect(jsonDecode(adapter.bodies[adapter.requests[1]]!), {
      'name': 'Research',
      'sortOrder': 10,
    });
    expect(jsonDecode(adapter.bodies[adapter.requests[2]]!), {
      'name': 'Archive',
      'parentId': 'root',
      'sortOrder': 0,
    });
  });
}

Liquid2Api _api(RecordingFolderAdapter adapter) {
  final dio = Dio(BaseOptions(baseUrl: 'http://api.test'));
  dio.httpClientAdapter = adapter;
  return Liquid2Api(dio: dio, interceptors: const []);
}

class RecordingFolderAdapter implements HttpClientAdapter {
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
    if (options.method == 'GET') {
      return {
        'items': [_folder('folder_1', 'Inbox', systemRole: 'inbox')],
      };
    }
    if (options.method == 'POST') {
      return _folder('folder_2', 'Research');
    }
    if (options.method == 'PATCH') {
      return _folder('folder_1', 'Archive', parentId: 'root');
    }
    return {};
  }
}

Map<String, Object?> _folder(
  String id,
  String name, {
  String? parentId,
  String? systemRole,
}) {
  return {
    'id': id,
    'name': name,
    'parentId': parentId,
    'sortOrder': 0,
    'systemRole': systemRole,
    'createdAt': _now,
    'updatedAt': _now,
    'children': [],
  };
}

const _now = 1760000000000;
