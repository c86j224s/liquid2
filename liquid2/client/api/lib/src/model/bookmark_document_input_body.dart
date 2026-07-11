//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'bookmark_document_input_body.g.dart';

/// BookmarkDocumentInputBody
///
/// Properties:
/// * [folderId]
/// * [tagIds]
/// * [title]
/// * [url]
@BuiltValue()
abstract class BookmarkDocumentInputBody implements Built<BookmarkDocumentInputBody, BookmarkDocumentInputBodyBuilder> {
  @BuiltValueField(wireName: r'folderId')
  String? get folderId;

  @BuiltValueField(wireName: r'tagIds')
  BuiltList<String>? get tagIds;

  @BuiltValueField(wireName: r'title')
  String? get title;

  @BuiltValueField(wireName: r'url')
  String get url;

  BookmarkDocumentInputBody._();

  factory BookmarkDocumentInputBody([void updates(BookmarkDocumentInputBodyBuilder b)]) = _$BookmarkDocumentInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(BookmarkDocumentInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<BookmarkDocumentInputBody> get serializer => _$BookmarkDocumentInputBodySerializer();
}

class _$BookmarkDocumentInputBodySerializer implements PrimitiveSerializer<BookmarkDocumentInputBody> {
  @override
  final Iterable<Type> types = const [BookmarkDocumentInputBody, _$BookmarkDocumentInputBody];

  @override
  final String wireName = r'BookmarkDocumentInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    BookmarkDocumentInputBody object, {
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
    if (object.title != null) {
      yield r'title';
      yield serializers.serialize(
        object.title,
        specifiedType: const FullType(String),
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
    BookmarkDocumentInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required BookmarkDocumentInputBodyBuilder result,
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
        case r'title':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.title = valueDes;
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
  BookmarkDocumentInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = BookmarkDocumentInputBodyBuilder();
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
