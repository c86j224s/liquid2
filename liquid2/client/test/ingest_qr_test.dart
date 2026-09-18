import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_client/features/ingest/ingest_qr_scanner.dart';

void main() {
  test('QR accepts complete HTTP URLs without embedded credentials', () {
    expect(
      qrUrl(' https://example.com/a?q=1#x '),
      'https://example.com/a?q=1#x',
    );
    expect(qrUrl('http://100.64.0.1:6011'), 'http://100.64.0.1:6011');
    for (final value in [
      null,
      '',
      'hello',
      'javascript:alert(1)',
      'https://',
      'https://user:pass@example.com',
      'https://a.test https://b.test',
    ]) {
      expect(qrUrl(value), isNull);
    }
  });
  testWidgets('Scanner entry is Android-only', (tester) async {
    final controller = TextEditingController();
    addTearDown(controller.dispose);
    addTearDown(() => debugDefaultTargetPlatformOverride = null);
    for (final platform in [TargetPlatform.android, TargetPlatform.macOS]) {
      await tester.pumpWidget(const SizedBox.shrink());
      debugDefaultTargetPlatformOverride = platform;
      await tester.pumpWidget(
        const ProviderScope(
          child: MaterialApp(home: Scaffold(body: QrQuickSaveButton())),
        ),
      );
      expect(
        find.byTooltip('QR 스캔'),
        platform == TargetPlatform.android ? findsOneWidget : findsNothing,
      );
    }
    debugDefaultTargetPlatformOverride = null;
  });
}
