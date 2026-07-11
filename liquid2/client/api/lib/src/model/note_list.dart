//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:liquid2_api/src/model/document_note.dart';
import 'package:built_collection/built_collection.dart';
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'note_list.g.dart';

/// NoteList
///
/// Properties:
/// * [items]
@BuiltValue()
abstract class NoteList implements Built<NoteList, NoteListBuilder> {
  @BuiltValueField(wireName: r'items')
  BuiltList<DocumentNote>? get items;

  NoteList._();

  factory NoteList([void updates(NoteListBuilder b)]) = _$NoteList;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(NoteListBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<NoteList> get serializer => _$NoteListSerializer();
}

class _$NoteListSerializer implements PrimitiveSerializer<NoteList> {
  @override
  final Iterable<Type> types = const [NoteList, _$NoteList];

  @override
  final String wireName = r'NoteList';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    NoteList object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'items';
    yield object.items == null ? null : serializers.serialize(
      object.items,
      specifiedType: const FullType.nullable(BuiltList, [FullType(DocumentNote)]),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    NoteList object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required NoteListBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'items':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(BuiltList, [FullType(DocumentNote)]),
          ) as BuiltList<DocumentNote>?;
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
  NoteList deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = NoteListBuilder();
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
