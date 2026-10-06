import 'dart:convert';

import 'package:crypto/crypto.dart';

// 조각 QR 형식: L2Q1|<SHA-256 앞 16 hex>|<조각 번호>|<조각 수>|<base64 조각>
// 해시가 전송 식별자를 겸해 다른 전송의 조각이 섞이지 않는다. 64비트면 손상 검출에 충분하고
// 장면마다 싣는 부담이 작다.
const qrStreamPrefix = 'L2Q1|';
// 280px 박스에서 스크린샷으로도 400바이트 조각은 읽히지 않았다. 실측으로 정한다.
const qrStreamChunkBytes = 250;

String _digest(List<int> bytes) =>
    sha256.convert(bytes).toString().substring(0, 16);

List<String> encodeQrStream(String text) {
  final bytes = utf8.encode(text);
  final hash = _digest(bytes);
  final count = (bytes.length / qrStreamChunkBytes).ceil();
  return [
    for (var i = 0; i < count; i++)
      '$qrStreamPrefix$hash|$i|$count|${base64.encode(bytes.sublist(i * qrStreamChunkBytes, ((i + 1) * qrStreamChunkBytes).clamp(0, bytes.length)))}',
  ];
}

// ponytail: 순환 반복이라 놓친 조각은 다음 바퀴를 기다린다. 실측 대기가 길면 분수 부호로 바꾼다.
class QrStreamReceiver {
  String? _hash;
  final _chunks = <int, List<int>>{};
  int _count = 0;
  String? result;

  int get received => _chunks.length;
  int get count => _count;

  static bool isFrame(String value) => value.startsWith(qrStreamPrefix);

  void add(String frame) {
    if (result != null || !isFrame(frame)) return;
    final parts = frame.substring(qrStreamPrefix.length).split('|');
    if (parts.length != 4) return;
    final index = int.tryParse(parts[1]);
    final count = int.tryParse(parts[2]);
    if (index == null || count == null || index < 0 || index >= count) return;
    _hash ??= parts[0];
    if (parts[0] != _hash) return;
    _count = count;
    try {
      _chunks[index] = base64.decode(parts[3]);
    } on FormatException {
      return;
    }
    if (_chunks.length < count) return;
    final bytes = [for (var i = 0; i < count; i++) ..._chunks[i]!];
    if (_digest(bytes) != _hash) {
      // 어느 조각이 틀렸는지 알 수 없으니 비우고 다음 바퀴에서 다시 모은다.
      _chunks.clear();
      return;
    }
    result = utf8.decode(bytes);
  }
}
