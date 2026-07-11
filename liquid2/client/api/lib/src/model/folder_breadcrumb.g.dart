// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'folder_breadcrumb.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$FolderBreadcrumb extends FolderBreadcrumb {
  @override
  final String id;
  @override
  final String name;

  factory _$FolderBreadcrumb(
          [void Function(FolderBreadcrumbBuilder)? updates]) =>
      (FolderBreadcrumbBuilder()..update(updates))._build();

  _$FolderBreadcrumb._({required this.id, required this.name}) : super._();
  @override
  FolderBreadcrumb rebuild(void Function(FolderBreadcrumbBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  FolderBreadcrumbBuilder toBuilder() =>
      FolderBreadcrumbBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is FolderBreadcrumb && id == other.id && name == other.name;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, name.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'FolderBreadcrumb')
          ..add('id', id)
          ..add('name', name))
        .toString();
  }
}

class FolderBreadcrumbBuilder
    implements Builder<FolderBreadcrumb, FolderBreadcrumbBuilder> {
  _$FolderBreadcrumb? _$v;

  String? _id;
  String? get id => _$this._id;
  set id(String? id) => _$this._id = id;

  String? _name;
  String? get name => _$this._name;
  set name(String? name) => _$this._name = name;

  FolderBreadcrumbBuilder() {
    FolderBreadcrumb._defaults(this);
  }

  FolderBreadcrumbBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _id = $v.id;
      _name = $v.name;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(FolderBreadcrumb other) {
    _$v = other as _$FolderBreadcrumb;
  }

  @override
  void update(void Function(FolderBreadcrumbBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  FolderBreadcrumb build() => _build();

  _$FolderBreadcrumb _build() {
    final _$result = _$v ??
        _$FolderBreadcrumb._(
          id: BuiltValueNullFieldError.checkNotNull(
              id, r'FolderBreadcrumb', 'id'),
          name: BuiltValueNullFieldError.checkNotNull(
              name, r'FolderBreadcrumb', 'name'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
