import 'package:liquid2_api/liquid2_api.dart';
import 'package:liquid2_client/data/library_repository.dart';

mixin FakeLibraryNotesRepository implements LibraryRepository {
  final notes = <DocumentNote>[];

  @override
  Future<List<DocumentNote>> listNotes(String documentId) async => notes;

  @override
  Future<DocumentNote> createNote({
    required String documentId,
    required String body,
    required String format,
  }) async {
    final note = DocumentNote(
      (b) => b
        ..id = 'note_${notes.length + 1}'
        ..documentId = documentId
        ..body = body
        ..format = format
        ..createdAt = _now
        ..updatedAt = _now,
    );
    notes.add(note);
    return note;
  }
}

const _now = 1760000000000;
