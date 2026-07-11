import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';

import 'fake_folder_repository.dart';
import 'fake_library_repository.dart';
import 'test_viewports.dart';

void main() {
  testWidgets('creates a folder from the library folder manager', (
    tester,
  ) async {
    await setDesktopViewport(tester);
    final libraryRepository = FakeLibraryRepository();
    final folderRepository = FakeFolderRepository(
      folders: [
        fakeFolder('folder_1', 'Inbox', systemRole: 'inbox'),
        fakeFolder('folder_trash', 'Trash', systemRole: 'trash'),
      ],
    );
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          libraryRepositoryProvider.overrideWithValue(libraryRepository),
          folderRepositoryProvider.overrideWithValue(folderRepository),
        ],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    await tester.ensureVisible(find.byTooltip('Manage folders'));
    await tester.tap(find.byTooltip('Manage folders'));
    await tester.pumpAndSettle();

    await tester.tap(find.byTooltip('Create folder'));
    await tester.pumpAndSettle();
    await tester.enterText(
      find.widgetWithText(TextFormField, 'Name'),
      'Projects',
    );
    await tester.tap(find.widgetWithText(FilledButton, 'Save'));
    await tester.pumpAndSettle();

    expect(folderRepository.created.single.name, 'Projects');
    expect(find.text('Projects'), findsOneWidget);
  });
}
