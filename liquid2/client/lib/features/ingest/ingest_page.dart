import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/providers.dart';
import '../../data/folder_tree.dart';
import '../../data/library_repository.dart';
import 'ingest_filters.dart';
import 'ingest_mode.dart';
import 'ingest_mode_selector.dart';
import 'ingest_text_fields.dart';
import 'ingest_upload_file.dart';
import 'ingest_upload_picker.dart';

class IngestPage extends ConsumerStatefulWidget {
  const IngestPage({super.key});

  @override
  ConsumerState<IngestPage> createState() => _IngestPageState();
}

class _IngestPageState extends ConsumerState<IngestPage> {
  final _urlController = TextEditingController();
  final _titleController = TextEditingController();
  final _selectedTagIds = <String>{};
  IngestMode _mode = IngestMode.bookmark;
  String? _folderId;
  String? _filename;
  Uint8List? _fileBytes;
  var _saving = false;

  @override
  void dispose() {
    _urlController.dispose();
    _titleController.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final snapshot = ref.watch(librarySnapshotProvider).value;
    final folders = flattenAssignableFolderTree(
      snapshot?.folders ?? const <Folder>[],
    );
    final tags = snapshot?.tags ?? const <Tag>[];
    return Scaffold(
      appBar: AppBar(
        leading: IconButton(
          tooltip: 'Back',
          onPressed: () => context.go('/'),
          icon: const Icon(Icons.arrow_back),
        ),
        title: const Text('Ingest'),
      ),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          IngestModeSelector(
            mode: _mode,
            onChanged: (value) => setState(() => _mode = value),
          ),
          const SizedBox(height: 20),
          if (_mode == IngestMode.upload)
            IngestUploadPicker(fileName: _filename, onPick: _pickFile)
          else
            IngestUrlField(controller: _urlController),
          if (_mode != IngestMode.scrape) ...[
            const SizedBox(height: 12),
            IngestTitleField(controller: _titleController),
          ],
          const SizedBox(height: 16),
          IngestFolderSelect(
            folders: folders,
            selectedFolderId: _folderId,
            onChanged: (value) => setState(() => _folderId = value),
          ),
          const SizedBox(height: 16),
          IngestTagSelect(
            tags: tags,
            selectedTagIds: _selectedTagIds,
            onChanged: (ids) => setState(
              () => _selectedTagIds
                ..clear()
                ..addAll(ids),
            ),
          ),
          const SizedBox(height: 24),
          FilledButton.icon(
            onPressed: _saving ? null : _submit,
            icon: _saving
                ? const SizedBox.square(
                    dimension: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.add),
            label: Text(_saving ? 'Creating' : 'Create'),
          ),
        ],
      ),
    );
  }

  Future<void> _pickFile() async {
    try {
      final file = await pickIngestUploadFile();
      if (file == null || !mounted) {
        return;
      }
      setState(() {
        _filename = file.filename;
        _fileBytes = file.bytes;
      });
    } on IngestUploadPickException catch (error) {
      _clearFile();
      _showSnackBar(error.message);
    }
  }

  Future<void> _submit() async {
    setState(() => _saving = true);
    try {
      final repository = ref.read(libraryRepositoryProvider);
      final detail = switch (_mode) {
        IngestMode.bookmark => await repository.bookmarkUrl(
          url: _requiredUrl(),
          title: _titleController.text,
          folderId: _folderId,
          tagIds: _selectedTagIds.toList(),
        ),
        IngestMode.scrape => await repository.scrapeUrl(
          url: _requiredUrl(),
          folderId: _folderId,
          tagIds: _selectedTagIds.toList(),
        ),
        IngestMode.upload => await repository.uploadFile(
          UploadFileInput(
            filename: _filename ?? 'upload.bin',
            bytes: _requiredFileBytes(),
            title: _titleController.text,
            folderId: _folderId,
            tagIds: _selectedTagIds.toList(),
          ),
        ),
      };
      ref.invalidate(librarySnapshotProvider);
      if (mounted) {
        context.go('/documents/${detail.document.id}');
      }
    } catch (error) {
      if (mounted) {
        ScaffoldMessenger.of(
          context,
        ).showSnackBar(SnackBar(content: Text(error.toString())));
      }
    } finally {
      if (mounted) {
        setState(() => _saving = false);
      }
    }
  }

  String _requiredUrl() {
    final url = _urlController.text.trim();
    if (url.isEmpty) {
      throw StateError('URL is required.');
    }
    return url;
  }

  Uint8List _requiredFileBytes() {
    final bytes = _fileBytes;
    if (bytes == null) {
      throw StateError('File is required.');
    }
    return bytes;
  }

  void _clearFile() {
    if (!mounted) {
      return;
    }
    setState(() {
      _filename = null;
      _fileBytes = null;
    });
  }

  void _showSnackBar(String message) {
    if (!mounted) {
      return;
    }
    ScaffoldMessenger.of(
      context,
    ).showSnackBar(SnackBar(content: Text(message)));
  }
}
