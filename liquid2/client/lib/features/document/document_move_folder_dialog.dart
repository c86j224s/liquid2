import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/providers.dart';
import '../../data/folder_tree.dart';
import '../../domain/folder_system_role.dart';
import '../../shared/action_feedback.dart';

Future<bool> showDocumentMoveFolderDialog(
  BuildContext context,
  DocumentMetadata document,
) async {
  final moved = await showDialog<bool>(
    context: context,
    builder: (context) => _DocumentMoveFolderDialog(document: document),
  );
  return moved ?? false;
}

class _DocumentMoveFolderDialog extends ConsumerStatefulWidget {
  const _DocumentMoveFolderDialog({required this.document});

  final DocumentMetadata document;

  @override
  ConsumerState<_DocumentMoveFolderDialog> createState() =>
      _DocumentMoveFolderDialogState();
}

class _DocumentMoveFolderDialogState
    extends ConsumerState<_DocumentMoveFolderDialog> {
  late Future<List<FolderTreeItem>> _foldersFuture;
  String? _selectedFolderId;
  var _saving = false;

  @override
  void initState() {
    super.initState();
    _foldersFuture = _loadFolders();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Move document'),
      content: SizedBox(
        width: 360,
        child: FutureBuilder<List<FolderTreeItem>>(
          future: _foldersFuture,
          builder: (context, snapshot) {
            if (snapshot.connectionState != ConnectionState.done) {
              return const SizedBox(
                height: 96,
                child: Center(child: CircularProgressIndicator()),
              );
            }
            if (snapshot.hasError) {
              return Text(snapshot.error.toString());
            }
            final folders = snapshot.data ?? const [];
            if (folders.isEmpty) {
              return const Text('No folders available.');
            }
            return DropdownButtonFormField<String>(
              initialValue: _selectedFolderId,
              decoration: const InputDecoration(labelText: 'Folder'),
              items: [
                for (final item in folders)
                  DropdownMenuItem(
                    value: item.folder.id,
                    child: Text('${'  ' * item.depth}${item.folder.name}'),
                  ),
              ],
              onChanged: _saving
                  ? null
                  : (value) => setState(() => _selectedFolderId = value),
            );
          },
        ),
      ),
      actions: [
        TextButton(
          onPressed: _saving ? null : () => Navigator.of(context).pop(false),
          child: const Text('Cancel'),
        ),
        FilledButton(
          onPressed: _saving || _selectedFolderId == null ? null : _move,
          child: _saving
              ? const SizedBox.square(
                  dimension: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Text('Move'),
        ),
      ],
    );
  }

  Future<List<FolderTreeItem>> _loadFolders() async {
    final folders = await ref.read(folderRepositoryProvider).listFolders();
    final items = flattenManualDocumentFolderTree(
      folders,
    ).where((item) => item.folder.id != widget.document.folderId).toList();
    if (items.isNotEmpty) {
      _selectedFolderId = items.first.folder.id;
    }
    return items;
  }

  Future<void> _move() {
    return runUiAction(context, () async {
      final folderId = _selectedFolderId;
      if (folderId == null) return;
      setState(() => _saving = true);
      try {
        await ref
            .read(libraryRepositoryProvider)
            .moveDocumentToFolder(widget.document.id, folderId);
        ref
          ..invalidate(documentDetailProvider(widget.document.id))
          ..invalidate(librarySnapshotProvider);
        if (!mounted) return;
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(const SnackBar(content: Text('Moved document.')));
        Navigator.of(context).pop(true);
      } finally {
        if (mounted) setState(() => _saving = false);
      }
    });
  }
}

List<FolderTreeItem> flattenManualDocumentFolderTree(List<Folder> folders) {
  return [
    for (final folder in folders) ..._flattenManualDocumentFolder(folder, 0),
  ];
}

Iterable<FolderTreeItem> _flattenManualDocumentFolder(
  Folder folder,
  int depth,
) sync* {
  if (folder.systemRole == FolderSystemRole.feeds ||
      folder.systemRole == FolderSystemRole.trash) {
    return;
  }
  yield FolderTreeItem(folder: folder, depth: depth);
  for (final child in folder.children?.toList() ?? const <Folder>[]) {
    yield* _flattenManualDocumentFolder(child, depth + 1);
  }
}
