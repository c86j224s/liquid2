import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../../app/providers.dart';
import '../../data/library_filters.dart';

class LibraryViewSelector extends ConsumerWidget {
  const LibraryViewSelector({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final filters = ref.watch(libraryFiltersProvider);
    final controller = ref.read(libraryFiltersProvider.notifier);
    return Wrap(
      spacing: 8,
      runSpacing: 8,
      children: [
        _ViewChip(
          icon: Icons.library_books,
          label: 'All',
          selected: filters.view == LibraryViewPreset.all,
          onSelected: () => controller.setView(LibraryViewPreset.all),
        ),
        _ViewChip(
          icon: Icons.mark_email_unread,
          label: 'Unread',
          selected: filters.view == LibraryViewPreset.unread,
          onSelected: () => controller.setView(LibraryViewPreset.unread),
        ),
        _ViewChip(
          icon: Icons.star,
          label: 'Rated',
          selected: filters.view == LibraryViewPreset.rated,
          onSelected: () => controller.setView(LibraryViewPreset.rated),
        ),
        _ViewChip(
          icon: Icons.schedule,
          label: 'Recent',
          selected: filters.view == LibraryViewPreset.recent,
          onSelected: () => controller.setView(LibraryViewPreset.recent),
        ),
      ],
    );
  }
}

class _ViewChip extends StatelessWidget {
  const _ViewChip({
    required this.icon,
    required this.label,
    required this.selected,
    required this.onSelected,
  });

  final IconData icon;
  final String label;
  final bool selected;
  final VoidCallback onSelected;

  @override
  Widget build(BuildContext context) {
    return ChoiceChip(
      avatar: Icon(icon, size: 18),
      label: Text(label),
      selected: selected,
      onSelected: (_) => onSelected(),
    );
  }
}
