import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/providers.dart';
import '../../shared/action_feedback.dart';
import 'document_tag_providers.dart';

class DocumentTagCreator extends ConsumerStatefulWidget {
  const DocumentTagCreator({
    required this.documentId,
    required this.assigned,
    super.key,
  });

  final String documentId;
  final List<Tag> assigned;

  @override
  ConsumerState<DocumentTagCreator> createState() => _DocumentTagCreatorState();
}

class _DocumentTagCreatorState extends ConsumerState<DocumentTagCreator> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final isCreating = ref.watch(
      documentTagCreatingProvider(widget.documentId),
    );
    return Row(
      children: [
        Expanded(
          child: TextField(
            key: const Key('document-tag-input'),
            controller: _controller,
            enabled: !isCreating,
            decoration: const InputDecoration(labelText: 'New tag'),
            textInputAction: TextInputAction.done,
            onSubmitted: (_) => _submit(context),
          ),
        ),
        IconButton(
          tooltip: 'Create tag',
          onPressed: isCreating ? null : () => _submit(context),
          icon: const Icon(Icons.add),
        ),
      ],
    );
  }

  void _submit(BuildContext context) {
    runUiAction(context, _createTag);
  }

  Future<void> _createTag() async {
    final name = _controller.text.trim();
    if (name.isEmpty) {
      return;
    }
    final creating = ref.read(
      documentTagCreatingProvider(widget.documentId).notifier,
    );
    if (creating.state) {
      return;
    }
    creating.state = true;
    try {
      final tag = await ref.read(tagRepositoryProvider).createTag(name);
      final next = {..._readSelectedTagIds(), tag.id};
      ref.read(documentTagSelectionProvider(widget.documentId).notifier).state =
          next;
      final detail = await ref
          .read(libraryRepositoryProvider)
          .replaceTags(widget.documentId, next.toList());
      ref.read(documentTagSelectionProvider(widget.documentId).notifier).state =
          detail.tags?.map((tag) => tag.id).toSet();
      _controller.clear();
    } catch (_) {
      ref.read(documentTagSelectionProvider(widget.documentId).notifier).state =
          null;
      rethrow;
    } finally {
      creating.state = false;
    }
    ref
      ..invalidate(documentDetailProvider(widget.documentId))
      ..invalidate(librarySnapshotProvider);
  }

  Set<String> _readSelectedTagIds() {
    final pending = ref.read(documentTagSelectionProvider(widget.documentId));
    if (pending != null) {
      return pending;
    }
    final latest = ref.read(documentDetailProvider(widget.documentId)).value;
    return (latest?.tags?.toList() ?? widget.assigned)
        .map((tag) => tag.id)
        .toSet();
  }
}
