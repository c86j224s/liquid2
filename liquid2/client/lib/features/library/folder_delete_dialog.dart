import 'package:flutter/material.dart';
import 'package:liquid2_api/liquid2_api.dart';

Future<bool> confirmFolderDelete(BuildContext context, Folder folder) async {
  final confirmed = await showDialog<bool>(
    context: context,
    builder: (context) => AlertDialog(
      title: const Text('Delete folder'),
      content: Text(folder.name),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(false),
          child: const Text('Cancel'),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context).pop(true),
          child: const Text('Delete'),
        ),
      ],
    ),
  );
  return confirmed == true;
}
