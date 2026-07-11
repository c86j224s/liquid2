// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'job_list.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$JobList extends JobList {
  @override
  final BuiltList<Job>? items;

  factory _$JobList([void Function(JobListBuilder)? updates]) =>
      (JobListBuilder()..update(updates))._build();

  _$JobList._({this.items}) : super._();
  @override
  JobList rebuild(void Function(JobListBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  JobListBuilder toBuilder() => JobListBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is JobList && items == other.items;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, items.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'JobList')..add('items', items))
        .toString();
  }
}

class JobListBuilder implements Builder<JobList, JobListBuilder> {
  _$JobList? _$v;

  ListBuilder<Job>? _items;
  ListBuilder<Job> get items => _$this._items ??= ListBuilder<Job>();
  set items(ListBuilder<Job>? items) => _$this._items = items;

  JobListBuilder() {
    JobList._defaults(this);
  }

  JobListBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _items = $v.items?.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(JobList other) {
    _$v = other as _$JobList;
  }

  @override
  void update(void Function(JobListBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  JobList build() => _build();

  _$JobList _build() {
    _$JobList _$result;
    try {
      _$result = _$v ??
          _$JobList._(
            items: _items?.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'items';
        _items?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'JobList', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
