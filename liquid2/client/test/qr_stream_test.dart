import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/shared/qr_stream.dart';

void main() {
  final text = '네트워크 없이 옮기는 장문 ${'가나다 abc ' * 300}끝';

  test(
    'frames arriving shuffled, repeated and late still rebuild the text',
    () {
      final frames = encodeQrStream(text);
      expect(frames.length, greaterThan(3));
      final receiver = QrStreamReceiver();

      final shuffled = [...frames.skip(1), ...frames.reversed]..shuffle();
      for (final frame in shuffled) {
        receiver.add(frame);
      }
      expect(receiver.result, text);
      expect(receiver.received, frames.length);
    },
  );

  test('a missing frame holds the result until the next loop brings it', () {
    final frames = encodeQrStream(text);
    final receiver = QrStreamReceiver();
    frames.skip(1).forEach(receiver.add);
    expect(receiver.result, isNull);
    expect(receiver.received, frames.length - 1);

    receiver.add(frames.first);
    expect(receiver.result, text);
  });

  test('frames from another transfer and plain QR text are ignored', () {
    final frames = encodeQrStream(text);
    final other = encodeQrStream('다른 전송 ${'x' * 1000}');
    final receiver = QrStreamReceiver();
    receiver.add(frames.first);
    other.forEach(receiver.add);
    receiver.add('https://example.com');
    receiver.add('L2Q1|broken');
    expect(receiver.received, 1);

    frames.forEach(receiver.add);
    expect(receiver.result, text);
  });

  test('a corrupted chunk fails the hash and collection starts over', () {
    final frames = encodeQrStream(text);
    final parts = frames.first.split('|');
    final bad = [...parts.take(4), 'AAAA${parts[4].substring(4)}'].join('|');
    final receiver = QrStreamReceiver();

    receiver.add(bad);
    frames.skip(1).forEach(receiver.add);
    expect(receiver.result, isNull);
    expect(receiver.received, 0);

    frames.forEach(receiver.add);
    expect(receiver.result, text);
  });
}
