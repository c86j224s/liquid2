// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'note_list.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$NoteList extends NoteList {
  @override
  final BuiltList<DocumentNote>? items;

  factory _$NoteList([void Function(NoteListBuilder)? updates]) =>
      (NoteListBuilder()..update(updates))._build();

  _$NoteList._({this.items}) : super._();
  @override
  NoteList rebuild(void Function(NoteListBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  NoteListBuilder toBuilder() => NoteListBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is NoteList && items == other.items;
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
    return (newBuiltValueToStringHelper(r'NoteList')..add('items', items))
        .toString();
  }
}

class NoteListBuilder implements Builder<NoteList, NoteListBuilder> {
  _$NoteList? _$v;

  ListBuilder<DocumentNote>? _items;
  ListBuilder<DocumentNote> get items =>
      _$this._items ??= ListBuilder<DocumentNote>();
  set items(ListBuilder<DocumentNote>? items) => _$this._items = items;

  NoteListBuilder() {
    NoteList._defaults(this);
  }

  NoteListBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _items = $v.items?.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(NoteList other) {
    _$v = other as _$NoteList;
  }

  @override
  void update(void Function(NoteListBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  NoteList build() => _build();

  _$NoteList _build() {
    _$NoteList _$result;
    try {
      _$result = _$v ??
          _$NoteList._(
            items: _items?.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'items';
        _items?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'NoteList', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
