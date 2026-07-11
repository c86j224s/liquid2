import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/app_theme.dart';
import 'library_filters_panel.dart';

void showLibraryFilterSheet(
  BuildContext context, {
  required List<Folder> folders,
  required List<Tag> tags,
}) {
  showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    useSafeArea: true,
    shape: const RoundedRectangleBorder(
      borderRadius: BorderRadius.vertical(top: AppRadius.lg),
    ),
    builder: (_) => DraggableScrollableSheet(
      initialChildSize: 0.75,
      maxChildSize: 0.95,
      minChildSize: 0.4,
      expand: false,
      builder: (_, scrollController) => Column(
        children: [
          const _SheetHandle(),
          Expanded(
            child: LibraryFiltersPanel(
              folders: folders,
              tags: tags,
              scrollController: scrollController,
            ),
          ),
        ],
      ),
    ),
  );
}

class _SheetHandle extends StatelessWidget {
  const _SheetHandle();

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: AppSpacing.sm),
      child: Center(
        child: Container(
          width: 36,
          height: 4,
          decoration: BoxDecoration(
            color: Theme.of(context).colorScheme.outline,
            borderRadius: const BorderRadius.all(AppRadius.pill),
          ),
        ),
      ),
    );
  }
}
