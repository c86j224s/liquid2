import 'package:liquid2_api/liquid2_api.dart';

abstract class TagRepository {
  Future<Tag> createTag(String name);
}
