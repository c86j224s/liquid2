// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'feed_list_output_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$FeedListOutputBody extends FeedListOutputBody {
  @override
  final BuiltList<Feed>? items;

  factory _$FeedListOutputBody(
          [void Function(FeedListOutputBodyBuilder)? updates]) =>
      (FeedListOutputBodyBuilder()..update(updates))._build();

  _$FeedListOutputBody._({this.items}) : super._();
  @override
  FeedListOutputBody rebuild(
          void Function(FeedListOutputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  FeedListOutputBodyBuilder toBuilder() =>
      FeedListOutputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is FeedListOutputBody && items == other.items;
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
    return (newBuiltValueToStringHelper(r'FeedListOutputBody')
          ..add('items', items))
        .toString();
  }
}

class FeedListOutputBodyBuilder
    implements Builder<FeedListOutputBody, FeedListOutputBodyBuilder> {
  _$FeedListOutputBody? _$v;

  ListBuilder<Feed>? _items;
  ListBuilder<Feed> get items => _$this._items ??= ListBuilder<Feed>();
  set items(ListBuilder<Feed>? items) => _$this._items = items;

  FeedListOutputBodyBuilder() {
    FeedListOutputBody._defaults(this);
  }

  FeedListOutputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _items = $v.items?.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(FeedListOutputBody other) {
    _$v = other as _$FeedListOutputBody;
  }

  @override
  void update(void Function(FeedListOutputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  FeedListOutputBody build() => _build();

  _$FeedListOutputBody _build() {
    _$FeedListOutputBody _$result;
    try {
      _$result = _$v ??
          _$FeedListOutputBody._(
            items: _items?.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'items';
        _items?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'FeedListOutputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
