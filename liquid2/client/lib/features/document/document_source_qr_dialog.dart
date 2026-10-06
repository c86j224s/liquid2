import 'package:flutter/material.dart';

import '../../shared/qr_code_box.dart';

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
            SizedBox(
              height: 280,
              child: QrCodeBox(
                data: url,
                semanticsLabel: 'QR code for the source URL',
                tooLongMessage: 'This URL is too long to display as a QR code.',
                invalidMessage: 'Unable to display this URL as a QR code.',
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
