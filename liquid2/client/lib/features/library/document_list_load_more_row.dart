import 'package:flutter/material.dart';

import '../../app/app_theme.dart';

class LoadMoreRow extends StatelessWidget {
  const LoadMoreRow({
    required this.hasMore,
    required this.isLoadingMore,
    required this.onPressed,
    this.error,
    super.key,
  });

  final bool hasMore;
  final bool isLoadingMore;
  final Object? error;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    if (!hasMore && error == null) return const SizedBox.shrink();
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: AppSpacing.md),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (error != null) const _LoadMoreErrorLabel(),
            TextButton.icon(
              onPressed: isLoadingMore ? null : onPressed,
              icon: isLoadingMore
                  ? const SizedBox.square(
                      dimension: 14,
                      child: CircularProgressIndicator(strokeWidth: 1.5),
                    )
                  : const Icon(Icons.expand_more, size: 16),
              label: Text(isLoadingMore ? 'Loading…' : 'Load more'),
            ),
          ],
        ),
      ),
    );
  }
}

class _LoadMoreErrorLabel extends StatelessWidget {
  const _LoadMoreErrorLabel();

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: AppSpacing.sm),
      child: Text(
        'Could not load more documents.',
        style: TextStyle(
          color: Theme.of(context).colorScheme.error,
          fontSize: 12,
        ),
      ),
    );
  }
}
