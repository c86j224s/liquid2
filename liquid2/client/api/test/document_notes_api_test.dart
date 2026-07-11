import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for DocumentNotesApi
void main() {
  final instance = Liquid2Api().getDocumentNotesApi();

  group(DocumentNotesApi, () {
    // Create document note
    //
    //Future<DocumentNote> createDocumentNote(String id, NoteBodyInputBody noteBodyInputBody) async
    test('test createDocumentNote', () async {
      // TODO
    });

    // Soft-delete document note
    //
    //Future<DeletedOutputBody> deleteDocumentNote(String id, String noteId) async
    test('test deleteDocumentNote', () async {
      // TODO
    });

    // List document notes
    //
    //Future<NoteList> listDocumentNotes(String id) async
    test('test listDocumentNotes', () async {
      // TODO
    });

    // Update document note
    //
    //Future<DocumentNote> updateDocumentNote(String id, String noteId, UpdateNoteInputBody updateNoteInputBody) async
    test('test updateDocumentNote', () async {
      // TODO
    });

  });
}
