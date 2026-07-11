//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'update_note_input_body.g.dart';

/// UpdateNoteInputBody
///
/// Properties:
/// * [body]
/// * [format]
@BuiltValue()
abstract class UpdateNoteInputBody implements Built<UpdateNoteInputBody, UpdateNoteInputBodyBuilder> {
  @BuiltValueField(wireName: r'body')
  String get body;

  @BuiltValueField(wireName: r'format')
  UpdateNoteInputBodyFormatEnum get format;
  // enum formatEnum {  text,  markdown,  };

  UpdateNoteInputBody._();

  factory UpdateNoteInputBody([void updates(UpdateNoteInputBodyBuilder b)]) = _$UpdateNoteInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UpdateNoteInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UpdateNoteInputBody> get serializer => _$UpdateNoteInputBodySerializer();
}

class _$UpdateNoteInputBodySerializer implements PrimitiveSerializer<UpdateNoteInputBody> {
  @override
  final Iterable<Type> types = const [UpdateNoteInputBody, _$UpdateNoteInputBody];

  @override
  final String wireName = r'UpdateNoteInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UpdateNoteInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'body';
    yield serializers.serialize(
      object.body,
      specifiedType: const FullType(String),
    );
    yield r'format';
    yield serializers.serialize(
      object.format,
      specifiedType: const FullType(UpdateNoteInputBodyFormatEnum),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    UpdateNoteInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UpdateNoteInputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'body':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.body = valueDes;
          break;
        case r'format':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(UpdateNoteInputBodyFormatEnum),
          ) as UpdateNoteInputBodyFormatEnum;
          result.format = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  UpdateNoteInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UpdateNoteInputBodyBuilder();
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

class UpdateNoteInputBodyFormatEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'text')
  static const UpdateNoteInputBodyFormatEnum text = _$updateNoteInputBodyFormatEnum_text;
  @BuiltValueEnumConst(wireName: r'markdown')
  static const UpdateNoteInputBodyFormatEnum markdown = _$updateNoteInputBodyFormatEnum_markdown;

  static Serializer<UpdateNoteInputBodyFormatEnum> get serializer => _$updateNoteInputBodyFormatEnumSerializer;

  const UpdateNoteInputBodyFormatEnum._(String name): super(name);

  static BuiltSet<UpdateNoteInputBodyFormatEnum> get values => _$updateNoteInputBodyFormatEnumValues;
  static UpdateNoteInputBodyFormatEnum valueOf(String name) => _$updateNoteInputBodyFormatEnumValueOf(name);
}
