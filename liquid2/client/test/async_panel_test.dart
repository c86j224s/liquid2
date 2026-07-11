import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_client/shared/async_panel.dart';

void main() {
  testWidgets('shows selectable fallback when copying an error fails', (
    tester,
  ) async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(SystemChannels.platform, (call) async {
          if (call.method == 'Clipboard.setData') {
            throw PlatformException(code: 'clipboard_failed');
          }
          return null;
        });
    addTearDown(() {
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(SystemChannels.platform, null);
    });

    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: AsyncPanel<void>(
            value: const AsyncError<void>('api offline', StackTrace.empty),
            builder: (_) => const SizedBox.shrink(),
          ),
        ),
      ),
    );

    await tester.tap(find.text('Copy error'));
    await tester.pumpAndSettle();

    expect(find.text('Copy failed'), findsOneWidget);
    expect(find.byType(SelectableText), findsWidgets);
    expect(find.text('api offline'), findsWidgets);
  });
}
