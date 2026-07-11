import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'api_config.dart';
import 'app_router.dart';
import 'app_theme.dart';
import 'environment_badge.dart';
import 'providers.dart';

class Liquid2App extends ConsumerWidget {
  const Liquid2App({super.key});

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return MaterialApp.router(
      title: 'Liquid2',
      theme: AppTheme.light(),
      darkTheme: AppTheme.dark(),
      themeMode: ref.watch(themeModeProvider),
      routerConfig: ref.watch(appRouterProvider),
      builder: (context, child) => EnvironmentBadgeOverlay(
        label: configuredLiquid2EnvironmentLabel,
        child: child ?? const SizedBox.shrink(),
      ),
    );
  }
}
