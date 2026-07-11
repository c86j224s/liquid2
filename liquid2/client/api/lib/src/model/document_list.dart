//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:liquid2_api/src/model/document_summary.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'document_list.g.dart';

/// DocumentList
///
/// Properties:
/// * [items]
/// * [nextCursor]
/// * [totalCount]
@BuiltValue()
abstract class DocumentList implements Built<DocumentList, DocumentListBuilder> {
  @BuiltValueField(wireName: r'items')
  BuiltList<DocumentSummary>? get items;

  @BuiltValueField(wireName: r'nextCursor')
  String? get nextCursor;

  @BuiltValueField(wireName: r'totalCount')
  int get totalCount;

  DocumentList._();

  factory DocumentList([void updates(DocumentListBuilder b)]) = _$DocumentList;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(DocumentListBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<DocumentList> get serializer => _$DocumentListSerializer();
}

class _$DocumentListSerializer implements PrimitiveSerializer<DocumentList> {
  @override
  final Iterable<Type> types = const [DocumentList, _$DocumentList];

  @override
  final String wireName = r'DocumentList';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    DocumentList object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'items';
    yield object.items == null ? null : serializers.serialize(
      object.items,
      specifiedType: const FullType.nullable(BuiltList, [FullType(DocumentSummary)]),
    );
    yield r'nextCursor';
    yield object.nextCursor == null ? null : serializers.serialize(
      object.nextCursor,
      specifiedType: const FullType.nullable(String),
    );
    yield r'totalCount';
    yield serializers.serialize(
      object.totalCount,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    DocumentList object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required DocumentListBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'items':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(DocumentSummary)]),
          ) as BuiltList<DocumentSummary>?;
          if (valueDes == null) continue;
          result.items.replace(valueDes);
          break;
        case r'nextCursor':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.nextCursor = valueDes;
          break;
        case r'totalCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.totalCount = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  DocumentList deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = DocumentListBuilder();
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
