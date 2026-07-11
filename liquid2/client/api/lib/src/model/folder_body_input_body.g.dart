// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'folder_body_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$FolderBodyInputBody extends FolderBodyInputBody {
  @override
  final String name;
  @override
  final String? parentId;
  @override
  final int sortOrder;

  factory _$FolderBodyInputBody(
          [void Function(FolderBodyInputBodyBuilder)? updates]) =>
      (FolderBodyInputBodyBuilder()..update(updates))._build();

  _$FolderBodyInputBody._(
      {required this.name, this.parentId, required this.sortOrder})
      : super._();
  @override
  FolderBodyInputBody rebuild(
          void Function(FolderBodyInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  FolderBodyInputBodyBuilder toBuilder() =>
      FolderBodyInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is FolderBodyInputBody &&
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
    return (newBuiltValueToStringHelper(r'FolderBodyInputBody')
          ..add('name', name)
          ..add('parentId', parentId)
          ..add('sortOrder', sortOrder))
        .toString();
  }
}

class FolderBodyInputBodyBuilder
    implements Builder<FolderBodyInputBody, FolderBodyInputBodyBuilder> {
  _$FolderBodyInputBody? _$v;

  String? _name;
  String? get name => _$this._name;
  set name(String? name) => _$this._name = name;

  String? _parentId;
  String? get parentId => _$this._parentId;
  set parentId(String? parentId) => _$this._parentId = parentId;

  int? _sortOrder;
  int? get sortOrder => _$this._sortOrder;
  set sortOrder(int? sortOrder) => _$this._sortOrder = sortOrder;

  FolderBodyInputBodyBuilder() {
    FolderBodyInputBody._defaults(this);
  }

  FolderBodyInputBodyBuilder get _$this {
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
  void replace(FolderBodyInputBody other) {
    _$v = other as _$FolderBodyInputBody;
  }

  @override
  void update(void Function(FolderBodyInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  FolderBodyInputBody build() => _build();

  _$FolderBodyInputBody _build() {
    final _$result = _$v ??
        _$FolderBodyInputBody._(
          name: BuiltValueNullFieldError.checkNotNull(
              name, r'FolderBodyInputBody', 'name'),
          parentId: parentId,
          sortOrder: BuiltValueNullFieldError.checkNotNull(
              sortOrder, r'FolderBodyInputBody', 'sortOrder'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
