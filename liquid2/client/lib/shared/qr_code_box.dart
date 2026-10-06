import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:qr_flutter/qr_flutter.dart';

// 바이트 모드·오류 정정 L의 최대 용량이다. 라이브러리는 초과를 페인트 시점에 감지한다.
const qrMaxBytes = 2953;

class QrCodeBox extends StatelessWidget {
  const QrCodeBox({
    required this.data,
    required this.semanticsLabel,
    required this.tooLongMessage,
    required this.invalidMessage,
    super.key,
  });

  final String data;
  final String semanticsLabel;
  final String tooLongMessage;
  final String invalidMessage;

  @override
  Widget build(BuildContext context) {
    final validation = utf8.encode(data).length > qrMaxBytes
        ? QrValidationResult(status: QrValidationStatus.contentTooLong)
        : QrValidator.validate(data: data);
    // 테마와 관계없이 흰 여백과 검은 모듈을 유지해 스캔 대비를 보장한다.
    return validation.isValid
        ? QrImageView.withQr(
            qr: validation.qrCode!,
            backgroundColor: Colors.white,
            padding: const EdgeInsets.all(20),
            semanticsLabel: semanticsLabel,
          )
        : Center(
            child: Text(
              validation.status == QrValidationStatus.contentTooLong
                  ? tooLongMessage
                  : invalidMessage,
            ),
          );
  }
}
