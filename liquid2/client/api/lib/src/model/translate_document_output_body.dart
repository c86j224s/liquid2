//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:liquid2_api/src/model/job.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'translate_document_output_body.g.dart';

/// TranslateDocumentOutputBody
///
/// Properties:
/// * [job]
@BuiltValue()
abstract class TranslateDocumentOutputBody implements Built<TranslateDocumentOutputBody, TranslateDocumentOutputBodyBuilder> {
  @BuiltValueField(wireName: r'job')
  Job get job;

  TranslateDocumentOutputBody._();

  factory TranslateDocumentOutputBody([void updates(TranslateDocumentOutputBodyBuilder b)]) = _$TranslateDocumentOutputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TranslateDocumentOutputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TranslateDocumentOutputBody> get serializer => _$TranslateDocumentOutputBodySerializer();
}

class _$TranslateDocumentOutputBodySerializer implements PrimitiveSerializer<TranslateDocumentOutputBody> {
  @override
  final Iterable<Type> types = const [TranslateDocumentOutputBody, _$TranslateDocumentOutputBody];

  @override
  final String wireName = r'TranslateDocumentOutputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TranslateDocumentOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'job';
    yield serializers.serialize(
      object.job,
      specifiedType: const FullType(Job),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TranslateDocumentOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TranslateDocumentOutputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'job':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(Job),
          ) as Job;
          result.job.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TranslateDocumentOutputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TranslateDocumentOutputBodyBuilder();
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
