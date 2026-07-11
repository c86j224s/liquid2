// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'feed.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$Feed extends Feed {
  @override
  final int createdAt;
  @override
  final bool enabled;
  @override
  final String? folderId;
  @override
  final String id;
  @override
  final int? lastCheckedAt;
  @override
  final String? title;
  @override
  final int updatedAt;
  @override
  final String url;

  factory _$Feed([void Function(FeedBuilder)? updates]) =>
      (FeedBuilder()..update(updates))._build();

  _$Feed._(
      {required this.createdAt,
      required this.enabled,
      this.folderId,
      required this.id,
      this.lastCheckedAt,
      this.title,
      required this.updatedAt,
      required this.url})
      : super._();
  @override
  Feed rebuild(void Function(FeedBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  FeedBuilder toBuilder() => FeedBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is Feed &&
        createdAt == other.createdAt &&
        enabled == other.enabled &&
        folderId == other.folderId &&
        id == other.id &&
        lastCheckedAt == other.lastCheckedAt &&
        title == other.title &&
        updatedAt == other.updatedAt &&
        url == other.url;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, createdAt.hashCode);
    _$hash = $jc(_$hash, enabled.hashCode);
    _$hash = $jc(_$hash, folderId.hashCode);
    _$hash = $jc(_$hash, id.hashCode);
    _$hash = $jc(_$hash, lastCheckedAt.hashCode);
    _$hash = $jc(_$hash, title.hashCode);
    _$hash = $jc(_$hash, updatedAt.hashCode);
    _$hash = $jc(_$hash, url.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'Feed')
          ..add('createdAt', createdAt)
          ..add('enabled', enabled)
          ..add('folderId', folderId)
          ..add('id', id)
          ..add('lastCheckedAt', lastCheckedAt)
          ..add('title', title)
          ..add('updatedAt', updatedAt)
          ..add('url', url))
        .toString();
  }
}

class FeedBuilder implements Builder<Feed, FeedBuilder> {
  _$Feed? _$v;

  int? _createdAt;
  int? get createdAt => _$this._createdAt;
  set createdAt(int? createdAt) => _$this._createdAt = createdAt;

  bool? _enabled;
  bool? get enabled => _$this._enabled;
  set enabled(bool? enabled) => _$this._enabled = enabled;

  String? _folderId;
  String? get folderId => _$this._folderId;
  set folderId(String? folderId) => _$this._folderId = folderId;

  String? _id;
  String? get id => _$this._id;
  set id(String? id) => _$this._id = id;

  int? _lastCheckedAt;
  int? get lastCheckedAt => _$this._lastCheckedAt;
  set lastCheckedAt(int? lastCheckedAt) =>
      _$this._lastCheckedAt = lastCheckedAt;

  String? _title;
  String? get title => _$this._title;
  set title(String? title) => _$this._title = title;

  int? _updatedAt;
  int? get updatedAt => _$this._updatedAt;
  set updatedAt(int? updatedAt) => _$this._updatedAt = updatedAt;

  String? _url;
  String? get url => _$this._url;
  set url(String? url) => _$this._url = url;

  FeedBuilder() {
    Feed._defaults(this);
  }

  FeedBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _createdAt = $v.createdAt;
      _enabled = $v.enabled;
      _folderId = $v.folderId;
      _id = $v.id;
      _lastCheckedAt = $v.lastCheckedAt;
      _title = $v.title;
      _updatedAt = $v.updatedAt;
      _url = $v.url;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(Feed other) {
    _$v = other as _$Feed;
  }

  @override
  void update(void Function(FeedBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  Feed build() => _build();

  _$Feed _build() {
    final _$result = _$v ??
        _$Feed._(
          createdAt: BuiltValueNullFieldError.checkNotNull(
              createdAt, r'Feed', 'createdAt'),
          enabled: BuiltValueNullFieldError.checkNotNull(
              enabled, r'Feed', 'enabled'),
          folderId: folderId,
          id: BuiltValueNullFieldError.checkNotNull(id, r'Feed', 'id'),
          lastCheckedAt: lastCheckedAt,
          title: title,
          updatedAt: BuiltValueNullFieldError.checkNotNull(
              updatedAt, r'Feed', 'updatedAt'),
          url: BuiltValueNullFieldError.checkNotNull(url, r'Feed', 'url'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
