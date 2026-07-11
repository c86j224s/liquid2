import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/app_theme.dart';
import '../../app/providers.dart';
import '../../shared/async_panel.dart';
import '../../shared/formatters.dart';
import 'document_actions_bar.dart';
import 'document_content_view.dart';
import 'document_scroll_actions.dart';
import 'document_tag_editor.dart';
import 'document_translation_panel.dart';
import 'note_section.dart';

class DocumentDetailPage extends ConsumerStatefulWidget {
  const DocumentDetailPage({required this.id, super.key});

  final String id;

  @override
  ConsumerState<DocumentDetailPage> createState() => _DocumentDetailPageState();
}

class _DocumentDetailPageState extends ConsumerState<DocumentDetailPage> {
  final _scrollController = ScrollController();

  @override
  void dispose() {
    _scrollController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final detail = ref.watch(documentDetailProvider(widget.id));
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'Back',
          onPressed: () => context.go('/'),
          icon: const Icon(Icons.arrow_back),
        ),
        title: const Text('Document'),
        actions: [
          IconButton(
            tooltip: 'Refresh',
            onPressed: () {
              ref
                ..invalidate(documentDetailProvider(widget.id))
                ..invalidate(documentNotesProvider(widget.id));
            },
            icon: const Icon(Icons.refresh),
          ),
        ],
      ),
      body: AsyncPanel(
        value: detail,
        builder: (data) => _DocumentDetailBody(
          detail: data,
          scrollController: _scrollController,
        ),
      ),
      floatingActionButton: DocumentScrollActions(
        controller: _scrollController,
        heroTagPrefix: 'document-detail-${widget.id}',
      ),
    );
  }
}

class _DocumentDetailBody extends StatelessWidget {
  const _DocumentDetailBody({
    required this.detail,
    required this.scrollController,
  });

  final DocumentDetail detail;
  final ScrollController scrollController;

  @override
  Widget build(BuildContext context) {
    final document = detail.document;
    final contents = detail.contents?.toList() ?? const <DocumentContent>[];
    final tags = detail.tags?.toList() ?? const <Tag>[];
    final folderPath = folderPathLabel(
      detail.folderPath?.map((folder) => folder.name) ?? const <String>[],
    );

    return Center(
      child: ConstrainedBox(
        constraints: const BoxConstraints(maxWidth: kDetailMaxWidth),
        child: ListView(
          key: const Key('document-detail-scroll-view'),
          controller: scrollController,
          padding: const EdgeInsets.symmetric(
            horizontal: AppSpacing.x2l,
            vertical: AppSpacing.x2l,
          ),
          children: [
            // Title + meta header
            Text(
              document.title,
              style: Theme.of(context).textTheme.headlineMedium,
            ),
            const SizedBox(height: AppSpacing.sm),
            _MetaLine(document: document, folderPath: folderPath),
            const SizedBox(height: AppSpacing.x2l),
            DocumentActionsBar(document: document),
            const _SectionDivider(),
            const _SectionHeader('Tags'),
            const SizedBox(height: AppSpacing.md),
            DocumentTagEditor(documentId: document.id, assigned: tags),
            const _SectionDivider(),
            const _SectionHeader('Content'),
            const SizedBox(height: AppSpacing.md),
            DocumentContentView(contents: contents),
            const _SectionDivider(),
            NoteSection(documentId: document.id),
            const _SectionDivider(),
            DocumentTranslationPanel(
              documentId: document.id,
              contents: contents,
            ),
          ],
        ),
      ),
    );
  }
}

class _MetaLine extends StatelessWidget {
  const _MetaLine({required this.document, required this.folderPath});

  final DocumentMetadata document;
  final String folderPath;

  @override
  Widget build(BuildContext context) {
    final style = Theme.of(context).textTheme.bodySmall;
    final parts = <String>[
      compactKind(document.kind),
      if (folderPath.isNotEmpty) folderPath,
      documentTimeLabel(
        updatedAt: document.updatedAt,
        publishedAt: document.publishedAt,
      ),
    ];
    return Text(parts.join('  ·  '), style: style);
  }
}

class _SectionDivider extends StatelessWidget {
  const _SectionDivider();

  @override
  Widget build(BuildContext context) {
    return const Padding(
      padding: EdgeInsets.symmetric(vertical: AppSpacing.x2l),
      child: Divider(height: 1),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader(this.label);

  final String label;

  @override
  Widget build(BuildContext context) {
    return Text(label, style: Theme.of(context).textTheme.titleMedium);
  }
}
