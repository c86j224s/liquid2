import 'package:flutter_test/flutter_test.dart';
import 'package:liquid2_client/shared/formatters.dart';

void main() {
  test('formatMillis shows relative recent times', () {
    final now = DateTime(2026, 6, 15, 12);

    expect(formatMillis(now.millisecondsSinceEpoch, now: now), 'just now');
    expect(
      formatMillis(
        now.subtract(const Duration(minutes: 5)).millisecondsSinceEpoch,
        now: now,
      ),
      '5m ago',
    );
    expect(
      formatMillis(
        now.subtract(const Duration(hours: 3)).millisecondsSinceEpoch,
        now: now,
      ),
      '3h ago',
    );
    expect(
      formatMillis(
        now.subtract(const Duration(days: 2)).millisecondsSinceEpoch,
        now: now,
      ),
      '2d ago',
    );
  });

  test('formatMillis shows older values with date and time', () {
    final now = DateTime(2026, 6, 15, 12);
    final old = DateTime(2026, 6, 1, 9, 7);

    expect(
      formatMillis(old.millisecondsSinceEpoch, now: now),
      '2026-06-01 09:07',
    );
  });

  test('formatFutureMillis shows upcoming times', () {
    final now = DateTime(2026, 6, 15, 12);

    expect(
      formatFutureMillis(
        now.add(const Duration(minutes: 5)).millisecondsSinceEpoch,
        now: now,
      ),
      'in 5m',
    );
    expect(
      formatFutureMillis(
        now.add(const Duration(hours: 2)).millisecondsSinceEpoch,
        now: now,
      ),
      'in 2h',
    );
  });

  test('documentTimeLabel prefers source publication time', () {
    final now = DateTime(2026, 6, 15, 12);
    final published = now.subtract(const Duration(hours: 2));
    final updated = now.subtract(const Duration(minutes: 5));

    expect(
      documentTimeLabel(
        updatedAt: updated.millisecondsSinceEpoch,
        publishedAt: published.millisecondsSinceEpoch,
        now: now,
      ),
      'Published 2h ago',
    );
    expect(
      documentTimeLabel(updatedAt: updated.millisecondsSinceEpoch, now: now),
      'Updated 5m ago',
    );
  });

  test('readableContent strips html tags and decodes entities', () {
    final summary = readableContent(
      '<ul><li>Build <strong>small</strong> tools &amp; ship</li>'
      '<li>Keep&nbsp;notes</li></ul>',
      format: 'html',
    );

    expect(summary, '- Build small tools & ship\n- Keep notes');
  });

  test('readableContent keeps plain text readable', () {
    final summary = readableContent('  Stored   document body  ');

    expect(summary, 'Stored document body');
  });
}
