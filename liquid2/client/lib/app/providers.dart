import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/legacy.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../data/api_feed_repository.dart';
import '../data/api_folder_repository.dart';
import '../data/api_library_repository.dart';
import '../data/feed_repository.dart';
import '../data/folder_repository.dart';
import '../data/library_filters.dart';
import '../data/library_repository.dart';
import '../data/library_snapshot.dart';
import 'api_config.dart';

final themeModeProvider = StateProvider<ThemeMode>((ref) => ThemeMode.system);

final apiBaseUrlProvider = Provider<String>(
  (ref) => resolveLiquid2ApiBaseUrl(),
);

final liquid2ApiProvider = Provider<Liquid2Api>((ref) {
  return Liquid2Api(basePathOverride: ref.watch(apiBaseUrlProvider));
});

final libraryRepositoryProvider = Provider<LibraryRepository>((ref) {
  return ApiLibraryRepository(ref.watch(liquid2ApiProvider));
});

final feedRepositoryProvider = Provider<FeedRepository>((ref) {
  return ApiFeedRepository(ref.watch(liquid2ApiProvider));
});

final folderRepositoryProvider = Provider<FolderRepository>((ref) {
  return ApiFolderRepository(ref.watch(liquid2ApiProvider));
});

final libraryFiltersProvider =
    NotifierProvider<LibraryFiltersController, LibraryFilters>(
      LibraryFiltersController.new,
    );

class LibraryFiltersController extends Notifier<LibraryFilters> {
  @override
  LibraryFilters build() => const LibraryFilters();

  void setQuery(String query) {
    final trimmed = query.trim();
    state = state.copyWith(query: trimmed.isEmpty ? null : trimmed);
  }

  void setView(LibraryViewPreset view) {
    state = switch (view) {
      LibraryViewPreset.all => state.copyWith(
        read: DocumentReadFilter.all,
        ratingMin: null,
        sort: null,
        view: view,
      ),
      LibraryViewPreset.unread => state.copyWith(
        read: DocumentReadFilter.unread,
        ratingMin: null,
        sort: null,
        view: view,
      ),
      LibraryViewPreset.rated => state.copyWith(
        read: DocumentReadFilter.all,
        ratingMin: 1,
        sort: DocumentSort.ratingDesc,
        view: view,
      ),
      LibraryViewPreset.recent => state.copyWith(
        read: DocumentReadFilter.all,
        ratingMin: null,
        sort: DocumentSort.recent,
        view: view,
      ),
    };
  }

  void setRead(DocumentReadFilter read) {
    state = state.copyWith(read: read, sort: null, view: null);
  }

  void setFolder(String? folderId) {
    state = state.copyWith(folderId: folderId);
  }

  void setTag(String? tagSlug) {
    state = state.copyWith(tagSlug: tagSlug);
  }

  void setRatingMin(int? ratingMin) {
    state = state.copyWith(ratingMin: ratingMin, sort: null, view: null);
  }
}

final librarySnapshotProvider =
    AsyncNotifierProvider<LibrarySnapshotController, LibrarySnapshot>(
      LibrarySnapshotController.new,
    );

class LibrarySnapshotController extends AsyncNotifier<LibrarySnapshot> {
  @override
  Future<LibrarySnapshot> build() {
    final filters = ref.watch(libraryFiltersProvider);
    return ref.watch(libraryRepositoryProvider).loadLibrary(filters);
  }

  Future<void> loadMore() async {
    final current = state.value;
    final cursor = current?.nextCursor;
    if (current == null || cursor == null || current.isLoadingMore) {
      return;
    }
    final filters = ref.read(libraryFiltersProvider);
    state = AsyncData(current.copyWith(isLoadingMore: true));
    try {
      final page = await ref
          .read(libraryRepositoryProvider)
          .loadLibrary(filters, cursor: cursor);
      final latest = state.value;
      if (latest == null ||
          latest.nextCursor != cursor ||
          ref.read(libraryFiltersProvider) != filters) {
        return;
      }
      state = AsyncData(latest.appendDocumentPage(page));
    } catch (error) {
      final latest = state.value;
      if (latest != null &&
          latest.nextCursor == cursor &&
          ref.read(libraryFiltersProvider) == filters) {
        state = AsyncData(
          latest.copyWith(isLoadingMore: false, loadMoreError: error),
        );
      }
    }
  }
}

final documentDetailProvider = FutureProvider.family<DocumentDetail, String>((
  ref,
  id,
) {
  return ref.watch(libraryRepositoryProvider).getDocument(id);
});

final documentNotesProvider = FutureProvider.family<List<DocumentNote>, String>(
  (ref, id) {
    return ref.watch(libraryRepositoryProvider).listNotes(id);
  },
);

final documentTagSelectionProvider = StateProvider.family<Set<String>?, String>(
  (ref, id) => null,
);

final documentTagSavingProvider = StateProvider.family<bool, String>(
  (ref, id) => false,
);
