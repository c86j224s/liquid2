//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'app_settings.g.dart';

/// AppSettings
///
/// Properties:
/// * [feedNextPollAt]
/// * [feedPollIntervalSeconds]
/// * [feedSchedulerEnabled]
/// * [updatedAt]
@BuiltValue()
abstract class AppSettings implements Built<AppSettings, AppSettingsBuilder> {
  @BuiltValueField(wireName: r'feedNextPollAt')
  int? get feedNextPollAt;

  @BuiltValueField(wireName: r'feedPollIntervalSeconds')
  int get feedPollIntervalSeconds;

  @BuiltValueField(wireName: r'feedSchedulerEnabled')
  bool get feedSchedulerEnabled;

  @BuiltValueField(wireName: r'updatedAt')
  int get updatedAt;

  AppSettings._();

  factory AppSettings([void updates(AppSettingsBuilder b)]) = _$AppSettings;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(AppSettingsBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<AppSettings> get serializer => _$AppSettingsSerializer();
}

class _$AppSettingsSerializer implements PrimitiveSerializer<AppSettings> {
  @override
  final Iterable<Type> types = const [AppSettings, _$AppSettings];

  @override
  final String wireName = r'AppSettings';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    AppSettings object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'feedNextPollAt';
    yield object.feedNextPollAt == null ? null : serializers.serialize(
      object.feedNextPollAt,
      specifiedType: const FullType.nullable(int),
    );
    yield r'feedPollIntervalSeconds';
    yield serializers.serialize(
      object.feedPollIntervalSeconds,
      specifiedType: const FullType(int),
    );
    yield r'feedSchedulerEnabled';
    yield serializers.serialize(
      object.feedSchedulerEnabled,
      specifiedType: const FullType(bool),
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
    AppSettings object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required AppSettingsBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'feedNextPollAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(int),
          ) as int?;
          if (valueDes == null) continue;
          result.feedNextPollAt = valueDes;
          break;
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
  AppSettings deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = AppSettingsBuilder();
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
