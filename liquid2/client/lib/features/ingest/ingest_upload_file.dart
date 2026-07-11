import 'dart:typed_data';

import 'package:file_picker/file_picker.dart';

const maxIngestUploadBytes = 1024 * 1024;

class PickedIngestUploadFile {
  const PickedIngestUploadFile({required this.filename, required this.bytes});

  final String filename;
  final Uint8List bytes;
}

class IngestUploadPickException implements Exception {
  const IngestUploadPickException(this.message);

  final String message;

  @override
  String toString() => message;
}

Future<PickedIngestUploadFile?> pickIngestUploadFile() async {
  final result = await FilePicker.pickFiles(
    allowMultiple: false,
    withData: false,
    withReadStream: true,
    type: FileType.custom,
    allowedExtensions: const ['txt', 'md', 'markdown', 'html', 'htm', 'pdf'],
  );
  final file = result?.files.single;
  if (file == null) {
    return null;
  }
  if (file.size > maxIngestUploadBytes) {
    throw const IngestUploadPickException('File must be 1 MB or smaller.');
  }
  final stream = file.readStream;
  if (stream == null) {
    throw const IngestUploadPickException('Unable to read the selected file.');
  }
  return PickedIngestUploadFile(
    filename: file.name,
    bytes: await readIngestUploadBytes(stream),
  );
}

Future<Uint8List> readIngestUploadBytes(Stream<List<int>> stream) async {
  final builder = BytesBuilder(copy: false);
  var total = 0;
  await for (final chunk in stream) {
    total += chunk.length;
    if (total > maxIngestUploadBytes) {
      throw const IngestUploadPickException('File must be 1 MB or smaller.');
    }
    builder.add(chunk);
  }
  return builder.takeBytes();
}
