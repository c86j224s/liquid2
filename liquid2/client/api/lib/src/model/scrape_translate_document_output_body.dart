//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:liquid2_api/src/model/job.dart';
import 'package:liquid2_api/src/model/document_detail.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'scrape_translate_document_output_body.g.dart';

/// ScrapeTranslateDocumentOutputBody
///
/// Properties:
/// * [document]
/// * [job]
@BuiltValue()
abstract class ScrapeTranslateDocumentOutputBody implements Built<ScrapeTranslateDocumentOutputBody, ScrapeTranslateDocumentOutputBodyBuilder> {
  @BuiltValueField(wireName: r'document')
  DocumentDetail get document;

  @BuiltValueField(wireName: r'job')
  Job get job;

  ScrapeTranslateDocumentOutputBody._();

  factory ScrapeTranslateDocumentOutputBody([void updates(ScrapeTranslateDocumentOutputBodyBuilder b)]) = _$ScrapeTranslateDocumentOutputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ScrapeTranslateDocumentOutputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ScrapeTranslateDocumentOutputBody> get serializer => _$ScrapeTranslateDocumentOutputBodySerializer();
}

class _$ScrapeTranslateDocumentOutputBodySerializer implements PrimitiveSerializer<ScrapeTranslateDocumentOutputBody> {
  @override
  final Iterable<Type> types = const [ScrapeTranslateDocumentOutputBody, _$ScrapeTranslateDocumentOutputBody];

  @override
  final String wireName = r'ScrapeTranslateDocumentOutputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ScrapeTranslateDocumentOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'document';
    yield serializers.serialize(
      object.document,
      specifiedType: const FullType(DocumentDetail),
    );
    yield r'job';
    yield serializers.serialize(
      object.job,
      specifiedType: const FullType(Job),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ScrapeTranslateDocumentOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ScrapeTranslateDocumentOutputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'document':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DocumentDetail),
          ) as DocumentDetail;
          result.document.replace(valueDes);
          break;
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
  ScrapeTranslateDocumentOutputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ScrapeTranslateDocumentOutputBodyBuilder();
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
