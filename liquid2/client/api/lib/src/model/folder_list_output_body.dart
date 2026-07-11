//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:liquid2_api/src/model/folder.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'folder_list_output_body.g.dart';

/// FolderListOutputBody
///
/// Properties:
/// * [items]
@BuiltValue()
abstract class FolderListOutputBody implements Built<FolderListOutputBody, FolderListOutputBodyBuilder> {
  @BuiltValueField(wireName: r'items')
  BuiltList<Folder>? get items;

  FolderListOutputBody._();

  factory FolderListOutputBody([void updates(FolderListOutputBodyBuilder b)]) = _$FolderListOutputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FolderListOutputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FolderListOutputBody> get serializer => _$FolderListOutputBodySerializer();
}

class _$FolderListOutputBodySerializer implements PrimitiveSerializer<FolderListOutputBody> {
  @override
  final Iterable<Type> types = const [FolderListOutputBody, _$FolderListOutputBody];

  @override
  final String wireName = r'FolderListOutputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FolderListOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'items';
    yield object.items == null ? null : serializers.serialize(
      object.items,
      specifiedType: const FullType.nullable(BuiltList, [FullType(Folder)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FolderListOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FolderListOutputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'items':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(Folder)]),
          ) as BuiltList<Folder>?;
          if (valueDes == null) continue;
          result.items.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FolderListOutputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FolderListOutputBodyBuilder();
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
