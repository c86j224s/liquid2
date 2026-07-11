// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'document_metadata.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$DocumentMetadata extends DocumentMetadata {
  @override
  final String? canonicalUrl;
  @override
  final int createdAt;
  @override
  final int? deletedAt;
  @override
  final String? folderId;
  @override
  final String id;
  @override
  final String kind;
  @override
  final String? language;
  @override
  final int? publishedAt;
  @override
  final int? rating;
  @override
  final int? readAt;
  @override
  final String? sourceUrl;
  @override
  final String status;
  @override
  final String title;
  @override
  final int updatedAt;

  factory _$DocumentMetadata(
          [void Function(DocumentMetadataBuilder)? updates]) =>
      (DocumentMetadataBuilder()..update(updates))._build();

  _$DocumentMetadata._(
      {this.canonicalUrl,
      required this.createdAt,
      this.deletedAt,
      this.folderId,
      required this.id,
      required this.kind,
      this.language,
      this.publishedAt,
      this.rating,
      this.readAt,
      this.sourceUrl,
      required this.status,
      required this.title,
      required this.updatedAt})
      : super._();
  @override
  DocumentMetadata rebuild(void Function(DocumentMetadataBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  DocumentMetadataBuilder toBuilder() =>
      DocumentMetadataBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is DocumentMetadata &&
        canonicalUrl == other.canonicalUrl &&
        createdAt == other.createdAt &&
        deletedAt == other.deletedAt &&
        folderId == other.folderId &&
        id == other.id &&
        kind == other.kind &&
        language == other.language &&
        publishedAt == other.publishedAt &&
        rating == other.rating &&
        readAt == other.readAt &&
        sourceUrl == other.sourceUrl &&
        status == other.status &&
        title == other.title &&
        updatedAt == other.updatedAt;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, canonicalUrl.hashCode);
    _$hash = $jc(_$hash, createdAt.hashCode);
    _$hash = $jc(_$hash, deletedAt.hashCode);
    _$hash = $jc(_$hash, folderId.hashCode);
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, kind.hashCode);
    _$hash = $jc(_$hash, language.hashCode);
    _$hash = $jc(_$hash, publishedAt.hashCode);
    _$hash = $jc(_$hash, rating.hashCode);
    _$hash = $jc(_$hash, readAt.hashCode);
    _$hash = $jc(_$hash, sourceUrl.hashCode);
    _$hash = $jc(_$hash, status.hashCode);
    _$hash = $jc(_$hash, title.hashCode);
    _$hash = $jc(_$hash, updatedAt.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'DocumentMetadata')
          ..add('canonicalUrl', canonicalUrl)
          ..add('createdAt', createdAt)
          ..add('deletedAt', deletedAt)
          ..add('folderId', folderId)
          ..add('id', id)
          ..add('kind', kind)
          ..add('language', language)
          ..add('publishedAt', publishedAt)
          ..add('rating', rating)
          ..add('readAt', readAt)
          ..add('sourceUrl', sourceUrl)
          ..add('status', status)
          ..add('title', title)
          ..add('updatedAt', updatedAt))
        .toString();
  }
}

class DocumentMetadataBuilder
    implements Builder<DocumentMetadata, DocumentMetadataBuilder> {
  _$DocumentMetadata? _$v;

  String? _canonicalUrl;
  String? get canonicalUrl => _$this._canonicalUrl;
  set canonicalUrl(String? canonicalUrl) => _$this._canonicalUrl = canonicalUrl;

  int? _createdAt;
  int? get createdAt => _$this._createdAt;
  set createdAt(int? createdAt) => _$this._createdAt = createdAt;

  int? _deletedAt;
  int? get deletedAt => _$this._deletedAt;
  set deletedAt(int? deletedAt) => _$this._deletedAt = deletedAt;

  String? _folderId;
  String? get folderId => _$this._folderId;
  set folderId(String? folderId) => _$this._folderId = folderId;

  String? _id;
  String? get id => _$this._id;
  set id(String? id) => _$this._id = id;

  String? _kind;
  String? get kind => _$this._kind;
  set kind(String? kind) => _$this._kind = kind;

  String? _language;
  String? get language => _$this._language;
  set language(String? language) => _$this._language = language;

  int? _publishedAt;
  int? get publishedAt => _$this._publishedAt;
  set publishedAt(int? publishedAt) => _$this._publishedAt = publishedAt;

  int? _rating;
  int? get rating => _$this._rating;
  set rating(int? rating) => _$this._rating = rating;

  int? _readAt;
  int? get readAt => _$this._readAt;
  set readAt(int? readAt) => _$this._readAt = readAt;

  String? _sourceUrl;
  String? get sourceUrl => _$this._sourceUrl;
  set sourceUrl(String? sourceUrl) => _$this._sourceUrl = sourceUrl;

  String? _status;
  String? get status => _$this._status;
  set status(String? status) => _$this._status = status;

  String? _title;
  String? get title => _$this._title;
  set title(String? title) => _$this._title = title;

  int? _updatedAt;
  int? get updatedAt => _$this._updatedAt;
  set updatedAt(int? updatedAt) => _$this._updatedAt = updatedAt;

  DocumentMetadataBuilder() {
    DocumentMetadata._defaults(this);
  }

  DocumentMetadataBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _canonicalUrl = $v.canonicalUrl;
      _createdAt = $v.createdAt;
      _deletedAt = $v.deletedAt;
      _folderId = $v.folderId;
      _id = $v.id;
      _kind = $v.kind;
      _language = $v.language;
      _publishedAt = $v.publishedAt;
      _rating = $v.rating;
      _readAt = $v.readAt;
      _sourceUrl = $v.sourceUrl;
      _status = $v.status;
      _title = $v.title;
      _updatedAt = $v.updatedAt;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(DocumentMetadata other) {
    _$v = other as _$DocumentMetadata;
  }

  @override
  void update(void Function(DocumentMetadataBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  DocumentMetadata build() => _build();

  _$DocumentMetadata _build() {
    final _$result = _$v ??
        _$DocumentMetadata._(
          canonicalUrl: canonicalUrl,
          createdAt: BuiltValueNullFieldError.checkNotNull(
              createdAt, r'DocumentMetadata', 'createdAt'),
          deletedAt: deletedAt,
          folderId: folderId,
          id: BuiltValueNullFieldError.checkNotNull(
              id, r'DocumentMetadata', 'id'),
          kind: BuiltValueNullFieldError.checkNotNull(
              kind, r'DocumentMetadata', 'kind'),
          language: language,
          publishedAt: publishedAt,
          rating: rating,
          readAt: readAt,
          sourceUrl: sourceUrl,
          status: BuiltValueNullFieldError.checkNotNull(
              status, r'DocumentMetadata', 'status'),
          title: BuiltValueNullFieldError.checkNotNull(
              title, r'DocumentMetadata', 'title'),
          updatedAt: BuiltValueNullFieldError.checkNotNull(
              updatedAt, r'DocumentMetadata', 'updatedAt'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
