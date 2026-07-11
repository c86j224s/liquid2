import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/providers.dart';
import '../../shared/action_feedback.dart';
import 'document_tag_creator.dart';

class DocumentTagEditor extends ConsumerWidget {
  const DocumentTagEditor({
    required this.documentId,
    required this.assigned,
    super.key,
  });

  final String documentId;
  final List<Tag> assigned;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final snapshot = ref.watch(librarySnapshotProvider);
    return snapshot.when(
      data: (data) {
        final assignedIds = _selectedTagIds(ref);
        final isSaving = ref.watch(documentTagSavingProvider(documentId));
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            DocumentTagCreator(documentId: documentId, assigned: assigned),
            const SizedBox(height: 12),
            if (data.tags.isEmpty)
              const Text('No tags yet.')
            else
              Wrap(
                spacing: 8,
                runSpacing: 8,
                children: [
                  for (final tag in data.tags)
                    FilterChip(
                      label: Text(tag.name),
                      selected: assignedIds.contains(tag.id),
                      onSelected: isSaving
                          ? null
                          : (_) => runUiAction(
                              context,
                              () => _toggleTag(ref, tag.id),
                            ),
                    ),
                ],
              ),
          ],
        );
      },
      error: (_, _) => const Text('Tags unavailable. Refresh to retry.'),
      loading: () => const Text('Loading tags...'),
    );
  }

  Future<void> _toggleTag(WidgetRef ref, String tagId) async {
    final saving = ref.read(documentTagSavingProvider(documentId).notifier);
    if (saving.state) {
      return;
    }
    final next = {..._readSelectedTagIds(ref)};
    if (!next.add(tagId)) {
      next.remove(tagId);
    }
    saving.state = true;
    ref.read(documentTagSelectionProvider(documentId).notifier).state = next;
    try {
      final detail = await ref
          .read(libraryRepositoryProvider)
          .replaceTags(documentId, next.toList());
      ref.read(documentTagSelectionProvider(documentId).notifier).state = detail
          .tags
          ?.map((tag) => tag.id)
          .toSet();
    } catch (_) {
      ref.read(documentTagSelectionProvider(documentId).notifier).state = null;
      rethrow;
    } finally {
      saving.state = false;
    }
    ref
      ..invalidate(documentDetailProvider(documentId))
      ..invalidate(librarySnapshotProvider);
  }

  Set<String> _selectedTagIds(WidgetRef ref) {
    final pending = ref.watch(documentTagSelectionProvider(documentId));
    if (pending != null) {
      return pending;
    }
    final latest = ref.watch(documentDetailProvider(documentId)).value;
    return (latest?.tags?.toList() ?? assigned).map((tag) => tag.id).toSet();
  }

  Set<String> _readSelectedTagIds(WidgetRef ref) {
    final pending = ref.read(documentTagSelectionProvider(documentId));
    if (pending != null) {
      return pending;
    }
    final latest = ref.read(documentDetailProvider(documentId)).value;
    return (latest?.tags?.toList() ?? assigned).map((tag) => tag.id).toSet();
  }
}
