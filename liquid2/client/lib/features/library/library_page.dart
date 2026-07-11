import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../../app/providers.dart';
import '../../shared/async_panel.dart';
import 'document_list_panel.dart';
import 'library_filters_panel.dart';
import 'library_mobile_filter_bar.dart';

// Cycles through System → Light → Dark → System.
class _ThemeModeButton extends ConsumerWidget {
  const _ThemeModeButton();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final mode = ref.watch(themeModeProvider);
    final (icon, tooltip, next) = switch (mode) {
      ThemeMode.system => (Icons.brightness_auto, 'System theme', ThemeMode.light),
      ThemeMode.light  => (Icons.light_mode_outlined, 'Light theme', ThemeMode.dark),
      ThemeMode.dark   => (Icons.dark_mode_outlined, 'Dark theme', ThemeMode.system),
    };
    return IconButton(
      tooltip: tooltip,
      icon: Icon(icon),
      onPressed: () => ref.read(themeModeProvider.notifier).state = next,
    );
  }
}

class LibraryPage extends ConsumerWidget {
  const LibraryPage({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final snapshot = ref.watch(librarySnapshotProvider);
    return Scaffold(
      appBar: AppBar(
        title: const Text('Liquid2'),
        actions: [
          const _ThemeModeButton(),
          IconButton(
            tooltip: 'Feeds',
            onPressed: () => context.go('/feeds'),
            icon: const Icon(Icons.rss_feed),
          ),
          IconButton(
            tooltip: 'Ingest',
            onPressed: () => context.go('/ingest'),
            icon: const Icon(Icons.add),
          ),
          IconButton(
            tooltip: 'Refresh',
            onPressed: () => ref.invalidate(librarySnapshotProvider),
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: AsyncPanel(
        value: snapshot,
        onRetry: () => ref.invalidate(librarySnapshotProvider),
        builder: (data) {
          return LayoutBuilder(
            builder: (context, constraints) {
              final filters = LibraryFiltersPanel(
                folders: data.folders,
                tags: data.tags,
              );
              final list = DocumentListPanel(
                documents: data.documents,
                hasMore: data.hasMoreDocuments,
                isLoadingMore: data.isLoadingMore,
                totalCount: data.totalCount,
                loadMoreError: data.loadMoreError,
                onLoadMore: () {
                  ref.read(librarySnapshotProvider.notifier).loadMore();
                },
              );
              if (constraints.maxWidth < 820) {
                return Column(
                  children: [
                    LibraryMobileFilterBar(
                      folders: data.folders,
                      tags: data.tags,
                    ),
                    ActiveFilterChips(
                      folders: data.folders,
                      tags: data.tags,
                    ),
                    const Divider(height: 1),
                    Expanded(child: list),
                  ],
                );
              }
              return Row(
                children: [
                  SizedBox(width: 320, child: filters),
                  const VerticalDivider(width: 1),
                  Expanded(child: list),
                ],
              );
            },
          );
        },
      ),
    );
  }
}
