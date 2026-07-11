// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'export_artifact.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$ExportArtifact extends ExportArtifact {
  @override
  final int blobCount;
  @override
  final int createdAt;
  @override
  final int documentCount;
  @override
  final String? downloadUrl;
  @override
  final String id;
  @override
  final int manifestVersion;
  @override
  final String sha256;
  @override
  final int sizeBytes;

  factory _$ExportArtifact([void Function(ExportArtifactBuilder)? updates]) =>
      (ExportArtifactBuilder()..update(updates))._build();

  _$ExportArtifact._(
      {required this.blobCount,
      required this.createdAt,
      required this.documentCount,
      this.downloadUrl,
      required this.id,
      required this.manifestVersion,
      required this.sha256,
      required this.sizeBytes})
      : super._();
  @override
  ExportArtifact rebuild(void Function(ExportArtifactBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  ExportArtifactBuilder toBuilder() => ExportArtifactBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is ExportArtifact &&
        blobCount == other.blobCount &&
        createdAt == other.createdAt &&
        documentCount == other.documentCount &&
        downloadUrl == other.downloadUrl &&
        id == other.id &&
        manifestVersion == other.manifestVersion &&
        sha256 == other.sha256 &&
        sizeBytes == other.sizeBytes;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, blobCount.hashCode);
    _$hash = $jc(_$hash, createdAt.hashCode);
    _$hash = $jc(_$hash, documentCount.hashCode);
    _$hash = $jc(_$hash, downloadUrl.hashCode);
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, manifestVersion.hashCode);
    _$hash = $jc(_$hash, sha256.hashCode);
    _$hash = $jc(_$hash, sizeBytes.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'ExportArtifact')
          ..add('blobCount', blobCount)
          ..add('createdAt', createdAt)
          ..add('documentCount', documentCount)
          ..add('downloadUrl', downloadUrl)
          ..add('id', id)
          ..add('manifestVersion', manifestVersion)
          ..add('sha256', sha256)
          ..add('sizeBytes', sizeBytes))
        .toString();
  }
}

class ExportArtifactBuilder
    implements Builder<ExportArtifact, ExportArtifactBuilder> {
  _$ExportArtifact? _$v;

  int? _blobCount;
  int? get blobCount => _$this._blobCount;
  set blobCount(int? blobCount) => _$this._blobCount = blobCount;

  int? _createdAt;
  int? get createdAt => _$this._createdAt;
  set createdAt(int? createdAt) => _$this._createdAt = createdAt;

  int? _documentCount;
  int? get documentCount => _$this._documentCount;
  set documentCount(int? documentCount) =>
      _$this._documentCount = documentCount;

  String? _downloadUrl;
  String? get downloadUrl => _$this._downloadUrl;
  set downloadUrl(String? downloadUrl) => _$this._downloadUrl = downloadUrl;

  String? _id;
  String? get id => _$this._id;
  set id(String? id) => _$this._id = id;

  int? _manifestVersion;
  int? get manifestVersion => _$this._manifestVersion;
  set manifestVersion(int? manifestVersion) =>
      _$this._manifestVersion = manifestVersion;

  String? _sha256;
  String? get sha256 => _$this._sha256;
  set sha256(String? sha256) => _$this._sha256 = sha256;

  int? _sizeBytes;
  int? get sizeBytes => _$this._sizeBytes;
  set sizeBytes(int? sizeBytes) => _$this._sizeBytes = sizeBytes;

  ExportArtifactBuilder() {
    ExportArtifact._defaults(this);
  }

  ExportArtifactBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _blobCount = $v.blobCount;
      _createdAt = $v.createdAt;
      _documentCount = $v.documentCount;
      _downloadUrl = $v.downloadUrl;
      _id = $v.id;
      _manifestVersion = $v.manifestVersion;
      _sha256 = $v.sha256;
      _sizeBytes = $v.sizeBytes;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(ExportArtifact other) {
    _$v = other as _$ExportArtifact;
  }

  @override
  void update(void Function(ExportArtifactBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  ExportArtifact build() => _build();

  _$ExportArtifact _build() {
    final _$result = _$v ??
        _$ExportArtifact._(
          blobCount: BuiltValueNullFieldError.checkNotNull(
              blobCount, r'ExportArtifact', 'blobCount'),
          createdAt: BuiltValueNullFieldError.checkNotNull(
              createdAt, r'ExportArtifact', 'createdAt'),
          documentCount: BuiltValueNullFieldError.checkNotNull(
              documentCount, r'ExportArtifact', 'documentCount'),
          downloadUrl: downloadUrl,
          id: BuiltValueNullFieldError.checkNotNull(
              id, r'ExportArtifact', 'id'),
          manifestVersion: BuiltValueNullFieldError.checkNotNull(
              manifestVersion, r'ExportArtifact', 'manifestVersion'),
          sha256: BuiltValueNullFieldError.checkNotNull(
              sha256, r'ExportArtifact', 'sha256'),
          sizeBytes: BuiltValueNullFieldError.checkNotNull(
              sizeBytes, r'ExportArtifact', 'sizeBytes'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
