import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import '../features/document/document_detail_page.dart';
import '../features/feeds/feed_page.dart';
import '../features/ingest/ingest_page.dart';
import '../features/library/library_page.dart';

final appRouterProvider = Provider<GoRouter>((ref) {
  return GoRouter(
    routes: [
      GoRoute(
        path: '/',
        pageBuilder: (context, state) => _fadePage(
          key: state.pageKey,
          child: const LibraryPage(),
        ),
        routes: [
          GoRoute(
            path: 'ingest',
            pageBuilder: (context, state) => _fadePage(
              key: state.pageKey,
              child: const IngestPage(),
            ),
          ),
          GoRoute(
            path: 'feeds',
            pageBuilder: (context, state) => _fadePage(
              key: state.pageKey,
              child: const FeedPage(),
            ),
          ),
          GoRoute(
            path: 'documents/:id',
            pageBuilder: (context, state) => _fadePage(
              key: state.pageKey,
              child: DocumentDetailPage(id: state.pathParameters['id']!),
            ),
          ),
        ],
      ),
    ],
  );
});

CustomTransitionPage<void> _fadePage({
  required LocalKey key,
  required Widget child,
}) {
  return CustomTransitionPage<void>(
    key: key,
    child: child,
    transitionDuration: const Duration(milliseconds: 180),
    reverseTransitionDuration: const Duration(milliseconds: 120),
    transitionsBuilder: (context, animation, secondaryAnimation, child) {
      return FadeTransition(
        opacity: CurvedAnimation(parent: animation, curve: Curves.easeIn),
        child: child,
      );
    },
  );
}
