import 'package:flutter/material.dart';

import '../../app/app_theme.dart';

class DocumentSwipeActions extends StatelessWidget {
  const DocumentSwipeActions({
    required this.documentId,
    required this.onMarkRead,
    required this.onMoveToTrash,
    super.key,
  });

  final String documentId;
  final Future<void> Function() onMarkRead;
  final Future<void> Function() onMoveToTrash;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return ColoredBox(
      color: theme.colorScheme.surfaceContainerHighest,
      child: Align(
        alignment: Alignment.centerRight,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: AppSpacing.sm),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              _SwipeActionButton(
                key: Key('document-swipe-read-$documentId'),
                tooltip: 'Mark read',
                icon: Icons.done,
                color: theme.colorScheme.primary,
                backgroundColor: theme.colorScheme.primaryContainer,
                onPressed: onMarkRead,
              ),
              const SizedBox(width: AppSpacing.xs),
              _SwipeActionButton(
                key: Key('document-swipe-trash-$documentId'),
                tooltip: 'Move to trash',
                icon: Icons.delete_outline,
                color: theme.colorScheme.onErrorContainer,
                backgroundColor: theme.colorScheme.errorContainer,
                onPressed: onMoveToTrash,
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _SwipeActionButton extends StatelessWidget {
  const _SwipeActionButton({
    required super.key,
    required this.tooltip,
    required this.icon,
    required this.color,
    required this.backgroundColor,
    required this.onPressed,
  });

  final String tooltip;
  final IconData icon;
  final Color color;
  final Color backgroundColor;
  final Future<void> Function() onPressed;

  @override
  Widget build(BuildContext context) {
    return IconButton(
      tooltip: tooltip,
      style: IconButton.styleFrom(
        fixedSize: const Size(36, 36),
        minimumSize: const Size(36, 36),
        padding: EdgeInsets.zero,
        foregroundColor: color,
        backgroundColor: backgroundColor,
      ),
      onPressed: () {
        onPressed();
      },
      icon: Icon(icon),
    );
  }
}
