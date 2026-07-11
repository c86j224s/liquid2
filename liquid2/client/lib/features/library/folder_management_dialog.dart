import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/providers.dart';
import '../../data/folder_tree.dart';
import '../../domain/folder_system_role.dart';
import '../../shared/action_feedback.dart';
import 'folder_delete_dialog.dart';
import 'folder_form_dialog.dart';

Future<void> showFolderManagementDialog({
  required BuildContext context,
  required List<Folder> folders,
}) {
  return showDialog<void>(
    context: context,
    builder: (context) => FolderManagementDialog(initialFolders: folders),
  );
}

class FolderManagementDialog extends ConsumerStatefulWidget {
  const FolderManagementDialog({required this.initialFolders, super.key});

  final List<Folder> initialFolders;

  @override
  ConsumerState<FolderManagementDialog> createState() {
    return _FolderManagementDialogState();
  }
}

class _FolderManagementDialogState
    extends ConsumerState<FolderManagementDialog> {
  late List<Folder> _folders;
  var _loading = false;

  @override
  void initState() {
    super.initState();
    _folders = widget.initialFolders;
  }

  @override
  Widget build(BuildContext context) {
    final items = flattenFolderTree(_folders);
    return AlertDialog(
      title: const Text('Manage folders'),
      content: SizedBox(
        width: 520,
        child: ConstrainedBox(
          constraints: const BoxConstraints(maxHeight: 520),
          child: _loading
              ? const Center(child: CircularProgressIndicator())
              : ListView(
                  shrinkWrap: true,
                  children: [
                    if (items.isEmpty)
                      const Padding(
                        padding: EdgeInsets.symmetric(vertical: 16),
                        child: Text('No folders'),
                      ),
                    for (final item in items)
                      _FolderRow(item: item, host: this),
                  ],
                ),
        ),
      ),
      actions: [
        IconButton(
          tooltip: 'Refresh',
          onPressed: _loading ? null : () => runUiAction(context, _refresh),
          icon: const Icon(Icons.refresh),
        ),
        IconButton(
          tooltip: 'Create folder',
          onPressed: _loading ? null : _createFolder,
          icon: const Icon(Icons.create_new_folder_outlined),
        ),
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Close'),
        ),
      ],
    );
  }

  Future<void> _refresh() async {
    setState(() => _loading = true);
    try {
      final folders = await ref.read(folderRepositoryProvider).listFolders();
      if (!mounted) {
        return;
      }
      setState(() {
        _folders = folders;
        _loading = false;
      });
    } catch (_) {
      if (mounted) {
        setState(() => _loading = false);
      }
      rethrow;
    }
  }

  Future<void> _createFolder() async {
    final input = await showFolderFormDialog(
      context: context,
      folders: _folders,
    );
    if (input == null || !mounted) {
      return;
    }
    await runUiAction(context, () async {
      await ref.read(folderRepositoryProvider).createFolder(input);
      await _refreshAfterMutation();
    });
  }

  Future<void> editFolder(Folder folder) async {
    final input = await showFolderFormDialog(
      context: context,
      folders: _folders,
      folder: folder,
    );
    if (input == null || !mounted) {
      return;
    }
    await runUiAction(context, () async {
      await ref.read(folderRepositoryProvider).updateFolder(folder.id, input);
      await _refreshAfterMutation();
    });
  }

  Future<void> deleteFolder(Folder folder) async {
    final confirmed = await confirmFolderDelete(context, folder);
    if (!confirmed || !mounted) {
      return;
    }
    await runUiAction(context, () async {
      await ref.read(folderRepositoryProvider).deleteFolder(folder.id);
      await _refreshAfterMutation();
    });
  }

  Future<void> _refreshAfterMutation() async {
    await _refresh();
    ref.invalidate(librarySnapshotProvider);
  }
}

class _FolderRow extends StatelessWidget {
  const _FolderRow({required this.item, required this.host});

  final FolderTreeItem item;
  final _FolderManagementDialogState host;

  @override
  Widget build(BuildContext context) {
    final folder = item.folder;
    final role = folder.systemRole;
    final system = role != null && role.isNotEmpty;
    return ListTile(
      contentPadding: EdgeInsets.only(left: 16.0 + item.depth * 18, right: 4),
      leading: Icon(_folderIcon(folder)),
      title: Text(folder.name, maxLines: 1, overflow: TextOverflow.ellipsis),
      subtitle: system ? Text(role) : null,
      trailing: system
          ? null
          : Wrap(
              children: [
                IconButton(
                  tooltip: 'Edit',
                  onPressed: () => host.editFolder(folder),
                  icon: const Icon(Icons.edit_outlined),
                ),
                IconButton(
                  tooltip: 'Delete',
                  onPressed: () => host.deleteFolder(folder),
                  icon: const Icon(Icons.delete_outline),
                ),
              ],
            ),
    );
  }
}

IconData _folderIcon(Folder folder) {
  if (folder.systemRole == FolderSystemRole.feeds) {
    return Icons.rss_feed;
  }
  return folder.systemRole == FolderSystemRole.trash
      ? Icons.delete_outline
      : Icons.folder;
}
