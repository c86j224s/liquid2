import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/features/document/document_actions_bar.dart';
import 'package:liquid2_client/features/document/document_source_qr_dialog.dart';
import 'package:qr_flutter/qr_flutter.dart';

void main() {
  for (final kind in ['bookmark', 'scraped_article']) {
    testWidgets('$kind QR uses source URL rather than canonical URL', (
      tester,
    ) async {
      const source = 'https://example.com/original?q=one%20two&n=2#part';
      await tester.pumpWidget(_host(source: source, kind: kind));
      await tester.tap(find.text('Show QR code'));
      await tester.pumpAndSettle();

      await _expectQrPayload(tester, source);
      expect(find.text(source), findsOneWidget);
      await tester.tap(find.text('Close'));
      await tester.pumpAndSettle();
      expect(find.byType(QrImageView), findsNothing);
    });
  }

  for (final source in [null, '', ' ', 'file:///page', 'https:/page']) {
    testWidgets('falls back to canonical URL for source $source', (
      tester,
    ) async {
      await tester.pumpWidget(_host(source: source));
      await tester.tap(find.text('Show QR code'));
      await tester.pumpAndSettle();
      await _expectQrPayload(tester, _canonical);
    });
    testWidgets('hides QR without a usable URL: $source', (tester) async {
      await tester.pumpWidget(_host(source: source, canonical: null));
      expect(find.text('Show QR code'), findsNothing);
    });
  }

  for (final brightness in Brightness.values) {
    testWidgets('QR dialog fits a small $brightness screen and large text', (
      tester,
    ) async {
      tester.view.physicalSize = const Size(320, 480);
      tester.view.devicePixelRatio = 1;
      addTearDown(tester.view.resetPhysicalSize);
      addTearDown(tester.view.resetDevicePixelRatio);
      await tester.pumpWidget(
        MaterialApp(
          theme: ThemeData(brightness: brightness),
          builder: (context, child) => MediaQuery(
            data: MediaQuery.of(
              context,
            ).copyWith(textScaler: const TextScaler.linear(2)),
            child: child!,
          ),
          home: Scaffold(
            body: Builder(
              builder: (context) => TextButton(
                onPressed: () =>
                    showDocumentSourceQrDialog(context, Uri.parse(_canonical)),
                child: const Text('Show QR code'),
              ),
            ),
          ),
        ),
      );
      await tester.tap(find.text('Show QR code'));
      await tester.pumpAndSettle();
      expect(tester.takeException(), isNull);
      final qr = tester.widget<QrImageView>(find.byType(QrImageView));
      expect(qr.backgroundColor, Colors.white);
      expect(qr.semanticsLabel, 'QR code for the source URL');
      await tester.tap(find.text('Close'));
      await tester.pumpAndSettle();
      expect(find.byType(AlertDialog), findsNothing);
    });
  }

  testWidgets('oversized URL shows an error without losing the URL', (
    tester,
  ) async {
    final source = 'https://example.com/?q=${'a' * 4000}';
    await tester.pumpWidget(_host(source: source));
    await tester.tap(find.text('Show QR code'));
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
    expect(
      find.text('This URL is too long to display as a QR code.'),
      findsOneWidget,
    );
    expect(find.text(source), findsOneWidget);
  });
}

Future<void> _expectQrPayload(WidgetTester tester, String url) async {
  final paint = tester.widget<CustomPaint>(
    find.descendant(
      of: find.byType(QrImageView),
      matching: find.byWidgetPredicate(
        (widget) => widget is CustomPaint && widget.painter is QrPainter,
      ),
    ),
  );
  final actual = paint.painter! as QrPainter;
  final expected = QrPainter(
    data: url,
    version: QrVersions.auto,
    gapless: true,
  );
  await tester.runAsync(() async {
    final actualBytes = await actual.toImageData(256);
    final expectedBytes = await expected.toImageData(256);
    expect(actualBytes, isNotNull);
    expect(expectedBytes, isNotNull);
    expect(
      actualBytes!.buffer.asUint8List(),
      expectedBytes!.buffer.asUint8List(),
    );
  });
}

const _canonical = 'https://example.com/canonical';

Widget _host({
  String? source,
  String? canonical = _canonical,
  String kind = 'scraped_article',
}) {
  return ProviderScope(
    child: MaterialApp(
      home: Scaffold(
        body: DocumentActionsBar(
          document: DocumentMetadata(
            (b) => b
              ..id = 'doc_qr'
              ..title = 'QR test'
              ..kind = kind
              ..status = 'unread'
              ..createdAt = 1760000000000
              ..updatedAt = 1760000000000
              ..sourceUrl = source
              ..canonicalUrl = canonical,
          ),
        ),
      ),
    ),
  );
}
