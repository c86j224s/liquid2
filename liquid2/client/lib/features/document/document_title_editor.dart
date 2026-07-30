import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/app_theme.dart';
import '../../app/providers.dart';
import '../../shared/action_feedback.dart';

class DocumentTitleHeader extends ConsumerWidget {
  const DocumentTitleHeader({required this.document, super.key});

  final DocumentMetadata document;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Expanded(
          child: Text(
            document.title,
            style: Theme.of(context).textTheme.headlineMedium,
          ),
        ),
        const SizedBox(width: AppSpacing.sm),
        IconButton(
          tooltip: 'Edit title',
          onPressed: () => _editTitle(context, ref),
          icon: const Icon(Icons.edit_outlined),
        ),
      ],
    );
  }

  Future<void> _editTitle(BuildContext context, WidgetRef ref) async {
    final title = await showDocumentTitleDialog(
      context: context,
      initialTitle: document.title,
    );
    if (title == null || title == document.title.trim()) {
      return;
    }
    if (!context.mounted) {
      return;
    }
    await runUiAction(context, () async {
      await ref
          .read(libraryRepositoryProvider)
          .renameDocument(document.id, title);
      ref
        ..invalidate(documentDetailProvider(document.id))
        ..invalidate(librarySnapshotProvider);
      if (context.mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('Updated title.')));
      }
    });
  }
}

Future<String?> showDocumentTitleDialog({
  required BuildContext context,
  required String initialTitle,
}) {
  return showDialog<String>(
    context: context,
    builder: (context) => _DocumentTitleDialog(initialTitle: initialTitle),
  );
}

class _DocumentTitleDialog extends StatefulWidget {
  const _DocumentTitleDialog({required this.initialTitle});

  final String initialTitle;

  @override
  State<_DocumentTitleDialog> createState() => _DocumentTitleDialogState();
}

class _DocumentTitleDialogState extends State<_DocumentTitleDialog> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _controller;

  @override
  void initState() {
    super.initState();
    _controller = TextEditingController(text: widget.initialTitle);
  }

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Edit title'),
      content: Form(
        key: _formKey,
        child: SizedBox(
          width: 420,
          child: TextFormField(
            controller: _controller,
            autofocus: true,
            decoration: const InputDecoration(labelText: 'Title'),
            validator: _validateTitle,
            onFieldSubmitted: (_) => _submit(),
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(onPressed: _submit, child: const Text('Save')),
      ],
    );
  }

  String? _validateTitle(String? value) {
    if ((value ?? '').trim().isEmpty) {
      return 'Title is required.';
    }
    return null;
  }

  void _submit() {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    Navigator.of(context).pop(_controller.text.trim());
  }
}
