//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:liquid2_api/src/model/tag.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'tag_list_output_body.g.dart';

/// TagListOutputBody
///
/// Properties:
/// * [items]
@BuiltValue()
abstract class TagListOutputBody implements Built<TagListOutputBody, TagListOutputBodyBuilder> {
  @BuiltValueField(wireName: r'items')
  BuiltList<Tag>? get items;

  TagListOutputBody._();

  factory TagListOutputBody([void updates(TagListOutputBodyBuilder b)]) = _$TagListOutputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TagListOutputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TagListOutputBody> get serializer => _$TagListOutputBodySerializer();
}

class _$TagListOutputBodySerializer implements PrimitiveSerializer<TagListOutputBody> {
  @override
  final Iterable<Type> types = const [TagListOutputBody, _$TagListOutputBody];

  @override
  final String wireName = r'TagListOutputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TagListOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'items';
    yield object.items == null ? null : serializers.serialize(
      object.items,
      specifiedType: const FullType.nullable(BuiltList, [FullType(Tag)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TagListOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TagListOutputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'items':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(Tag)]),
          ) as BuiltList<Tag>?;
          if (valueDes == null) continue;
          result.items.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TagListOutputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TagListOutputBodyBuilder();
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
