//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:liquid2_api/src/model/folder_breadcrumb.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'document_summary.g.dart';

/// DocumentSummary
///
/// Properties:
/// * [canonicalUrl]
/// * [createdAt]
/// * [deletedAt]
/// * [folderId]
/// * [folderPath]
/// * [id]
/// * [kind]
/// * [language]
/// * [publishedAt]
/// * [rating]
/// * [readAt]
/// * [sourceUrl]
/// * [status]
/// * [tags]
/// * [title]
/// * [updatedAt]
@BuiltValue()
abstract class DocumentSummary implements Built<DocumentSummary, DocumentSummaryBuilder> {
  @BuiltValueField(wireName: r'canonicalUrl')
  String? get canonicalUrl;

  @BuiltValueField(wireName: r'createdAt')
  int get createdAt;

  @BuiltValueField(wireName: r'deletedAt')
  int? get deletedAt;

  @BuiltValueField(wireName: r'folderId')
  String? get folderId;

  @BuiltValueField(wireName: r'folderPath')
  BuiltList<FolderBreadcrumb>? get folderPath;

  @BuiltValueField(wireName: r'id')
  String get id;

  @BuiltValueField(wireName: r'kind')
  String get kind;

  @BuiltValueField(wireName: r'language')
  String? get language;

  @BuiltValueField(wireName: r'publishedAt')
  int? get publishedAt;

  @BuiltValueField(wireName: r'rating')
  int? get rating;

  @BuiltValueField(wireName: r'readAt')
  int? get readAt;

  @BuiltValueField(wireName: r'sourceUrl')
  String? get sourceUrl;

  @BuiltValueField(wireName: r'status')
  String get status;

  @BuiltValueField(wireName: r'tags')
  BuiltList<String>? get tags;

  @BuiltValueField(wireName: r'title')
  String get title;

  @BuiltValueField(wireName: r'updatedAt')
  int get updatedAt;

  DocumentSummary._();

  factory DocumentSummary([void updates(DocumentSummaryBuilder b)]) = _$DocumentSummary;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(DocumentSummaryBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<DocumentSummary> get serializer => _$DocumentSummarySerializer();
}

class _$DocumentSummarySerializer implements PrimitiveSerializer<DocumentSummary> {
  @override
  final Iterable<Type> types = const [DocumentSummary, _$DocumentSummary];

  @override
  final String wireName = r'DocumentSummary';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    DocumentSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'canonicalUrl';
    yield object.canonicalUrl == null ? null : serializers.serialize(
      object.canonicalUrl,
      specifiedType: const FullType.nullable(String),
    );
    yield r'createdAt';
    yield serializers.serialize(
      object.createdAt,
      specifiedType: const FullType(int),
    );
    yield r'deletedAt';
    yield object.deletedAt == null ? null : serializers.serialize(
      object.deletedAt,
      specifiedType: const FullType.nullable(int),
    );
    yield r'folderId';
    yield object.folderId == null ? null : serializers.serialize(
      object.folderId,
      specifiedType: const FullType.nullable(String),
    );
    yield r'folderPath';
    yield object.folderPath == null ? null : serializers.serialize(
      object.folderPath,
      specifiedType: const FullType.nullable(BuiltList, [FullType(FolderBreadcrumb)]),
    );
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'kind';
    yield serializers.serialize(
      object.kind,
      specifiedType: const FullType(String),
    );
    yield r'language';
    yield object.language == null ? null : serializers.serialize(
      object.language,
      specifiedType: const FullType.nullable(String),
    );
    yield r'publishedAt';
    yield object.publishedAt == null ? null : serializers.serialize(
      object.publishedAt,
      specifiedType: const FullType.nullable(int),
    );
    yield r'rating';
    yield object.rating == null ? null : serializers.serialize(
      object.rating,
      specifiedType: const FullType.nullable(int),
    );
    yield r'readAt';
    yield object.readAt == null ? null : serializers.serialize(
      object.readAt,
      specifiedType: const FullType.nullable(int),
    );
    yield r'sourceUrl';
    yield object.sourceUrl == null ? null : serializers.serialize(
      object.sourceUrl,
      specifiedType: const FullType.nullable(String),
    );
    yield r'status';
    yield serializers.serialize(
      object.status,
      specifiedType: const FullType(String),
    );
    yield r'tags';
    yield object.tags == null ? null : serializers.serialize(
      object.tags,
      specifiedType: const FullType.nullable(BuiltList, [FullType(String)]),
    );
    yield r'title';
    yield serializers.serialize(
      object.title,
      specifiedType: const FullType(String),
    );
    yield r'updatedAt';
    yield serializers.serialize(
      object.updatedAt,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    DocumentSummary object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required DocumentSummaryBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'canonicalUrl':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.canonicalUrl = valueDes;
          break;
        case r'createdAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.createdAt = valueDes;
          break;
        case r'deletedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.deletedAt = valueDes;
          break;
        case r'folderId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.folderId = valueDes;
          break;
        case r'folderPath':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(FolderBreadcrumb)]),
          ) as BuiltList<FolderBreadcrumb>?;
          if (valueDes == null) continue;
          result.folderPath.replace(valueDes);
          break;
        case r'id':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.id = valueDes;
          break;
        case r'kind':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.kind = valueDes;
          break;
        case r'language':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.language = valueDes;
          break;
        case r'publishedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.publishedAt = valueDes;
          break;
        case r'rating':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.rating = valueDes;
          break;
        case r'readAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.readAt = valueDes;
          break;
        case r'sourceUrl':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.sourceUrl = valueDes;
          break;
        case r'status':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.status = valueDes;
          break;
        case r'tags':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(String)]),
          ) as BuiltList<String>?;
          if (valueDes == null) continue;
          result.tags.replace(valueDes);
          break;
        case r'title':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.title = valueDes;
          break;
        case r'updatedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.updatedAt = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  DocumentSummary deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = DocumentSummaryBuilder();
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
