//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:liquid2_api/src/model/backup_artifact.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'backup_output_body.g.dart';

/// BackupOutputBody
///
/// Properties:
/// * [backup]
@BuiltValue()
abstract class BackupOutputBody implements Built<BackupOutputBody, BackupOutputBodyBuilder> {
  @BuiltValueField(wireName: r'backup')
  BackupArtifact get backup;

  BackupOutputBody._();

  factory BackupOutputBody([void updates(BackupOutputBodyBuilder b)]) = _$BackupOutputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(BackupOutputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<BackupOutputBody> get serializer => _$BackupOutputBodySerializer();
}

class _$BackupOutputBodySerializer implements PrimitiveSerializer<BackupOutputBody> {
  @override
  final Iterable<Type> types = const [BackupOutputBody, _$BackupOutputBody];

  @override
  final String wireName = r'BackupOutputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    BackupOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'backup';
    yield serializers.serialize(
      object.backup,
      specifiedType: const FullType(BackupArtifact),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    BackupOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required BackupOutputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'backup':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(BackupArtifact),
          ) as BackupArtifact;
          result.backup.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  BackupOutputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = BackupOutputBodyBuilder();
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
