import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../shared/formatters.dart';

class TranslationJobLine extends StatelessWidget {
  const TranslationJobLine({
    required this.job,
    required this.refreshing,
    required this.onRefresh,
    super.key,
  });

  final Job job;
  final bool refreshing;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Row(
      children: [
        Icon(_statusIcon(job.status), color: _statusColor(colors, job.status)),
        const SizedBox(width: 8),
        Expanded(child: Text('${compactKind(job.kind)} · ${job.status}')),
        Text('Updated ${formatMillis(job.updatedAt)}'),
        IconButton(
          tooltip: 'Refresh translation status',
          onPressed: refreshing ? null : onRefresh,
          icon: refreshing
              ? const SizedBox.square(
                  dimension: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.refresh),
        ),
      ],
    );
  }

  IconData _statusIcon(String status) {
    return switch (status) {
      'completed' => Icons.check_circle,
      'failed' => Icons.error,
      'running' => Icons.sync,
      _ => Icons.schedule,
    };
  }

  Color _statusColor(ColorScheme colors, String status) {
    return switch (status) {
      'completed' => colors.primary,
      'failed' => colors.error,
      'running' => colors.tertiary,
      _ => colors.outline,
    };
  }
}
