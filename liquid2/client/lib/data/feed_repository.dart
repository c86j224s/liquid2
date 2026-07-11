import 'package:liquid2_api/liquid2_api.dart';

import 'feed_snapshot.dart';

abstract class FeedRepository {
  Future<FeedSnapshot> loadFeeds();

  Future<Feed> createFeed(FeedInput input);

  Future<Feed> updateFeed(String id, FeedInput input);

  Future<void> deleteFeed(String id);

  Future<Job> refreshFeed(String id);

  Future<AppSettings> updateSettings(FeedSettingsInput input);
}

class FeedInput {
  const FeedInput({
    required this.url,
    this.title,
    this.folderId,
    this.enabled = true,
  });

  final String url;
  final String? title;
  final String? folderId;
  final bool enabled;

  factory FeedInput.fromFeed(Feed feed, {required bool enabled}) {
    return FeedInput(
      url: feed.url,
      title: feed.title,
      folderId: feed.folderId,
      enabled: enabled,
    );
  }
}

class FeedSettingsInput {
  const FeedSettingsInput({
    required this.feedSchedulerEnabled,
    required this.feedPollIntervalSeconds,
  });

  final bool feedSchedulerEnabled;
  final int feedPollIntervalSeconds;
}
