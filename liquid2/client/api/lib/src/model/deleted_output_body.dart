//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'deleted_output_body.g.dart';

/// DeletedOutputBody
///
/// Properties:
/// * [deleted]
/// * [deletedAt]
@BuiltValue()
abstract class DeletedOutputBody implements Built<DeletedOutputBody, DeletedOutputBodyBuilder> {
  @BuiltValueField(wireName: r'deleted')
  bool get deleted;

  @BuiltValueField(wireName: r'deletedAt')
  int get deletedAt;

  DeletedOutputBody._();

  factory DeletedOutputBody([void updates(DeletedOutputBodyBuilder b)]) = _$DeletedOutputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(DeletedOutputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<DeletedOutputBody> get serializer => _$DeletedOutputBodySerializer();
}

class _$DeletedOutputBodySerializer implements PrimitiveSerializer<DeletedOutputBody> {
  @override
  final Iterable<Type> types = const [DeletedOutputBody, _$DeletedOutputBody];

  @override
  final String wireName = r'DeletedOutputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    DeletedOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'deleted';
    yield serializers.serialize(
      object.deleted,
      specifiedType: const FullType(bool),
    );
    yield r'deletedAt';
    yield serializers.serialize(
      object.deletedAt,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    DeletedOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required DeletedOutputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'deleted':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.deleted = valueDes;
          break;
        case r'deletedAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.deletedAt = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  DeletedOutputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = DeletedOutputBodyBuilder();
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
