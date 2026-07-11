import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import 'feed_card.dart';

class FeedListPanel extends StatelessWidget {
  const FeedListPanel({
    required this.feeds,
    required this.folders,
    required this.refreshingFeedIds,
    required this.onCreate,
    required this.onEdit,
    required this.onToggle,
    required this.onDelete,
    required this.onRefresh,
    super.key,
  });

  final List<Feed> feeds;
  final List<Folder> folders;
  final Set<String> refreshingFeedIds;
  final VoidCallback onCreate;
  final ValueChanged<Feed> onEdit;
  final ValueChanged<Feed> onToggle;
  final ValueChanged<Feed> onDelete;
  final ValueChanged<Feed> onRefresh;

  @override
  Widget build(BuildContext context) {
    if (feeds.isEmpty) {
      return Center(
        child: FilledButton.icon(
          onPressed: onCreate,
          icon: const Icon(Icons.rss_feed),
          label: const Text('Add feed'),
        ),
      );
    }
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: feeds.length,
      separatorBuilder: (_, _) => const SizedBox(height: 12),
      itemBuilder: (context, index) {
        final feed = feeds[index];
        return FeedCard(
          feed: feed,
          folderName: _folderName(feed.folderId),
          refreshing: refreshingFeedIds.contains(feed.id),
          onEdit: () => onEdit(feed),
          onToggle: () => onToggle(feed),
          onDelete: () => onDelete(feed),
          onRefresh: () => onRefresh(feed),
        );
      },
    );
  }

  String? _folderName(String? id) {
    if (id == null) {
      return null;
    }
    for (final folder in folders) {
      final match = _findFolder(folder, id);
      if (match != null) {
        return match.name;
      }
    }
    return null;
  }

  Folder? _findFolder(Folder folder, String id) {
    if (folder.id == id) {
      return folder;
    }
    for (final child in folder.children?.toList() ?? const <Folder>[]) {
      final match = _findFolder(child, id);
      if (match != null) {
        return match;
      }
    }
    return null;
  }
}
