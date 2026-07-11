import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/feed_repository.dart';
import 'package:liquid2_client/data/feed_snapshot.dart';

class FakeFeedRepository implements FeedRepository {
  final feeds = <Feed>[_feed()];
  final jobs = <Job>[_job(status: 'failed', error: 'job failed')];
  final inputs = <FeedInput>[];
  final settingsInputs = <FeedSettingsInput>[];
  var deletedFeedIds = <String>[];
  var refreshCount = 0;
  var settings = _settings();

  @override
  Future<FeedSnapshot> loadFeeds() async {
    return FeedSnapshot(
      feeds: [...feeds],
      folders: [_folder()],
      jobs: [...jobs],
      settings: settings,
    );
  }

  @override
  Future<Feed> createFeed(FeedInput input) async {
    inputs.add(input);
    final feed = _feed(
      id: 'feed_${feeds.length + 1}',
      url: input.url,
      title: input.title,
      enabled: input.enabled,
      folderId: input.folderId,
    );
    feeds.add(feed);
    return feed;
  }

  @override
  Future<Feed> updateFeed(String id, FeedInput input) async {
    inputs.add(input);
    final index = feeds.indexWhere((feed) => feed.id == id);
    final updated = _feed(
      id: id,
      url: input.url,
      title: input.title,
      enabled: input.enabled,
      folderId: input.folderId,
    );
    feeds[index] = updated;
    return updated;
  }

  @override
  Future<void> deleteFeed(String id) async {
    deletedFeedIds.add(id);
    feeds.removeWhere((feed) => feed.id == id);
  }

  @override
  Future<Job> refreshFeed(String id) async {
    refreshCount++;
    final job = _job(status: 'queued', error: null);
    jobs.insert(0, job);
    return job;
  }

  @override
  Future<AppSettings> updateSettings(FeedSettingsInput input) async {
    settingsInputs.add(input);
    settings = _settings(
      enabled: input.feedSchedulerEnabled,
      interval: input.feedPollIntervalSeconds,
    );
    return settings;
  }
}

Feed _feed({
  String id = 'feed_1',
  String url = 'https://example.com/feed.xml',
  String? title = 'Example Feed',
  bool enabled = true,
  String? folderId = 'folder_1',
}) {
  return Feed(
    (b) => b
      ..id = id
      ..url = url
      ..title = title
      ..enabled = enabled
      ..folderId = folderId
      ..createdAt = _now
      ..updatedAt = _now,
  );
}

Folder _folder() {
  return Folder(
    (b) => b
      ..id = 'folder_1'
      ..name = 'Inbox'
      ..sortOrder = 0
      ..createdAt = _now
      ..updatedAt = _now,
  );
}

Job _job({required String status, String? error}) {
  return Job(
    (b) => b
      ..id = 'job_1'
      ..kind = 'poll_feed'
      ..status = status
      ..error = error
      ..attempts = 1
      ..createdAt = _now
      ..updatedAt = _now,
  );
}

AppSettings _settings({bool enabled = false, int interval = 7200}) {
  return AppSettings(
    (b) => b
      ..feedSchedulerEnabled = enabled
      ..feedPollIntervalSeconds = interval
      ..feedNextPollAt = enabled ? _now + interval * 1000 : null
      ..updatedAt = _now,
  );
}

const _now = 1760000000000;
