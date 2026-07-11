import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/data/library_filters.dart';

void main() {
  test('manual preset-field changes clear hidden sort state', () {
    final container = ProviderContainer();
    addTearDown(container.dispose);
    final controller = container.read(libraryFiltersProvider.notifier);

    controller.setView(LibraryViewPreset.rated);
    expect(
      container.read(libraryFiltersProvider).sort,
      DocumentSort.ratingDesc,
    );

    controller.setRatingMin(3);
    var filters = container.read(libraryFiltersProvider);
    expect(filters.view, isNull);
    expect(filters.ratingMin, 3);
    expect(filters.sort, isNull);

    controller.setView(LibraryViewPreset.recent);
    expect(container.read(libraryFiltersProvider).sort, DocumentSort.recent);

    controller.setRead(DocumentReadFilter.unread);
    filters = container.read(libraryFiltersProvider);
    expect(filters.view, isNull);
    expect(filters.read, DocumentReadFilter.unread);
    expect(filters.sort, isNull);
  });
}
