// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'feed_refresh_output_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$FeedRefreshOutputBody extends FeedRefreshOutputBody {
  @override
  final Job job;

  factory _$FeedRefreshOutputBody(
          [void Function(FeedRefreshOutputBodyBuilder)? updates]) =>
      (FeedRefreshOutputBodyBuilder()..update(updates))._build();

  _$FeedRefreshOutputBody._({required this.job}) : super._();
  @override
  FeedRefreshOutputBody rebuild(
          void Function(FeedRefreshOutputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  FeedRefreshOutputBodyBuilder toBuilder() =>
      FeedRefreshOutputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is FeedRefreshOutputBody && job == other.job;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, job.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'FeedRefreshOutputBody')
          ..add('job', job))
        .toString();
  }
}

class FeedRefreshOutputBodyBuilder
    implements Builder<FeedRefreshOutputBody, FeedRefreshOutputBodyBuilder> {
  _$FeedRefreshOutputBody? _$v;

  JobBuilder? _job;
  JobBuilder get job => _$this._job ??= JobBuilder();
  set job(JobBuilder? job) => _$this._job = job;

  FeedRefreshOutputBodyBuilder() {
    FeedRefreshOutputBody._defaults(this);
  }

  FeedRefreshOutputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _job = $v.job.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(FeedRefreshOutputBody other) {
    _$v = other as _$FeedRefreshOutputBody;
  }

  @override
  void update(void Function(FeedRefreshOutputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  FeedRefreshOutputBody build() => _build();

  _$FeedRefreshOutputBody _build() {
    _$FeedRefreshOutputBody _$result;
    try {
      _$result = _$v ??
          _$FeedRefreshOutputBody._(
            job: job.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'job';
        job.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'FeedRefreshOutputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
