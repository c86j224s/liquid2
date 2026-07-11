import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../data/folder_repository.dart';
import '../../data/folder_tree.dart';

Future<FolderMutationInput?> showFolderFormDialog({
  required BuildContext context,
  required List<Folder> folders,
  Folder? folder,
}) {
  return showDialog<FolderMutationInput>(
    context: context,
    builder: (context) => FolderFormDialog(folders: folders, folder: folder),
  );
}

class FolderFormDialog extends StatefulWidget {
  const FolderFormDialog({required this.folders, this.folder, super.key});

  final List<Folder> folders;
  final Folder? folder;

  @override
  State<FolderFormDialog> createState() => _FolderFormDialogState();
}

class _FolderFormDialogState extends State<FolderFormDialog> {
  final _formKey = GlobalKey<FormState>();
  late final TextEditingController _nameController;
  late final TextEditingController _sortOrderController;
  String? _parentId;

  @override
  void initState() {
    super.initState();
    final folder = widget.folder;
    _nameController = TextEditingController(text: folder?.name ?? '');
    _sortOrderController = TextEditingController(
      text: '${folder?.sortOrder ?? 0}',
    );
    _parentId = _editableParentID(folder);
  }

  @override
  void dispose() {
    _nameController.dispose();
    _sortOrderController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final parents = _parentCandidates(widget.folders, widget.folder);
    return AlertDialog(
      title: Text(widget.folder == null ? 'Create folder' : 'Edit folder'),
      content: Form(
        key: _formKey,
        child: SizedBox(
          width: 420,
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              TextFormField(
                controller: _nameController,
                decoration: const InputDecoration(labelText: 'Name'),
                validator: _validateName,
              ),
              const SizedBox(height: 12),
              DropdownButtonFormField<String?>(
                initialValue: _parentId,
                decoration: const InputDecoration(labelText: 'Parent'),
                items: [
                  const DropdownMenuItem(
                    value: null,
                    child: Text('Root level'),
                  ),
                  for (final item in parents)
                    DropdownMenuItem(
                      value: item.folder.id,
                      child: Text('${'  ' * item.depth}${item.folder.name}'),
                    ),
                ],
                onChanged: (value) => setState(() => _parentId = value),
              ),
              const SizedBox(height: 12),
              TextFormField(
                controller: _sortOrderController,
                decoration: const InputDecoration(labelText: 'Sort order'),
                keyboardType: TextInputType.number,
                validator: _validateSortOrder,
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
        FilledButton(onPressed: _submit, child: const Text('Save')),
      ],
    );
  }

  String? _validateName(String? value) {
    if ((value ?? '').trim().isEmpty) {
      return 'Name is required.';
    }
    return null;
  }

  String? _validateSortOrder(String? value) {
    if (int.tryParse((value ?? '').trim()) == null) {
      return 'Enter a number.';
    }
    return null;
  }

  void _submit() {
    if (!(_formKey.currentState?.validate() ?? false)) {
      return;
    }
    Navigator.of(context).pop(
      FolderMutationInput(
        name: _nameController.text,
        parentId: _parentId,
        sortOrder: int.parse(_sortOrderController.text.trim()),
      ),
    );
  }
}

String? _editableParentID(Folder? folder) {
  if (folder == null || folder.parentId?.isEmpty == true) {
    return null;
  }
  return folder.parentId;
}

List<FolderTreeItem> _parentCandidates(List<Folder> folders, Folder? current) {
  final blocked = _blockedParentIDs(current);
  return flattenAssignableFolderTree(
    folders,
  ).where((item) => !blocked.contains(item.folder.id)).toList();
}

Set<String> _blockedParentIDs(Folder? current) {
  if (current == null) {
    return const {};
  }
  return {current.id, ..._descendantIDs(current)};
}

Iterable<String> _descendantIDs(Folder folder) sync* {
  for (final child in folder.children?.toList() ?? const <Folder>[]) {
    yield child.id;
    yield* _descendantIDs(child);
  }
}
