import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/providers.dart';
import '../../data/feed_repository.dart';
import '../../data/feed_snapshot.dart';

final feedDashboardProvider =
    AsyncNotifierProvider<FeedDashboardController, FeedSnapshot>(
      FeedDashboardController.new,
    );

class FeedDashboardController extends AsyncNotifier<FeedSnapshot> {
  @override
  Future<FeedSnapshot> build() {
    return ref.watch(feedRepositoryProvider).loadFeeds();
  }

  Future<void> createFeed(FeedInput input) async {
    await ref.read(feedRepositoryProvider).createFeed(input);
    ref.invalidateSelf();
  }

  Future<void> updateFeed(String id, FeedInput input) async {
    await ref.read(feedRepositoryProvider).updateFeed(id, input);
    ref.invalidateSelf();
  }

  Future<void> deleteFeed(String id) async {
    await ref.read(feedRepositoryProvider).deleteFeed(id);
    ref.invalidateSelf();
  }

  Future<Job> refreshFeed(String id) async {
    final job = await ref.read(feedRepositoryProvider).refreshFeed(id);
    ref.invalidate(librarySnapshotProvider);
    ref.invalidateSelf();
    return job;
  }

  Future<void> updateSettings(FeedSettingsInput input) async {
    final settings = await ref
        .read(feedRepositoryProvider)
        .updateSettings(input);
    final current = state.value;
    if (current == null) {
      ref.invalidateSelf();
      return;
    }
    state = AsyncData(current.copyWith(settings: settings));
  }
}
