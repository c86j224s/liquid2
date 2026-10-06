import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/features/library/text_qr_dialog.dart';
import 'package:liquid2_client/shared/qr_code_box.dart';
import 'package:liquid2_client/shared/qr_stream.dart';
import 'package:qr_flutter/qr_flutter.dart';

void main() {
  testWidgets('typed text becomes a QR code with a byte count', (tester) async {
    await _open(tester);
    expect(find.byType(QrImageView), findsNothing);

    await tester.enterText(find.byKey(const Key('text-qr-input')), '안녕 qr');
    await tester.pumpAndSettle();

    expect(find.text('9 / 2953 bytes'), findsOneWidget);
    final qr = tester.widget<QrImageView>(find.byType(QrImageView));
    expect(qr.semanticsLabel, 'QR code for the entered text');
  });

  testWidgets('text over one QR cycles through stream frames', (tester) async {
    await _open(tester);
    final text = '가' * 985;
    await tester.enterText(find.byKey(const Key('text-qr-input')), text);
    await tester.pump();

    final frames = encodeQrStream(text);
    expect(find.text('Frame 1 / ${frames.length}'), findsOneWidget);
    expect(_qrData(tester), frames[0]);

    await tester.pump(const Duration(milliseconds: 250));
    expect(find.text('Frame 2 / ${frames.length}'), findsOneWidget);
    expect(_qrData(tester), frames[1]);

    await tester.tap(find.text('Close'));
    await tester.pumpAndSettle();
  });
}

Future<void> _open(WidgetTester tester) async {
  await tester.pumpWidget(
    MaterialApp(
      home: Scaffold(
        body: Builder(
          builder: (context) => TextButton(
            onPressed: () => showTextQrDialog(context),
            child: const Text('Open'),
          ),
        ),
      ),
    ),
  );
  await tester.tap(find.text('Open'));
  await tester.pumpAndSettle();
}

String _qrData(WidgetTester tester) =>
    tester.widget<QrCodeBox>(find.byType(QrCodeBox)).data;
