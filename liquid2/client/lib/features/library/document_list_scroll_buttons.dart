import 'package:flutter/material.dart';

import '../../app/app_theme.dart';

class ScrollButtons extends StatelessWidget {
  const ScrollButtons({required this.onTop, required this.onBottom, super.key});

  final VoidCallback onTop;
  final VoidCallback onBottom;

  @override
  Widget build(BuildContext context) {
    final bg = Theme.of(context).colorScheme.onSurface;
    return Opacity(
      opacity: 0.25,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        spacing: AppSpacing.xs,
        children: [
          _ScrollFab(
            icon: Icons.keyboard_arrow_up_rounded,
            onTap: onTop,
            bg: bg,
          ),
          _ScrollFab(
            icon: Icons.keyboard_arrow_down_rounded,
            onTap: onBottom,
            bg: bg,
          ),
        ],
      ),
    );
  }
}

class _ScrollFab extends StatelessWidget {
  const _ScrollFab({required this.icon, required this.onTap, required this.bg});

  final IconData icon;
  final VoidCallback onTap;
  final Color bg;

  @override
  Widget build(BuildContext context) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        width: 36,
        height: 36,
        decoration: BoxDecoration(
          color: bg,
          borderRadius: const BorderRadius.all(AppRadius.md),
        ),
        child: Icon(
          icon,
          size: 20,
          color: Theme.of(context).colorScheme.surface,
        ),
      ),
    );
  }
}
