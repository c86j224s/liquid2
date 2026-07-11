// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'update_note_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

const UpdateNoteInputBodyFormatEnum _$updateNoteInputBodyFormatEnum_text =
    const UpdateNoteInputBodyFormatEnum._('text');
const UpdateNoteInputBodyFormatEnum _$updateNoteInputBodyFormatEnum_markdown =
    const UpdateNoteInputBodyFormatEnum._('markdown');

UpdateNoteInputBodyFormatEnum _$updateNoteInputBodyFormatEnumValueOf(
    String name) {
  switch (name) {
    case 'text':
      return _$updateNoteInputBodyFormatEnum_text;
    case 'markdown':
      return _$updateNoteInputBodyFormatEnum_markdown;
    default:
      throw ArgumentError(name);
  }
}

final BuiltSet<UpdateNoteInputBodyFormatEnum>
    _$updateNoteInputBodyFormatEnumValues = BuiltSet<
        UpdateNoteInputBodyFormatEnum>(const <UpdateNoteInputBodyFormatEnum>[
  _$updateNoteInputBodyFormatEnum_text,
  _$updateNoteInputBodyFormatEnum_markdown,
]);

Serializer<UpdateNoteInputBodyFormatEnum>
    _$updateNoteInputBodyFormatEnumSerializer =
    _$UpdateNoteInputBodyFormatEnumSerializer();

class _$UpdateNoteInputBodyFormatEnumSerializer
    implements PrimitiveSerializer<UpdateNoteInputBodyFormatEnum> {
  static const Map<String, Object> _toWire = const <String, Object>{
    'text': 'text',
    'markdown': 'markdown',
  };
  static const Map<Object, String> _fromWire = const <Object, String>{
    'text': 'text',
    'markdown': 'markdown',
  };

  @override
  final Iterable<Type> types = const <Type>[UpdateNoteInputBodyFormatEnum];
  @override
  final String wireName = 'UpdateNoteInputBodyFormatEnum';

  @override
  Object serialize(
          Serializers serializers, UpdateNoteInputBodyFormatEnum object,
          {FullType specifiedType = FullType.unspecified}) =>
      _toWire[object.name] ?? object.name;

  @override
  UpdateNoteInputBodyFormatEnum deserialize(
          Serializers serializers, Object serialized,
          {FullType specifiedType = FullType.unspecified}) =>
      UpdateNoteInputBodyFormatEnum.valueOf(
          _fromWire[serialized] ?? (serialized is String ? serialized : ''));
}

class _$UpdateNoteInputBody extends UpdateNoteInputBody {
  @override
  final String body;
  @override
  final UpdateNoteInputBodyFormatEnum format;

  factory _$UpdateNoteInputBody(
          [void Function(UpdateNoteInputBodyBuilder)? updates]) =>
      (UpdateNoteInputBodyBuilder()..update(updates))._build();

  _$UpdateNoteInputBody._({required this.body, required this.format})
      : super._();
  @override
  UpdateNoteInputBody rebuild(
          void Function(UpdateNoteInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  UpdateNoteInputBodyBuilder toBuilder() =>
      UpdateNoteInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is UpdateNoteInputBody &&
        body == other.body &&
        format == other.format;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, body.hashCode);
    _$hash = $jc(_$hash, format.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'UpdateNoteInputBody')
          ..add('body', body)
          ..add('format', format))
        .toString();
  }
}

class UpdateNoteInputBodyBuilder
    implements Builder<UpdateNoteInputBody, UpdateNoteInputBodyBuilder> {
  _$UpdateNoteInputBody? _$v;

  String? _body;
  String? get body => _$this._body;
  set body(String? body) => _$this._body = body;

  UpdateNoteInputBodyFormatEnum? _format;
  UpdateNoteInputBodyFormatEnum? get format => _$this._format;
  set format(UpdateNoteInputBodyFormatEnum? format) => _$this._format = format;

  UpdateNoteInputBodyBuilder() {
    UpdateNoteInputBody._defaults(this);
  }

  UpdateNoteInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _body = $v.body;
      _format = $v.format;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(UpdateNoteInputBody other) {
    _$v = other as _$UpdateNoteInputBody;
  }

  @override
  void update(void Function(UpdateNoteInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  UpdateNoteInputBody build() => _build();

  _$UpdateNoteInputBody _build() {
    final _$result = _$v ??
        _$UpdateNoteInputBody._(
          body: BuiltValueNullFieldError.checkNotNull(
              body, r'UpdateNoteInputBody', 'body'),
          format: BuiltValueNullFieldError.checkNotNull(
              format, r'UpdateNoteInputBody', 'format'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
