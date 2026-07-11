// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'document_summary.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$DocumentSummary extends DocumentSummary {
  @override
  final String? canonicalUrl;
  @override
  final int createdAt;
  @override
  final int? deletedAt;
  @override
  final String? folderId;
  @override
  final BuiltList<FolderBreadcrumb>? folderPath;
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
  final BuiltList<String>? tags;
  @override
  final String title;
  @override
  final int updatedAt;

  factory _$DocumentSummary([void Function(DocumentSummaryBuilder)? updates]) =>
      (DocumentSummaryBuilder()..update(updates))._build();

  _$DocumentSummary._(
      {this.canonicalUrl,
      required this.createdAt,
      this.deletedAt,
      this.folderId,
      this.folderPath,
      required this.id,
      required this.kind,
      this.language,
      this.publishedAt,
      this.rating,
      this.readAt,
      this.sourceUrl,
      required this.status,
      this.tags,
      required this.title,
      required this.updatedAt})
      : super._();
  @override
  DocumentSummary rebuild(void Function(DocumentSummaryBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  DocumentSummaryBuilder toBuilder() => DocumentSummaryBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is DocumentSummary &&
        canonicalUrl == other.canonicalUrl &&
        createdAt == other.createdAt &&
        deletedAt == other.deletedAt &&
        folderId == other.folderId &&
        folderPath == other.folderPath &&
        id == other.id &&
        kind == other.kind &&
        language == other.language &&
        publishedAt == other.publishedAt &&
        rating == other.rating &&
        readAt == other.readAt &&
        sourceUrl == other.sourceUrl &&
        status == other.status &&
        tags == other.tags &&
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
    _$hash = $jc(_$hash, folderPath.hashCode);
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, kind.hashCode);
    _$hash = $jc(_$hash, language.hashCode);
    _$hash = $jc(_$hash, publishedAt.hashCode);
    _$hash = $jc(_$hash, rating.hashCode);
    _$hash = $jc(_$hash, readAt.hashCode);
    _$hash = $jc(_$hash, sourceUrl.hashCode);
    _$hash = $jc(_$hash, status.hashCode);
    _$hash = $jc(_$hash, tags.hashCode);
    _$hash = $jc(_$hash, title.hashCode);
    _$hash = $jc(_$hash, updatedAt.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'DocumentSummary')
          ..add('canonicalUrl', canonicalUrl)
          ..add('createdAt', createdAt)
          ..add('deletedAt', deletedAt)
          ..add('folderId', folderId)
          ..add('folderPath', folderPath)
          ..add('id', id)
          ..add('kind', kind)
          ..add('language', language)
          ..add('publishedAt', publishedAt)
          ..add('rating', rating)
          ..add('readAt', readAt)
          ..add('sourceUrl', sourceUrl)
          ..add('status', status)
          ..add('tags', tags)
          ..add('title', title)
          ..add('updatedAt', updatedAt))
        .toString();
  }
}

class DocumentSummaryBuilder
    implements Builder<DocumentSummary, DocumentSummaryBuilder> {
  _$DocumentSummary? _$v;

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

  ListBuilder<FolderBreadcrumb>? _folderPath;
  ListBuilder<FolderBreadcrumb> get folderPath =>
      _$this._folderPath ??= ListBuilder<FolderBreadcrumb>();
  set folderPath(ListBuilder<FolderBreadcrumb>? folderPath) =>
      _$this._folderPath = folderPath;

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

  ListBuilder<String>? _tags;
  ListBuilder<String> get tags => _$this._tags ??= ListBuilder<String>();
  set tags(ListBuilder<String>? tags) => _$this._tags = tags;

  String? _title;
  String? get title => _$this._title;
  set title(String? title) => _$this._title = title;

  int? _updatedAt;
  int? get updatedAt => _$this._updatedAt;
  set updatedAt(int? updatedAt) => _$this._updatedAt = updatedAt;

  DocumentSummaryBuilder() {
    DocumentSummary._defaults(this);
  }

  DocumentSummaryBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _canonicalUrl = $v.canonicalUrl;
      _createdAt = $v.createdAt;
      _deletedAt = $v.deletedAt;
      _folderId = $v.folderId;
      _folderPath = $v.folderPath?.toBuilder();
      _id = $v.id;
      _kind = $v.kind;
      _language = $v.language;
      _publishedAt = $v.publishedAt;
      _rating = $v.rating;
      _readAt = $v.readAt;
      _sourceUrl = $v.sourceUrl;
      _status = $v.status;
      _tags = $v.tags?.toBuilder();
      _title = $v.title;
      _updatedAt = $v.updatedAt;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(DocumentSummary other) {
    _$v = other as _$DocumentSummary;
  }

  @override
  void update(void Function(DocumentSummaryBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  DocumentSummary build() => _build();

  _$DocumentSummary _build() {
    _$DocumentSummary _$result;
    try {
      _$result = _$v ??
          _$DocumentSummary._(
            canonicalUrl: canonicalUrl,
            createdAt: BuiltValueNullFieldError.checkNotNull(
                createdAt, r'DocumentSummary', 'createdAt'),
            deletedAt: deletedAt,
            folderId: folderId,
            folderPath: _folderPath?.build(),
            id: BuiltValueNullFieldError.checkNotNull(
                id, r'DocumentSummary', 'id'),
            kind: BuiltValueNullFieldError.checkNotNull(
                kind, r'DocumentSummary', 'kind'),
            language: language,
            publishedAt: publishedAt,
            rating: rating,
            readAt: readAt,
            sourceUrl: sourceUrl,
            status: BuiltValueNullFieldError.checkNotNull(
                status, r'DocumentSummary', 'status'),
            tags: _tags?.build(),
            title: BuiltValueNullFieldError.checkNotNull(
                title, r'DocumentSummary', 'title'),
            updatedAt: BuiltValueNullFieldError.checkNotNull(
                updatedAt, r'DocumentSummary', 'updatedAt'),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'folderPath';
        _folderPath?.build();

        _$failedField = 'tags';
        _tags?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'DocumentSummary', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
