// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'blob_metadata.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$BlobMetadata extends BlobMetadata {
  @override
  final int createdAt;
  @override
  final String filename;
  @override
  final String id;
  @override
  final String mimeType;
  @override
  final int size;

  factory _$BlobMetadata([void Function(BlobMetadataBuilder)? updates]) =>
      (BlobMetadataBuilder()..update(updates))._build();

  _$BlobMetadata._(
      {required this.createdAt,
      required this.filename,
      required this.id,
      required this.mimeType,
      required this.size})
      : super._();
  @override
  BlobMetadata rebuild(void Function(BlobMetadataBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  BlobMetadataBuilder toBuilder() => BlobMetadataBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is BlobMetadata &&
        createdAt == other.createdAt &&
        filename == other.filename &&
        id == other.id &&
        mimeType == other.mimeType &&
        size == other.size;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, createdAt.hashCode);
    _$hash = $jc(_$hash, filename.hashCode);
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, mimeType.hashCode);
    _$hash = $jc(_$hash, size.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'BlobMetadata')
          ..add('createdAt', createdAt)
          ..add('filename', filename)
          ..add('id', id)
          ..add('mimeType', mimeType)
          ..add('size', size))
        .toString();
  }
}

class BlobMetadataBuilder
    implements Builder<BlobMetadata, BlobMetadataBuilder> {
  _$BlobMetadata? _$v;

  int? _createdAt;
  int? get createdAt => _$this._createdAt;
  set createdAt(int? createdAt) => _$this._createdAt = createdAt;

  String? _filename;
  String? get filename => _$this._filename;
  set filename(String? filename) => _$this._filename = filename;

  String? _id;
  String? get id => _$this._id;
  set id(String? id) => _$this._id = id;

  String? _mimeType;
  String? get mimeType => _$this._mimeType;
  set mimeType(String? mimeType) => _$this._mimeType = mimeType;

  int? _size;
  int? get size => _$this._size;
  set size(int? size) => _$this._size = size;

  BlobMetadataBuilder() {
    BlobMetadata._defaults(this);
  }

  BlobMetadataBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _createdAt = $v.createdAt;
      _filename = $v.filename;
      _id = $v.id;
      _mimeType = $v.mimeType;
      _size = $v.size;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(BlobMetadata other) {
    _$v = other as _$BlobMetadata;
  }

  @override
  void update(void Function(BlobMetadataBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  BlobMetadata build() => _build();

  _$BlobMetadata _build() {
    final _$result = _$v ??
        _$BlobMetadata._(
          createdAt: BuiltValueNullFieldError.checkNotNull(
              createdAt, r'BlobMetadata', 'createdAt'),
          filename: BuiltValueNullFieldError.checkNotNull(
              filename, r'BlobMetadata', 'filename'),
          id: BuiltValueNullFieldError.checkNotNull(id, r'BlobMetadata', 'id'),
          mimeType: BuiltValueNullFieldError.checkNotNull(
              mimeType, r'BlobMetadata', 'mimeType'),
          size: BuiltValueNullFieldError.checkNotNull(
              size, r'BlobMetadata', 'size'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
