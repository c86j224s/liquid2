import 'dart:convert';

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
  test('QR stream text uploads as txt titled by its first line', () {
    final text = '\n  회의록 ${'긴 제목 ' * 20}\n본문';
    final input = qrTextUpload(text);
    expect(input.filename, 'qr-transfer.txt');
    expect(utf8.decode(input.bytes), text);
    expect(input.title, startsWith('회의록 긴 제목'));
    expect(input.title!.length, 80);
    expect(qrTextUpload('  \n').title, 'QR 텍스트');
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
