// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'scrape_document_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$ScrapeDocumentInputBody extends ScrapeDocumentInputBody {
  @override
  final String? folderId;
  @override
  final BuiltList<String>? tagIds;
  @override
  final String url;

  factory _$ScrapeDocumentInputBody(
          [void Function(ScrapeDocumentInputBodyBuilder)? updates]) =>
      (ScrapeDocumentInputBodyBuilder()..update(updates))._build();

  _$ScrapeDocumentInputBody._({this.folderId, this.tagIds, required this.url})
      : super._();
  @override
  ScrapeDocumentInputBody rebuild(
          void Function(ScrapeDocumentInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  ScrapeDocumentInputBodyBuilder toBuilder() =>
      ScrapeDocumentInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is ScrapeDocumentInputBody &&
        folderId == other.folderId &&
        tagIds == other.tagIds &&
        url == other.url;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, folderId.hashCode);
    _$hash = $jc(_$hash, tagIds.hashCode);
    _$hash = $jc(_$hash, url.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'ScrapeDocumentInputBody')
          ..add('folderId', folderId)
          ..add('tagIds', tagIds)
          ..add('url', url))
        .toString();
  }
}

class ScrapeDocumentInputBodyBuilder
    implements
        Builder<ScrapeDocumentInputBody, ScrapeDocumentInputBodyBuilder> {
  _$ScrapeDocumentInputBody? _$v;

  String? _folderId;
  String? get folderId => _$this._folderId;
  set folderId(String? folderId) => _$this._folderId = folderId;

  ListBuilder<String>? _tagIds;
  ListBuilder<String> get tagIds => _$this._tagIds ??= ListBuilder<String>();
  set tagIds(ListBuilder<String>? tagIds) => _$this._tagIds = tagIds;

  String? _url;
  String? get url => _$this._url;
  set url(String? url) => _$this._url = url;

  ScrapeDocumentInputBodyBuilder() {
    ScrapeDocumentInputBody._defaults(this);
  }

  ScrapeDocumentInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _folderId = $v.folderId;
      _tagIds = $v.tagIds?.toBuilder();
      _url = $v.url;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(ScrapeDocumentInputBody other) {
    _$v = other as _$ScrapeDocumentInputBody;
  }

  @override
  void update(void Function(ScrapeDocumentInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  ScrapeDocumentInputBody build() => _build();

  _$ScrapeDocumentInputBody _build() {
    _$ScrapeDocumentInputBody _$result;
    try {
      _$result = _$v ??
          _$ScrapeDocumentInputBody._(
            folderId: folderId,
            tagIds: _tagIds?.build(),
            url: BuiltValueNullFieldError.checkNotNull(
                url, r'ScrapeDocumentInputBody', 'url'),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'tagIds';
        _tagIds?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'ScrapeDocumentInputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
