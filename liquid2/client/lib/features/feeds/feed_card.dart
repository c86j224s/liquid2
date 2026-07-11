import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../shared/formatters.dart';

class FeedCard extends StatelessWidget {
  const FeedCard({
    required this.feed,
    required this.refreshing,
    required this.onEdit,
    required this.onToggle,
    required this.onDelete,
    required this.onRefresh,
    this.folderName,
    super.key,
  });

  final Feed feed;
  final String? folderName;
  final bool refreshing;
  final VoidCallback onEdit;
  final VoidCallback onToggle;
  final VoidCallback onDelete;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Card(
      margin: EdgeInsets.zero,
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(
                  feed.enabled ? Icons.rss_feed : Icons.pause_circle_outline,
                  color: feed.enabled ? colors.primary : colors.outline,
                ),
                const SizedBox(width: 12),
                Expanded(child: _FeedTitle(feed: feed)),
                _FeedActions(
                  enabled: feed.enabled,
                  refreshing: refreshing,
                  onEdit: onEdit,
                  onToggle: onToggle,
                  onDelete: onDelete,
                  onRefresh: onRefresh,
                ),
              ],
            ),
            const SizedBox(height: 10),
            Wrap(
              spacing: 8,
              runSpacing: 8,
              children: [
                _InfoChip(
                  icon: Icons.folder_outlined,
                  label: folderName ?? 'No folder',
                ),
                _InfoChip(
                  icon: Icons.schedule,
                  label: 'Checked ${formatMillis(feed.lastCheckedAt)}',
                ),
                _InfoChip(
                  icon: feed.enabled ? Icons.check_circle : Icons.pause,
                  label: feed.enabled ? 'Enabled' : 'Disabled',
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _FeedTitle extends StatelessWidget {
  const _FeedTitle({required this.feed});

  final Feed feed;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          feed.title?.trim().isNotEmpty == true ? feed.title! : feed.url,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: Theme.of(context).textTheme.titleMedium,
        ),
        const SizedBox(height: 4),
        Text(
          feed.url,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
          style: Theme.of(context).textTheme.bodySmall,
        ),
      ],
    );
  }
}

class _FeedActions extends StatelessWidget {
  const _FeedActions({
    required this.enabled,
    required this.refreshing,
    required this.onEdit,
    required this.onToggle,
    required this.onDelete,
    required this.onRefresh,
  });

  final bool enabled;
  final bool refreshing;
  final VoidCallback onEdit;
  final VoidCallback onToggle;
  final VoidCallback onDelete;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: 2,
      children: [
        IconButton(
          tooltip: 'Refresh feed',
          onPressed: enabled && !refreshing ? onRefresh : null,
          icon: refreshing
              ? const SizedBox.square(
                  dimension: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.refresh),
        ),
        IconButton(
          tooltip: enabled ? 'Disable feed' : 'Enable feed',
          onPressed: onToggle,
          icon: Icon(enabled ? Icons.pause : Icons.play_arrow),
        ),
        IconButton(
          tooltip: 'Edit feed',
          onPressed: onEdit,
          icon: const Icon(Icons.edit),
        ),
        IconButton(
          tooltip: 'Delete feed',
          onPressed: onDelete,
          icon: const Icon(Icons.delete_outline),
        ),
      ],
    );
  }
}

class _InfoChip extends StatelessWidget {
  const _InfoChip({required this.icon, required this.label});

  final IconData icon;
  final String label;

  @override
  Widget build(BuildContext context) {
    return Chip(
      avatar: Icon(icon, size: 16),
      label: Text(label),
      visualDensity: VisualDensity.compact,
    );
  }
}
