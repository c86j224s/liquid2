import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for FeedsApi
void main() {
  final instance = Liquid2Api().getFeedsApi();

  group(FeedsApi, () {
    // Create feed
    //
    //Future<Feed> createFeed(CreateFeedInputBody createFeedInputBody) async
    test('test createFeed', () async {
      // TODO
    });

    // Delete feed
    //
    //Future deleteFeed(String id) async
    test('test deleteFeed', () async {
      // TODO
    });

    // List feeds
    //
    //Future<FeedListOutputBody> listFeeds() async
    test('test listFeeds', () async {
      // TODO
    });

    // Refresh feed
    //
    //Future<FeedRefreshOutputBody> refreshFeed(String id) async
    test('test refreshFeed', () async {
      // TODO
    });

    // Update feed
    //
    //Future<Feed> updateFeed(String id, UpdateFeedInputBody updateFeedInputBody) async
    test('test updateFeed', () async {
      // TODO
    });

  });
}
