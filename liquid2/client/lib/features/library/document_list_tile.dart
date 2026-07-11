import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/app_theme.dart';
import '../../shared/formatters.dart';
import '../../shared/star_rating.dart';

class DocumentListTile extends StatelessWidget {
  const DocumentListTile({required this.document, super.key});

  final DocumentSummary document;

  @override
  Widget build(BuildContext context) {
    final isRead = document.status == 'read';
    final theme = Theme.of(context);
    final primary = theme.colorScheme.primary;
    final tags = document.tags?.toList() ?? const <String>[];
    final folderPath = folderPathLabel(
      document.folderPath?.map((f) => f.name) ?? const <String>[],
    );

    return InkWell(
      onTap: () => context.go('/documents/${document.id}'),
      borderRadius: const BorderRadius.all(AppRadius.md),
      child: Container(
        decoration: BoxDecoration(
          border: Border(
            left: isRead
                ? BorderSide.none
                : BorderSide(color: primary, width: kUnreadBorderWidth),
          ),
          borderRadius: const BorderRadius.all(AppRadius.md),
          color: theme.colorScheme.surface,
        ),
        padding: const EdgeInsets.fromLTRB(
          AppSpacing.md,
          AppSpacing.md,
          AppSpacing.lg,
          AppSpacing.md,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              document.title,
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
              style: theme.textTheme.titleSmall?.copyWith(
                fontWeight: isRead ? FontWeight.w400 : FontWeight.w600,
                color: isRead
                    ? theme.colorScheme.onSurface.withValues(alpha: 0.6)
                    : null,
              ),
            ),
            const SizedBox(height: AppSpacing.xs),
            _MetaRow(document: document, folderPath: folderPath),
            if (tags.isNotEmpty) ...[
              const SizedBox(height: AppSpacing.xs),
              _TagRow(tags: tags),
            ],
          ],
        ),
      ),
    );
  }
}

class _MetaRow extends StatelessWidget {
  const _MetaRow({required this.document, required this.folderPath});

  final DocumentSummary document;
  final String folderPath;

  @override
  Widget build(BuildContext context) {
    final style = Theme.of(context).textTheme.bodySmall;
    final parts = <String>[
      compactKind(document.kind),
      if (folderPath.isNotEmpty) folderPath,
      documentTimeLabel(
        updatedAt: document.updatedAt,
        publishedAt: document.publishedAt,
      ),
    ];

    return Row(
      children: [
        Expanded(
          child: Text(
            parts.join('  ·  '),
            style: style,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ),
        if (document.rating != null) ...[
          const SizedBox(width: AppSpacing.sm),
          StarRatingDisplay(rating: document.rating!),
        ],
      ],
    );
  }
}

class _TagRow extends StatelessWidget {
  const _TagRow({required this.tags});

  final List<String> tags;

  @override
  Widget build(BuildContext context) {
    return Wrap(
      spacing: AppSpacing.xs,
      runSpacing: AppSpacing.xs,
      children: [for (final tag in tags.take(5)) Chip(label: Text(tag))],
    );
  }
}
