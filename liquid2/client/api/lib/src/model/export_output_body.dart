//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:liquid2_api/src/model/export_artifact.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'export_output_body.g.dart';

/// ExportOutputBody
///
/// Properties:
/// * [export_]
@BuiltValue()
abstract class ExportOutputBody implements Built<ExportOutputBody, ExportOutputBodyBuilder> {
  @BuiltValueField(wireName: r'export')
  ExportArtifact get export_;

  ExportOutputBody._();

  factory ExportOutputBody([void updates(ExportOutputBodyBuilder b)]) = _$ExportOutputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ExportOutputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ExportOutputBody> get serializer => _$ExportOutputBodySerializer();
}

class _$ExportOutputBodySerializer implements PrimitiveSerializer<ExportOutputBody> {
  @override
  final Iterable<Type> types = const [ExportOutputBody, _$ExportOutputBody];

  @override
  final String wireName = r'ExportOutputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ExportOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'export';
    yield serializers.serialize(
      object.export_,
      specifiedType: const FullType(ExportArtifact),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ExportOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ExportOutputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'export':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(ExportArtifact),
          ) as ExportArtifact;
          result.export_.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ExportOutputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ExportOutputBodyBuilder();
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
