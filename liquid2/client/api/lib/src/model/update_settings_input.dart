//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'update_settings_input.g.dart';

/// UpdateSettingsInput
///
/// Properties:
/// * [feedPollIntervalSeconds]
/// * [feedSchedulerEnabled]
@BuiltValue()
abstract class UpdateSettingsInput implements Built<UpdateSettingsInput, UpdateSettingsInputBuilder> {
  @BuiltValueField(wireName: r'feedPollIntervalSeconds')
  int? get feedPollIntervalSeconds;

  @BuiltValueField(wireName: r'feedSchedulerEnabled')
  bool? get feedSchedulerEnabled;

  UpdateSettingsInput._();

  factory UpdateSettingsInput([void updates(UpdateSettingsInputBuilder b)]) = _$UpdateSettingsInput;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(UpdateSettingsInputBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<UpdateSettingsInput> get serializer => _$UpdateSettingsInputSerializer();
}

class _$UpdateSettingsInputSerializer implements PrimitiveSerializer<UpdateSettingsInput> {
  @override
  final Iterable<Type> types = const [UpdateSettingsInput, _$UpdateSettingsInput];

  @override
  final String wireName = r'UpdateSettingsInput';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    UpdateSettingsInput object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.feedPollIntervalSeconds != null) {
      yield r'feedPollIntervalSeconds';
      yield serializers.serialize(
        object.feedPollIntervalSeconds,
        specifiedType: const FullType(int),
      );
    }
    if (object.feedSchedulerEnabled != null) {
      yield r'feedSchedulerEnabled';
      yield serializers.serialize(
        object.feedSchedulerEnabled,
        specifiedType: const FullType(bool),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    UpdateSettingsInput object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required UpdateSettingsInputBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'feedPollIntervalSeconds':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.feedPollIntervalSeconds = valueDes;
          break;
        case r'feedSchedulerEnabled':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(bool),
          ) as bool;
          result.feedSchedulerEnabled = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  UpdateSettingsInput deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = UpdateSettingsInputBuilder();
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
