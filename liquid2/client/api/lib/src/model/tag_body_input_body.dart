//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'tag_body_input_body.g.dart';

/// TagBodyInputBody
///
/// Properties:
/// * [name]
@BuiltValue()
abstract class TagBodyInputBody implements Built<TagBodyInputBody, TagBodyInputBodyBuilder> {
  @BuiltValueField(wireName: r'name')
  String get name;

  TagBodyInputBody._();

  factory TagBodyInputBody([void updates(TagBodyInputBodyBuilder b)]) = _$TagBodyInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TagBodyInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TagBodyInputBody> get serializer => _$TagBodyInputBodySerializer();
}

class _$TagBodyInputBodySerializer implements PrimitiveSerializer<TagBodyInputBody> {
  @override
  final Iterable<Type> types = const [TagBodyInputBody, _$TagBodyInputBody];

  @override
  final String wireName = r'TagBodyInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TagBodyInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TagBodyInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TagBodyInputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TagBodyInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TagBodyInputBodyBuilder();
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
