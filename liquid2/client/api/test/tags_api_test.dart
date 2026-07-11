import 'package:test/test.dart';
import 'package:liquid2_api/liquid2_api.dart';


/// tests for TagsApi
void main() {
  final instance = Liquid2Api().getTagsApi();

  group(TagsApi, () {
    // Create tag
    //
    //Future<Tag> createTag(TagBodyInputBody tagBodyInputBody) async
    test('test createTag', () async {
      // TODO
    });

    // List tags
    //
    //Future<TagListOutputBody> listTags() async
    test('test listTags', () async {
      // TODO
    });

    // Replace document tags
    //
    //Future<DocumentDetail> replaceDocumentTags(String id, ReplaceTagsInputBody replaceTagsInputBody) async
    test('test replaceDocumentTags', () async {
      // TODO
    });

  });
}
