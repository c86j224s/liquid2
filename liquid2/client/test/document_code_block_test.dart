import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/features/document/document_code_block.dart';
import 'package:liquid2_client/features/document/document_content_view.dart';

void main() {
  testWidgets('markdown code blocks overflow horizontally', (tester) async {
    final code = 'final value = "${'a'.padRight(180, 'a')}";';
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SizedBox(
            width: 240,
            child: DocumentContentView(
              contents: [
                DocumentContent(
                  (b) => b
                    ..id = 'content_1'
                    ..role = 'extracted'
                    ..format = 'markdown'
                    ..content = '```dart\n$code\n```',
                ),
              ],
            ),
          ),
        ),
      ),
    );

    final horizontalScrollable = find.descendant(
      of: find.byKey(documentCodeBlockScrollKey),
      matching: find.byType(Scrollable),
    );
    final state = tester.state<ScrollableState>(horizontalScrollable);
    expect(state.position.maxScrollExtent, greaterThan(0));

    await tester.drag(
      find.byKey(documentCodeBlockScrollKey),
      const Offset(-80, 0),
    );
    await tester.pumpAndSettle();
    expect(state.position.pixels, greaterThan(0));
  });
}
