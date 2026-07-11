// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'document_note.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$DocumentNote extends DocumentNote {
  @override
  final String body;
  @override
  final int createdAt;
  @override
  final int? deletedAt;
  @override
  final String documentId;
  @override
  final String format;
  @override
  final String id;
  @override
  final int updatedAt;

  factory _$DocumentNote([void Function(DocumentNoteBuilder)? updates]) =>
      (DocumentNoteBuilder()..update(updates))._build();

  _$DocumentNote._(
      {required this.body,
      required this.createdAt,
      this.deletedAt,
      required this.documentId,
      required this.format,
      required this.id,
      required this.updatedAt})
      : super._();
  @override
  DocumentNote rebuild(void Function(DocumentNoteBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  DocumentNoteBuilder toBuilder() => DocumentNoteBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is DocumentNote &&
        body == other.body &&
        createdAt == other.createdAt &&
        deletedAt == other.deletedAt &&
        documentId == other.documentId &&
        format == other.format &&
        id == other.id &&
        updatedAt == other.updatedAt;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, body.hashCode);
    _$hash = $jc(_$hash, createdAt.hashCode);
    _$hash = $jc(_$hash, deletedAt.hashCode);
    _$hash = $jc(_$hash, documentId.hashCode);
    _$hash = $jc(_$hash, format.hashCode);
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, updatedAt.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'DocumentNote')
          ..add('body', body)
          ..add('createdAt', createdAt)
          ..add('deletedAt', deletedAt)
          ..add('documentId', documentId)
          ..add('format', format)
          ..add('id', id)
          ..add('updatedAt', updatedAt))
        .toString();
  }
}

class DocumentNoteBuilder
    implements Builder<DocumentNote, DocumentNoteBuilder> {
  _$DocumentNote? _$v;

  String? _body;
  String? get body => _$this._body;
  set body(String? body) => _$this._body = body;

  int? _createdAt;
  int? get createdAt => _$this._createdAt;
  set createdAt(int? createdAt) => _$this._createdAt = createdAt;

  int? _deletedAt;
  int? get deletedAt => _$this._deletedAt;
  set deletedAt(int? deletedAt) => _$this._deletedAt = deletedAt;

  String? _documentId;
  String? get documentId => _$this._documentId;
  set documentId(String? documentId) => _$this._documentId = documentId;

  String? _format;
  String? get format => _$this._format;
  set format(String? format) => _$this._format = format;

  String? _id;
  String? get id => _$this._id;
  set id(String? id) => _$this._id = id;

  int? _updatedAt;
  int? get updatedAt => _$this._updatedAt;
  set updatedAt(int? updatedAt) => _$this._updatedAt = updatedAt;

  DocumentNoteBuilder() {
    DocumentNote._defaults(this);
  }

  DocumentNoteBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _body = $v.body;
      _createdAt = $v.createdAt;
      _deletedAt = $v.deletedAt;
      _documentId = $v.documentId;
      _format = $v.format;
      _id = $v.id;
      _updatedAt = $v.updatedAt;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(DocumentNote other) {
    _$v = other as _$DocumentNote;
  }

  @override
  void update(void Function(DocumentNoteBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  DocumentNote build() => _build();

  _$DocumentNote _build() {
    final _$result = _$v ??
        _$DocumentNote._(
          body: BuiltValueNullFieldError.checkNotNull(
              body, r'DocumentNote', 'body'),
          createdAt: BuiltValueNullFieldError.checkNotNull(
              createdAt, r'DocumentNote', 'createdAt'),
          deletedAt: deletedAt,
          documentId: BuiltValueNullFieldError.checkNotNull(
              documentId, r'DocumentNote', 'documentId'),
          format: BuiltValueNullFieldError.checkNotNull(
              format, r'DocumentNote', 'format'),
          id: BuiltValueNullFieldError.checkNotNull(id, r'DocumentNote', 'id'),
          updatedAt: BuiltValueNullFieldError.checkNotNull(
              updatedAt, r'DocumentNote', 'updatedAt'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
