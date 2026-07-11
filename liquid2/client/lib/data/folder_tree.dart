import 'package:liquid2_api/liquid2_api.dart';

import '../domain/folder_system_role.dart';

class FolderTreeItem {
  const FolderTreeItem({required this.folder, required this.depth});

  final Folder folder;
  final int depth;
}

List<FolderTreeItem> flattenFolderTree(List<Folder> folders) {
  return [for (final folder in folders) ..._flattenFolder(folder, 0)];
}

List<FolderTreeItem> flattenAssignableFolderTree(List<Folder> folders) {
  return [
    for (final folder in folders)
      ..._flattenFolder(folder, 0, skipTrashBranch: true),
  ];
}

Iterable<FolderTreeItem> _flattenFolder(
  Folder folder,
  int depth, {
  bool skipTrashBranch = false,
  bool underTrash = false,
}) sync* {
  final inTrashBranch =
      underTrash || folder.systemRole == FolderSystemRole.trash;
  if (!skipTrashBranch || !inTrashBranch) {
    yield FolderTreeItem(folder: folder, depth: depth);
  }
  for (final child in folder.children?.toList() ?? const <Folder>[]) {
    yield* _flattenFolder(
      child,
      depth + 1,
      skipTrashBranch: skipTrashBranch,
      underTrash: inTrashBranch,
    );
  }
}
