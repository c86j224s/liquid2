import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:mobile_scanner/mobile_scanner.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import '../../app/android_share_shell.dart';
import '../../app/providers.dart';
import '../../data/library_repository.dart';
import '../../shared/qr_stream.dart';

String? qrUrl(String? value) {
  if (value == null) return null;
  final text = value.trim();
  if (RegExp(r'\s').hasMatch(text)) return null;
  final uri = Uri.tryParse(text);
  if (uri == null ||
      (uri.scheme != 'http' && uri.scheme != 'https') ||
      uri.host.isEmpty ||
      uri.userInfo.isNotEmpty) {
    return null;
  }
  return text;
}

typedef QrScan = ({String value, bool isText});

UploadFileInput qrTextUpload(String text) {
  final firstLine = text
      .split('\n')
      .map((line) => line.trim())
      .firstWhere((line) => line.isNotEmpty, orElse: () => 'QR 텍스트');
  return UploadFileInput(
    filename: 'qr-transfer.txt',
    bytes: utf8.encode(text),
    title: firstLine.length > 80 ? firstLine.substring(0, 80) : firstLine,
  );
}

class QrQuickSaveButton extends ConsumerStatefulWidget {
  const QrQuickSaveButton({super.key});
  @override
  ConsumerState<QrQuickSaveButton> createState() => _QrQuickSaveButtonState();
}

class _QrQuickSaveButtonState extends ConsumerState<QrQuickSaveButton> {
  bool busy = false;

  Future<void> scan() async {
    if (busy) return;
    setState(() => busy = true);
    try {
      final scanned = await Navigator.of(context).push<QrScan>(
        MaterialPageRoute(builder: (_) => const IngestQrScanner()),
      );
      if (!mounted || scanned == null) return;
      await showDialog<void>(
        context: context,
        barrierDismissible: false,
        builder: (_) => _QrSaveDialog(scanned: scanned),
      );
    } finally {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    if (kIsWeb || defaultTargetPlatform != TargetPlatform.android) {
      return const SizedBox.shrink();
    }
    return IconButton(
      tooltip: 'QR 스캔',
      icon: const Icon(Icons.qr_code_scanner),
      onPressed: busy ? null : scan,
    );
  }
}

class _QrSaveDialog extends ConsumerStatefulWidget {
  const _QrSaveDialog({required this.scanned});
  final QrScan scanned;
  @override
  ConsumerState<_QrSaveDialog> createState() => _QrSaveDialogState();
}

class _QrSaveDialogState extends ConsumerState<_QrSaveDialog> {
  bool busy = true;
  @override
  void initState() {
    super.initState();
    Future.microtask(save);
  }

  Future<void> save() async {
    if (!mounted) return;
    setState(() => busy = true);
    try {
      final repository = ref.read(libraryRepositoryProvider);
      final String done;
      if (widget.scanned.isText) {
        await repository.uploadFile(qrTextUpload(widget.scanned.value));
        done = 'QR로 받은 텍스트 저장됨';
      } else {
        final scraped = await saveSharedUrl(repository, widget.scanned.value);
        done = scraped ? '스크랩 저장됨' : '링크만 저장됨 · 원문 스크랩 실패';
      }
      if (!mounted) return;
      ref.invalidate(librarySnapshotProvider);
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(done)));
      Navigator.of(context).pop();
    } catch (_) {
      if (mounted) setState(() => busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final value = widget.scanned.value;
    final isText = widget.scanned.isText;
    return PopScope(
      canPop: !busy,
      child: AlertDialog(
        title: Text(busy ? (isText ? '저장 중…' : '스크랩 중…') : '저장하지 못했습니다'),
        scrollable: true,
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (busy) const LinearProgressIndicator(),
            SelectableText(
              isText && value.length > 200
                  ? '${value.substring(0, 200)}… (${value.length}자)'
                  : value,
            ),
            Text(
              busy
                  ? (isText ? '받은 텍스트를 문서로 저장합니다.' : '실패하면 링크만 저장합니다.')
                  : '서버 연결을 확인한 뒤 재시도하세요.',
            ),
          ],
        ),
        actions: busy
            ? null
            : [
                // 닫으면 받은 내용이 사라지므로 서버에 닿지 않을 때 옮길 길을 남긴다.
                TextButton(
                  onPressed: () =>
                      Clipboard.setData(ClipboardData(text: value)),
                  child: const Text('복사'),
                ),
                TextButton(
                  onPressed: () => Navigator.of(context).pop(),
                  child: const Text('닫기'),
                ),
                FilledButton(onPressed: save, child: const Text('다시 시도')),
              ],
      ),
    );
  }
}

class IngestQrScanner extends StatefulWidget {
  const IngestQrScanner({super.key});

  @override
  State<IngestQrScanner> createState() => _IngestQrScannerState();
}

class _IngestQrScannerState extends State<IngestQrScanner> {
  bool completed = false;
  String? message;
  final stream = QrStreamReceiver();

  void detect(BarcodeCapture capture) {
    if (completed || !mounted) return;
    for (final barcode in capture.barcodes) {
      if (barcode.format != BarcodeFormat.qrCode) continue;
      final raw = barcode.rawValue;
      if (raw != null && QrStreamReceiver.isFrame(raw)) {
        stream.add(raw);
        final text = stream.result;
        if (text != null) {
          completed = true;
          Navigator.of(context).pop((value: text, isText: true));
          return;
        }
        setState(() => message = null);
        continue;
      }
      final value = qrUrl(raw);
      if (value != null) {
        completed = true;
        Navigator.of(context).pop((value: value, isText: false));
        return;
      }
    }
    if (stream.count > 0) return;
    if (message == null) {
      setState(() => message = 'HTTP(S) URL이 담긴 QR을 읽어 주세요.');
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('QR 스캔'),
        leading: IconButton(
          tooltip: '취소',
          icon: const Icon(Icons.close),
          onPressed: () {
            completed = true;
            Navigator.of(context).pop();
          },
        ),
      ),
      body: Column(
        children: [
          Expanded(
            // The internal controller owns permission requests, lifecycle
            // pause/resume, and camera stop/disposal when this route closes.
            child: MobileScanner(
              onDetect: detect,
              errorBuilder: (context, error) => const Center(
                child: Padding(
                  padding: EdgeInsets.all(24),
                  child: Text(
                    '카메라를 사용할 수 없습니다. 카메라 권한과 기기 상태를 확인하세요. '
                    '권한을 거부했다면 앱 설정에서 허용한 뒤 다시 열어 주세요.',
                  ),
                ),
              ),
            ),
          ),
          SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.all(16),
              child: stream.count > 0
                  ? Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        LinearProgressIndicator(
                          value: stream.received / stream.count,
                        ),
                        const SizedBox(height: 8),
                        Text(
                          '텍스트 수신 중 ${stream.received} / ${stream.count} 조각 · '
                          '화면을 계속 비춰 주세요.',
                        ),
                      ],
                    )
                  : Text(message ?? 'QR을 읽으면 자동으로 스크랩하고, 실패하면 링크만 저장합니다.'),
            ),
          ),
        ],
      ),
    );
  }
}
