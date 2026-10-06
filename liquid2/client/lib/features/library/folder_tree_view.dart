import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../data/folder_tree.dart';
import '../../domain/folder_system_role.dart';
import 'library_pane_focus.dart';

class FolderTreeView extends ConsumerStatefulWidget {
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
  ConsumerState<FolderTreeView> createState() => _FolderTreeViewState();
}

class _FolderTreeViewState extends ConsumerState<FolderTreeView> {
  final _rowKeys = <int, GlobalKey>{};

  void _revealCursor(int row) {
    if (ref.read(libraryPaneProvider) != LibraryPane.folders) return;
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted) return;
      final target = _rowKeys[row]?.currentContext;
      if (target == null) return;
      Scrollable.ensureVisible(
        target,
        duration: const Duration(milliseconds: 160),
        curve: Curves.easeOut,
        alignment: 0.5,
      );
    });
  }

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    ref.listen(libraryFolderCursorProvider, (_, next) => _revealCursor(next));
    ref.listen(libraryPaneProvider, (_, next) {
      if (next == LibraryPane.folders) {
        _revealCursor(ref.read(libraryFolderCursorProvider));
      }
    });
    final cursor = ref.watch(libraryPaneProvider) == LibraryPane.folders
        ? ref.watch(libraryFolderCursorProvider)
        : -1;
    return Material(
      color: Colors.transparent,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          _keyed(
            0,
            FolderTreeRow(
              key: const Key('folder-tree-row-0'),
              label: 'All documents',
              depth: 0,
              selected: widget.selectedFolderId == null,
              isCursor: cursor == 0,
              icon: Icons.library_books,
              onTap: () => widget.onSelected(null),
            ),
          ),
          const SizedBox(height: 8),
          const _SectionLabel('Folders'),
          const SizedBox(height: 6),
          for (var index = 0; index < widget.items.length; index++)
            _keyed(
              index + 1,
              FolderTreeRow(
                key: Key('folder-tree-row-${index + 1}'),
                label: widget.items[index].folder.name,
                depth: widget.items[index].depth,
                selected:
                    widget.selectedFolderId == widget.items[index].folder.id,
                isCursor: cursor == index + 1,
                icon: _folderIcon(
                  widget.items[index].folder,
                  widget.selectedFolderId,
                ),
                onTap: () => widget.onSelected(widget.items[index].folder.id),
              ),
            ),
          if (widget.items.isEmpty)
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

  Widget _keyed(int row, Widget child) => KeyedSubtree(
    key: _rowKeys.putIfAbsent(row, GlobalKey.new),
    child: child,
  );
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

class FolderTreeRow extends StatelessWidget {
  const FolderTreeRow({
    required this.label,
    required this.depth,
    required this.selected,
    required this.isCursor,
    required this.icon,
    required this.onTap,
    super.key,
  });

  final String label;
  final int depth;
  final bool selected;
  final bool isCursor;
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
            border: isCursor
                ? Border.all(color: colors.primary, width: 1.5)
                : null,
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
