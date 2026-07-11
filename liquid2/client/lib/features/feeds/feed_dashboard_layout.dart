import 'package:flutter/material.dart';

class FeedDashboardLayout extends StatelessWidget {
  const FeedDashboardLayout({
    required this.feeds,
    required this.jobs,
    super.key,
  });

  final Widget feeds;
  final Widget jobs;

  @override
  Widget build(BuildContext context) {
    return LayoutBuilder(
      builder: (context, constraints) {
        if (constraints.maxWidth < 900) {
          return Column(
            children: [
              Expanded(flex: 3, child: feeds),
              const Divider(height: 1),
              Expanded(flex: 2, child: jobs),
            ],
          );
        }
        return Row(
          children: [
            Expanded(flex: 3, child: feeds),
            const VerticalDivider(width: 1),
            Expanded(flex: 2, child: jobs),
          ],
        );
      },
    );
  }
}
