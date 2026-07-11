import 'package:liquid2_api/liquid2_api.dart';

import 'folder_repository.dart';

class ApiFolderRepository implements FolderRepository {
  const ApiFolderRepository(this.api);

  final Liquid2Api api;

  @override
  Future<List<Folder>> listFolders() async {
    final response = await api.getFoldersApi().listFolders();
    return response.data?.items?.toList() ?? const [];
  }

  @override
  Future<Folder> createFolder(FolderMutationInput input) async {
    final response = await api.getFoldersApi().createFolder(
      folderBodyInputBody: FolderBodyInputBody(
        (b) => b
          ..name = input.name
          ..parentId = _optionalText(input.parentId)
          ..sortOrder = input.sortOrder,
      ),
    );
    return _required(response.data);
  }

  @override
  Future<void> deleteFolder(
    String id, {
    String documentAction = 'reject_if_not_empty',
  }) {
    return api.getFoldersApi().deleteFolder(
      id: id,
      documentAction: documentAction,
    );
  }

  @override
  Future<Folder> updateFolder(String id, FolderMutationInput input) async {
    final response = await api.getFoldersApi().updateFolder(
      id: id,
      updateFolderInputBody: UpdateFolderInputBody(
        (b) => b
          ..name = input.name
          ..parentId = _optionalText(input.parentId)
          ..sortOrder = input.sortOrder,
      ),
    );
    return _required(response.data);
  }
}

T _required<T>(T? value) {
  if (value == null) {
    throw StateError('Folder response was empty.');
  }
  return value;
}

String? _optionalText(String? value) {
  final trimmed = value?.trim();
  return trimmed == null || trimmed.isEmpty ? null : trimmed;
}
