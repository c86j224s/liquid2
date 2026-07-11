import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../data/folder_tree.dart';

class IngestFolderSelect extends StatelessWidget {
  const IngestFolderSelect({
    required this.folders,
    required this.selectedFolderId,
    required this.onChanged,
    super.key,
  });

  final List<FolderTreeItem> folders;
  final String? selectedFolderId;
  final ValueChanged<String?> onChanged;

  @override
  Widget build(BuildContext context) {
    return DropdownButtonFormField<String?>(
      initialValue: selectedFolderId,
      decoration: const InputDecoration(
        border: OutlineInputBorder(),
        labelText: 'Folder',
      ),
      items: [
        const DropdownMenuItem(value: null, child: Text('No folder')),
        for (final item in folders)
          DropdownMenuItem(
            value: item.folder.id,
            child: Text('${'  ' * item.depth}${item.folder.name}'),
          ),
      ],
      onChanged: onChanged,
    );
  }
}

class IngestTagSelect extends StatelessWidget {
  const IngestTagSelect({
    required this.tags,
    required this.selectedTagIds,
    required this.onChanged,
    super.key,
  });

  final List<Tag> tags;
  final Set<String> selectedTagIds;
  final ValueChanged<Set<String>> onChanged;

  @override
  Widget build(BuildContext context) {
    if (tags.isEmpty) {
      return Text(
        'No tags',
        style: TextStyle(color: Theme.of(context).colorScheme.onSurfaceVariant),
      );
    }
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        for (final tag in tags)
          FilterChip(
            label: Text(tag.name),
            selected: selectedTagIds.contains(tag.id),
            onSelected: (selected) {
              final next = {...selectedTagIds};
              if (selected) {
                next.add(tag.id);
              } else {
                next.remove(tag.id);
              }
              onChanged(next);
            },
          ),
      ],
    );
  }
}
