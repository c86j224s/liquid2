import 'package:liquid2_api/liquid2_api.dart';

class FeedSnapshot {
  const FeedSnapshot({
    required this.feeds,
    required this.folders,
    required this.jobs,
    required this.settings,
  });

  final List<Feed> feeds;
  final List<Folder> folders;
  final List<Job> jobs;
  final AppSettings settings;

  FeedSnapshot copyWith({
    List<Feed>? feeds,
    List<Folder>? folders,
    List<Job>? jobs,
    AppSettings? settings,
  }) {
    return FeedSnapshot(
      feeds: feeds ?? this.feeds,
      folders: folders ?? this.folders,
      jobs: jobs ?? this.jobs,
      settings: settings ?? this.settings,
    );
  }
}
