//
// AUTO-GENERATED FILE, DO NOT MODIFY!
//

// ignore_for_file: unused_element
import 'package:built_value/built_value.dart';
import 'package:built_value/serializer.dart';

part 'export_artifact.g.dart';

/// ExportArtifact
///
/// Properties:
/// * [blobCount]
/// * [createdAt]
/// * [documentCount]
/// * [downloadUrl]
/// * [id]
/// * [manifestVersion]
/// * [sha256]
/// * [sizeBytes]
@BuiltValue()
abstract class ExportArtifact implements Built<ExportArtifact, ExportArtifactBuilder> {
  @BuiltValueField(wireName: r'blobCount')
  int get blobCount;

  @BuiltValueField(wireName: r'createdAt')
  int get createdAt;

  @BuiltValueField(wireName: r'documentCount')
  int get documentCount;

  @BuiltValueField(wireName: r'downloadUrl')
  String? get downloadUrl;

  @BuiltValueField(wireName: r'id')
  String get id;

  @BuiltValueField(wireName: r'manifestVersion')
  int get manifestVersion;

  @BuiltValueField(wireName: r'sha256')
  String get sha256;

  @BuiltValueField(wireName: r'sizeBytes')
  int get sizeBytes;

  ExportArtifact._();

  factory ExportArtifact([void updates(ExportArtifactBuilder b)]) = _$ExportArtifact;

  @BuiltValueHook(initializeBuilder: true)
  static void _defaults(ExportArtifactBuilder b) => b;

  @BuiltValueSerializer(custom: true)
  static Serializer<ExportArtifact> get serializer => _$ExportArtifactSerializer();
}

class _$ExportArtifactSerializer implements PrimitiveSerializer<ExportArtifact> {
  @override
  final Iterable<Type> types = const [ExportArtifact, _$ExportArtifact];

  @override
  final String wireName = r'ExportArtifact';

  Iterable<Object?> _serializeProperties(
    Serializers serializers,
    ExportArtifact object, {
    FullType specifiedType = FullType.unspecified,
  }) sync* {
    yield r'blobCount';
    yield serializers.serialize(
      object.blobCount,
      specifiedType: const FullType(int),
    );
    yield r'createdAt';
    yield serializers.serialize(
      object.createdAt,
      specifiedType: const FullType(int),
    );
    yield r'documentCount';
    yield serializers.serialize(
      object.documentCount,
      specifiedType: const FullType(int),
    );
    yield r'downloadUrl';
    yield object.downloadUrl == null ? null : serializers.serialize(
      object.downloadUrl,
      specifiedType: const FullType.nullable(String),
    );
    yield r'id';
    yield serializers.serialize(
      object.id,
      specifiedType: const FullType(String),
    );
    yield r'manifestVersion';
    yield serializers.serialize(
      object.manifestVersion,
      specifiedType: const FullType(int),
    );
    yield r'sha256';
    yield serializers.serialize(
      object.sha256,
      specifiedType: const FullType(String),
    );
    yield r'sizeBytes';
    yield serializers.serialize(
      object.sizeBytes,
      specifiedType: const FullType(int),
    );
  }

  @override
  Object serialize(
    Serializers serializers,
    ExportArtifact object, {
    FullType specifiedType = FullType.unspecified,
  }) {
    return _serializeProperties(serializers, object, specifiedType: specifiedType).toList();
  }

  void _deserializeProperties(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
    required List<Object?> serializedList,
    required ExportArtifactBuilder result,
    required List<Object?> unhandled,
  }) {
    for (var i = 0; i < serializedList.length; i += 2) {
      final key = serializedList[i] as String;
      final value = serializedList[i + 1];
      switch (key) {
        case r'blobCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.blobCount = valueDes;
          break;
        case r'createdAt':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.createdAt = valueDes;
          break;
        case r'documentCount':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.documentCount = valueDes;
          break;
        case r'downloadUrl':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType.nullable(String),
          ) as String?;
          if (valueDes == null) continue;
          result.downloadUrl = valueDes;
          break;
        case r'id':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.id = valueDes;
          break;
        case r'manifestVersion':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.manifestVersion = valueDes;
          break;
        case r'sha256':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(String),
          ) as String;
          result.sha256 = valueDes;
          break;
        case r'sizeBytes':
          final valueDes = serializers.deserialize(
            value,
            specifiedType: const FullType(int),
          ) as int;
          result.sizeBytes = valueDes;
          break;
        default:
          unhandled.add(key);
          unhandled.add(value);
          break;
      }
    }
  }

  @override
  ExportArtifact deserialize(
    Serializers serializers,
    Object serialized, {
    FullType specifiedType = FullType.unspecified,
  }) {
    final result = ExportArtifactBuilder();
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
