import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../app/app_theme.dart';
import '../../app/providers.dart';
import '../../shared/action_feedback.dart';
import '../../shared/star_rating.dart';
import 'document_move_folder_dialog.dart';
import 'document_rescrape_button.dart';

class DocumentActionsBar extends ConsumerWidget {
  const DocumentActionsBar({required this.document, super.key});

  final DocumentMetadata document;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final isRead = document.status == 'read';
    final sourceURL = _documentSourceURL(document);

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // Primary actions row
        Wrap(
          spacing: AppSpacing.sm,
          runSpacing: AppSpacing.sm,
          crossAxisAlignment: WrapCrossAlignment.center,
          children: [
            _ReadToggleButton(document: document, isRead: isRead),
            if (sourceURL != null)
              OutlinedButton.icon(
                onPressed: () =>
                    runUiAction(context, () => _openSource(sourceURL)),
                icon: const Icon(Icons.open_in_new, size: 16),
                label: const Text('Open source'),
              ),
            DocumentRescrapeButton(document: document),
            if (document.kind != 'rss_item')
              _MoveFolderButton(document: document),
            _TrashButton(document: document),
          ],
        ),
        const SizedBox(height: AppSpacing.md),
        // Rating row
        Row(
          children: [
            Text('Rating', style: Theme.of(context).textTheme.labelSmall),
            const SizedBox(width: AppSpacing.sm),
            StarRating(
              rating: document.rating,
              onChanged: (value) =>
                  runUiAction(context, () => _setRating(ref, value)),
            ),
          ],
        ),
      ],
    );
  }

  Future<void> _setRating(WidgetRef ref, int? rating) async {
    await ref.read(libraryRepositoryProvider).setRating(document.id, rating);
    ref
      ..invalidate(documentDetailProvider(document.id))
      ..invalidate(librarySnapshotProvider);
  }

  Future<void> _openSource(Uri url) async {
    final opened = await launchUrl(url, mode: LaunchMode.externalApplication);
    if (!opened) throw StateError('Unable to open source URL.');
  }
}

class _ReadToggleButton extends ConsumerWidget {
  const _ReadToggleButton({required this.document, required this.isRead});

  final DocumentMetadata document;
  final bool isRead;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    void onPressed() async {
      await runUiAction(context, () => _toggleRead(ref));
      if (context.mounted) context.go('/');
    }

    if (isRead) {
      return FilledButton.icon(
        onPressed: onPressed,
        icon: const Icon(Icons.mark_email_unread_outlined, size: 16),
        label: const Text('Mark unread'),
      );
    }
    return OutlinedButton.icon(
      onPressed: onPressed,
      icon: const Icon(Icons.done, size: 16),
      label: const Text('Mark read'),
    );
  }

  Future<void> _toggleRead(WidgetRef ref) async {
    final repo = ref.read(libraryRepositoryProvider);
    if (isRead) {
      await repo.markUnread(document.id);
    } else {
      await repo.markRead(document.id);
    }
    ref
      ..invalidate(documentDetailProvider(document.id))
      ..invalidate(librarySnapshotProvider);
  }
}

class _MoveFolderButton extends ConsumerWidget {
  const _MoveFolderButton({required this.document});

  final DocumentMetadata document;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return OutlinedButton.icon(
      onPressed: () => showDocumentMoveFolderDialog(context, document),
      icon: const Icon(Icons.drive_file_move_outline, size: 16),
      label: const Text('Move folder'),
    );
  }
}

class _TrashButton extends ConsumerWidget {
  const _TrashButton({required this.document});

  final DocumentMetadata document;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return IconButton(
      tooltip: 'Move to trash',
      onPressed: () async {
        await runUiAction(context, () => _moveToTrash(ref));
        if (context.mounted) context.go('/');
      },
      icon: const Icon(Icons.delete_outline),
    );
  }

  Future<void> _moveToTrash(WidgetRef ref) async {
    await ref.read(libraryRepositoryProvider).moveDocumentToTrash(document.id);
    ref
      ..invalidate(documentDetailProvider(document.id))
      ..invalidate(librarySnapshotProvider);
  }
}

Uri? _documentSourceURL(DocumentMetadata document) {
  return _parseHTTPURL(document.sourceUrl) ??
      _parseHTTPURL(document.canonicalUrl);
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
