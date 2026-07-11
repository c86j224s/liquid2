// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'document_detail.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$DocumentDetail extends DocumentDetail {
  @override
  final BuiltList<BlobMetadata>? blobs;
  @override
  final BuiltList<DocumentContent>? contents;
  @override
  final DocumentMetadata document;
  @override
  final BuiltList<FolderBreadcrumb>? folderPath;
  @override
  final BuiltList<Tag>? tags;

  factory _$DocumentDetail([void Function(DocumentDetailBuilder)? updates]) =>
      (DocumentDetailBuilder()..update(updates))._build();

  _$DocumentDetail._(
      {this.blobs,
      this.contents,
      required this.document,
      this.folderPath,
      this.tags})
      : super._();
  @override
  DocumentDetail rebuild(void Function(DocumentDetailBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  DocumentDetailBuilder toBuilder() => DocumentDetailBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is DocumentDetail &&
        blobs == other.blobs &&
        contents == other.contents &&
        document == other.document &&
        folderPath == other.folderPath &&
        tags == other.tags;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, blobs.hashCode);
    _$hash = $jc(_$hash, contents.hashCode);
    _$hash = $jc(_$hash, document.hashCode);
    _$hash = $jc(_$hash, folderPath.hashCode);
    _$hash = $jc(_$hash, tags.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'DocumentDetail')
          ..add('blobs', blobs)
          ..add('contents', contents)
          ..add('document', document)
          ..add('folderPath', folderPath)
          ..add('tags', tags))
        .toString();
  }
}

class DocumentDetailBuilder
    implements Builder<DocumentDetail, DocumentDetailBuilder> {
  _$DocumentDetail? _$v;

  ListBuilder<BlobMetadata>? _blobs;
  ListBuilder<BlobMetadata> get blobs =>
      _$this._blobs ??= ListBuilder<BlobMetadata>();
  set blobs(ListBuilder<BlobMetadata>? blobs) => _$this._blobs = blobs;

  ListBuilder<DocumentContent>? _contents;
  ListBuilder<DocumentContent> get contents =>
      _$this._contents ??= ListBuilder<DocumentContent>();
  set contents(ListBuilder<DocumentContent>? contents) =>
      _$this._contents = contents;

  DocumentMetadataBuilder? _document;
  DocumentMetadataBuilder get document =>
      _$this._document ??= DocumentMetadataBuilder();
  set document(DocumentMetadataBuilder? document) =>
      _$this._document = document;

  ListBuilder<FolderBreadcrumb>? _folderPath;
  ListBuilder<FolderBreadcrumb> get folderPath =>
      _$this._folderPath ??= ListBuilder<FolderBreadcrumb>();
  set folderPath(ListBuilder<FolderBreadcrumb>? folderPath) =>
      _$this._folderPath = folderPath;

  ListBuilder<Tag>? _tags;
  ListBuilder<Tag> get tags => _$this._tags ??= ListBuilder<Tag>();
  set tags(ListBuilder<Tag>? tags) => _$this._tags = tags;

  DocumentDetailBuilder() {
    DocumentDetail._defaults(this);
  }

  DocumentDetailBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _blobs = $v.blobs?.toBuilder();
      _contents = $v.contents?.toBuilder();
      _document = $v.document.toBuilder();
      _folderPath = $v.folderPath?.toBuilder();
      _tags = $v.tags?.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(DocumentDetail other) {
    _$v = other as _$DocumentDetail;
  }

  @override
  void update(void Function(DocumentDetailBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  DocumentDetail build() => _build();

  _$DocumentDetail _build() {
    _$DocumentDetail _$result;
    try {
      _$result = _$v ??
          _$DocumentDetail._(
            blobs: _blobs?.build(),
            contents: _contents?.build(),
            document: document.build(),
            folderPath: _folderPath?.build(),
            tags: _tags?.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'blobs';
        _blobs?.build();
        _$failedField = 'contents';
        _contents?.build();
        _$failedField = 'document';
        document.build();
        _$failedField = 'folderPath';
        _folderPath?.build();
        _$failedField = 'tags';
        _tags?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'DocumentDetail', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
