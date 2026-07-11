// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'bookmark_document_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$BookmarkDocumentInputBody extends BookmarkDocumentInputBody {
  @override
  final String? folderId;
  @override
  final BuiltList<String>? tagIds;
  @override
  final String? title;
  @override
  final String url;

  factory _$BookmarkDocumentInputBody(
          [void Function(BookmarkDocumentInputBodyBuilder)? updates]) =>
      (BookmarkDocumentInputBodyBuilder()..update(updates))._build();

  _$BookmarkDocumentInputBody._(
      {this.folderId, this.tagIds, this.title, required this.url})
      : super._();
  @override
  BookmarkDocumentInputBody rebuild(
          void Function(BookmarkDocumentInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  BookmarkDocumentInputBodyBuilder toBuilder() =>
      BookmarkDocumentInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is BookmarkDocumentInputBody &&
        folderId == other.folderId &&
        tagIds == other.tagIds &&
        title == other.title &&
        url == other.url;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, folderId.hashCode);
    _$hash = $jc(_$hash, tagIds.hashCode);
    _$hash = $jc(_$hash, title.hashCode);
    _$hash = $jc(_$hash, url.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'BookmarkDocumentInputBody')
          ..add('folderId', folderId)
          ..add('tagIds', tagIds)
          ..add('title', title)
          ..add('url', url))
        .toString();
  }
}

class BookmarkDocumentInputBodyBuilder
    implements
        Builder<BookmarkDocumentInputBody, BookmarkDocumentInputBodyBuilder> {
  _$BookmarkDocumentInputBody? _$v;

  String? _folderId;
  String? get folderId => _$this._folderId;
  set folderId(String? folderId) => _$this._folderId = folderId;

  ListBuilder<String>? _tagIds;
  ListBuilder<String> get tagIds => _$this._tagIds ??= ListBuilder<String>();
  set tagIds(ListBuilder<String>? tagIds) => _$this._tagIds = tagIds;

  String? _title;
  String? get title => _$this._title;
  set title(String? title) => _$this._title = title;

  String? _url;
  String? get url => _$this._url;
  set url(String? url) => _$this._url = url;

  BookmarkDocumentInputBodyBuilder() {
    BookmarkDocumentInputBody._defaults(this);
  }

  BookmarkDocumentInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _folderId = $v.folderId;
      _tagIds = $v.tagIds?.toBuilder();
      _title = $v.title;
      _url = $v.url;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(BookmarkDocumentInputBody other) {
    _$v = other as _$BookmarkDocumentInputBody;
  }

  @override
  void update(void Function(BookmarkDocumentInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  BookmarkDocumentInputBody build() => _build();

  _$BookmarkDocumentInputBody _build() {
    _$BookmarkDocumentInputBody _$result;
    try {
      _$result = _$v ??
          _$BookmarkDocumentInputBody._(
            folderId: folderId,
            tagIds: _tagIds?.build(),
            title: title,
            url: BuiltValueNullFieldError.checkNotNull(
                url, r'BookmarkDocumentInputBody', 'url'),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'tagIds';
        _tagIds?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'BookmarkDocumentInputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
