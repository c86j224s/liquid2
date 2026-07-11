import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/data/library_snapshot.dart';

void main() {
  test(
    'appendDocumentPage keeps first page count when cursor page skips count',
    () {
      const first = LibrarySnapshot(
        documents: [],
        folders: [],
        tags: [],
        totalCount: 12,
        nextCursor: 'next',
      );
      const page = LibrarySnapshot(
        documents: [],
        folders: [],
        tags: [],
        totalCount: -1,
      );

      expect(first.appendDocumentPage(page).totalCount, 12);
    },
  );
}
