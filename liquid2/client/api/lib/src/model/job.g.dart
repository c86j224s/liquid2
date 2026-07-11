// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'job.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$Job extends Job {
  @override
  final int attempts;
  @override
  final int createdAt;
  @override
  final String? error;
  @override
  final int? finishedAt;
  @override
  final String id;
  @override
  final String kind;
  @override
  final int? startedAt;
  @override
  final String status;
  @override
  final int updatedAt;

  factory _$Job([void Function(JobBuilder)? updates]) =>
      (JobBuilder()..update(updates))._build();

  _$Job._(
      {required this.attempts,
      required this.createdAt,
      this.error,
      this.finishedAt,
      required this.id,
      required this.kind,
      this.startedAt,
      required this.status,
      required this.updatedAt})
      : super._();
  @override
  Job rebuild(void Function(JobBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  JobBuilder toBuilder() => JobBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is Job &&
        attempts == other.attempts &&
        createdAt == other.createdAt &&
        error == other.error &&
        finishedAt == other.finishedAt &&
        id == other.id &&
        kind == other.kind &&
        startedAt == other.startedAt &&
        status == other.status &&
        updatedAt == other.updatedAt;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, attempts.hashCode);
    _$hash = $jc(_$hash, createdAt.hashCode);
    _$hash = $jc(_$hash, error.hashCode);
    _$hash = $jc(_$hash, finishedAt.hashCode);
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, kind.hashCode);
    _$hash = $jc(_$hash, startedAt.hashCode);
    _$hash = $jc(_$hash, status.hashCode);
    _$hash = $jc(_$hash, updatedAt.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'Job')
          ..add('attempts', attempts)
          ..add('createdAt', createdAt)
          ..add('error', error)
          ..add('finishedAt', finishedAt)
          ..add('id', id)
          ..add('kind', kind)
          ..add('startedAt', startedAt)
          ..add('status', status)
          ..add('updatedAt', updatedAt))
        .toString();
  }
}

class JobBuilder implements Builder<Job, JobBuilder> {
  _$Job? _$v;

  int? _attempts;
  int? get attempts => _$this._attempts;
  set attempts(int? attempts) => _$this._attempts = attempts;

  int? _createdAt;
  int? get createdAt => _$this._createdAt;
  set createdAt(int? createdAt) => _$this._createdAt = createdAt;

  String? _error;
  String? get error => _$this._error;
  set error(String? error) => _$this._error = error;

  int? _finishedAt;
  int? get finishedAt => _$this._finishedAt;
  set finishedAt(int? finishedAt) => _$this._finishedAt = finishedAt;

  String? _id;
  String? get id => _$this._id;
  set id(String? id) => _$this._id = id;

  String? _kind;
  String? get kind => _$this._kind;
  set kind(String? kind) => _$this._kind = kind;

  int? _startedAt;
  int? get startedAt => _$this._startedAt;
  set startedAt(int? startedAt) => _$this._startedAt = startedAt;

  String? _status;
  String? get status => _$this._status;
  set status(String? status) => _$this._status = status;

  int? _updatedAt;
  int? get updatedAt => _$this._updatedAt;
  set updatedAt(int? updatedAt) => _$this._updatedAt = updatedAt;

  JobBuilder() {
    Job._defaults(this);
  }

  JobBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _attempts = $v.attempts;
      _createdAt = $v.createdAt;
      _error = $v.error;
      _finishedAt = $v.finishedAt;
      _id = $v.id;
      _kind = $v.kind;
      _startedAt = $v.startedAt;
      _status = $v.status;
      _updatedAt = $v.updatedAt;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(Job other) {
    _$v = other as _$Job;
  }

  @override
  void update(void Function(JobBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  Job build() => _build();

  _$Job _build() {
    final _$result = _$v ??
        _$Job._(
          attempts: BuiltValueNullFieldError.checkNotNull(
              attempts, r'Job', 'attempts'),
          createdAt: BuiltValueNullFieldError.checkNotNull(
              createdAt, r'Job', 'createdAt'),
          error: error,
          finishedAt: finishedAt,
          id: BuiltValueNullFieldError.checkNotNull(id, r'Job', 'id'),
          kind: BuiltValueNullFieldError.checkNotNull(kind, r'Job', 'kind'),
          startedAt: startedAt,
          status:
              BuiltValueNullFieldError.checkNotNull(status, r'Job', 'status'),
          updatedAt: BuiltValueNullFieldError.checkNotNull(
              updatedAt, r'Job', 'updatedAt'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
