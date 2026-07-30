import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/app_theme.dart';
import 'document_list_swipe_actions.dart';
import 'document_list_tile_body.dart';

class DocumentListTile extends StatefulWidget {
  const DocumentListTile({
    required this.document,
    required this.isSwipeOpen,
    required this.onSwipeOpen,
    required this.onSwipeClose,
    required this.onMarkRead,
    required this.onMoveToTrash,
    required this.onDismissed,
    super.key,
  });

  final DocumentSummary document;
  final bool isSwipeOpen;
  final VoidCallback onSwipeOpen;
  final VoidCallback onSwipeClose;
  final Future<bool> Function() onMarkRead;
  final Future<bool> Function() onMoveToTrash;
  final VoidCallback onDismissed;

  @override
  State<DocumentListTile> createState() => _DocumentListTileState();
}

class _DocumentListTileState extends State<DocumentListTile> {
  static const _actionWidth = 92.0;
  static const _openThreshold = 42.0;
  static const _exitDuration = Duration(milliseconds: 220);
  static const _collapseDuration = Duration(milliseconds: 120);

  var _dragOffset = 0.0;
  var _dragging = false;
  var _exiting = false;
  var _collapsed = false;

  @override
  void didUpdateWidget(DocumentListTile oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (!widget.isSwipeOpen && oldWidget.isSwipeOpen) {
      _dragOffset = 0;
    }
  }

  void _handleDragStart(DragStartDetails details) {
    if (_exiting) return;
    _dragging = true;
    _dragOffset = widget.isSwipeOpen ? _actionWidth : 0;
  }

  void _handleDragUpdate(DragUpdateDetails details) {
    if (_exiting) return;
    final nextOffset = (_dragOffset - details.delta.dx).clamp(
      0.0,
      _actionWidth,
    );
    setState(() => _dragOffset = nextOffset);
  }

  void _handleDragEnd(DragEndDetails details) {
    if (_exiting) return;
    final shouldOpen =
        _dragOffset >= _openThreshold ||
        details.primaryVelocity != null && details.primaryVelocity! < -350;
    setState(() {
      _dragging = false;
      _dragOffset = shouldOpen ? _actionWidth : 0;
    });
    if (shouldOpen) {
      widget.onSwipeOpen();
    } else {
      widget.onSwipeClose();
    }
  }

  void _handleTap(BuildContext context) {
    if (_exiting) return;
    if (widget.isSwipeOpen) {
      widget.onSwipeClose();
      return;
    }
    context.go('/documents/${widget.document.id}');
  }

  Future<void> _runDismissAction(Future<bool> Function() action) async {
    if (_exiting) return;
    final success = await action();
    if (!mounted || !success) return;
    setState(() => _exiting = true);
    await Future<void>.delayed(_exitDuration);
    if (!mounted) return;
    setState(() => _collapsed = true);
    await Future<void>.delayed(_collapseDuration);
    if (mounted) widget.onDismissed();
  }

  @override
  Widget build(BuildContext context) {
    final width = MediaQuery.sizeOf(context).width;
    final offset = _exiting
        ? width + _actionWidth
        : widget.isSwipeOpen
        ? _actionWidth
        : _dragOffset;
    final showActions = widget.isSwipeOpen || _dragOffset > 0 || _exiting;
    final duration = _dragging
        ? Duration.zero
        : _exiting
        ? _exitDuration
        : const Duration(milliseconds: 160);
    final curve = _exiting ? Curves.easeInCubic : Curves.easeOutCubic;

    return AnimatedSize(
      duration: _collapseDuration,
      curve: Curves.easeOutCubic,
      alignment: Alignment.topCenter,
      child: _collapsed
          ? const SizedBox.shrink()
          : ClipRRect(
              borderRadius: const BorderRadius.all(AppRadius.md),
              child: Stack(
                children: [
                  if (showActions)
                    Positioned.fill(
                      child: DocumentSwipeActions(
                        documentId: widget.document.id,
                        onMarkRead: () => _runDismissAction(widget.onMarkRead),
                        onMoveToTrash: () =>
                            _runDismissAction(widget.onMoveToTrash),
                      ),
                    ),
                  GestureDetector(
                    onHorizontalDragStart: _handleDragStart,
                    onHorizontalDragUpdate: _handleDragUpdate,
                    onHorizontalDragEnd: _handleDragEnd,
                    child: AnimatedContainer(
                      duration: duration,
                      curve: curve,
                      transform: Matrix4.translationValues(-offset, 0, 0),
                      child: AnimatedOpacity(
                        duration: duration,
                        curve: curve,
                        opacity: _exiting ? 0 : 1,
                        child: DocumentTileSurface(
                          key: Key('document-tile-${widget.document.id}'),
                          document: widget.document,
                          onTap: () => _handleTap(context),
                        ),
                      ),
                    ),
                  ),
                ],
              ),
            ),
    );
  }
}
