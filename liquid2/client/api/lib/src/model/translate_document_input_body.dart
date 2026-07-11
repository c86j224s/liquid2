//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'translate_document_input_body.g.dart';

/// TranslateDocumentInputBody
///
/// Properties:
/// * [sourceContentId]
/// * [targetLanguage]
@BuiltValue()
abstract class TranslateDocumentInputBody implements Built<TranslateDocumentInputBody, TranslateDocumentInputBodyBuilder> {
  @BuiltValueField(wireName: r'sourceContentId')
  String get sourceContentId;

  @BuiltValueField(wireName: r'targetLanguage')
  String get targetLanguage;

  TranslateDocumentInputBody._();

  factory TranslateDocumentInputBody([void updates(TranslateDocumentInputBodyBuilder b)]) = _$TranslateDocumentInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(TranslateDocumentInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<TranslateDocumentInputBody> get serializer => _$TranslateDocumentInputBodySerializer();
}

class _$TranslateDocumentInputBodySerializer implements PrimitiveSerializer<TranslateDocumentInputBody> {
  @override
  final Iterable<Type> types = const [TranslateDocumentInputBody, _$TranslateDocumentInputBody];

  @override
  final String wireName = r'TranslateDocumentInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    TranslateDocumentInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'sourceContentId';
    yield serializers.serialize(
      object.sourceContentId,
      specifiedType: const FullType(String),
    );
    yield r'targetLanguage';
    yield serializers.serialize(
      object.targetLanguage,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    TranslateDocumentInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required TranslateDocumentInputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'sourceContentId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.sourceContentId = valueDes;
          break;
        case r'targetLanguage':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.targetLanguage = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  TranslateDocumentInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = TranslateDocumentInputBodyBuilder();
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
