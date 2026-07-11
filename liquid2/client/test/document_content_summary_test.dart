import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/app/liquid2_app.dart';
import 'package:liquid2_client/app/providers.dart';

import 'fake_library_repository.dart';

void main() {
  testWidgets('shows readable content instead of raw html', (tester) async {
    await tester.pumpWidget(
      ProviderScope(
        overrides: [
          libraryRepositoryProvider.overrideWithValue(_HtmlContentRepository()),
        ],
        child: const Liquid2App(),
      ),
    );
    await tester.pumpAndSettle();

    await tester.tap(find.text('SQLite notes'));
    await tester.pumpAndSettle();

    expect(find.text('- Build small tools & ship'), findsOneWidget);
    expect(find.textContaining('<strong>'), findsNothing);
    expect(find.textContaining('&amp;'), findsNothing);
  });
}

class _HtmlContentRepository extends FakeLibraryRepository {
  @override
  Future<DocumentDetail> getDocument(String id) async {
    return DocumentDetail(
      (b) => b
        ..document.replace(_metadata())
        ..contents.add(
          DocumentContent(
            (c) => c
              ..id = 'content_1'
              ..role = 'extracted'
              ..format = 'html'
              ..content =
                  '<ul><li>Build <strong>small</strong> tools &amp; ship</li></ul>',
          ),
        ),
    );
  }

  DocumentMetadata _metadata() {
    return DocumentMetadata(
      (b) => b
        ..id = 'doc_1'
        ..title = 'SQLite notes'
        ..kind = 'rss_item'
        ..status = 'unread'
        ..createdAt = _now
        ..updatedAt = _now,
    );
  }
}

const _now = 1760000000000;
