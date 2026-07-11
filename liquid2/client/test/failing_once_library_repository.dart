import 'package:liquid2_client/data/library_filters.dart';
import 'package:liquid2_client/data/library_snapshot.dart';

import 'fake_library_repository.dart';

class FailingOnceLibraryRepository extends FakeLibraryRepository {
  var loadAttempts = 0;

  @override
  Future<LibrarySnapshot> loadLibrary(
    LibraryFilters filters, {
    String? cursor,
  }) {
    if (loadAttempts++ == 0) {
      throw StateError('api offline');
    }
    return super.loadLibrary(filters, cursor: cursor);
  }
}
