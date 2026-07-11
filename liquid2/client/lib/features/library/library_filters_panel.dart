import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/app_theme.dart';
import '../../app/providers.dart';
import '../../data/folder_tree.dart';
import '../../data/library_filters.dart';
import 'folder_management_dialog.dart';
import 'folder_tree_view.dart';
import 'library_search_field.dart';
import 'library_view_selector.dart';

class LibraryFiltersPanel extends ConsumerWidget {
  const LibraryFiltersPanel({
    required this.folders,
    required this.tags,
    this.scrollController,
    super.key,
  });

  final List<Folder> folders;
  final List<Tag> tags;
  final ScrollController? scrollController;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final filters = ref.watch(libraryFiltersProvider);
    final controller = ref.read(libraryFiltersProvider.notifier);
    final folderItems = flattenFolderTree(folders);

    return ColoredBox(
      color: Theme.of(context).colorScheme.surfaceContainerHighest,
      child: SingleChildScrollView(
        controller: scrollController,
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.lg,
          AppSpacing.md,
          AppSpacing.lg,
          AppSpacing.x2l,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const LibrarySearchField(),
            const SidebarSectionHeader('Views'),
            const LibraryViewSelector(),
            const SidebarSectionHeader('Status'),
            SegmentedButton<DocumentReadFilter>(
              segments: const [
                ButtonSegment(
                  value: DocumentReadFilter.unread,
                  label: Text('Unread'),
                ),
                ButtonSegment(
                  value: DocumentReadFilter.read,
                  label: Text('Read'),
                ),
              ],
              selected: {
                filters.read == DocumentReadFilter.all
                    ? DocumentReadFilter.unread
                    : filters.read,
              },
              onSelectionChanged: (value) => controller.setRead(value.first),
            ),
            Row(
              children: [
                const Expanded(child: SidebarSectionHeader('Folders')),
                IconButton(
                  tooltip: 'Manage folders',
                  onPressed: () => showFolderManagementDialog(
                    context: context,
                    folders: folders,
                  ),
                  icon: const Icon(Icons.drive_file_move_outline),
                ),
              ],
            ),
            FolderTreeView(
              items: folderItems,
              selectedFolderId: filters.folderId,
              onSelected: controller.setFolder,
            ),
            const SidebarSectionHeader('Tags'),
            Wrap(
              spacing: AppSpacing.xs,
              runSpacing: AppSpacing.xs,
              children: [
                FilterChip(
                  label: const Text('All'),
                  selected: filters.tagSlug == null,
                  onSelected: (_) => controller.setTag(null),
                ),
                for (final tag in tags)
                  FilterChip(
                    label: Text(tag.name),
                    selected: filters.tagSlug == tag.slug,
                    onSelected: (_) => controller.setTag(tag.slug),
                  ),
              ],
            ),
            const SidebarSectionHeader('Rating'),
            Wrap(
              spacing: AppSpacing.xs,
              runSpacing: AppSpacing.xs,
              children: [
                ChoiceChip(
                  label: const Text('Any'),
                  selected: filters.ratingMin == null,
                  onSelected: (_) => controller.setRatingMin(null),
                ),
                for (var rating = 1; rating <= 5; rating++)
                  ChoiceChip(
                    label: Text('$rating+'),
                    selected: filters.ratingMin == rating,
                    onSelected: (_) => controller.setRatingMin(rating),
                  ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
