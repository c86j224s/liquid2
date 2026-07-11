// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'document_list.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$DocumentList extends DocumentList {
  @override
  final BuiltList<DocumentSummary>? items;
  @override
  final String? nextCursor;
  @override
  final int totalCount;

  factory _$DocumentList([void Function(DocumentListBuilder)? updates]) =>
      (DocumentListBuilder()..update(updates))._build();

  _$DocumentList._({this.items, this.nextCursor, required this.totalCount})
      : super._();
  @override
  DocumentList rebuild(void Function(DocumentListBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  DocumentListBuilder toBuilder() => DocumentListBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is DocumentList &&
        items == other.items &&
        nextCursor == other.nextCursor &&
        totalCount == other.totalCount;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, items.hashCode);
    _$hash = $jc(_$hash, nextCursor.hashCode);
    _$hash = $jc(_$hash, totalCount.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'DocumentList')
          ..add('items', items)
          ..add('nextCursor', nextCursor)
          ..add('totalCount', totalCount))
        .toString();
  }
}

class DocumentListBuilder
    implements Builder<DocumentList, DocumentListBuilder> {
  _$DocumentList? _$v;

  ListBuilder<DocumentSummary>? _items;
  ListBuilder<DocumentSummary> get items =>
      _$this._items ??= ListBuilder<DocumentSummary>();
  set items(ListBuilder<DocumentSummary>? items) => _$this._items = items;

  String? _nextCursor;
  String? get nextCursor => _$this._nextCursor;
  set nextCursor(String? nextCursor) => _$this._nextCursor = nextCursor;

  int? _totalCount;
  int? get totalCount => _$this._totalCount;
  set totalCount(int? totalCount) => _$this._totalCount = totalCount;

  DocumentListBuilder() {
    DocumentList._defaults(this);
  }

  DocumentListBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _items = $v.items?.toBuilder();
      _nextCursor = $v.nextCursor;
      _totalCount = $v.totalCount;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(DocumentList other) {
    _$v = other as _$DocumentList;
  }

  @override
  void update(void Function(DocumentListBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  DocumentList build() => _build();

  _$DocumentList _build() {
    _$DocumentList _$result;
    try {
      _$result = _$v ??
          _$DocumentList._(
            items: _items?.build(),
            nextCursor: nextCursor,
            totalCount: BuiltValueNullFieldError.checkNotNull(
                totalCount, r'DocumentList', 'totalCount'),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'items';
        _items?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'DocumentList', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
