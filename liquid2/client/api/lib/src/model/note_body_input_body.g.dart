// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'note_body_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

const NoteBodyInputBodyFormatEnum _$noteBodyInputBodyFormatEnum_text =
    const NoteBodyInputBodyFormatEnum._('text');
const NoteBodyInputBodyFormatEnum _$noteBodyInputBodyFormatEnum_markdown =
    const NoteBodyInputBodyFormatEnum._('markdown');

NoteBodyInputBodyFormatEnum _$noteBodyInputBodyFormatEnumValueOf(String name) {
  switch (name) {
    case 'text':
      return _$noteBodyInputBodyFormatEnum_text;
    case 'markdown':
      return _$noteBodyInputBodyFormatEnum_markdown;
    default:
      throw ArgumentError(name);
  }
}

final BuiltSet<NoteBodyInputBodyFormatEnum>
    _$noteBodyInputBodyFormatEnumValues =
    BuiltSet<NoteBodyInputBodyFormatEnum>(const <NoteBodyInputBodyFormatEnum>[
  _$noteBodyInputBodyFormatEnum_text,
  _$noteBodyInputBodyFormatEnum_markdown,
]);

Serializer<NoteBodyInputBodyFormatEnum>
    _$noteBodyInputBodyFormatEnumSerializer =
    _$NoteBodyInputBodyFormatEnumSerializer();

class _$NoteBodyInputBodyFormatEnumSerializer
    implements PrimitiveSerializer<NoteBodyInputBodyFormatEnum> {
  static const Map<String, Object> _toWire = const <String, Object>{
    'text': 'text',
    'markdown': 'markdown',
  };
  static const Map<Object, String> _fromWire = const <Object, String>{
    'text': 'text',
    'markdown': 'markdown',
  };

  @override
  final Iterable<Type> types = const <Type>[NoteBodyInputBodyFormatEnum];
  @override
  final String wireName = 'NoteBodyInputBodyFormatEnum';

  @override
  Object serialize(Serializers serializers, NoteBodyInputBodyFormatEnum object,
          {FullType specifiedType = FullType.unspecified}) =>
      _toWire[object.name] ?? object.name;

  @override
  NoteBodyInputBodyFormatEnum deserialize(
          Serializers serializers, Object serialized,
          {FullType specifiedType = FullType.unspecified}) =>
      NoteBodyInputBodyFormatEnum.valueOf(
          _fromWire[serialized] ?? (serialized is String ? serialized : ''));
}

class _$NoteBodyInputBody extends NoteBodyInputBody {
  @override
  final String body;
  @override
  final NoteBodyInputBodyFormatEnum format;

  factory _$NoteBodyInputBody(
          [void Function(NoteBodyInputBodyBuilder)? updates]) =>
      (NoteBodyInputBodyBuilder()..update(updates))._build();

  _$NoteBodyInputBody._({required this.body, required this.format}) : super._();
  @override
  NoteBodyInputBody rebuild(void Function(NoteBodyInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  NoteBodyInputBodyBuilder toBuilder() =>
      NoteBodyInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is NoteBodyInputBody &&
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
    return (newBuiltValueToStringHelper(r'NoteBodyInputBody')
          ..add('body', body)
          ..add('format', format))
        .toString();
  }
}

class NoteBodyInputBodyBuilder
    implements Builder<NoteBodyInputBody, NoteBodyInputBodyBuilder> {
  _$NoteBodyInputBody? _$v;

  String? _body;
  String? get body => _$this._body;
  set body(String? body) => _$this._body = body;

  NoteBodyInputBodyFormatEnum? _format;
  NoteBodyInputBodyFormatEnum? get format => _$this._format;
  set format(NoteBodyInputBodyFormatEnum? format) => _$this._format = format;

  NoteBodyInputBodyBuilder() {
    NoteBodyInputBody._defaults(this);
  }

  NoteBodyInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _body = $v.body;
      _format = $v.format;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(NoteBodyInputBody other) {
    _$v = other as _$NoteBodyInputBody;
  }

  @override
  void update(void Function(NoteBodyInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  NoteBodyInputBody build() => _build();

  _$NoteBodyInputBody _build() {
    final _$result = _$v ??
        _$NoteBodyInputBody._(
          body: BuiltValueNullFieldError.checkNotNull(
              body, r'NoteBodyInputBody', 'body'),
          format: BuiltValueNullFieldError.checkNotNull(
              format, r'NoteBodyInputBody', 'format'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
