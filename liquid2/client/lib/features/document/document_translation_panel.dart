import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/providers.dart';
import '../../data/library_repository.dart';
import 'document_translation_helpers.dart';
import 'document_translation_jobs.dart';
import 'translation_job_line.dart';

part 'document_translation_panel_state.dart';

class DocumentTranslationPanel extends ConsumerStatefulWidget {
  const DocumentTranslationPanel({
    required this.documentId,
    required this.contents,
    super.key,
  });

  final String documentId;
  final List<DocumentContent> contents;
  @override
  ConsumerState<DocumentTranslationPanel> createState() =>
      _DocumentTranslationPanelState();
}

class _DocumentTranslationPanelState
    extends ConsumerState<DocumentTranslationPanel> {
  final _targetController = TextEditingController(text: 'ko');
  String? _sourceContentId;
  String? _languageError;
  String? _notice;
  Timer? _pollTimer;
  String? _pollKey;
  String? _pollJobId;
  var _submitting = false;
  var _refreshing = false;

  @override
  void initState() {
    super.initState();
    _targetController.addListener(_targetLanguageChanged);
  }

  @override
  void dispose() {
    _pollTimer?.cancel();
    _targetController.removeListener(_targetLanguageChanged);
    _targetController.dispose();
    super.dispose();
  }

  void _update(VoidCallback callback) => setState(callback);

  @override
  Widget build(BuildContext context) {
    final sources = translationSourceContents(widget.contents);
    final selected = selectedTranslationSourceId(sources, _sourceContentId);
    final language = _normalizedLanguage();
    final jobKey = selected == null
        ? null
        : documentTranslationJobKey(
            documentId: widget.documentId,
            sourceContentId: selected,
            targetLanguage: language,
          );
    final job = jobKey == null
        ? null
        : ref.watch(
            documentTranslationJobsProvider.select((jobs) => jobs[jobKey]),
          );
    final translationActive = _isActiveJob(job);
    _syncPolling(jobKey, job);
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(16),
      decoration: BoxDecoration(
        border: Border.all(color: Theme.of(context).dividerColor),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text('Translation', style: Theme.of(context).textTheme.titleMedium),
          const SizedBox(height: 12),
          Wrap(
            spacing: 12,
            runSpacing: 12,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              SizedBox(
                width: 280,
                child: DropdownButtonFormField<String>(
                  initialValue: selected,
                  isExpanded: true,
                  decoration: const InputDecoration(
                    labelText: 'Source content',
                  ),
                  items: [
                    for (final content in sources)
                      translationContentMenuItem(content),
                  ],
                  onChanged: sources.isEmpty
                      ? null
                      : (value) => setState(() => _sourceContentId = value),
                ),
              ),
              SizedBox(
                width: 180,
                child: TextField(
                  key: const Key('translation-target-language'),
                  controller: _targetController,
                  decoration: InputDecoration(
                    labelText: 'Target language',
                    errorText: _languageError,
                  ),
                  textInputAction: TextInputAction.done,
                  onSubmitted: (_) => _enqueue(jobKey, selected),
                ),
              ),
              FilledButton.icon(
                onPressed: _submitting || translationActive || selected == null
                    ? null
                    : () => _enqueue(jobKey, selected),
                icon: _submitting
                    ? const SizedBox.square(
                        dimension: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : translationActive
                    ? const Icon(Icons.sync)
                    : const Icon(Icons.translate),
                label: Text(translationActive ? 'Translating' : 'Translate'),
              ),
            ],
          ),
          if (_notice != null) ...[
            const SizedBox(height: 12),
            Text(_notice!, style: Theme.of(context).textTheme.bodySmall),
          ],
          if (job != null) ...[
            const SizedBox(height: 12),
            TranslationJobLine(
              job: job,
              refreshing: _refreshing,
              onRefresh: () => _refreshJob(jobKey!),
            ),
          ],
        ],
      ),
    );
  }
}
