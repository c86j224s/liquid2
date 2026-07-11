import 'package:liquid2_api/liquid2_api.dart';

class LibrarySnapshot {
  const LibrarySnapshot({
    required this.documents,
    required this.folders,
    required this.tags,
    required this.totalCount,
    this.nextCursor,
    this.isLoadingMore = false,
    this.loadMoreError,
  });

  final List<DocumentSummary> documents;
  final List<Folder> folders;
  final List<Tag> tags;
  final int totalCount;
  final String? nextCursor;
  final bool isLoadingMore;
  final Object? loadMoreError;

  bool get hasMoreDocuments => nextCursor != null;

  LibrarySnapshot copyWith({
    List<DocumentSummary>? documents,
    List<Folder>? folders,
    List<Tag>? tags,
    int? totalCount,
    Object? nextCursor = _sentinel,
    bool? isLoadingMore,
    Object? loadMoreError = _sentinel,
  }) {
    return LibrarySnapshot(
      documents: documents ?? this.documents,
      folders: folders ?? this.folders,
      tags: tags ?? this.tags,
      totalCount: totalCount ?? this.totalCount,
      nextCursor: nextCursor == _sentinel
          ? this.nextCursor
          : nextCursor as String?,
      isLoadingMore: isLoadingMore ?? this.isLoadingMore,
      loadMoreError: loadMoreError == _sentinel
          ? this.loadMoreError
          : loadMoreError,
    );
  }

  LibrarySnapshot appendDocumentPage(LibrarySnapshot page) {
    return copyWith(
      documents: [...documents, ...page.documents],
      totalCount: page.totalCount < 0 ? totalCount : page.totalCount,
      nextCursor: page.nextCursor,
      isLoadingMore: false,
      loadMoreError: null,
    );
  }
}

const _sentinel = Object();
