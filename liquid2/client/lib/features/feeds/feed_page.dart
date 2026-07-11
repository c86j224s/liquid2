import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../data/feed_repository.dart';
import '../../data/feed_snapshot.dart';
import '../../shared/action_feedback.dart';
import '../../shared/async_panel.dart';
import 'feed_dashboard_controller.dart';
import 'feed_dashboard_layout.dart';
import 'feed_delete_dialog.dart';
import 'feed_form_dialog.dart';
import 'feed_jobs_panel.dart';
import 'feed_list_panel.dart';
import 'feed_scheduler_settings_panel.dart';
import 'feed_section_header.dart';

class FeedPage extends ConsumerStatefulWidget {
  const FeedPage({super.key});

  @override
  ConsumerState<FeedPage> createState() => _FeedPageState();
}

class _FeedPageState extends ConsumerState<FeedPage> {
  final _refreshingFeedIds = <String>{};

  @override
  Widget build(BuildContext context) {
    final snapshot = ref.watch(feedDashboardProvider);
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'Back',
          onPressed: () => context.go('/'),
          icon: const Icon(Icons.arrow_back),
        ),
        title: const Text('Feeds'),
        actions: [
          IconButton(
            tooltip: 'Add feed',
            onPressed: () => _showCreate(snapshot.value),
            icon: const Icon(Icons.add),
          ),
          IconButton(
            tooltip: 'Refresh',
            onPressed: () => ref.invalidate(feedDashboardProvider),
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: AsyncPanel(
        value: snapshot,
        onRetry: () => ref.invalidate(feedDashboardProvider),
        builder: _buildDashboard,
      ),
    );
  }

  Widget _buildDashboard(FeedSnapshot snapshot) {
    return FeedDashboardLayout(
      feeds: _feedsPanel(snapshot),
      jobs: _jobsPanel(snapshot),
    );
  }

  Widget _feedsPanel(FeedSnapshot snapshot) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        FeedSectionHeader(
          title: 'Feeds',
          action: FilledButton.icon(
            onPressed: () => _showCreate(snapshot),
            icon: const Icon(Icons.add),
            label: const Text('Add feed'),
          ),
        ),
        FeedSchedulerSettingsPanel(
          settings: snapshot.settings,
          onEnabledChanged: (enabled) => _updateScheduler(enabled: enabled),
          onIntervalChanged: (seconds) => _updateScheduler(interval: seconds),
        ),
        Expanded(
          child: FeedListPanel(
            feeds: snapshot.feeds,
            folders: snapshot.folders,
            refreshingFeedIds: _refreshingFeedIds,
            onCreate: () => _showCreate(snapshot),
            onEdit: (feed) => _showEdit(snapshot, feed),
            onToggle: _toggleFeed,
            onDelete: _deleteFeed,
            onRefresh: _refreshFeed,
          ),
        ),
      ],
    );
  }

  Widget _jobsPanel(FeedSnapshot snapshot) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        const FeedSectionHeader(title: 'Recent RSS jobs'),
        Expanded(child: FeedJobsPanel(jobs: snapshot.jobs)),
      ],
    );
  }

  Future<void> _showCreate(FeedSnapshot? snapshot) async {
    final input = await showFeedFormDialog(
      context: context,
      folders: snapshot?.folders ?? const [],
    );
    if (input == null || !mounted) {
      return;
    }
    await runUiAction(context, () async {
      await ref.read(feedDashboardProvider.notifier).createFeed(input);
    });
  }

  Future<void> _showEdit(FeedSnapshot snapshot, Feed feed) async {
    final input = await showFeedFormDialog(
      context: context,
      folders: snapshot.folders,
      feed: feed,
    );
    if (input == null || !mounted) {
      return;
    }
    await runUiAction(context, () async {
      await ref.read(feedDashboardProvider.notifier).updateFeed(feed.id, input);
    });
  }

  Future<void> _toggleFeed(Feed feed) {
    return runUiAction(context, () async {
      await ref
          .read(feedDashboardProvider.notifier)
          .updateFeed(
            feed.id,
            FeedInput.fromFeed(feed, enabled: !feed.enabled),
          );
    });
  }

  Future<void> _deleteFeed(Feed feed) async {
    if (!await confirmFeedDelete(context, feed) || !mounted) {
      return;
    }
    await runUiAction(context, () async {
      await ref.read(feedDashboardProvider.notifier).deleteFeed(feed.id);
    });
  }

  Future<void> _refreshFeed(Feed feed) async {
    setState(() => _refreshingFeedIds.add(feed.id));
    await runUiAction(context, () async {
      final job = await ref
          .read(feedDashboardProvider.notifier)
          .refreshFeed(feed.id);
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text('Refresh queued: ${job.id}')));
      }
    });
    if (mounted) {
      setState(() => _refreshingFeedIds.remove(feed.id));
    }
  }

  Future<void> _updateScheduler({bool? enabled, int? interval}) {
    final settings = ref.read(feedDashboardProvider).value?.settings;
    if (settings == null) {
      return Future.value();
    }
    return runUiAction(context, () async {
      await ref
          .read(feedDashboardProvider.notifier)
          .updateSettings(
            FeedSettingsInput(
              feedSchedulerEnabled: enabled ?? settings.feedSchedulerEnabled,
              feedPollIntervalSeconds:
                  interval ?? settings.feedPollIntervalSeconds,
            ),
          );
    });
  }
}
