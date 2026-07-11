import 'package:liquid2_api/liquid2_api.dart';

import 'feed_repository.dart';
import 'feed_snapshot.dart';

class ApiFeedRepository implements FeedRepository {
  const ApiFeedRepository(this.api);

  final Liquid2Api api;

  @override
  Future<FeedSnapshot> loadFeeds() async {
    final (feeds, folders, jobs, settings) = await (
      api.getFeedsApi().listFeeds(),
      api.getFoldersApi().listFolders(),
      api.getJobsApi().listJobs(kind: 'poll_feed', limit: 10),
      api.getSettingsApi().getSettings(),
    ).wait;
    return FeedSnapshot(
      feeds: feeds.data?.items?.toList() ?? const [],
      folders: folders.data?.items?.toList() ?? const [],
      jobs: jobs.data?.items?.toList() ?? const [],
      settings: _required(settings.data, 'Settings response was empty.'),
    );
  }

  @override
  Future<Feed> createFeed(FeedInput input) async {
    final response = await api.getFeedsApi().createFeed(
      createFeedInputBody: CreateFeedInputBody(
        (b) => b
          ..url = input.url.trim()
          ..title = _optionalText(input.title)
          ..folderId = _optionalText(input.folderId)
          ..enabled = input.enabled,
      ),
    );
    return _required(response.data, 'Feed response was empty.');
  }

  @override
  Future<Feed> updateFeed(String id, FeedInput input) async {
    final response = await api.getFeedsApi().updateFeed(
      id: id,
      updateFeedInputBody: UpdateFeedInputBody(
        (b) => b
          ..url = input.url.trim()
          ..title = _textOrEmpty(input.title)
          ..folderId = _textOrEmpty(input.folderId)
          ..enabled = input.enabled,
      ),
    );
    return _required(response.data, 'Feed response was empty.');
  }

  @override
  Future<void> deleteFeed(String id) async {
    await api.getFeedsApi().deleteFeed(id: id);
  }

  @override
  Future<Job> refreshFeed(String id) async {
    final response = await api.getFeedsApi().refreshFeed(id: id);
    return _required(response.data?.job, 'Feed refresh response was empty.');
  }

  @override
  Future<AppSettings> updateSettings(FeedSettingsInput input) async {
    final response = await api.getSettingsApi().updateSettings(
      updateSettingsInput: UpdateSettingsInput(
        (b) => b
          ..feedSchedulerEnabled = input.feedSchedulerEnabled
          ..feedPollIntervalSeconds = input.feedPollIntervalSeconds,
      ),
    );
    return _required(response.data, 'Settings response was empty.');
  }
}

T _required<T>(T? value, String message) {
  if (value == null) {
    throw StateError(message);
  }
  return value;
}

String? _optionalText(String? value) {
  final trimmed = value?.trim();
  if (trimmed == null || trimmed.isEmpty) {
    return null;
  }
  return trimmed;
}

String _textOrEmpty(String? value) => value?.trim() ?? '';
