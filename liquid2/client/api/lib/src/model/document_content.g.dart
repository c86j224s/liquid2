// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'document_content.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$DocumentContent extends DocumentContent {
  @override
  final String content;
  @override
  final String format;
  @override
  final String id;
  @override
  final String? language;
  @override
  final String role;

  factory _$DocumentContent([void Function(DocumentContentBuilder)? updates]) =>
      (DocumentContentBuilder()..update(updates))._build();

  _$DocumentContent._(
      {required this.content,
      required this.format,
      required this.id,
      this.language,
      required this.role})
      : super._();
  @override
  DocumentContent rebuild(void Function(DocumentContentBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  DocumentContentBuilder toBuilder() => DocumentContentBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is DocumentContent &&
        content == other.content &&
        format == other.format &&
        id == other.id &&
        language == other.language &&
        role == other.role;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, content.hashCode);
    _$hash = $jc(_$hash, format.hashCode);
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, language.hashCode);
    _$hash = $jc(_$hash, role.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'DocumentContent')
          ..add('content', content)
          ..add('format', format)
          ..add('id', id)
          ..add('language', language)
          ..add('role', role))
        .toString();
  }
}

class DocumentContentBuilder
    implements Builder<DocumentContent, DocumentContentBuilder> {
  _$DocumentContent? _$v;

  String? _content;
  String? get content => _$this._content;
  set content(String? content) => _$this._content = content;

  String? _format;
  String? get format => _$this._format;
  set format(String? format) => _$this._format = format;

  String? _id;
  String? get id => _$this._id;
  set id(String? id) => _$this._id = id;

  String? _language;
  String? get language => _$this._language;
  set language(String? language) => _$this._language = language;

  String? _role;
  String? get role => _$this._role;
  set role(String? role) => _$this._role = role;

  DocumentContentBuilder() {
    DocumentContent._defaults(this);
  }

  DocumentContentBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _content = $v.content;
      _format = $v.format;
      _id = $v.id;
      _language = $v.language;
      _role = $v.role;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(DocumentContent other) {
    _$v = other as _$DocumentContent;
  }

  @override
  void update(void Function(DocumentContentBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  DocumentContent build() => _build();

  _$DocumentContent _build() {
    final _$result = _$v ??
        _$DocumentContent._(
          content: BuiltValueNullFieldError.checkNotNull(
              content, r'DocumentContent', 'content'),
          format: BuiltValueNullFieldError.checkNotNull(
              format, r'DocumentContent', 'format'),
          id: BuiltValueNullFieldError.checkNotNull(
              id, r'DocumentContent', 'id'),
          language: language,
          role: BuiltValueNullFieldError.checkNotNull(
              role, r'DocumentContent', 'role'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
