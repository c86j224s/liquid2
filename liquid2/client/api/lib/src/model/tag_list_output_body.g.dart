// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'tag_list_output_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$TagListOutputBody extends TagListOutputBody {
  @override
  final BuiltList<Tag>? items;

  factory _$TagListOutputBody(
          [void Function(TagListOutputBodyBuilder)? updates]) =>
      (TagListOutputBodyBuilder()..update(updates))._build();

  _$TagListOutputBody._({this.items}) : super._();
  @override
  TagListOutputBody rebuild(void Function(TagListOutputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  TagListOutputBodyBuilder toBuilder() =>
      TagListOutputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is TagListOutputBody && items == other.items;
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
    return (newBuiltValueToStringHelper(r'TagListOutputBody')
          ..add('items', items))
        .toString();
  }
}

class TagListOutputBodyBuilder
    implements Builder<TagListOutputBody, TagListOutputBodyBuilder> {
  _$TagListOutputBody? _$v;

  ListBuilder<Tag>? _items;
  ListBuilder<Tag> get items => _$this._items ??= ListBuilder<Tag>();
  set items(ListBuilder<Tag>? items) => _$this._items = items;

  TagListOutputBodyBuilder() {
    TagListOutputBody._defaults(this);
  }

  TagListOutputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _items = $v.items?.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(TagListOutputBody other) {
    _$v = other as _$TagListOutputBody;
  }

  @override
  void update(void Function(TagListOutputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  TagListOutputBody build() => _build();

  _$TagListOutputBody _build() {
    _$TagListOutputBody _$result;
    try {
      _$result = _$v ??
          _$TagListOutputBody._(
            items: _items?.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'items';
        _items?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'TagListOutputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
