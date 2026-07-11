// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'update_feed_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$UpdateFeedInputBody extends UpdateFeedInputBody {
  @override
  final bool? enabled;
  @override
  final String? folderId;
  @override
  final String? title;
  @override
  final String? url;

  factory _$UpdateFeedInputBody(
          [void Function(UpdateFeedInputBodyBuilder)? updates]) =>
      (UpdateFeedInputBodyBuilder()..update(updates))._build();

  _$UpdateFeedInputBody._({this.enabled, this.folderId, this.title, this.url})
      : super._();
  @override
  UpdateFeedInputBody rebuild(
          void Function(UpdateFeedInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  UpdateFeedInputBodyBuilder toBuilder() =>
      UpdateFeedInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is UpdateFeedInputBody &&
        enabled == other.enabled &&
        folderId == other.folderId &&
        title == other.title &&
        url == other.url;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, enabled.hashCode);
    _$hash = $jc(_$hash, folderId.hashCode);
    _$hash = $jc(_$hash, title.hashCode);
    _$hash = $jc(_$hash, url.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'UpdateFeedInputBody')
          ..add('enabled', enabled)
          ..add('folderId', folderId)
          ..add('title', title)
          ..add('url', url))
        .toString();
  }
}

class UpdateFeedInputBodyBuilder
    implements Builder<UpdateFeedInputBody, UpdateFeedInputBodyBuilder> {
  _$UpdateFeedInputBody? _$v;

  bool? _enabled;
  bool? get enabled => _$this._enabled;
  set enabled(bool? enabled) => _$this._enabled = enabled;

  String? _folderId;
  String? get folderId => _$this._folderId;
  set folderId(String? folderId) => _$this._folderId = folderId;

  String? _title;
  String? get title => _$this._title;
  set title(String? title) => _$this._title = title;

  String? _url;
  String? get url => _$this._url;
  set url(String? url) => _$this._url = url;

  UpdateFeedInputBodyBuilder() {
    UpdateFeedInputBody._defaults(this);
  }

  UpdateFeedInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _enabled = $v.enabled;
      _folderId = $v.folderId;
      _title = $v.title;
      _url = $v.url;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(UpdateFeedInputBody other) {
    _$v = other as _$UpdateFeedInputBody;
  }

  @override
  void update(void Function(UpdateFeedInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  UpdateFeedInputBody build() => _build();

  _$UpdateFeedInputBody _build() {
    final _$result = _$v ??
        _$UpdateFeedInputBody._(
          enabled: enabled,
          folderId: folderId,
          title: title,
          url: url,
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
