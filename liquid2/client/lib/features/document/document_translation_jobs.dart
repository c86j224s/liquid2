import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:liquid2_api/liquid2_api.dart';

final documentTranslationJobsProvider =
    NotifierProvider<DocumentTranslationJobs, Map<String, Job>>(
      DocumentTranslationJobs.new,
    );

class DocumentTranslationJobs extends Notifier<Map<String, Job>> {
  @override
  Map<String, Job> build() => const {};

  void track(String key, Job job) {
    state = {...state, key: job};
  }

  void forget(String key) {
    if (!state.containsKey(key)) {
      return;
    }
    final next = {...state};
    next.remove(key);
    state = next;
  }
}

String documentTranslationJobKey({
  required String documentId,
  required String sourceContentId,
  required String targetLanguage,
}) {
  return '$documentId\n$sourceContentId\n$targetLanguage';
}
