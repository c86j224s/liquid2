import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';

import 'fake_feed_repository.dart';
import 'fake_library_repository.dart';

void main() {
  testWidgets('manages feeds from the feed screen', (tester) async {
    final feeds = FakeFeedRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          libraryRepositoryProvider.overrideWithValue(FakeLibraryRepository()),
          feedRepositoryProvider.overrideWithValue(feeds),
        ],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Feeds'));
    await tester.pumpAndSettle();

    expect(find.text('Example Feed'), findsOneWidget);
    expect(find.textContaining('job failed'), findsOneWidget);
    expect(find.text('Auto refresh RSS'), findsOneWidget);
    expect(find.textContaining('Next check'), findsOneWidget);

    await tester.tap(find.byTooltip('Enable RSS auto refresh'));
    await tester.pumpAndSettle();
    expect(feeds.settingsInputs.last.feedSchedulerEnabled, isTrue);

    await tester.tap(find.byType(DropdownButtonFormField<int>));
    await tester.pumpAndSettle();
    await tester.tap(find.text('15m').last);
    await tester.pumpAndSettle();
    expect(feeds.settingsInputs.last.feedPollIntervalSeconds, 900);

    await tester.tap(find.byTooltip('Add feed').first);
    await tester.pumpAndSettle();
    await tester.enterText(
      find.byType(TextFormField).at(0),
      'https://example.com/next.xml',
    );
    await tester.enterText(find.byType(TextFormField).at(1), 'Next Feed');
    await tester.tap(find.text('Create'));
    await tester.pumpAndSettle();

    expect(feeds.inputs.last.url, 'https://example.com/next.xml');
    expect(find.text('Next Feed'), findsOneWidget);

    await tester.tap(find.byTooltip('Refresh feed').first);
    await tester.pumpAndSettle();
    expect(feeds.refreshCount, 1);
    expect(find.textContaining('Refresh queued'), findsOneWidget);

    await tester.tap(find.byTooltip('Disable feed').first);
    await tester.pumpAndSettle();
    expect(feeds.inputs.last.enabled, isFalse);

    await tester.tap(find.byTooltip('Delete feed').first);
    await tester.pumpAndSettle();
    await tester.tap(find.text('Delete'));
    await tester.pumpAndSettle();
    expect(feeds.deletedFeedIds, contains('feed_1'));
  });
}
