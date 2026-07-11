import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../shared/formatters.dart';

List<DocumentContent> translationSourceContents(
  List<DocumentContent> contents,
) {
  final primary = contents
      .where(
        (content) => content.id.isNotEmpty && content.role != 'translation',
      )
      .toList();
  return primary.isEmpty
      ? contents.where((content) => content.id.isNotEmpty).toList()
      : primary;
}

String? selectedTranslationSourceId(
  List<DocumentContent> sources,
  String? current,
) {
  if (sources.isEmpty) {
    return null;
  }
  if (sources.any((content) => content.id == current)) {
    return current;
  }
  return sources.first.id;
}

String? translationLanguageError(String language) {
  if (language.isEmpty) {
    return 'Required';
  }
  final valid = RegExp(r'^[a-zA-Z]{2,8}(-[a-zA-Z0-9]{2,8})*$');
  return valid.hasMatch(language) ? null : 'Invalid language';
}

String translationContentLabel(DocumentContent content) {
  final language = content.language == null ? '' : ' · ${content.language}';
  return '${compactKind(content.role)} · ${content.format}$language';
}

DropdownMenuItem<String> translationContentMenuItem(DocumentContent content) {
  return DropdownMenuItem(
    value: content.id,
    child: Text(
      translationContentLabel(content),
      overflow: TextOverflow.ellipsis,
    ),
  );
}
