import 'package:liquid2_api/liquid2_api.dart';

abstract class FolderRepository {
  Future<List<Folder>> listFolders();

  Future<Folder> createFolder(FolderMutationInput input);

  Future<Folder> updateFolder(String id, FolderMutationInput input);

  Future<void> deleteFolder(
    String id, {
    String documentAction = 'reject_if_not_empty',
  });
}

class FolderMutationInput {
  const FolderMutationInput({
    required this.name,
    this.parentId,
    this.sortOrder = 0,
  });

  final String name;
  final String? parentId;
  final int sortOrder;
}
