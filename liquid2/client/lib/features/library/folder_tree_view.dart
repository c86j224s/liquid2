import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../data/folder_tree.dart';
import '../../domain/folder_system_role.dart';

class FolderTreeView extends StatelessWidget {
  const FolderTreeView({
    required this.items,
    required this.selectedFolderId,
    required this.onSelected,
    super.key,
  });

  final List<FolderTreeItem> items;
  final String? selectedFolderId;
  final ValueChanged<String?> onSelected;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Material(
      color: Colors.transparent,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _FilterRow(
            label: 'All documents',
            depth: 0,
            selected: selectedFolderId == null,
            icon: Icons.library_books,
            onTap: () => onSelected(null),
          ),
          const SizedBox(height: 8),
          const _SectionLabel('Folders'),
          const SizedBox(height: 6),
          for (final item in items)
            _FilterRow(
              label: item.folder.name,
              depth: item.depth,
              selected: selectedFolderId == item.folder.id,
              icon: _folderIcon(item.folder, selectedFolderId),
              onTap: () => onSelected(item.folder.id),
            ),
          if (items.isEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 8),
              child: Text(
                'No folders',
                style: TextStyle(color: colors.onSurfaceVariant),
              ),
            ),
        ],
      ),
    );
  }
}

IconData _folderIcon(Folder folder, String? selectedFolderId) {
  if (folder.systemRole == FolderSystemRole.feeds) {
    return Icons.rss_feed;
  }
  if (folder.systemRole == FolderSystemRole.trash) {
    return Icons.delete_outline;
  }
  return selectedFolderId == folder.id ? Icons.folder_open : Icons.folder;
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.label);

  final String label;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.only(left: 8),
      child: Text(
        label,
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
          color: colors.onSurfaceVariant,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _FilterRow extends StatelessWidget {
  const _FilterRow({
    required this.label,
    required this.depth,
    required this.selected,
    required this.icon,
    required this.onTap,
  });

  final String label;
  final int depth;
  final bool selected;
  final IconData icon;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final leftPadding = 8.0 + depth * 18.0;
    return Padding(
      padding: const EdgeInsets.only(bottom: 4),
      child: InkWell(
        borderRadius: BorderRadius.circular(8),
        onTap: onTap,
        child: Container(
          height: 40,
          padding: EdgeInsets.only(left: leftPadding, right: 8),
          decoration: BoxDecoration(
            color: selected ? colors.secondaryContainer : Colors.transparent,
            borderRadius: BorderRadius.circular(8),
          ),
          child: Row(
            children: [
              Icon(
                icon,
                size: 18,
                color: selected
                    ? colors.onSecondaryContainer
                    : colors.onSurfaceVariant,
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: selected ? colors.onSecondaryContainer : null,
                    fontWeight: selected ? FontWeight.w600 : FontWeight.w400,
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
