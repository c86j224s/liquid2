import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../shared/formatters.dart';

class FeedSchedulerSettingsPanel extends StatelessWidget {
  const FeedSchedulerSettingsPanel({
    required this.settings,
    required this.onEnabledChanged,
    required this.onIntervalChanged,
    super.key,
  });

  final AppSettings settings;
  final ValueChanged<bool> onEnabledChanged;
  final ValueChanged<int> onIntervalChanged;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    final intervals = _intervalOptions(settings.feedPollIntervalSeconds);
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
      child: Card(
        margin: EdgeInsets.zero,
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Wrap(
            spacing: 16,
            runSpacing: 12,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              Icon(
                Icons.schedule,
                color: settings.feedSchedulerEnabled
                    ? colors.primary
                    : colors.outline,
              ),
              Text(
                'Auto refresh RSS',
                style: Theme.of(context).textTheme.titleMedium,
              ),
              Tooltip(
                message: settings.feedSchedulerEnabled
                    ? 'Disable RSS auto refresh'
                    : 'Enable RSS auto refresh',
                child: Switch(
                  value: settings.feedSchedulerEnabled,
                  onChanged: onEnabledChanged,
                ),
              ),
              SizedBox(
                width: 150,
                child: DropdownButtonFormField<int>(
                  initialValue: settings.feedPollIntervalSeconds,
                  decoration: const InputDecoration(
                    labelText: 'Interval',
                    isDense: true,
                  ),
                  items: intervals
                      .map(
                        (entry) => DropdownMenuItem(
                          value: entry.seconds,
                          child: Text(entry.label),
                        ),
                      )
                      .toList(),
                  onChanged: (value) {
                    if (value != null) {
                      onIntervalChanged(value);
                    }
                  },
                ),
              ),
              _NextPollLabel(settings: settings),
            ],
          ),
        ),
      ),
    );
  }
}

class _NextPollLabel extends StatelessWidget {
  const _NextPollLabel({required this.settings});

  final AppSettings settings;

  @override
  Widget build(BuildContext context) {
    final nextPoll = settings.feedSchedulerEnabled
        ? formatFutureMillis(settings.feedNextPollAt)
        : '-';
    return Text(
      'Next check $nextPoll',
      style: Theme.of(context).textTheme.bodySmall,
    );
  }
}

List<_IntervalOption> _intervalOptions(int currentSeconds) {
  if (_intervals.any((entry) => entry.seconds == currentSeconds)) {
    return _intervals;
  }
  return [
    _IntervalOption(currentSeconds, _formatInterval(currentSeconds)),
    ..._intervals,
  ];
}

const _intervals = [
  _IntervalOption(60, '1m'),
  _IntervalOption(300, '5m'),
  _IntervalOption(900, '15m'),
  _IntervalOption(1800, '30m'),
  _IntervalOption(3600, '1h'),
  _IntervalOption(7200, '2h'),
  _IntervalOption(21600, '6h'),
];

class _IntervalOption {
  const _IntervalOption(this.seconds, this.label);

  final int seconds;
  final String label;
}

String _formatInterval(int seconds) {
  if (seconds % 3600 == 0) {
    return '${seconds ~/ 3600}h';
  }
  if (seconds % 60 == 0) {
    return '${seconds ~/ 60}m';
  }
  return '${seconds}s';
}
