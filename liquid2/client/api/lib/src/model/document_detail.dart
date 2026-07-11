//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:liquid2_api/src/model/blob_metadata.dart';
import 'package:built_collection/built_collection.dart';
import 'package:liquid2_api/src/model/tag.dart';
import 'package:liquid2_api/src/model/document_content.dart';
import 'package:liquid2_api/src/model/folder_breadcrumb.dart';
import 'package:liquid2_api/src/model/document_metadata.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'document_detail.g.dart';

/// DocumentDetail
///
/// Properties:
/// * [blobs]
/// * [contents]
/// * [document]
/// * [folderPath]
/// * [tags]
@BuiltValue()
abstract class DocumentDetail implements Built<DocumentDetail, DocumentDetailBuilder> {
  @BuiltValueField(wireName: r'blobs')
  BuiltList<BlobMetadata>? get blobs;

  @BuiltValueField(wireName: r'contents')
  BuiltList<DocumentContent>? get contents;

  @BuiltValueField(wireName: r'document')
  DocumentMetadata get document;

  @BuiltValueField(wireName: r'folderPath')
  BuiltList<FolderBreadcrumb>? get folderPath;

  @BuiltValueField(wireName: r'tags')
  BuiltList<Tag>? get tags;

  DocumentDetail._();

  factory DocumentDetail([void updates(DocumentDetailBuilder b)]) = _$DocumentDetail;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(DocumentDetailBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<DocumentDetail> get serializer => _$DocumentDetailSerializer();
}

class _$DocumentDetailSerializer implements PrimitiveSerializer<DocumentDetail> {
  @override
  final Iterable<Type> types = const [DocumentDetail, _$DocumentDetail];

  @override
  final String wireName = r'DocumentDetail';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    DocumentDetail object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'blobs';
    yield object.blobs == null ? null : serializers.serialize(
      object.blobs,
      specifiedType: const FullType.nullable(BuiltList, [FullType(BlobMetadata)]),
    );
    yield r'contents';
    yield object.contents == null ? null : serializers.serialize(
      object.contents,
      specifiedType: const FullType.nullable(BuiltList, [FullType(DocumentContent)]),
    );
    yield r'document';
    yield serializers.serialize(
      object.document,
      specifiedType: const FullType(DocumentMetadata),
    );
    yield r'folderPath';
    yield object.folderPath == null ? null : serializers.serialize(
      object.folderPath,
      specifiedType: const FullType.nullable(BuiltList, [FullType(FolderBreadcrumb)]),
    );
    yield r'tags';
    yield object.tags == null ? null : serializers.serialize(
      object.tags,
      specifiedType: const FullType.nullable(BuiltList, [FullType(Tag)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    DocumentDetail object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required DocumentDetailBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'blobs':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(BlobMetadata)]),
          ) as BuiltList<BlobMetadata>?;
          if (valueDes == null) continue;
          result.blobs.replace(valueDes);
          break;
        case r'contents':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(DocumentContent)]),
          ) as BuiltList<DocumentContent>?;
          if (valueDes == null) continue;
          result.contents.replace(valueDes);
          break;
        case r'document':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(DocumentMetadata),
          ) as DocumentMetadata;
          result.document.replace(valueDes);
          break;
        case r'folderPath':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(FolderBreadcrumb)]),
          ) as BuiltList<FolderBreadcrumb>?;
          if (valueDes == null) continue;
          result.folderPath.replace(valueDes);
          break;
        case r'tags':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(Tag)]),
          ) as BuiltList<Tag>?;
          if (valueDes == null) continue;
          result.tags.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  DocumentDetail deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = DocumentDetailBuilder();
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
