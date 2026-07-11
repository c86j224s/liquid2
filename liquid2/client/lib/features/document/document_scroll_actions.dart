import 'package:flutter/material.dart';

import '../../app/app_theme.dart';

class DocumentScrollActions extends StatelessWidget {
  const DocumentScrollActions({
    required this.controller,
    required this.heroTagPrefix,
    super.key,
  });

  final ScrollController controller;
  final String heroTagPrefix;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        FloatingActionButton.small(
          heroTag: '$heroTagPrefix-top',
          tooltip: 'Scroll to top',
          onPressed: _scrollToTop,
          child: const Icon(Icons.keyboard_double_arrow_up),
        ),
        const SizedBox(height: AppSpacing.sm),
        FloatingActionButton.small(
          heroTag: '$heroTagPrefix-bottom',
          tooltip: 'Scroll to bottom',
          onPressed: _scrollToBottom,
          child: const Icon(Icons.keyboard_double_arrow_down),
        ),
      ],
    );
  }

  void _scrollToTop() {
    if (!controller.hasClients) return;
    _scrollTo(controller.position.minScrollExtent);
  }

  void _scrollToBottom() {
    if (!controller.hasClients) return;
    _scrollTo(controller.position.maxScrollExtent);
  }

  void _scrollTo(double offset) {
    controller.animateTo(
      offset,
      duration: const Duration(milliseconds: 240),
      curve: Curves.easeOutCubic,
    );
  }
}
