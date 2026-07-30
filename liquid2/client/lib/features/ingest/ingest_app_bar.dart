import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

class IngestAppBar extends StatelessWidget implements PreferredSizeWidget {
  const IngestAppBar({super.key});

  @override
  Size get preferredSize => const Size.fromHeight(kToolbarHeight);

  @override
  Widget build(BuildContext context) {
    return AppBar(
      leading: IconButton(
        tooltip: 'Back',
        onPressed: () => context.go('/'),
        icon: const Icon(Icons.arrow_back),
      ),
      title: const Text('Ingest'),
    );
  }
}
