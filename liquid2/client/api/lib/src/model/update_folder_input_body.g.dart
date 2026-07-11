// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'update_folder_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$UpdateFolderInputBody extends UpdateFolderInputBody {
  @override
  final String name;
  @override
  final String? parentId;
  @override
  final int sortOrder;

  factory _$UpdateFolderInputBody(
          [void Function(UpdateFolderInputBodyBuilder)? updates]) =>
      (UpdateFolderInputBodyBuilder()..update(updates))._build();

  _$UpdateFolderInputBody._(
      {required this.name, this.parentId, required this.sortOrder})
      : super._();
  @override
  UpdateFolderInputBody rebuild(
          void Function(UpdateFolderInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  UpdateFolderInputBodyBuilder toBuilder() =>
      UpdateFolderInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is UpdateFolderInputBody &&
        name == other.name &&
        parentId == other.parentId &&
        sortOrder == other.sortOrder;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, name.hashCode);
    _$hash = $jc(_$hash, parentId.hashCode);
    _$hash = $jc(_$hash, sortOrder.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'UpdateFolderInputBody')
          ..add('name', name)
          ..add('parentId', parentId)
          ..add('sortOrder', sortOrder))
        .toString();
  }
}

class UpdateFolderInputBodyBuilder
    implements Builder<UpdateFolderInputBody, UpdateFolderInputBodyBuilder> {
  _$UpdateFolderInputBody? _$v;

  String? _name;
  String? get name => _$this._name;
  set name(String? name) => _$this._name = name;

  String? _parentId;
  String? get parentId => _$this._parentId;
  set parentId(String? parentId) => _$this._parentId = parentId;

  int? _sortOrder;
  int? get sortOrder => _$this._sortOrder;
  set sortOrder(int? sortOrder) => _$this._sortOrder = sortOrder;

  UpdateFolderInputBodyBuilder() {
    UpdateFolderInputBody._defaults(this);
  }

  UpdateFolderInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _name = $v.name;
      _parentId = $v.parentId;
      _sortOrder = $v.sortOrder;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(UpdateFolderInputBody other) {
    _$v = other as _$UpdateFolderInputBody;
  }

  @override
  void update(void Function(UpdateFolderInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  UpdateFolderInputBody build() => _build();

  _$UpdateFolderInputBody _build() {
    final _$result = _$v ??
        _$UpdateFolderInputBody._(
          name: BuiltValueNullFieldError.checkNotNull(
              name, r'UpdateFolderInputBody', 'name'),
          parentId: parentId,
          sortOrder: BuiltValueNullFieldError.checkNotNull(
              sortOrder, r'UpdateFolderInputBody', 'sortOrder'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
