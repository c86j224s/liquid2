import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/app_theme.dart';
import 'document_list_load_more_row.dart';
import 'document_list_scroll_buttons.dart';
import 'document_list_tile.dart';

class DocumentListPanel extends StatefulWidget {
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
  State<DocumentListPanel> createState() => _DocumentListPanelState();
}

class _DocumentListPanelState extends State<DocumentListPanel> {
  final _scrollController = ScrollController();

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

  @override
  Widget build(BuildContext context) {
    if (widget.documents.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(AppSpacing.x3l),
          child: Text(
            'No documents match the current filters.',
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ),
      );
    }
    return Stack(
      children: [
        ListView.separated(
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
            return DocumentListTile(document: widget.documents[index - 1]);
          },
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
