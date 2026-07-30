import 'package:flutter/material.dart';

class IngestSubmitButton extends StatelessWidget {
  const IngestSubmitButton({
    required this.saving,
    required this.label,
    required this.onPressed,
    super.key,
  });

  final bool saving;
  final String label;
  final VoidCallback onPressed;

  @override
  Widget build(BuildContext context) {
    return FilledButton.icon(
      onPressed: saving ? null : onPressed,
      icon: saving
          ? const SizedBox.square(
              dimension: 16,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Icon(Icons.add),
      label: Text(saving ? 'Creating' : label),
    );
  }
}
