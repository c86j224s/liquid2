import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/folder_repository.dart';

class FakeFolderRepository implements FolderRepository {
  FakeFolderRepository({List<Folder>? folders}) : folders = folders ?? [];

  final List<Folder> folders;
  final created = <FolderMutationInput>[];
  final updated = <String, FolderMutationInput>{};
  final deleted = <String>[];

  @override
  Future<List<Folder>> listFolders() async => folders;

  @override
  Future<Folder> createFolder(FolderMutationInput input) async {
    created.add(input);
    final folder = _folder('folder_${folders.length + 1}', input.name);
    folders.add(folder);
    return folder;
  }

  @override
  Future<void> deleteFolder(
    String id, {
    String documentAction = 'reject_if_not_empty',
  }) async {
    deleted.add(id);
    folders.removeWhere((folder) => folder.id == id);
  }

  @override
  Future<Folder> updateFolder(String id, FolderMutationInput input) async {
    updated[id] = input;
    final folder = _folder(id, input.name, parentId: input.parentId);
    final index = folders.indexWhere((folder) => folder.id == id);
    if (index >= 0) {
      folders[index] = folder;
    }
    return folder;
  }
}

Folder fakeFolder(
  String id,
  String name, {
  String? parentId,
  String? systemRole,
  List<Folder> children = const [],
}) {
  return _folder(
    id,
    name,
    parentId: parentId,
    systemRole: systemRole,
    children: children,
  );
}

Folder _folder(
  String id,
  String name, {
  String? parentId,
  String? systemRole,
  List<Folder> children = const [],
}) {
  return Folder(
    (b) => b
      ..id = id
      ..name = name
      ..parentId = parentId
      ..systemRole = systemRole
      ..sortOrder = 0
      ..createdAt = _now
      ..updatedAt = _now
      ..children.addAll(children),
  );
}

const _now = 1760000000000;
