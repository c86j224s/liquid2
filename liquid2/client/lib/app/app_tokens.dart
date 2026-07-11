import 'package:flutter/material.dart';

abstract final class AppColors {
  static const teal = Color(0xFF2F6F73);
  static const tealLight = Color(0xFF4ECDC4);

  static const backgroundLight = Color(0xFFFAFAF8);
  static const surfaceLight = Color(0xFFFFFFFF);
  static const surfaceVariantLight = Color(0xFFF4F4F2);
  static const borderLight = Color(0xFFE4E4E2);

  static const backgroundDark = Color(0xFF111110);
  static const surfaceDark = Color(0xFF1C1C1B);
  static const surfaceVariantDark = Color(0xFF252524);
  static const borderDark = Color(0xFF2D2D2C);

  static const error = Color(0xFFDC2626);
}

abstract final class AppSpacing {
  static const xs = 4.0;
  static const sm = 8.0;
  static const md = 12.0;
  static const lg = 16.0;
  static const xl = 20.0;
  static const x2l = 24.0;
  static const x3l = 32.0;
  static const x4l = 48.0;
}

abstract final class AppRadius {
  static const sm = Radius.circular(4);
  static const md = Radius.circular(8);
  static const lg = Radius.circular(12);
  static const pill = Radius.circular(99);
}

const double kDetailMaxWidth = 680.0;
const double kUnreadBorderWidth = 3.0;

class SidebarSectionHeader extends StatelessWidget {
  const SidebarSectionHeader(this.label, {super.key});

  final String label;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(top: AppSpacing.lg, bottom: AppSpacing.sm),
      child: Text(
        label.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall,
      ),
    );
  }
}
