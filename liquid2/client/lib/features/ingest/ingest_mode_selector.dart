import 'package:flutter/material.dart';

import 'ingest_mode.dart';

class IngestModeSelector extends StatelessWidget {
  const IngestModeSelector({
    required this.mode,
    required this.onChanged,
    super.key,
  });

  final IngestMode mode;
  final ValueChanged<IngestMode> onChanged;

  @override
  Widget build(BuildContext context) {
    return SegmentedButton<IngestMode>(
      segments: const [
        ButtonSegment(
          value: IngestMode.bookmark,
          icon: Icon(Icons.bookmark_add_outlined),
          label: Text('Bookmark'),
        ),
        ButtonSegment(
          value: IngestMode.scrape,
          icon: Icon(Icons.travel_explore),
          label: Text('Scrape'),
        ),
        ButtonSegment(
          value: IngestMode.upload,
          icon: Icon(Icons.upload_file),
          label: Text('Upload'),
        ),
      ],
      selected: {mode},
      onSelectionChanged: (value) => onChanged(value.first),
    );
  }
}
