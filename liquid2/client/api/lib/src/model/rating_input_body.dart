//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'rating_input_body.g.dart';

/// RatingInputBody
///
/// Properties:
/// * [rating]
@BuiltValue()
abstract class RatingInputBody implements Built<RatingInputBody, RatingInputBodyBuilder> {
  @BuiltValueField(wireName: r'rating')
  int? get rating;

  RatingInputBody._();

  factory RatingInputBody([void updates(RatingInputBodyBuilder b)]) = _$RatingInputBody;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(RatingInputBodyBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<RatingInputBody> get serializer => _$RatingInputBodySerializer();
}

class _$RatingInputBodySerializer implements PrimitiveSerializer<RatingInputBody> {
  @override
  final Iterable<Type> types = const [RatingInputBody, _$RatingInputBody];

  @override
  final String wireName = r'RatingInputBody';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    RatingInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    if (object.rating != null) {
      yield r'rating';
      yield serializers.serialize(
        object.rating,
        specifiedType: const FullType(int),
      );
    }
  }

  @override
  Object serialize(
    Serializers serializers,
    RatingInputBody object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required RatingInputBodyBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'rating':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.rating = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  RatingInputBody deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = RatingInputBodyBuilder();
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
