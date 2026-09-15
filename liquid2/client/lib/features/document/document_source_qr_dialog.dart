import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:qr_flutter/qr_flutter.dart';

Future<void> showDocumentSourceQrDialog(BuildContext context, Uri sourceURL) {
  return showDialog<void>(
    context: context,
    builder: (context) => _DocumentSourceQrDialog(url: sourceURL.toString()),
  );
}

class _DocumentSourceQrDialog extends StatelessWidget {
  const _DocumentSourceQrDialog({required this.url});

  final String url;

  @override
  Widget build(BuildContext context) {
    // 바이트 모드·오류 정정 L의 최대 용량이다. 라이브러리는 초과를 페인트 시점에 감지한다.
    final validation = utf8.encode(url).length > 2953
        ? QrValidationResult(status: QrValidationStatus.contentTooLong)
        : QrValidator.validate(data: url);
    return AlertDialog(
      title: const Text('Source URL QR code'),
      scrollable: true,
      content: SizedBox(
        width: 320,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text('Scan with another device to open the original page.'),
            const SizedBox(height: 16),
            // 테마와 관계없이 흰 여백과 검은 모듈을 유지해 스캔 대비를 보장한다.
            SizedBox(
              height: 280,
              child: validation.isValid
                  ? QrImageView.withQr(
                      qr: validation.qrCode!,
                      backgroundColor: Colors.white,
                      padding: const EdgeInsets.all(20),
                      semanticsLabel: 'QR code for the source URL',
                    )
                  : Center(
                      child: Text(
                        validation.status == QrValidationStatus.contentTooLong
                            ? 'This URL is too long to display as a QR code.'
                            : 'Unable to display this URL as a QR code.',
                      ),
                    ),
            ),
            const SizedBox(height: 16),
            SelectableText(url),
          ],
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Close'),
        ),
      ],
    );
  }
}
