// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'backup_output_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$BackupOutputBody extends BackupOutputBody {
  @override
  final BackupArtifact backup;

  factory _$BackupOutputBody(
          [void Function(BackupOutputBodyBuilder)? updates]) =>
      (BackupOutputBodyBuilder()..update(updates))._build();

  _$BackupOutputBody._({required this.backup}) : super._();
  @override
  BackupOutputBody rebuild(void Function(BackupOutputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  BackupOutputBodyBuilder toBuilder() =>
      BackupOutputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is BackupOutputBody && backup == other.backup;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, backup.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'BackupOutputBody')
          ..add('backup', backup))
        .toString();
  }
}

class BackupOutputBodyBuilder
    implements Builder<BackupOutputBody, BackupOutputBodyBuilder> {
  _$BackupOutputBody? _$v;

  BackupArtifactBuilder? _backup;
  BackupArtifactBuilder get backup =>
      _$this._backup ??= BackupArtifactBuilder();
  set backup(BackupArtifactBuilder? backup) => _$this._backup = backup;

  BackupOutputBodyBuilder() {
    BackupOutputBody._defaults(this);
  }

  BackupOutputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _backup = $v.backup.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(BackupOutputBody other) {
    _$v = other as _$BackupOutputBody;
  }

  @override
  void update(void Function(BackupOutputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  BackupOutputBody build() => _build();

  _$BackupOutputBody _build() {
    _$BackupOutputBody _$result;
    try {
      _$result = _$v ??
          _$BackupOutputBody._(
            backup: backup.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'backup';
        backup.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'BackupOutputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
