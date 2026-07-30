import 'package:flutter/material.dart';

import 'ingest_mode.dart';

class IngestModeSelector extends StatelessWidget {
  const IngestModeSelector({
    required this.source,
    required this.onChanged,
    super.key,
  });

  final IngestSource source;
  final ValueChanged<IngestSource> onChanged;

  @override
  Widget build(BuildContext context) {
    return SegmentedButton<IngestSource>(
      segments: const [
        ButtonSegment(
          value: IngestSource.url,
          icon: Icon(Icons.link),
          label: Text('URL'),
        ),
        ButtonSegment(
          value: IngestSource.file,
          icon: Icon(Icons.upload_file),
          label: Text('File'),
        ),
      ],
      selected: {source},
      onSelectionChanged: (value) => onChanged(value.first),
    );
  }
}

class UrlSaveModeSelector extends StatelessWidget {
  const UrlSaveModeSelector({
    required this.mode,
    required this.onChanged,
    super.key,
  });

  final UrlSaveMode mode;
  final ValueChanged<UrlSaveMode> onChanged;

  @override
  Widget build(BuildContext context) {
    return SegmentedButton<UrlSaveMode>(
      segments: const [
        ButtonSegment(
          value: UrlSaveMode.linkOnly,
          icon: Icon(Icons.bookmark_add_outlined),
          label: Text('Link only'),
        ),
        ButtonSegment(
          value: UrlSaveMode.pageCopy,
          icon: Icon(Icons.travel_explore),
          label: Text('Save page'),
        ),
      ],
      selected: {mode},
      onSelectionChanged: (value) => onChanged(value.first),
    );
  }
}
