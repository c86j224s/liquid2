import 'package:liquid2_api/liquid2_api.dart';

import 'tag_repository.dart';

class ApiTagRepository implements TagRepository {
  const ApiTagRepository(this.api);

  final Liquid2Api api;

  @override
  Future<Tag> createTag(String name) async {
    final response = await api.getTagsApi().createTag(
      tagBodyInputBody: TagBodyInputBody((b) => b.name = name.trim()),
    );
    final tag = response.data;
    if (tag == null) {
      throw StateError('Tag response was empty.');
    }
    return tag;
  }
}
