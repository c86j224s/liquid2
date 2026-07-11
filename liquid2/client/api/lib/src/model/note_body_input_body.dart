//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'note_body_input_body.g.dart';

/// NoteBodyInputBody
///
/// Properties:
/// * [body]
/// * [format]
@BuiltValue()
abstract class NoteBodyInputBody implements Built<NoteBodyInputBody, NoteBodyInputBodyBuilder> {
  @BuiltValueField(wireName: r'body')
  String get body;

  @BuiltValueField(wireName: r'format')
  NoteBodyInputBodyFormatEnum get format;
  // enum formatEnum {  text,  markdown,  };

  NoteBodyInputBody._();

  factory NoteBodyInputBody([void updates(NoteBodyInputBodyBuilder b)]) = _$NoteBodyInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(NoteBodyInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<NoteBodyInputBody> get serializer => _$NoteBodyInputBodySerializer();
}

class _$NoteBodyInputBodySerializer implements PrimitiveSerializer<NoteBodyInputBody> {
  @override
  final Iterable<Type> types = const [NoteBodyInputBody, _$NoteBodyInputBody];

  @override
  final String wireName = r'NoteBodyInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    NoteBodyInputBody object, {
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
      specifiedType: const FullType(NoteBodyInputBodyFormatEnum),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    NoteBodyInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required NoteBodyInputBodyBuilder result,
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
            specifiedType: const FullType(NoteBodyInputBodyFormatEnum),
          ) as NoteBodyInputBodyFormatEnum;
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
  NoteBodyInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = NoteBodyInputBodyBuilder();
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

class NoteBodyInputBodyFormatEnum extends EnumClass {

  @BuiltValueEnumConst(wireName: r'text')
  static const NoteBodyInputBodyFormatEnum text = _$noteBodyInputBodyFormatEnum_text;
  @BuiltValueEnumConst(wireName: r'markdown')
  static const NoteBodyInputBodyFormatEnum markdown = _$noteBodyInputBodyFormatEnum_markdown;

  static Serializer<NoteBodyInputBodyFormatEnum> get serializer => _$noteBodyInputBodyFormatEnumSerializer;

  const NoteBodyInputBodyFormatEnum._(String name): super(name);

  static BuiltSet<NoteBodyInputBodyFormatEnum> get values => _$noteBodyInputBodyFormatEnumValues;
  static NoteBodyInputBodyFormatEnum valueOf(String name) => _$noteBodyInputBodyFormatEnumValueOf(name);
}
