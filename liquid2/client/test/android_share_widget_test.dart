import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_client/app/android_share_shell.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  testWidgets(
    'First setup stays editable on failure and saves only after success',
    (tester) async {
      const channel = MethodChannel('liquid2/share');
      final saved = <String>[];
      final checked = <String>[];
      tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(channel, (
        call,
      ) async {
        if (call.method == 'getServer') return '';
        if (call.method == 'setServer') saved.add(call.arguments as String);
        return null;
      });
      addTearDown(
        () => tester.binding.defaultBinaryMessenger.setMockMethodCallHandler(
          channel,
          null,
        ),
      );
      await tester.pumpWidget(
        ProviderScope(
          overrides: [
            androidServerCheckProvider.overrideWithValue((origin) async {
              checked.add(origin);
              if (origin == 'https://wrong.example') {
                throw StateError('Unavailable');
              }
            }),
          ],
          child: MaterialApp(
            home: const Scaffold(body: Text('Library')),
            builder: (context, child) => AndroidShareShell(child: child!),
          ),
        ),
      );
      await tester.pumpAndSettle();
      expect(
        tester.widget<TextField>(find.byType(TextField)).controller!.text,
        isEmpty,
      );
      expect(find.text('Library'), findsNothing);
      expect(find.text('닫기'), findsNothing);
      expect(checked, isEmpty);
      await tester.enterText(find.byType(TextField), 'ftp://wrong.example');
      await tester.tap(find.text('연결 설정 저장'));
      await tester.pumpAndSettle();
      expect(checked, isEmpty);
      expect(saved, isEmpty);
      await tester.enterText(find.byType(TextField), 'https://wrong.example');
      await tester.tap(find.text('연결 설정 저장'));
      await tester.pumpAndSettle();
      expect(saved, isEmpty);
      expect(find.textContaining('설정은 저장되지 않았습니다'), findsOneWidget);
      expect(find.byType(TextField), findsOneWidget);
      expect(find.text('Library'), findsNothing);
      await tester.enterText(find.byType(TextField), 'http://100.64.0.1:6011/');
      await tester.tap(find.text('연결 설정 저장'));
      await tester.pumpAndSettle();
      expect(saved, ['http://100.64.0.1:6011']);
      expect(checked, ['https://wrong.example', 'http://100.64.0.1:6011']);
      expect(find.byType(TextField), findsNothing);
      expect(find.text('Library'), findsOneWidget);
      expect(tester.takeException(), isNull);
    },
  );
}
