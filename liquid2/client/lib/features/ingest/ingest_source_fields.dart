import 'package:flutter/material.dart';

import 'ingest_mode.dart';
import 'ingest_mode_selector.dart';
import 'ingest_text_fields.dart';
import 'ingest_upload_picker.dart';

class IngestSourceFields extends StatelessWidget {
  const IngestSourceFields({
    required this.source,
    required this.urlSaveMode,
    required this.urlController,
    required this.titleController,
    required this.fileName,
    required this.onUrlSaveModeChanged,
    required this.onPickFile,
    super.key,
  });

  final IngestSource source;
  final UrlSaveMode urlSaveMode;
  final TextEditingController urlController;
  final TextEditingController titleController;
  final String? fileName;
  final ValueChanged<UrlSaveMode> onUrlSaveModeChanged;
  final VoidCallback onPickFile;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        if (source == IngestSource.url) ...[
          IngestUrlField(controller: urlController),
          const SizedBox(height: 12),
          UrlSaveModeSelector(
            mode: urlSaveMode,
            onChanged: onUrlSaveModeChanged,
          ),
        ] else
          IngestUploadPicker(fileName: fileName, onPick: onPickFile),
        if (_showsTitleField) ...[
          const SizedBox(height: 12),
          IngestTitleField(
            controller: titleController,
            label: source == IngestSource.url
                ? 'Title override (optional)'
                : 'Title (optional)',
          ),
        ],
      ],
    );
  }

  bool get _showsTitleField {
    return source == IngestSource.file ||
        (source == IngestSource.url && urlSaveMode == UrlSaveMode.linkOnly);
  }
}
