//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'form_file.g.dart';

/// FormFile
///
/// Properties:
/// * [contentType]
/// * [filename]
/// * [isSet]
/// * [size]
@BuiltValue()
abstract class FormFile implements Built<FormFile, FormFileBuilder> {
  @BuiltValueField(wireName: r'ContentType')
  String get contentType;

  @BuiltValueField(wireName: r'Filename')
  String get filename;

  @BuiltValueField(wireName: r'IsSet')
  bool get isSet;

  @BuiltValueField(wireName: r'Size')
  int get size;

  FormFile._();

  factory FormFile([void updates(FormFileBuilder b)]) = _$FormFile;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FormFileBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FormFile> get serializer => _$FormFileSerializer();
}

class _$FormFileSerializer implements PrimitiveSerializer<FormFile> {
  @override
  final Iterable<Type> types = const [FormFile, _$FormFile];

  @override
  final String wireName = r'FormFile';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FormFile object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'ContentType';
    yield serializers.serialize(
      object.contentType,
      specifiedType: const FullType(String),
    );
    yield r'Filename';
    yield serializers.serialize(
      object.filename,
      specifiedType: const FullType(String),
    );
    yield r'IsSet';
    yield serializers.serialize(
      object.isSet,
      specifiedType: const FullType(bool),
    );
    yield r'Size';
    yield serializers.serialize(
      object.size,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FormFile object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FormFileBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'ContentType':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.contentType = valueDes;
          break;
        case r'Filename':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.filename = valueDes;
          break;
        case r'IsSet':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.isSet = valueDes;
          break;
        case r'Size':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.size = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FormFile deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FormFileBuilder();
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
