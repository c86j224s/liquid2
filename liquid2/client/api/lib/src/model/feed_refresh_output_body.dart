//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:liquid2_api/src/model/job.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'feed_refresh_output_body.g.dart';

/// FeedRefreshOutputBody
///
/// Properties:
/// * [job]
@BuiltValue()
abstract class FeedRefreshOutputBody implements Built<FeedRefreshOutputBody, FeedRefreshOutputBodyBuilder> {
  @BuiltValueField(wireName: r'job')
  Job get job;

  FeedRefreshOutputBody._();

  factory FeedRefreshOutputBody([void updates(FeedRefreshOutputBodyBuilder b)]) = _$FeedRefreshOutputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FeedRefreshOutputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FeedRefreshOutputBody> get serializer => _$FeedRefreshOutputBodySerializer();
}

class _$FeedRefreshOutputBodySerializer implements PrimitiveSerializer<FeedRefreshOutputBody> {
  @override
  final Iterable<Type> types = const [FeedRefreshOutputBody, _$FeedRefreshOutputBody];

  @override
  final String wireName = r'FeedRefreshOutputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FeedRefreshOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'job';
    yield serializers.serialize(
      object.job,
      specifiedType: const FullType(Job),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FeedRefreshOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FeedRefreshOutputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'job':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(Job),
          ) as Job;
          result.job.replace(valueDes);
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FeedRefreshOutputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FeedRefreshOutputBodyBuilder();
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
