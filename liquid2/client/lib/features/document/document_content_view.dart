import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_markdown_plus/flutter_markdown_plus.dart';
import 'package:liquid2_api/liquid2_api.dart';
import 'package:url_launcher/url_launcher.dart';

import '../../app/app_theme.dart';
import '../../shared/action_feedback.dart';
import '../../shared/formatters.dart';
import 'document_code_block.dart';

typedef DocumentLinkLauncher = Future<bool> Function(Uri url);

class DocumentContentView extends StatelessWidget {
  const DocumentContentView({
    required this.contents,
    this.launchLink = _defaultLaunchLink,
    super.key,
  });

  final List<DocumentContent> contents;
  final DocumentLinkLauncher launchLink;

  @override
  Widget build(BuildContext context) {
    if (contents.isEmpty) {
      return Text(
        'No content captured yet.',
        style: Theme.of(context).textTheme.bodySmall,
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        for (final content in contents)
          _ContentPanel(content: content, launchLink: launchLink),
      ],
    );
  }
}

class _ContentPanel extends StatelessWidget {
  const _ContentPanel({required this.content, required this.launchLink});

  final DocumentContent content;
  final DocumentLinkLauncher launchLink;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      margin: const EdgeInsets.only(bottom: AppSpacing.md),
      padding: const EdgeInsets.all(AppSpacing.lg),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest,
        borderRadius: const BorderRadius.all(AppRadius.md),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(_contentCaption(content), style: theme.textTheme.labelSmall),
          const SizedBox(height: AppSpacing.sm),
          _ContentBody(content: content, launchLink: launchLink),
        ],
      ),
    );
  }
}

class _ContentBody extends StatelessWidget {
  const _ContentBody({required this.content, required this.launchLink});

  final DocumentContent content;
  final DocumentLinkLauncher launchLink;

  @override
  Widget build(BuildContext context) {
    if (_isMarkdownFormat(content.format)) {
      final theme = Theme.of(context);
      return MarkdownBody(
        data: content.content.trim(),
        selectable: true,
        softLineBreak: true,
        styleSheet: MarkdownStyleSheet.fromTheme(theme).copyWith(
          codeblockPadding: EdgeInsets.zero,
          codeblockDecoration: const BoxDecoration(),
          p: theme.textTheme.bodyMedium,
          listBullet: theme.textTheme.bodyMedium,
        ),
        builders: documentMarkdownBuilders,
        onTapLink: (text, href, title) {
          unawaited(
            runUiAction(context, () => _openMarkdownLink(href, launchLink)),
          );
        },
      );
    }
    return SelectableText(
      readableBodyContent(content.content, format: content.format),
      style: Theme.of(context).textTheme.bodyMedium,
    );
  }
}

bool _isMarkdownFormat(String format) {
  final value = format.toLowerCase();
  return value == 'markdown' || value == 'text/markdown';
}

String _contentCaption(DocumentContent content) {
  final language = content.language == null ? '' : ' · ${content.language}';
  return '${compactKind(content.role)} · ${content.format}$language';
}

Future<void> _openMarkdownLink(
  String? href,
  DocumentLinkLauncher launchLink,
) async {
  final uri = _externalHTTPURL(href);
  if (uri == null) return;
  final opened = await launchLink(uri);
  if (!opened) throw StateError('Unable to open Markdown link.');
}

Future<bool> _defaultLaunchLink(Uri uri) {
  return launchUrl(uri, mode: LaunchMode.externalApplication);
}

Uri? _externalHTTPURL(String? rawURL) {
  final value = rawURL?.trim();
  if (value == null || value.isEmpty) return null;
  final uri = Uri.tryParse(value);
  if (uri == null || (uri.scheme != 'http' && uri.scheme != 'https')) {
    return null;
  }
  return uri;
}
