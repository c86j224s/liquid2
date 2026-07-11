import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for FoldersApi
void main() {
  final instance = Liquid2Api().getFoldersApi();

  group(FoldersApi, () {
    // Create folder
    //
    //Future<Folder> createFolder(FolderBodyInputBody folderBodyInputBody) async
    test('test createFolder', () async {
      // TODO
    });

    // Delete folder
    //
    //Future deleteFolder(String id, { String documentAction }) async
    test('test deleteFolder', () async {
      // TODO
    });

    // List folder tree
    //
    //Future<FolderListOutputBody> listFolders() async
    test('test listFolders', () async {
      // TODO
    });

    // Update folder
    //
    //Future<Folder> updateFolder(String id, UpdateFolderInputBody updateFolderInputBody) async
    test('test updateFolder', () async {
      // TODO
    });

  });
}
