import 'package:flutter/material.dart';

import 'app_tokens.dart';

class EnvironmentBadgeOverlay extends StatelessWidget {
  const EnvironmentBadgeOverlay({
    required this.label,
    required this.child,
    super.key,
  });

  final String label;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final normalized = label.trim();
    if (normalized.isEmpty) {
      return child;
    }

    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    return Stack(
      children: [
        child,
        Positioned(
          right: AppSpacing.sm,
          bottom: AppSpacing.sm,
          child: SafeArea(
            minimum: const EdgeInsets.all(AppSpacing.xs),
            child: IgnorePointer(
              child: DecoratedBox(
                decoration: BoxDecoration(
                  color: colors.tertiaryContainer.withValues(alpha: 0.88),
                  border: Border.all(color: colors.outlineVariant),
                  borderRadius: const BorderRadius.all(AppRadius.pill),
                  boxShadow: const [
                    BoxShadow(
                      color: Color(0x22000000),
                      blurRadius: 10,
                      offset: Offset(0, 3),
                    ),
                  ],
                ),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: AppSpacing.sm,
                    vertical: 2,
                  ),
                  child: Text(
                    normalized,
                    style: theme.textTheme.labelSmall?.copyWith(
                      color: colors.onTertiaryContainer,
                      fontWeight: FontWeight.w700,
                      letterSpacing: 0.4,
                      fontSize: 10,
                    ),
                  ),
                ),
              ),
            ),
          ),
        ),
      ],
    );
  }
}
