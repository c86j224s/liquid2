import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/environment_badge.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/data/library_filters.dart';

import 'failing_once_library_repository.dart';
import 'fake_library_repository.dart';
import 'test_viewports.dart';

void main() {
  testWidgets('browses and updates document detail', (tester) async {
    await setDesktopViewport(tester);
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();
    expect(find.text('SQLite notes'), findsOneWidget);
    expect(find.text('go'), findsWidgets);
    await tester.tap(find.text('SQLite notes'));
    await tester.pumpAndSettle();
    expect(find.text('Document'), findsOneWidget);
    expect(find.text('Mark read'), findsOneWidget);
    await tester.tap(find.byIcon(Icons.star_outline_rounded).last);
    await tester.pumpAndSettle();
    expect(repository.rating, 5);

    await tester.tap(find.byIcon(Icons.star_rounded).last);
    await tester.pumpAndSettle();
    expect(repository.rating, isNull);

    final noteField = find.widgetWithText(TextField, 'New note');
    await tester.dragUntilVisible(
      noteField,
      find.byType(ListView),
      const Offset(0, -500),
    );
    await tester.enterText(noteField, 'Follow up');
    await tester.tap(find.text('Add'));
    await tester.pumpAndSettle();
    expect(find.text('Follow up'), findsOneWidget);
  });

  testWidgets('loads next document page through repository', (tester) async {
    await setDesktopViewport(tester);
    final repository = FakeLibraryRepository(hasSecondPage: true);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('SQLite notes'), findsOneWidget);
    expect(find.text('Second document'), findsNothing);

    await tester.tap(find.text('Load more'));
    await tester.pumpAndSettle();

    expect(repository.requestedCursors, contains('page_2'));
    expect(find.text('Second document'), findsOneWidget);
  });

  testWidgets('shows recursive folder children as a directory tree', (
    tester,
  ) async {
    await setDesktopViewport(tester);
    final repository = FakeLibraryRepository(includeChildFolder: true);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('VIEWS'), findsOneWidget);
    expect(find.text('All documents'), findsOneWidget);
    expect(find.text('FOLDERS'), findsOneWidget);
    expect(find.text('Inbox'), findsOneWidget);
    expect(find.text('Research'), findsOneWidget);
    expect(find.byIcon(Icons.library_books), findsWidgets);
    expect(find.byIcon(Icons.folder), findsWidgets);
    expect(find.byIcon(Icons.subdirectory_arrow_right), findsNothing);
  });

  testWidgets('search field updates library filters after debounce', (
    tester,
  ) async {
    await setDesktopViewport(tester);
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    final initialLoads = repository.requestedFilters.length;
    await tester.enterText(
      find.byKey(const Key('library-search-field')),
      'missing',
    );
    await tester.pump(const Duration(milliseconds: 200));
    expect(repository.requestedFilters.length, initialLoads);

    await tester.pump(const Duration(milliseconds: 200));
    await tester.pumpAndSettle();

    expect(find.text('SQLite notes'), findsNothing);
    expect(repository.requestedFilters.last.query, 'missing');
  });

  testWidgets('fixed view presets compose library filters', (tester) async {
    await setDesktopViewport(tester);
    final repository = FakeLibraryRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.widgetWithText(ChoiceChip, 'Rated'));
    await tester.pumpAndSettle();

    final filters = repository.requestedFilters.last;
    expect(filters.view, LibraryViewPreset.rated);
    expect(filters.ratingMin, 1);
    expect(filters.sort, DocumentSort.ratingDesc);
  });

  testWidgets('retries the document list after an initial load failure', (
    tester,
  ) async {
    await setDesktopViewport(tester);
    final repository = FailingOnceLibraryRepository();
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.textContaining('api offline'), findsOneWidget);
    expect(find.text('Copy error'), findsOneWidget);

    final clipboardCalls = <MethodCall>[];
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(SystemChannels.platform, (call) async {
          clipboardCalls.add(call);
          return null;
        });
    addTearDown(() {
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(SystemChannels.platform, null);
    });

    await tester.tap(find.text('Copy error'));
    await tester.pumpAndSettle();

    expect(
      clipboardCalls.any(
        (call) =>
            call.method == 'Clipboard.setData' &&
            (call.arguments as Map<Object?, Object?>)['text']
                .toString()
                .contains('api offline'),
      ),
      isTrue,
    );

    await tester.tap(find.text('Retry'));
    await tester.pumpAndSettle();

    expect(find.text('SQLite notes'), findsOneWidget);
    expect(repository.loadAttempts, 2);
  });

  testWidgets(
    'shows optional environment badge without replacing app content',
    (tester) async {
      await tester.pumpWidget(
        const MaterialApp(
          home: EnvironmentBadgeOverlay(
            label: 'DEV',
            child: Text('Liquid2 content'),
          ),
        ),
      );

      expect(find.text('Liquid2 content'), findsOneWidget);
      expect(find.text('DEV'), findsOneWidget);
    },
  );
}
