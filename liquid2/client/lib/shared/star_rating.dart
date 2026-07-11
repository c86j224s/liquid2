import 'package:flutter/material.dart';

import '../app/app_theme.dart';

/// Tap a star to set the rating; tap the same star again to clear it.
class StarRating extends StatelessWidget {
  const StarRating({
    required this.rating,
    required this.onChanged,
    this.size = 20.0,
    super.key,
  });

  final int? rating;
  final ValueChanged<int?> onChanged;
  final double size;

  @override
  Widget build(BuildContext context) {
    final filled = Theme.of(context).colorScheme.primary;
    final empty = Theme.of(context).colorScheme.outline;
    return Row(
      mainAxisSize: MainAxisSize.min,
      spacing: AppSpacing.xs,
      children: [
        for (var i = 1; i <= 5; i++)
          GestureDetector(
            onTap: () => onChanged(rating == i ? null : i),
            child: Icon(
              i <= (rating ?? 0) ? Icons.star_rounded : Icons.star_outline_rounded,
              size: size,
              color: i <= (rating ?? 0) ? filled : empty,
            ),
          ),
      ],
    );
  }
}

/// Read-only compact star display (for list tiles).
class StarRatingDisplay extends StatelessWidget {
  const StarRatingDisplay({required this.rating, this.size = 14.0, super.key});

  final int rating;
  final double size;

  @override
  Widget build(BuildContext context) {
    final color = Theme.of(context).colorScheme.primary;
    return Row(
      mainAxisSize: MainAxisSize.min,
      spacing: 2,
      children: [
        for (var i = 1; i <= 5; i++)
          Icon(
            i <= rating ? Icons.star_rounded : Icons.star_outline_rounded,
            size: size,
            color: i <= rating ? color : Theme.of(context).colorScheme.outline,
          ),
      ],
    );
  }
}
