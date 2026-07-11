// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'deleted_output_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$DeletedOutputBody extends DeletedOutputBody {
  @override
  final bool deleted;
  @override
  final int deletedAt;

  factory _$DeletedOutputBody(
          [void Function(DeletedOutputBodyBuilder)? updates]) =>
      (DeletedOutputBodyBuilder()..update(updates))._build();

  _$DeletedOutputBody._({required this.deleted, required this.deletedAt})
      : super._();
  @override
  DeletedOutputBody rebuild(void Function(DeletedOutputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  DeletedOutputBodyBuilder toBuilder() =>
      DeletedOutputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is DeletedOutputBody &&
        deleted == other.deleted &&
        deletedAt == other.deletedAt;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, deleted.hashCode);
    _$hash = $jc(_$hash, deletedAt.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'DeletedOutputBody')
          ..add('deleted', deleted)
          ..add('deletedAt', deletedAt))
        .toString();
  }
}

class DeletedOutputBodyBuilder
    implements Builder<DeletedOutputBody, DeletedOutputBodyBuilder> {
  _$DeletedOutputBody? _$v;

  bool? _deleted;
  bool? get deleted => _$this._deleted;
  set deleted(bool? deleted) => _$this._deleted = deleted;

  int? _deletedAt;
  int? get deletedAt => _$this._deletedAt;
  set deletedAt(int? deletedAt) => _$this._deletedAt = deletedAt;

  DeletedOutputBodyBuilder() {
    DeletedOutputBody._defaults(this);
  }

  DeletedOutputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _deleted = $v.deleted;
      _deletedAt = $v.deletedAt;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(DeletedOutputBody other) {
    _$v = other as _$DeletedOutputBody;
  }

  @override
  void update(void Function(DeletedOutputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  DeletedOutputBody build() => _build();

  _$DeletedOutputBody _build() {
    final _$result = _$v ??
        _$DeletedOutputBody._(
          deleted: BuiltValueNullFieldError.checkNotNull(
              deleted, r'DeletedOutputBody', 'deleted'),
          deletedAt: BuiltValueNullFieldError.checkNotNull(
              deletedAt, r'DeletedOutputBody', 'deletedAt'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
