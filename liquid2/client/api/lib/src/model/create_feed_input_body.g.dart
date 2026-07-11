// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_feed_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$CreateFeedInputBody extends CreateFeedInputBody {
  @override
  final bool? enabled;
  @override
  final String? folderId;
  @override
  final String? title;
  @override
  final String url;

  factory _$CreateFeedInputBody(
          [void Function(CreateFeedInputBodyBuilder)? updates]) =>
      (CreateFeedInputBodyBuilder()..update(updates))._build();

  _$CreateFeedInputBody._(
      {this.enabled, this.folderId, this.title, required this.url})
      : super._();
  @override
  CreateFeedInputBody rebuild(
          void Function(CreateFeedInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  CreateFeedInputBodyBuilder toBuilder() =>
      CreateFeedInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is CreateFeedInputBody &&
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
    return (newBuiltValueToStringHelper(r'CreateFeedInputBody')
          ..add('enabled', enabled)
          ..add('folderId', folderId)
          ..add('title', title)
          ..add('url', url))
        .toString();
  }
}

class CreateFeedInputBodyBuilder
    implements Builder<CreateFeedInputBody, CreateFeedInputBodyBuilder> {
  _$CreateFeedInputBody? _$v;

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

  CreateFeedInputBodyBuilder() {
    CreateFeedInputBody._defaults(this);
  }

  CreateFeedInputBodyBuilder get _$this {
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
  void replace(CreateFeedInputBody other) {
    _$v = other as _$CreateFeedInputBody;
  }

  @override
  void update(void Function(CreateFeedInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  CreateFeedInputBody build() => _build();

  _$CreateFeedInputBody _build() {
    final _$result = _$v ??
        _$CreateFeedInputBody._(
          enabled: enabled,
          folderId: folderId,
          title: title,
          url: BuiltValueNullFieldError.checkNotNull(
              url, r'CreateFeedInputBody', 'url'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
