import 'package:flutter/material.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/app_theme.dart';
import '../../app/providers.dart';
import 'document_list_load_more_row.dart';
import 'document_list_scroll_buttons.dart';
import 'document_list_tile.dart';
import 'library_hint_mode.dart';
import 'library_keyboard_cursor.dart';

class DocumentListPanel extends ConsumerStatefulWidget {
  const DocumentListPanel({
    required this.documents,
    required this.hasMore,
    required this.isLoadingMore,
    required this.totalCount,
    required this.onLoadMore,
    this.loadMoreError,
    super.key,
  });

  final List<DocumentSummary> documents;
  final bool hasMore;
  final bool isLoadingMore;
  final int totalCount;
  final Object? loadMoreError;
  final VoidCallback onLoadMore;

  @override
  ConsumerState<DocumentListPanel> createState() => _DocumentListPanelState();
}

class _DocumentListPanelState extends ConsumerState<DocumentListPanel> {
  final _scrollController = ScrollController();
  final _rowKeys = <String, GlobalKey>{};
  String? _openSwipeDocumentId;

  @override
  void didUpdateWidget(DocumentListPanel oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (identical(widget.documents, oldWidget.documents)) return;
    final ids = {for (final document in widget.documents) document.id};
    _rowKeys.removeWhere((id, _) => !ids.contains(id));
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      ref.read(libraryCursorProvider.notifier).clampTo(widget.documents.length);
    });
  }

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  void _scrollToTop() {
    _scrollController.animateTo(
      0,
      duration: const Duration(milliseconds: 300),
      curve: Curves.easeOut,
    );
  }

  void _scrollToBottom() {
    _scrollController.animateTo(
      _scrollController.position.maxScrollExtent,
      duration: const Duration(milliseconds: 300),
      curve: Curves.easeOut,
    );
  }

  void _revealCursor(int index, {required bool downward, bool retry = true}) {
    if (index < 0 || index >= widget.documents.length) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final target = _rowKeys[widget.documents[index].id]?.currentContext;
      if (target == null) {
        if (!retry || !_scrollController.hasClients) return;
        final edge = index == 0
            ? 0.0
            : index == widget.documents.length - 1
            ? _scrollController.position.maxScrollExtent
            : null;
        if (edge == null) return;
        _scrollController.jumpTo(edge);
        _revealCursor(index, downward: downward, retry: false);
        return;
      }
      Scrollable.ensureVisible(
        target,
        duration: const Duration(milliseconds: 160),
        curve: Curves.easeOut,
        alignmentPolicy: downward
            ? ScrollPositionAlignmentPolicy.keepVisibleAtEnd
            : ScrollPositionAlignmentPolicy.keepVisibleAtStart,
      );
    });
  }

  void _openDocument(int index) {
    context.go('/documents/${widget.documents[index].id}');
  }

  List<LibraryHintTarget> _hintTargets() {
    final targets = <LibraryHintTarget>[];
    Rect? viewport;
    for (var index = 0; index < widget.documents.length; index++) {
      final rowContext = _rowKeys[widget.documents[index].id]?.currentContext;
      final box = rowContext?.findRenderObject() as RenderBox?;
      if (box == null || !box.attached || !box.hasSize) continue;
      viewport ??= _viewportRect(box);
      final rect = box.localToGlobal(Offset.zero) & box.size;
      if (viewport != null && !viewport.overlaps(rect)) continue;
      targets.add(LibraryHintTarget(index: index, rect: rect));
    }
    return targets;
  }

  Rect? _viewportRect(RenderBox row) {
    final viewport = RenderAbstractViewport.maybeOf(row);
    if (viewport is! RenderBox) return null;
    final box = viewport as RenderBox;
    if (!box.hasSize) return null;
    return box.localToGlobal(Offset.zero) & box.size;
  }

  void _closeSwipe() {
    if (_openSwipeDocumentId == null) return;
    setState(() => _openSwipeDocumentId = null);
  }

  Future<bool> _markRead(DocumentSummary document) async {
    try {
      final repo = ref.read(libraryRepositoryProvider);
      await repo.markRead(document.id);
      if (!mounted) return false;
      ref.invalidate(documentDetailProvider(document.id));
      return true;
    } catch (error) {
      _showActionError(error);
      return false;
    }
  }

  Future<bool> _moveToTrash(DocumentSummary document) async {
    try {
      final repo = ref.read(libraryRepositoryProvider);
      await repo.moveDocumentToTrash(document.id);
      if (!mounted) return false;
      ref.invalidate(documentDetailProvider(document.id));
      return true;
    } catch (error) {
      _showActionError(error);
      return false;
    }
  }

  void _removeDocument(DocumentSummary document) {
    _closeSwipe();
    ref.read(librarySnapshotProvider.notifier).removeDocument(document.id);
  }

  void _showActionError(Object error) {
    if (!mounted) return;
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(error.toString())));
  }

  @override
  Widget build(BuildContext context) {
    ref.listen(libraryCursorProvider, (previous, next) {
      _revealCursor(next, downward: next >= (previous ?? -1));
    });
    final cursor = ref.watch(libraryCursorProvider);
    if (widget.documents.isEmpty) {
      return LibraryKeyboardCursor(
        length: 0,
        onOpen: _openDocument,
        hintTargets: _hintTargets,
        child: Center(
          child: Padding(
            padding: const EdgeInsets.all(AppSpacing.x3l),
            child: Text(
              'No documents match the current filters.',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
        ),
      );
    }
    return Stack(
      children: [
        LibraryKeyboardCursor(
          length: widget.documents.length,
          onOpen: _openDocument,
          hintTargets: _hintTargets,
          child: GestureDetector(
            behavior: HitTestBehavior.translucent,
            onTap: _closeSwipe,
            child: ListView.separated(
              controller: _scrollController,
              padding: const EdgeInsets.symmetric(
                horizontal: AppSpacing.lg,
                vertical: AppSpacing.md,
              ),
              itemCount: widget.documents.length + 2,
              separatorBuilder: (_, _) => const SizedBox(height: AppSpacing.xs),
              itemBuilder: (context, index) {
                if (index == 0) {
                  return Padding(
                    padding: const EdgeInsets.only(
                      left: AppSpacing.xs,
                      bottom: AppSpacing.sm,
                    ),
                    child: Text(
                      _documentCountText(widget.totalCount),
                      style: Theme.of(context).textTheme.labelSmall,
                    ),
                  );
                }
                if (index == widget.documents.length + 1) {
                  return LoadMoreRow(
                    hasMore: widget.hasMore,
                    isLoadingMore: widget.isLoadingMore,
                    error: widget.loadMoreError,
                    onPressed: widget.onLoadMore,
                  );
                }
                final document = widget.documents[index - 1];
                return KeyedSubtree(
                  key: _rowKeys.putIfAbsent(document.id, GlobalKey.new),
                  child: DocumentListTile(
                    key: ValueKey('document-list-tile-${document.id}'),
                    document: document,
                    isCursor: cursor == index - 1,
                    isSwipeOpen: _openSwipeDocumentId == document.id,
                    onSwipeOpen: () =>
                        setState(() => _openSwipeDocumentId = document.id),
                    onSwipeClose: _closeSwipe,
                    onMarkRead: () => _markRead(document),
                    onMoveToTrash: () => _moveToTrash(document),
                    onDismissed: () => _removeDocument(document),
                  ),
                );
              },
            ),
          ),
        ),
        Positioned(
          right: AppSpacing.lg,
          bottom: AppSpacing.lg,
          child: ScrollButtons(onTop: _scrollToTop, onBottom: _scrollToBottom),
        ),
      ],
    );
  }
}

String _documentCountText(int count) =>
    count == 1 ? '1 DOCUMENT' : '$count DOCUMENTS';
