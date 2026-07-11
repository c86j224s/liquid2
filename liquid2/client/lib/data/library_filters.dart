enum DocumentReadFilter {
  all(null),
  unread('unread'),
  read('read');

  const DocumentReadFilter(this.apiValue);

  final String? apiValue;
}

enum DocumentSort {
  relevance('relevance'),
  recent('recent'),
  createdDesc('created_desc'),
  ratingDesc('rating_desc');

  const DocumentSort(this.apiValue);

  final String apiValue;
}

enum LibraryViewPreset { all, unread, rated, recent }

class LibraryFilters {
  const LibraryFilters({
    this.query,
    this.read = DocumentReadFilter.unread,
    this.folderId,
    this.includeFolderDescendants = true,
    this.tagSlug,
    this.ratingMin,
    this.sort,
    this.view = LibraryViewPreset.all,
  });

  final String? query;
  final DocumentReadFilter read;
  final String? folderId;
  final bool includeFolderDescendants;
  final String? tagSlug;
  final int? ratingMin;
  final DocumentSort? sort;
  final LibraryViewPreset? view;

  LibraryFilters copyWith({
    Object? query = _sentinel,
    DocumentReadFilter? read,
    Object? folderId = _sentinel,
    bool? includeFolderDescendants,
    Object? tagSlug = _sentinel,
    Object? ratingMin = _sentinel,
    Object? sort = _sentinel,
    Object? view = _sentinel,
  }) {
    return LibraryFilters(
      query: query == _sentinel ? this.query : query as String?,
      read: read ?? this.read,
      folderId: folderId == _sentinel ? this.folderId : folderId as String?,
      includeFolderDescendants:
          includeFolderDescendants ?? this.includeFolderDescendants,
      tagSlug: tagSlug == _sentinel ? this.tagSlug : tagSlug as String?,
      ratingMin: ratingMin == _sentinel ? this.ratingMin : ratingMin as int?,
      sort: sort == _sentinel ? this.sort : sort as DocumentSort?,
      view: view == _sentinel ? this.view : view as LibraryViewPreset?,
    );
  }

  @override
  bool operator ==(Object other) {
    return other is LibraryFilters &&
        other.query == query &&
        other.read == read &&
        other.folderId == folderId &&
        other.includeFolderDescendants == includeFolderDescendants &&
        other.tagSlug == tagSlug &&
        other.ratingMin == ratingMin &&
        other.sort == sort &&
        other.view == view;
  }

  @override
  int get hashCode => Object.hash(
    query,
    read,
    folderId,
    includeFolderDescendants,
    tagSlug,
    ratingMin,
    sort,
    view,
  );
}

const _sentinel = Object();
