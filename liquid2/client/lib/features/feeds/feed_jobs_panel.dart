import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../shared/formatters.dart';

class FeedJobsPanel extends StatelessWidget {
  const FeedJobsPanel({required this.jobs, super.key});

  final List<Job> jobs;

  @override
  Widget build(BuildContext context) {
    if (jobs.isEmpty) {
      return Center(
        child: Text(
          'No RSS jobs',
          style: TextStyle(color: Theme.of(context).colorScheme.outline),
        ),
      );
    }
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: jobs.length,
      separatorBuilder: (_, _) => const Divider(height: 20),
      itemBuilder: (context, index) => _JobRow(job: jobs[index]),
    );
  }
}

class _JobRow extends StatelessWidget {
  const _JobRow({required this.job});

  final Job job;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Row(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Icon(_statusIcon(job.status), color: _statusColor(colors, job.status)),
        const SizedBox(width: 12),
        Expanded(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                '${compactKind(job.kind)} · ${job.status}',
                style: Theme.of(context).textTheme.titleSmall,
              ),
              const SizedBox(height: 4),
              Text(
                'Updated ${formatMillis(job.updatedAt)} · attempts ${job.attempts}',
                style: Theme.of(context).textTheme.bodySmall,
              ),
              if (job.error != null) ...[
                const SizedBox(height: 4),
                SelectableText(
                  job.error!,
                  style: TextStyle(color: colors.error),
                ),
              ],
            ],
          ),
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
