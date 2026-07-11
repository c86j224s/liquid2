import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/providers.dart';
import 'package:liquid2_client/features/document/document_detail_page.dart';

import 'fake_library_repository.dart';

void main() {
  testWidgets('detail floating actions scroll to bottom and top', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(390, 500);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(() {
      tester.view.resetPhysicalSize();
      tester.view.resetDevicePixelRatio();
    });

    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          libraryRepositoryProvider.overrideWithValue(FakeLibraryRepository()),
        ],
        child: const MaterialApp(home: DocumentDetailPage(id: 'doc_1')),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.byTooltip('Scroll to top'), findsOneWidget);
    expect(find.byTooltip('Scroll to bottom'), findsOneWidget);
    expect(_scrollPixels(tester), 0);

    await tester.tap(find.byTooltip('Scroll to bottom'));
    await tester.pumpAndSettle();
    expect(_scrollPixels(tester), greaterThan(0));

    await tester.tap(find.byTooltip('Scroll to top'));
    await tester.pumpAndSettle();
    expect(_scrollPixels(tester), 0);
  });
}

double _scrollPixels(WidgetTester tester) {
  final listView = tester.widget<ListView>(
    find.byKey(const Key('document-detail-scroll-view')),
  );
  return listView.controller!.position.pixels;
}
