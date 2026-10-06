import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';

import '../../shared/qr_code_box.dart';
import '../../shared/qr_stream.dart';

// ponytail: 고정 간격. 받는 쪽 인식률을 실측해 조정한다.
const _frameInterval = Duration(milliseconds: 250);

Future<void> showTextQrDialog(BuildContext context) {
  return showDialog<void>(
    context: context,
    builder: (context) => const _TextQrDialog(),
  );
}

class _TextQrDialog extends StatefulWidget {
  const _TextQrDialog();

  @override
  State<_TextQrDialog> createState() => _TextQrDialogState();
}

class _TextQrDialogState extends State<_TextQrDialog> {
  final _controller = TextEditingController();
  List<String> _frames = const [];
  int _frame = 0;
  Timer? _timer;

  @override
  void dispose() {
    _timer?.cancel();
    _controller.dispose();
    super.dispose();
  }

  void _onChanged(String text) {
    _timer?.cancel();
    _timer = null;
    _frame = 0;
    _frames = utf8.encode(text).length > qrMaxBytes
        ? encodeQrStream(text)
        : const [];
    if (_frames.isNotEmpty) {
      _timer = Timer.periodic(
        _frameInterval,
        (_) => setState(() => _frame = (_frame + 1) % _frames.length),
      );
    }
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final text = _controller.text;
    final bytes = utf8.encode(text).length;
    final streaming = _frames.isNotEmpty;
    return AlertDialog(
      title: const Text('Text QR code'),
      scrollable: true,
      content: SizedBox(
        width: 320,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            TextField(
              key: const Key('text-qr-input'),
              controller: _controller,
              autofocus: true,
              minLines: 2,
              maxLines: 6,
              decoration: InputDecoration(
                hintText: 'Text to encode',
                helperText: streaming
                    ? '$bytes bytes · animated, scan with Liquid2'
                    : '$bytes / $qrMaxBytes bytes',
              ),
              onChanged: _onChanged,
            ),
            const SizedBox(height: 16),
            SizedBox(
              height: 280,
              child: text.isEmpty
                  ? const Center(child: Text('Type text to see its QR code.'))
                  : QrCodeBox(
                      data: streaming ? _frames[_frame] : text,
                      semanticsLabel: 'QR code for the entered text',
                      tooLongMessage:
                          'This text is too long to display as a QR code.',
                      invalidMessage:
                          'Unable to display this text as a QR code.',
                    ),
            ),
            if (streaming) ...[
              const SizedBox(height: 8),
              Text(
                'Frame ${_frame + 1} / ${_frames.length}',
                textAlign: TextAlign.center,
              ),
            ],
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
