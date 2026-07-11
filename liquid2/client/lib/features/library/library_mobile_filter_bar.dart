import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/app_theme.dart';
import '../../app/providers.dart';
import '../../data/library_filters.dart';
import 'library_filter_sheet.dart';
import 'library_search_field.dart';

/// Compact top bar for narrow screens: search + filter button.
class LibraryMobileFilterBar extends StatelessWidget {
  const LibraryMobileFilterBar({
    required this.folders,
    required this.tags,
    super.key,
  });

  final List<Folder> folders;
  final List<Tag> tags;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(
        AppSpacing.md,
        AppSpacing.sm,
        AppSpacing.sm,
        AppSpacing.sm,
      ),
      child: Row(
        children: [
          const Expanded(child: LibrarySearchField()),
          const SizedBox(width: AppSpacing.xs),
          IconButton(
            tooltip: 'Filters',
            onPressed: () => showLibraryFilterSheet(
              context,
              folders: folders,
              tags: tags,
            ),
            icon: const Icon(Icons.tune),
          ),
        ],
      ),
    );
  }
}

/// Horizontal scrollable row of active filter chips.
class ActiveFilterChips extends ConsumerWidget {
  const ActiveFilterChips({
    required this.folders,
    required this.tags,
    super.key,
  });

  final List<Folder> folders;
  final List<Tag> tags;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final filters = ref.watch(libraryFiltersProvider);
    final ctrl = ref.read(libraryFiltersProvider.notifier);

    final chips = <Widget>[
      if (filters.view != null && filters.view != LibraryViewPreset.all)
        _ActiveChip(
          label: _viewLabel(filters.view!),
          onRemove: () => ctrl.setView(LibraryViewPreset.all),
        ),
      if (filters.read == DocumentReadFilter.read)
        _ActiveChip(
          label: 'Read',
          onRemove: () => ctrl.setRead(DocumentReadFilter.unread),
        ),
      if (filters.folderId != null)
        _ActiveChip(
          label: _folderName(filters.folderId!, folders),
          onRemove: () => ctrl.setFolder(null),
        ),
      if (filters.tagSlug != null)
        _ActiveChip(
          label: _tagName(filters.tagSlug!, tags),
          onRemove: () => ctrl.setTag(null),
        ),
      if (filters.ratingMin != null)
        _ActiveChip(
          label: '${filters.ratingMin}+ ★',
          onRemove: () => ctrl.setRatingMin(null),
        ),
    ];

    if (chips.isEmpty) return const SizedBox.shrink();

    return SizedBox(
      height: 36,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: AppSpacing.md),
        itemCount: chips.length,
        separatorBuilder: (_, _) => const SizedBox(width: AppSpacing.xs),
        itemBuilder: (_, i) => chips[i],
      ),
    );
  }

  String _viewLabel(LibraryViewPreset view) => switch (view) {
    LibraryViewPreset.all    => 'All',
    LibraryViewPreset.unread => 'Unread',
    LibraryViewPreset.rated  => 'Rated',
    LibraryViewPreset.recent => 'Recent',
  };

  String _folderName(String id, List<Folder> folders) {
    for (final f in folders) {
      if (f.id == id) return f.name;
      for (final c in f.children ?? <Folder>[]) {
        if (c.id == id) return c.name;
      }
    }
    return 'Folder';
  }

  String _tagName(String slug, List<Tag> tags) {
    final match = tags.where((t) => t.slug == slug).firstOrNull;
    return match?.name ?? slug;
  }
}

class _ActiveChip extends StatelessWidget {
  const _ActiveChip({required this.label, required this.onRemove});

  final String label;
  final VoidCallback onRemove;

  @override
  Widget build(BuildContext context) {
    return InputChip(
      label: Text(label),
      onDeleted: onRemove,
      deleteIcon: const Icon(Icons.close, size: 14),
      visualDensity: VisualDensity.compact,
      materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
    );
  }
}
