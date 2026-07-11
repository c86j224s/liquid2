import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/providers.dart';
import '../../shared/action_feedback.dart';

class DocumentRescrapeButton extends ConsumerWidget {
  const DocumentRescrapeButton({required this.document, super.key});

  final DocumentMetadata document;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    if (!canRescrapeDocument(document)) {
      return const SizedBox.shrink();
    }
    return OutlinedButton.icon(
      onPressed: () => runUiAction(context, () => _rescrape(context, ref)),
      icon: const Icon(Icons.refresh, size: 16),
      label: const Text('Re-scrape'),
    );
  }

  Future<void> _rescrape(BuildContext context, WidgetRef ref) async {
    await ref.read(libraryRepositoryProvider).rescrapeDocument(document.id);
    ref
      ..invalidate(documentDetailProvider(document.id))
      ..invalidate(librarySnapshotProvider);
    if (!context.mounted) return;
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(const SnackBar(content: Text('Re-scraped document.')));
  }
}

bool canRescrapeDocument(DocumentMetadata document) {
  if (document.kind != 'scraped_article' && document.kind != 'rss_item') {
    return false;
  }
  return _parseHTTPURL(document.sourceUrl) != null ||
      _parseHTTPURL(document.canonicalUrl) != null;
}

Uri? _parseHTTPURL(String? rawURL) {
  final value = rawURL?.trim();
  if (value == null || value.isEmpty) return null;
  final uri = Uri.tryParse(value);
  if (uri == null || (uri.scheme != 'http' && uri.scheme != 'https')) {
    return null;
  }
  return uri;
}
