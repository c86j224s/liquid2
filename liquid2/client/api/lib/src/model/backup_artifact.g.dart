// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'backup_artifact.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$BackupArtifact extends BackupArtifact {
  @override
  final int createdAt;
  @override
  final String? downloadUrl;
  @override
  final String id;
  @override
  final int schemaVersion;
  @override
  final String sha256;
  @override
  final int sizeBytes;
  @override
  final String sourceType;

  factory _$BackupArtifact([void Function(BackupArtifactBuilder)? updates]) =>
      (BackupArtifactBuilder()..update(updates))._build();

  _$BackupArtifact._(
      {required this.createdAt,
      this.downloadUrl,
      required this.id,
      required this.schemaVersion,
      required this.sha256,
      required this.sizeBytes,
      required this.sourceType})
      : super._();
  @override
  BackupArtifact rebuild(void Function(BackupArtifactBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  BackupArtifactBuilder toBuilder() => BackupArtifactBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is BackupArtifact &&
        createdAt == other.createdAt &&
        downloadUrl == other.downloadUrl &&
        id == other.id &&
        schemaVersion == other.schemaVersion &&
        sha256 == other.sha256 &&
        sizeBytes == other.sizeBytes &&
        sourceType == other.sourceType;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, createdAt.hashCode);
    _$hash = $jc(_$hash, downloadUrl.hashCode);
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, schemaVersion.hashCode);
    _$hash = $jc(_$hash, sha256.hashCode);
    _$hash = $jc(_$hash, sizeBytes.hashCode);
    _$hash = $jc(_$hash, sourceType.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'BackupArtifact')
          ..add('createdAt', createdAt)
          ..add('downloadUrl', downloadUrl)
          ..add('id', id)
          ..add('schemaVersion', schemaVersion)
          ..add('sha256', sha256)
          ..add('sizeBytes', sizeBytes)
          ..add('sourceType', sourceType))
        .toString();
  }
}

class BackupArtifactBuilder
    implements Builder<BackupArtifact, BackupArtifactBuilder> {
  _$BackupArtifact? _$v;

  int? _createdAt;
  int? get createdAt => _$this._createdAt;
  set createdAt(int? createdAt) => _$this._createdAt = createdAt;

  String? _downloadUrl;
  String? get downloadUrl => _$this._downloadUrl;
  set downloadUrl(String? downloadUrl) => _$this._downloadUrl = downloadUrl;

  String? _id;
  String? get id => _$this._id;
  set id(String? id) => _$this._id = id;

  int? _schemaVersion;
  int? get schemaVersion => _$this._schemaVersion;
  set schemaVersion(int? schemaVersion) =>
      _$this._schemaVersion = schemaVersion;

  String? _sha256;
  String? get sha256 => _$this._sha256;
  set sha256(String? sha256) => _$this._sha256 = sha256;

  int? _sizeBytes;
  int? get sizeBytes => _$this._sizeBytes;
  set sizeBytes(int? sizeBytes) => _$this._sizeBytes = sizeBytes;

  String? _sourceType;
  String? get sourceType => _$this._sourceType;
  set sourceType(String? sourceType) => _$this._sourceType = sourceType;

  BackupArtifactBuilder() {
    BackupArtifact._defaults(this);
  }

  BackupArtifactBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _createdAt = $v.createdAt;
      _downloadUrl = $v.downloadUrl;
      _id = $v.id;
      _schemaVersion = $v.schemaVersion;
      _sha256 = $v.sha256;
      _sizeBytes = $v.sizeBytes;
      _sourceType = $v.sourceType;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(BackupArtifact other) {
    _$v = other as _$BackupArtifact;
  }

  @override
  void update(void Function(BackupArtifactBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  BackupArtifact build() => _build();

  _$BackupArtifact _build() {
    final _$result = _$v ??
        _$BackupArtifact._(
          createdAt: BuiltValueNullFieldError.checkNotNull(
              createdAt, r'BackupArtifact', 'createdAt'),
          downloadUrl: downloadUrl,
          id: BuiltValueNullFieldError.checkNotNull(
              id, r'BackupArtifact', 'id'),
          schemaVersion: BuiltValueNullFieldError.checkNotNull(
              schemaVersion, r'BackupArtifact', 'schemaVersion'),
          sha256: BuiltValueNullFieldError.checkNotNull(
              sha256, r'BackupArtifact', 'sha256'),
          sizeBytes: BuiltValueNullFieldError.checkNotNull(
              sizeBytes, r'BackupArtifact', 'sizeBytes'),
          sourceType: BuiltValueNullFieldError.checkNotNull(
              sourceType, r'BackupArtifact', 'sourceType'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
