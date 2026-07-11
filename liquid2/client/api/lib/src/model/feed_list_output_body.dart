//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_collection/built_collection.dart';
import 'package:liquid2_api/src/model/feed.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'feed_list_output_body.g.dart';

/// FeedListOutputBody
///
/// Properties:
/// * [items]
@BuiltValue()
abstract class FeedListOutputBody implements Built<FeedListOutputBody, FeedListOutputBodyBuilder> {
  @BuiltValueField(wireName: r'items')
  BuiltList<Feed>? get items;

  FeedListOutputBody._();

  factory FeedListOutputBody([void updates(FeedListOutputBodyBuilder b)]) = _$FeedListOutputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FeedListOutputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FeedListOutputBody> get serializer => _$FeedListOutputBodySerializer();
}

class _$FeedListOutputBodySerializer implements PrimitiveSerializer<FeedListOutputBody> {
  @override
  final Iterable<Type> types = const [FeedListOutputBody, _$FeedListOutputBody];

  @override
  final String wireName = r'FeedListOutputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FeedListOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'items';
    yield object.items == null ? null : serializers.serialize(
      object.items,
      specifiedType: const FullType.nullable(BuiltList, [FullType(Feed)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FeedListOutputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FeedListOutputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'items':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(Feed)]),
          ) as BuiltList<Feed>?;
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
  FeedListOutputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FeedListOutputBodyBuilder();
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
