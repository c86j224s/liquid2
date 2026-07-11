//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'folder_body_input_body.g.dart';

/// FolderBodyInputBody
///
/// Properties:
/// * [name]
/// * [parentId]
/// * [sortOrder]
@BuiltValue()
abstract class FolderBodyInputBody implements Built<FolderBodyInputBody, FolderBodyInputBodyBuilder> {
  @BuiltValueField(wireName: r'name')
  String get name;

  @BuiltValueField(wireName: r'parentId')
  String? get parentId;

  @BuiltValueField(wireName: r'sortOrder')
  int get sortOrder;

  FolderBodyInputBody._();

  factory FolderBodyInputBody([void updates(FolderBodyInputBodyBuilder b)]) = _$FolderBodyInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FolderBodyInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FolderBodyInputBody> get serializer => _$FolderBodyInputBodySerializer();
}

class _$FolderBodyInputBodySerializer implements PrimitiveSerializer<FolderBodyInputBody> {
  @override
  final Iterable<Type> types = const [FolderBodyInputBody, _$FolderBodyInputBody];

  @override
  final String wireName = r'FolderBodyInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FolderBodyInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
    if (object.parentId != null) {
      yield r'parentId';
      yield serializers.serialize(
        object.parentId,
        specifiedType: const FullType(String),
      );
    }
    yield r'sortOrder';
    yield serializers.serialize(
      object.sortOrder,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FolderBodyInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FolderBodyInputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
          break;
        case r'parentId':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.parentId = valueDes;
          break;
        case r'sortOrder':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.sortOrder = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FolderBodyInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FolderBodyInputBodyBuilder();
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
