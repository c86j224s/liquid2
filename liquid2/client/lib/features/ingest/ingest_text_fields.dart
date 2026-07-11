import 'package:flutter/material.dart';

class IngestUrlField extends StatelessWidget {
  const IngestUrlField({required this.controller, super.key});

  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      decoration: const InputDecoration(
        border: OutlineInputBorder(),
        labelText: 'URL',
      ),
      keyboardType: TextInputType.url,
    );
  }
}

class IngestTitleField extends StatelessWidget {
  const IngestTitleField({required this.controller, super.key});

  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: controller,
      decoration: const InputDecoration(
        border: OutlineInputBorder(),
        labelText: 'Title',
      ),
    );
  }
}
