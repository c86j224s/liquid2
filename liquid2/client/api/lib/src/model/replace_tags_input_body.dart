//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'replace_tags_input_body.g.dart';

/// ReplaceTagsInputBody
///
/// Properties:
/// * [tagIds] - Replacement tag IDs
@BuiltValue()
abstract class ReplaceTagsInputBody implements Built<ReplaceTagsInputBody, ReplaceTagsInputBodyBuilder> {
  /// Replacement tag IDs
  @BuiltValueField(wireName: r'tagIds')
  BuiltList<String>? get tagIds;

  ReplaceTagsInputBody._();

  factory ReplaceTagsInputBody([void updates(ReplaceTagsInputBodyBuilder b)]) = _$ReplaceTagsInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ReplaceTagsInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ReplaceTagsInputBody> get serializer => _$ReplaceTagsInputBodySerializer();
}

class _$ReplaceTagsInputBodySerializer implements PrimitiveSerializer<ReplaceTagsInputBody> {
  @override
  final Iterable<Type> types = const [ReplaceTagsInputBody, _$ReplaceTagsInputBody];

  @override
  final String wireName = r'ReplaceTagsInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ReplaceTagsInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'tagIds';
    yield object.tagIds == null ? null : serializers.serialize(
      object.tagIds,
      specifiedType: const FullType.nullable(BuiltList, [FullType(String)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ReplaceTagsInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ReplaceTagsInputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'tagIds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(String)]),
          ) as BuiltList<String>?;
          if (valueDes == null) continue;
          result.tagIds.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ReplaceTagsInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ReplaceTagsInputBodyBuilder();
    final serializedList = (serialized as Iterable<Object?>).toList();
    final unhandled = <Object?>[];
    _deserializeProperties(
      serializers,
      serialized,
      specifiedType: specifiedType,
      serializedList: serializedList,
      unhandled: unhandled,
      result: result,
    );
    return result.build();
  }
}
