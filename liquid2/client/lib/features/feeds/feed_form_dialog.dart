import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../data/feed_repository.dart';
import '../../data/folder_tree.dart';

Future<FeedInput?> showFeedFormDialog({
  required BuildContext context,
  required List<Folder> folders,
  Feed? feed,
}) {
  return showDialog<FeedInput>(
    context: context,
    builder: (context) => FeedFormDialog(folders: folders, feed: feed),
  );
}

class FeedFormDialog extends StatefulWidget {
  const FeedFormDialog({required this.folders, this.feed, super.key});

  final List<Folder> folders;
  final Feed? feed;

  @override
  State<FeedFormDialog> createState() => _FeedFormDialogState();
}

class _FeedFormDialogState extends State<FeedFormDialog> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _urlController;
  late final TextEditingController _titleController;
  late bool _enabled;
  String? _folderId;

  @override
  void initState() {
    super.initState();
    final feed = widget.feed;
    _urlController = TextEditingController(text: feed?.url ?? '');
    _titleController = TextEditingController(text: feed?.title ?? '');
    _folderId = feed?.folderId;
    _enabled = feed?.enabled ?? true;
  }

  @override
  void dispose() {
    _urlController.dispose();
    _titleController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final isEditing = widget.feed != null;
    final folders = isEditing
        ? flattenAssignableFolderTree(widget.folders)
        : const <FolderTreeItem>[];
    return AlertDialog(
      title: Text(isEditing ? 'Edit feed' : 'Add feed'),
      content: Form(
        key: _formKey,
        child: SizedBox(
          width: 420,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextFormField(
                controller: _urlController,
                decoration: const InputDecoration(labelText: 'Feed URL'),
                keyboardType: TextInputType.url,
                validator: _validateURL,
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _titleController,
                decoration: const InputDecoration(labelText: 'Title'),
              ),
              if (isEditing) ...[
                const SizedBox(height: 12),
                DropdownButtonFormField<String?>(
                  initialValue: _folderId,
                  decoration: const InputDecoration(labelText: 'Folder'),
                  items: [
                    const DropdownMenuItem(
                      value: null,
                      child: Text('No folder'),
                    ),
                    for (final item in folders)
                      DropdownMenuItem(
                        value: item.folder.id,
                        child: Text('${'  ' * item.depth}${item.folder.name}'),
                      ),
                  ],
                  onChanged: (value) => setState(() => _folderId = value),
                ),
              ],
              const SizedBox(height: 8),
              SwitchListTile(
                contentPadding: EdgeInsets.zero,
                title: const Text('Enabled'),
                value: _enabled,
                onChanged: (value) => setState(() => _enabled = value),
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(
          onPressed: _submit,
          child: Text(widget.feed == null ? 'Create' : 'Save'),
        ),
      ],
    );
  }

  String? _validateURL(String? value) {
    final text = value?.trim() ?? '';
    if (text.isEmpty) {
      return 'Feed URL is required.';
    }
    final parsed = Uri.tryParse(text);
    if (parsed == null || !parsed.hasAuthority) {
      return 'Enter a valid URL.';
    }
    if (parsed.scheme != 'http' && parsed.scheme != 'https') {
      return 'Use http or https.';
    }
    return null;
  }

  void _submit() {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    Navigator.of(context).pop(
      FeedInput(
        url: _urlController.text,
        title: _titleController.text,
        folderId: _folderId,
        enabled: _enabled,
      ),
    );
  }
}
