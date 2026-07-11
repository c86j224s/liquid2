//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'folder_breadcrumb.g.dart';

/// FolderBreadcrumb
///
/// Properties:
/// * [id]
/// * [name]
@BuiltValue()
abstract class FolderBreadcrumb implements Built<FolderBreadcrumb, FolderBreadcrumbBuilder> {
  @BuiltValueField(wireName: r'id')
  String get id;

  @BuiltValueField(wireName: r'name')
  String get name;

  FolderBreadcrumb._();

  factory FolderBreadcrumb([void updates(FolderBreadcrumbBuilder b)]) = _$FolderBreadcrumb;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(FolderBreadcrumbBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<FolderBreadcrumb> get serializer => _$FolderBreadcrumbSerializer();
}

class _$FolderBreadcrumbSerializer implements PrimitiveSerializer<FolderBreadcrumb> {
  @override
  final Iterable<Type> types = const [FolderBreadcrumb, _$FolderBreadcrumb];

  @override
  final String wireName = r'FolderBreadcrumb';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    FolderBreadcrumb object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'name';
    yield serializers.serialize(
      object.name,
      specifiedType: const FullType(String),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    FolderBreadcrumb object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required FolderBreadcrumbBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'id':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.id = valueDes;
          break;
        case r'name':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.name = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  FolderBreadcrumb deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = FolderBreadcrumbBuilder();
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
