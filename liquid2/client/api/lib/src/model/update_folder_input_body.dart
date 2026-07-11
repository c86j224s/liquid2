//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'update_folder_input_body.g.dart';

/// UpdateFolderInputBody
///
/// Properties:
/// * [name]
/// * [parentId]
/// * [sortOrder]
@BuiltValue()
abstract class UpdateFolderInputBody implements Built<UpdateFolderInputBody, UpdateFolderInputBodyBuilder> {
  @BuiltValueField(wireName: r'name')
  String get name;

  @BuiltValueField(wireName: r'parentId')
  String? get parentId;

  @BuiltValueField(wireName: r'sortOrder')
  int get sortOrder;

  UpdateFolderInputBody._();

  factory UpdateFolderInputBody([void updates(UpdateFolderInputBodyBuilder b)]) = _$UpdateFolderInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UpdateFolderInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UpdateFolderInputBody> get serializer => _$UpdateFolderInputBodySerializer();
}

class _$UpdateFolderInputBodySerializer implements PrimitiveSerializer<UpdateFolderInputBody> {
  @override
  final Iterable<Type> types = const [UpdateFolderInputBody, _$UpdateFolderInputBody];

  @override
  final String wireName = r'UpdateFolderInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UpdateFolderInputBody object, {
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
    UpdateFolderInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UpdateFolderInputBodyBuilder result,
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
  UpdateFolderInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UpdateFolderInputBodyBuilder();
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
