//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'create_export_input_body.g.dart';

/// CreateExportInputBody
///
/// Properties:
/// * [documentIds]
/// * [includeBlobs]
@BuiltValue()
abstract class CreateExportInputBody implements Built<CreateExportInputBody, CreateExportInputBodyBuilder> {
  @BuiltValueField(wireName: r'documentIds')
  BuiltList<String>? get documentIds;

  @BuiltValueField(wireName: r'includeBlobs')
  bool? get includeBlobs;

  CreateExportInputBody._();

  factory CreateExportInputBody([void updates(CreateExportInputBodyBuilder b)]) = _$CreateExportInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(CreateExportInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<CreateExportInputBody> get serializer => _$CreateExportInputBodySerializer();
}

class _$CreateExportInputBodySerializer implements PrimitiveSerializer<CreateExportInputBody> {
  @override
  final Iterable<Type> types = const [CreateExportInputBody, _$CreateExportInputBody];

  @override
  final String wireName = r'CreateExportInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    CreateExportInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.documentIds != null) {
      yield r'documentIds';
      yield serializers.serialize(
        object.documentIds,
        specifiedType: const FullType.nullable(BuiltList, [FullType(String)]),
      );
    }
    if (object.includeBlobs != null) {
      yield r'includeBlobs';
      yield serializers.serialize(
        object.includeBlobs,
        specifiedType: const FullType(bool),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    CreateExportInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required CreateExportInputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'documentIds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(String)]),
          ) as BuiltList<String>?;
          if (valueDes == null) continue;
          result.documentIds.replace(valueDes);
          break;
        case r'includeBlobs':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.includeBlobs = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  CreateExportInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = CreateExportInputBodyBuilder();
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
