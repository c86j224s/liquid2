import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:liquid2_api/liquid2_api.dart';

import '../../app/providers.dart';
import '../../data/folder_tree.dart';
import '../../data/library_repository.dart';
import 'ingest_app_bar.dart';
import 'ingest_filters.dart';
import 'ingest_mode.dart';
import 'ingest_mode_selector.dart';
import 'ingest_source_fields.dart';
import 'ingest_submit_button.dart';
import 'ingest_upload_file.dart';

class IngestPage extends ConsumerStatefulWidget {
  const IngestPage({super.key});

  @override
  ConsumerState<IngestPage> createState() => _IngestPageState();
}

class _IngestPageState extends ConsumerState<IngestPage> {
  final _urlController = TextEditingController();
  final _titleController = TextEditingController();
  final _selectedTagIds = <String>{};
  IngestSource _source = IngestSource.url;
  UrlSaveMode _urlSaveMode = UrlSaveMode.linkOnly;
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
      appBar: const IngestAppBar(),
      body: ListView(
        padding: const EdgeInsets.all(20),
        children: [
          IngestModeSelector(
            source: _source,
            onChanged: (value) => setState(() => _source = value),
          ),
          const SizedBox(height: 20),
          IngestSourceFields(
            source: _source,
            urlSaveMode: _urlSaveMode,
            urlController: _urlController,
            titleController: _titleController,
            fileName: _filename,
            onUrlSaveModeChanged: (value) =>
                setState(() => _urlSaveMode = value),
            onPickFile: _pickFile,
          ),
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
          IngestSubmitButton(
            saving: _saving,
            label: _submitLabel,
            onPressed: _submit,
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
      if (!mounted) {
        return;
      }
      setState(() {
        _filename = null;
        _fileBytes = null;
      });
      _showSnackBar(error.message);
    }
  }

  Future<void> _submit() async {
    setState(() => _saving = true);
    try {
      final repository = ref.read(libraryRepositoryProvider);
      final detail = await _createDocument(repository);
      ref.invalidate(librarySnapshotProvider);
      if (mounted) {
        context.go('/documents/${detail.document.id}');
      }
    } catch (error) {
      _showSnackBar(error.toString());
    } finally {
      if (mounted) {
        setState(() => _saving = false);
      }
    }
  }

  Future<DocumentDetail> _createDocument(LibraryRepository repository) {
    if (_source == IngestSource.file) {
      return repository.uploadFile(
        UploadFileInput(
          filename: _filename ?? 'upload.bin',
          bytes: _requiredFileBytes(),
          title: _titleController.text,
          folderId: _folderId,
          tagIds: _selectedTagIds.toList(),
        ),
      );
    }
    if (_urlSaveMode == UrlSaveMode.pageCopy) {
      return repository.scrapeUrl(
        url: _requiredUrl(),
        folderId: _folderId,
        tagIds: _selectedTagIds.toList(),
      );
    }
    return repository.bookmarkUrl(
      url: _requiredUrl(),
      title: _titleController.text,
      folderId: _folderId,
      tagIds: _selectedTagIds.toList(),
    );
  }

  String get _submitLabel {
    if (_source == IngestSource.file) {
      return 'Upload file';
    }
    return _urlSaveMode == UrlSaveMode.pageCopy ? 'Save page' : 'Save link';
  }

  String _requiredUrl() {
    final url = _urlController.text.trim();
    if (url.isNotEmpty) {
      return url;
    }
    throw StateError('URL is required.');
  }

  Uint8List _requiredFileBytes() {
    final bytes = _fileBytes;
    if (bytes != null) {
      return bytes;
    }
    throw StateError('File is required.');
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
