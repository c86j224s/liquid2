String formatMillis(int? millis, {DateTime? now}) {
  if (millis == null || millis <= 0) {
    return '-';
  }
  final value = DateTime.fromMillisecondsSinceEpoch(millis).toLocal();
  final current = (now ?? DateTime.now()).toLocal();
  final elapsed = current.difference(value);
  if (!elapsed.isNegative) {
    if (elapsed.inMinutes < 1) {
      return 'just now';
    }
    if (elapsed.inHours < 1) {
      return '${elapsed.inMinutes}m ago';
    }
    if (elapsed.inDays < 1) {
      return '${elapsed.inHours}h ago';
    }
    if (elapsed.inDays < 7) {
      return '${elapsed.inDays}d ago';
    }
  }
  return formatDateTime(value);
}

String formatFutureMillis(int? millis, {DateTime? now}) {
  if (millis == null || millis <= 0) {
    return '-';
  }
  final value = DateTime.fromMillisecondsSinceEpoch(millis).toLocal();
  final current = (now ?? DateTime.now()).toLocal();
  final remaining = value.difference(current);
  if (!remaining.isNegative) {
    if (remaining.inMinutes < 1) {
      return 'in <1m';
    }
    if (remaining.inHours < 1) {
      return 'in ${remaining.inMinutes}m';
    }
    if (remaining.inDays < 1) {
      return 'in ${remaining.inHours}h';
    }
    if (remaining.inDays < 7) {
      return 'in ${remaining.inDays}d';
    }
  }
  return formatDateTime(value);
}

String documentTimeLabel({
  required int updatedAt,
  int? publishedAt,
  DateTime? now,
}) {
  if (publishedAt != null && publishedAt > 0) {
    return 'Published ${formatMillis(publishedAt, now: now)}';
  }
  return 'Updated ${formatMillis(updatedAt, now: now)}';
}

String formatDateTime(DateTime value) {
  final local = value.toLocal();
  return '${local.year.toString().padLeft(4, '0')}-'
      '${local.month.toString().padLeft(2, '0')}-'
      '${local.day.toString().padLeft(2, '0')} '
      '${local.hour.toString().padLeft(2, '0')}:'
      '${local.minute.toString().padLeft(2, '0')}';
}

String compactKind(String kind) {
  return kind.replaceAll('_', ' ');
}

String folderPathLabel(Iterable<String> names) {
  return names.where((name) => name.trim().isNotEmpty).join(' / ');
}

String readableContent(String content, {String? format}) {
  final source = content.trim();
  if (source.isEmpty) {
    return '';
  }
  if (!_looksLikeHtml(source, format)) {
    return _decodeHtmlEntities(_collapseText(source));
  }
  final text = _htmlToReadableText(source);
  return _decodeHtmlEntities(_collapseText(text));
}

String readableBodyContent(String content, {String? format}) {
  final source = content.trim();
  if (source.isEmpty) {
    return '';
  }
  if (!_looksLikeHtml(source, format)) {
    return _decodeHtmlEntities(_normalizeBodyText(source));
  }
  final text = _htmlToReadableText(source);
  return _decodeHtmlEntities(_normalizeBodyText(text));
}

bool _looksLikeHtml(String content, String? format) {
  if (format?.toLowerCase() == 'html') {
    return true;
  }
  return RegExp(r'<[a-zA-Z][^>]*>').hasMatch(content);
}

String _htmlToReadableText(String source) {
  return source
      .replaceAll(
        RegExp(
          r'<script[^>]*>.*?</script>',
          caseSensitive: false,
          dotAll: true,
        ),
        ' ',
      )
      .replaceAll(
        RegExp(r'<style[^>]*>.*?</style>', caseSensitive: false, dotAll: true),
        ' ',
      )
      .replaceAll(RegExp(r'<br\s*/?>', caseSensitive: false), '\n')
      .replaceAll(
        RegExp(r'</(p|div|h[1-6]|blockquote)>', caseSensitive: false),
        '\n\n',
      )
      .replaceAll(RegExp(r'<li[^>]*>', caseSensitive: false), '\n- ')
      .replaceAll(RegExp(r'</li>', caseSensitive: false), '\n')
      .replaceAll(RegExp(r'</?(ul|ol)[^>]*>', caseSensitive: false), '\n')
      .replaceAll(RegExp(r'<[^>]+>'), ' ');
}

String _collapseText(String value) {
  return value
      .replaceAll('\r\n', '\n')
      .replaceAll('\r', '\n')
      .split('\n')
      .map((line) => line.replaceAll(RegExp(r'[ \t]+'), ' ').trim())
      .where((line) => line.isNotEmpty)
      .join('\n');
}

String _normalizeBodyText(String value) {
  return value
      .replaceAll('\r\n', '\n')
      .replaceAll('\r', '\n')
      .split('\n')
      .map((line) => line.replaceAll(RegExp(r'[ \t]+'), ' ').trim())
      .join('\n')
      .replaceAll(RegExp(r'\n{3,}'), '\n\n')
      .trim();
}

String _decodeHtmlEntities(String value) {
  return value.replaceAllMapped(RegExp(r'&(#x?[0-9a-fA-F]+|[a-zA-Z]+);'), (
    match,
  ) {
    final entity = match.group(1)!;
    if (entity.startsWith('#x') || entity.startsWith('#X')) {
      return _decodeCodePoint(entity.substring(2), radix: 16) ??
          match.group(0)!;
    }
    if (entity.startsWith('#')) {
      return _decodeCodePoint(entity.substring(1), radix: 10) ??
          match.group(0)!;
    }
    return _namedHtmlEntity(entity) ?? match.group(0)!;
  });
}

String? _decodeCodePoint(String value, {required int radix}) {
  final codePoint = int.tryParse(value, radix: radix);
  if (codePoint == null || codePoint <= 0 || codePoint > 0x10FFFF) {
    return null;
  }
  return String.fromCharCode(codePoint);
}

String? _namedHtmlEntity(String entity) {
  return switch (entity) {
    'amp' => '&',
    'apos' => "'",
    'gt' => '>',
    'lt' => '<',
    'nbsp' => ' ',
    'quot' => '"',
    _ => null,
  };
}
