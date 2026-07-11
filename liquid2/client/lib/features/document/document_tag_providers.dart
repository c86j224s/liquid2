import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_riverpod/legacy.dart';

import '../../app/providers.dart';
import '../../data/api_tag_repository.dart';
import '../../data/tag_repository.dart';

final tagRepositoryProvider = Provider<TagRepository>((ref) {
  return ApiTagRepository(ref.watch(liquid2ApiProvider));
});

final documentTagCreatingProvider = StateProvider.family<bool, String>(
  (ref, id) => false,
);
