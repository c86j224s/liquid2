//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'update_document_input_body.g.dart';

/// UpdateDocumentInputBody
///
/// Properties:
/// * [folderId]
/// * [title]
@BuiltValue()
abstract class UpdateDocumentInputBody implements Built<UpdateDocumentInputBody, UpdateDocumentInputBodyBuilder> {
  @BuiltValueField(wireName: r'folderId')
  String? get folderId;

  @BuiltValueField(wireName: r'title')
  String? get title;

  UpdateDocumentInputBody._();

  factory UpdateDocumentInputBody([void updates(UpdateDocumentInputBodyBuilder b)]) = _$UpdateDocumentInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UpdateDocumentInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UpdateDocumentInputBody> get serializer => _$UpdateDocumentInputBodySerializer();
}

class _$UpdateDocumentInputBodySerializer implements PrimitiveSerializer<UpdateDocumentInputBody> {
  @override
  final Iterable<Type> types = const [UpdateDocumentInputBody, _$UpdateDocumentInputBody];

  @override
  final String wireName = r'UpdateDocumentInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UpdateDocumentInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.folderId != null) {
      yield r'folderId';
      yield serializers.serialize(
        object.folderId,
        specifiedType: const FullType(String),
      );
    }
    if (object.title != null) {
      yield r'title';
      yield serializers.serialize(
        object.title,
        specifiedType: const FullType(String),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    UpdateDocumentInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UpdateDocumentInputBodyBuilder result,
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
        case r'title':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.title = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  UpdateDocumentInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UpdateDocumentInputBodyBuilder();
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
