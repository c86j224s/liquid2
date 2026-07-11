import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';

import 'fake_library_repository.dart';

void main() {
  testWidgets('shows the exact filtered document count', (tester) async {
    final repository = FakeLibraryRepository(hasSecondPage: true);
    await tester.pumpWidget(
      ProviderScope(
        overrides: [libraryRepositoryProvider.overrideWithValue(repository)],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    expect(find.text('2 DOCUMENTS'), findsOneWidget);
  });
}
