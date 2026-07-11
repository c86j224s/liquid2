// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'scrape_translate_document_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$ScrapeTranslateDocumentInputBody
    extends ScrapeTranslateDocumentInputBody {
  @override
  final String? folderId;
  @override
  final BuiltList<String>? tagIds;
  @override
  final String targetLanguage;
  @override
  final String url;

  factory _$ScrapeTranslateDocumentInputBody(
          [void Function(ScrapeTranslateDocumentInputBodyBuilder)? updates]) =>
      (ScrapeTranslateDocumentInputBodyBuilder()..update(updates))._build();

  _$ScrapeTranslateDocumentInputBody._(
      {this.folderId,
      this.tagIds,
      required this.targetLanguage,
      required this.url})
      : super._();
  @override
  ScrapeTranslateDocumentInputBody rebuild(
          void Function(ScrapeTranslateDocumentInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  ScrapeTranslateDocumentInputBodyBuilder toBuilder() =>
      ScrapeTranslateDocumentInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is ScrapeTranslateDocumentInputBody &&
        folderId == other.folderId &&
        tagIds == other.tagIds &&
        targetLanguage == other.targetLanguage &&
        url == other.url;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, folderId.hashCode);
    _$hash = $jc(_$hash, tagIds.hashCode);
    _$hash = $jc(_$hash, targetLanguage.hashCode);
    _$hash = $jc(_$hash, url.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'ScrapeTranslateDocumentInputBody')
          ..add('folderId', folderId)
          ..add('tagIds', tagIds)
          ..add('targetLanguage', targetLanguage)
          ..add('url', url))
        .toString();
  }
}

class ScrapeTranslateDocumentInputBodyBuilder
    implements
        Builder<ScrapeTranslateDocumentInputBody,
            ScrapeTranslateDocumentInputBodyBuilder> {
  _$ScrapeTranslateDocumentInputBody? _$v;

  String? _folderId;
  String? get folderId => _$this._folderId;
  set folderId(String? folderId) => _$this._folderId = folderId;

  ListBuilder<String>? _tagIds;
  ListBuilder<String> get tagIds => _$this._tagIds ??= ListBuilder<String>();
  set tagIds(ListBuilder<String>? tagIds) => _$this._tagIds = tagIds;

  String? _targetLanguage;
  String? get targetLanguage => _$this._targetLanguage;
  set targetLanguage(String? targetLanguage) =>
      _$this._targetLanguage = targetLanguage;

  String? _url;
  String? get url => _$this._url;
  set url(String? url) => _$this._url = url;

  ScrapeTranslateDocumentInputBodyBuilder() {
    ScrapeTranslateDocumentInputBody._defaults(this);
  }

  ScrapeTranslateDocumentInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _folderId = $v.folderId;
      _tagIds = $v.tagIds?.toBuilder();
      _targetLanguage = $v.targetLanguage;
      _url = $v.url;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(ScrapeTranslateDocumentInputBody other) {
    _$v = other as _$ScrapeTranslateDocumentInputBody;
  }

  @override
  void update(void Function(ScrapeTranslateDocumentInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  ScrapeTranslateDocumentInputBody build() => _build();

  _$ScrapeTranslateDocumentInputBody _build() {
    _$ScrapeTranslateDocumentInputBody _$result;
    try {
      _$result = _$v ??
          _$ScrapeTranslateDocumentInputBody._(
            folderId: folderId,
            tagIds: _tagIds?.build(),
            targetLanguage: BuiltValueNullFieldError.checkNotNull(
                targetLanguage,
                r'ScrapeTranslateDocumentInputBody',
                'targetLanguage'),
            url: BuiltValueNullFieldError.checkNotNull(
                url, r'ScrapeTranslateDocumentInputBody', 'url'),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'tagIds';
        _tagIds?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'ScrapeTranslateDocumentInputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
