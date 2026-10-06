import 'package:flutter/material.dart';

import '../../app/app_theme.dart';

const hintAlphabet = 'asdfghjklqwertyuiopzxcvbnm';

class LibraryHintTarget {
  const LibraryHintTarget({required this.index, required this.rect});

  final int index;
  final Rect rect;

  LibraryHintTarget shift(Offset offset) =>
      LibraryHintTarget(index: index, rect: rect.shift(offset));
}

List<String> buildHintLabels(int count) {
  if (count <= hintAlphabet.length) {
    return [for (var i = 0; i < count; i++) hintAlphabet[i]];
  }
  final labels = <String>[];
  for (final first in hintAlphabet.split('')) {
    for (final second in hintAlphabet.split('')) {
      if (labels.length == count) return labels;
      labels.add('$first$second');
    }
  }
  return labels;
}

class LibraryHintLayer extends StatelessWidget {
  const LibraryHintLayer({
    required this.targets,
    required this.labels,
    required this.input,
    super.key,
  });

  final List<LibraryHintTarget> targets;
  final List<String> labels;
  final String input;

  @override
  Widget build(BuildContext context) {
    return IgnorePointer(
      child: Stack(
        children: [
          for (var i = 0; i < targets.length; i++)
            if (labels[i].startsWith(input))
              Positioned.fromRect(
                rect: targets[i].rect,
                child: Align(
                  alignment: Alignment.centerRight,
                  child: Padding(
                    padding: const EdgeInsets.only(right: AppSpacing.sm),
                    child: LibraryHintLabel(
                      label: labels[i],
                      matched: input.length,
                    ),
                  ),
                ),
              ),
        ],
      ),
    );
  }
}

class LibraryHintLabel extends StatelessWidget {
  const LibraryHintLabel({
    required this.label,
    required this.matched,
    super.key,
  });

  final String label;
  final int matched;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final scheme = theme.colorScheme;
    final style = theme.textTheme.labelSmall?.copyWith(
      color: scheme.onPrimary,
      fontWeight: FontWeight.w700,
      letterSpacing: 0.5,
    );
    return DecoratedBox(
      decoration: BoxDecoration(
        color: scheme.primary,
        borderRadius: const BorderRadius.all(AppRadius.sm),
        border: Border.all(color: scheme.outline),
      ),
      child: Padding(
        padding: const EdgeInsets.symmetric(
          horizontal: AppSpacing.xs,
          vertical: 1,
        ),
        child: Text.rich(
          TextSpan(
            children: [
              TextSpan(
                text: label.substring(0, matched).toUpperCase(),
                style: style?.copyWith(color: scheme.inversePrimary),
              ),
              TextSpan(text: label.substring(matched).toUpperCase()),
            ],
          ),
          style: style,
        ),
      ),
    );
  }
}
