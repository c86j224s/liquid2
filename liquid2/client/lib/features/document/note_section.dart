import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/providers.dart';
import '../../shared/action_feedback.dart';
import '../../shared/formatters.dart';

class NoteSection extends ConsumerWidget {
  const NoteSection({required this.documentId, super.key});

  final String documentId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final notes = ref.watch(documentNotesProvider(documentId));
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text('Notes', style: Theme.of(context).textTheme.titleMedium),
        const SizedBox(height: 12),
        NoteComposer(documentId: documentId),
        const SizedBox(height: 16),
        notes.when(
          data: (items) => _NoteList(notes: items),
          loading: () => const LinearProgressIndicator(),
          error: (error, stackTrace) => Text(error.toString()),
        ),
      ],
    );
  }
}

class NoteComposer extends ConsumerStatefulWidget {
  const NoteComposer({required this.documentId, super.key});

  final String documentId;

  @override
  ConsumerState<NoteComposer> createState() => _NoteComposerState();
}

class _NoteComposerState extends ConsumerState<NoteComposer> {
  final _controller = TextEditingController();
  bool _saving = false;

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: TextField(
            controller: _controller,
            minLines: 2,
            maxLines: 4,
            decoration: const InputDecoration(labelText: 'New note'),
          ),
        ),
        const SizedBox(width: 8),
        FilledButton.icon(
          onPressed: _saving ? null : _save,
          icon: const Icon(Icons.add_comment_outlined),
          label: const Text('Add'),
        ),
      ],
    );
  }

  Future<void> _save() async {
    final body = _controller.text.trim();
    if (body.isEmpty) {
      return;
    }
    setState(() => _saving = true);
    await runUiAction(context, () async {
      await ref
          .read(libraryRepositoryProvider)
          .createNote(
            documentId: widget.documentId,
            body: body,
            format: 'markdown',
          );
      _controller.clear();
      ref.invalidate(documentNotesProvider(widget.documentId));
    });
    if (mounted) {
      setState(() => _saving = false);
    }
  }
}

class _NoteList extends StatelessWidget {
  const _NoteList({required this.notes});

  final List<DocumentNote> notes;

  @override
  Widget build(BuildContext context) {
    if (notes.isEmpty) {
      return const Text('No notes yet.');
    }
    return Column(
      children: [
        for (final note in notes)
          Card(
            margin: const EdgeInsets.only(bottom: 8),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(8),
            ),
            child: ListTile(
              title: Text(note.body),
              subtitle: Text('Updated ${formatMillis(note.updatedAt)}'),
            ),
          ),
      ],
    );
  }
}
