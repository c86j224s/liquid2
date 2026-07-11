part of 'document_translation_panel.dart';

extension _DocumentTranslationPanelActions on _DocumentTranslationPanelState {
  Future<void> _enqueue(String? jobKey, String? sourceContentId) async {
    final language = _normalizedLanguage();
    final error = translationLanguageError(language);
    _update(() {
      _languageError = error;
      _notice = null;
    });
    if (jobKey == null || sourceContentId == null || error != null) {
      return;
    }
    _update(() => _submitting = true);
    try {
      final job = await ref
          .read(libraryRepositoryProvider)
          .translateDocument(
            documentId: widget.documentId,
            sourceContentId: sourceContentId,
            targetLanguage: language,
          );
      if (!mounted) {
        return;
      }
      ref.read(documentTranslationJobsProvider.notifier).track(jobKey, job);
      _schedulePoll(jobKey, job);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Translation queued: ${job.id}')));
    } catch (error) {
      _showError(error);
    } finally {
      if (mounted) {
        _update(() => _submitting = false);
      }
    }
  }

  Future<void> _refreshJob(String jobKey) async {
    final current = ref.read(documentTranslationJobsProvider)[jobKey];
    if (current == null || _refreshing) {
      return;
    }
    _update(() => _refreshing = true);
    try {
      final job = await ref.read(libraryRepositoryProvider).getJob(current.id);
      final latest = ref.read(documentTranslationJobsProvider)[jobKey];
      if (!mounted || latest?.id != current.id) {
        return;
      }
      ref.read(documentTranslationJobsProvider.notifier).track(jobKey, job);
      if (job.status == 'completed') {
        ref
          ..invalidate(documentDetailProvider(widget.documentId))
          ..invalidate(librarySnapshotProvider);
      }
      _schedulePoll(jobKey, job);
    } catch (error) {
      _showError(error);
    } finally {
      if (mounted) {
        _update(() => _refreshing = false);
      }
    }
  }

  void _schedulePoll(String jobKey, Job job) {
    _pollTimer?.cancel();
    _pollKey = null;
    _pollJobId = null;
    if (!_isActiveJob(job)) {
      return;
    }
    _pollKey = jobKey;
    _pollJobId = job.id;
    _pollTimer = Timer(const Duration(seconds: 2), () => _refreshJob(jobKey));
  }

  String _normalizedLanguage() => _targetController.text.trim().toLowerCase();

  void _syncPolling(String? jobKey, Job? job) {
    if (jobKey == null || job == null || !_isActiveJob(job)) {
      return;
    }
    if (_pollKey == jobKey &&
        _pollJobId == job.id &&
        (_pollTimer?.isActive ?? false)) {
      return;
    }
    _schedulePoll(jobKey, job);
  }

  bool _isActiveJob(Job? job) {
    return job?.status == 'queued' || job?.status == 'running';
  }

  void _targetLanguageChanged() {
    if (!mounted) {
      return;
    }
    _update(() {
      _languageError = null;
      _notice = null;
    });
  }

  void _showError(Object error) {
    if (!mounted) {
      return;
    }
    final message = _translationErrorMessage(error);
    if (error is TranslationAlreadyRunningException) {
      ref.invalidate(documentDetailProvider(widget.documentId));
      _update(() => _notice = message);
    }
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }

  String _translationErrorMessage(Object error) {
    if (error is TranslationAlreadyRunningException) {
      return 'Translation is already queued or running. The result will appear here when the current job completes.';
    }
    return error.toString();
  }
}
