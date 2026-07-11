import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/data/folder_tree.dart';

import 'fake_folder_repository.dart';

void main() {
  test('assignable folder tree excludes trash branches', () {
    final trashChild = fakeFolder('folder_trash_child', 'Old trash');
    final folders = [
      fakeFolder('folder_inbox', 'Inbox'),
      fakeFolder(
        'folder_trash',
        'Trash',
        systemRole: 'trash',
        children: [trashChild],
      ),
    ];

    expect(flattenFolderTree(folders).map((item) => item.folder.name), [
      'Inbox',
      'Trash',
      'Old trash',
    ]);
    expect(
      flattenAssignableFolderTree(folders).map((item) => item.folder.name),
      ['Inbox'],
    );
  });
}
