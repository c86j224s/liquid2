// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'folder_list_output_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$FolderListOutputBody extends FolderListOutputBody {
  @override
  final BuiltList<Folder>? items;

  factory _$FolderListOutputBody(
          [void Function(FolderListOutputBodyBuilder)? updates]) =>
      (FolderListOutputBodyBuilder()..update(updates))._build();

  _$FolderListOutputBody._({this.items}) : super._();
  @override
  FolderListOutputBody rebuild(
          void Function(FolderListOutputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  FolderListOutputBodyBuilder toBuilder() =>
      FolderListOutputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is FolderListOutputBody && items == other.items;
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
    return (newBuiltValueToStringHelper(r'FolderListOutputBody')
          ..add('items', items))
        .toString();
  }
}

class FolderListOutputBodyBuilder
    implements Builder<FolderListOutputBody, FolderListOutputBodyBuilder> {
  _$FolderListOutputBody? _$v;

  ListBuilder<Folder>? _items;
  ListBuilder<Folder> get items => _$this._items ??= ListBuilder<Folder>();
  set items(ListBuilder<Folder>? items) => _$this._items = items;

  FolderListOutputBodyBuilder() {
    FolderListOutputBody._defaults(this);
  }

  FolderListOutputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _items = $v.items?.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(FolderListOutputBody other) {
    _$v = other as _$FolderListOutputBody;
  }

  @override
  void update(void Function(FolderListOutputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  FolderListOutputBody build() => _build();

  _$FolderListOutputBody _build() {
    _$FolderListOutputBody _$result;
    try {
      _$result = _$v ??
          _$FolderListOutputBody._(
            items: _items?.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'items';
        _items?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'FolderListOutputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
