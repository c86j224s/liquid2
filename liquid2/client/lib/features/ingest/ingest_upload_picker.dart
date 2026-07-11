import 'package:flutter/material.dart';

class IngestUploadPicker extends StatelessWidget {
  const IngestUploadPicker({
    required this.fileName,
    required this.onPick,
    super.key,
  });

  final String? fileName;
  final VoidCallback onPick;

  @override
  Widget build(BuildContext context) {
    return Row(
      children: [
        OutlinedButton.icon(
          onPressed: onPick,
          icon: const Icon(Icons.attach_file),
          label: const Text('Choose file'),
        ),
        const SizedBox(width: 12),
        Expanded(
          child: Text(
            fileName ?? 'No file selected',
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
          ),
        ),
      ],
    );
  }
}
