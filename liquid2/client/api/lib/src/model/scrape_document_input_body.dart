//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'scrape_document_input_body.g.dart';

/// ScrapeDocumentInputBody
///
/// Properties:
/// * [folderId]
/// * [tagIds]
/// * [url]
@BuiltValue()
abstract class ScrapeDocumentInputBody implements Built<ScrapeDocumentInputBody, ScrapeDocumentInputBodyBuilder> {
  @BuiltValueField(wireName: r'folderId')
  String? get folderId;

  @BuiltValueField(wireName: r'tagIds')
  BuiltList<String>? get tagIds;

  @BuiltValueField(wireName: r'url')
  String get url;

  ScrapeDocumentInputBody._();

  factory ScrapeDocumentInputBody([void updates(ScrapeDocumentInputBodyBuilder b)]) = _$ScrapeDocumentInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ScrapeDocumentInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ScrapeDocumentInputBody> get serializer => _$ScrapeDocumentInputBodySerializer();
}

class _$ScrapeDocumentInputBodySerializer implements PrimitiveSerializer<ScrapeDocumentInputBody> {
  @override
  final Iterable<Type> types = const [ScrapeDocumentInputBody, _$ScrapeDocumentInputBody];

  @override
  final String wireName = r'ScrapeDocumentInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ScrapeDocumentInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.folderId != null) {
      yield r'folderId';
      yield serializers.serialize(
        object.folderId,
        specifiedType: const FullType(String),
      );
    }
    if (object.tagIds != null) {
      yield r'tagIds';
      yield serializers.serialize(
        object.tagIds,
        specifiedType: const FullType.nullable(BuiltList, [FullType(String)]),
      );
    }
    yield r'url';
    yield serializers.serialize(
      object.url,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ScrapeDocumentInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ScrapeDocumentInputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'folderId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.folderId = valueDes;
          break;
        case r'tagIds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(String)]),
          ) as BuiltList<String>?;
          if (valueDes == null) continue;
          result.tagIds.replace(valueDes);
          break;
        case r'url':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.url = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ScrapeDocumentInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ScrapeDocumentInputBodyBuilder();
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
