// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'folder.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$Folder extends Folder {
  @override
  final BuiltList<Folder>? children;
  @override
  final int createdAt;
  @override
  final String id;
  @override
  final String name;
  @override
  final String? parentId;
  @override
  final int sortOrder;
  @override
  final String? systemRole;
  @override
  final int updatedAt;

  factory _$Folder([void Function(FolderBuilder)? updates]) =>
      (FolderBuilder()..update(updates))._build();

  _$Folder._(
      {this.children,
      required this.createdAt,
      required this.id,
      required this.name,
      this.parentId,
      required this.sortOrder,
      this.systemRole,
      required this.updatedAt})
      : super._();
  @override
  Folder rebuild(void Function(FolderBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  FolderBuilder toBuilder() => FolderBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is Folder &&
        children == other.children &&
        createdAt == other.createdAt &&
        id == other.id &&
        name == other.name &&
        parentId == other.parentId &&
        sortOrder == other.sortOrder &&
        systemRole == other.systemRole &&
        updatedAt == other.updatedAt;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, children.hashCode);
    _$hash = $jc(_$hash, createdAt.hashCode);
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, name.hashCode);
    _$hash = $jc(_$hash, parentId.hashCode);
    _$hash = $jc(_$hash, sortOrder.hashCode);
    _$hash = $jc(_$hash, systemRole.hashCode);
    _$hash = $jc(_$hash, updatedAt.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'Folder')
          ..add('children', children)
          ..add('createdAt', createdAt)
          ..add('id', id)
          ..add('name', name)
          ..add('parentId', parentId)
          ..add('sortOrder', sortOrder)
          ..add('systemRole', systemRole)
          ..add('updatedAt', updatedAt))
        .toString();
  }
}

class FolderBuilder implements Builder<Folder, FolderBuilder> {
  _$Folder? _$v;

  ListBuilder<Folder>? _children;
  ListBuilder<Folder> get children =>
      _$this._children ??= ListBuilder<Folder>();
  set children(ListBuilder<Folder>? children) => _$this._children = children;

  int? _createdAt;
  int? get createdAt => _$this._createdAt;
  set createdAt(int? createdAt) => _$this._createdAt = createdAt;

  String? _id;
  String? get id => _$this._id;
  set id(String? id) => _$this._id = id;

  String? _name;
  String? get name => _$this._name;
  set name(String? name) => _$this._name = name;

  String? _parentId;
  String? get parentId => _$this._parentId;
  set parentId(String? parentId) => _$this._parentId = parentId;

  int? _sortOrder;
  int? get sortOrder => _$this._sortOrder;
  set sortOrder(int? sortOrder) => _$this._sortOrder = sortOrder;

  String? _systemRole;
  String? get systemRole => _$this._systemRole;
  set systemRole(String? systemRole) => _$this._systemRole = systemRole;

  int? _updatedAt;
  int? get updatedAt => _$this._updatedAt;
  set updatedAt(int? updatedAt) => _$this._updatedAt = updatedAt;

  FolderBuilder() {
    Folder._defaults(this);
  }

  FolderBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _children = $v.children?.toBuilder();
      _createdAt = $v.createdAt;
      _id = $v.id;
      _name = $v.name;
      _parentId = $v.parentId;
      _sortOrder = $v.sortOrder;
      _systemRole = $v.systemRole;
      _updatedAt = $v.updatedAt;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(Folder other) {
    _$v = other as _$Folder;
  }

  @override
  void update(void Function(FolderBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  Folder build() => _build();

  _$Folder _build() {
    _$Folder _$result;
    try {
      _$result = _$v ??
          _$Folder._(
            children: _children?.build(),
            createdAt: BuiltValueNullFieldError.checkNotNull(
                createdAt, r'Folder', 'createdAt'),
            id: BuiltValueNullFieldError.checkNotNull(id, r'Folder', 'id'),
            name:
                BuiltValueNullFieldError.checkNotNull(name, r'Folder', 'name'),
            parentId: parentId,
            sortOrder: BuiltValueNullFieldError.checkNotNull(
                sortOrder, r'Folder', 'sortOrder'),
            systemRole: systemRole,
            updatedAt: BuiltValueNullFieldError.checkNotNull(
                updatedAt, r'Folder', 'updatedAt'),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'children';
        _children?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'Folder', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
