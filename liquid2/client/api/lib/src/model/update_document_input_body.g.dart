// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'update_document_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$UpdateDocumentInputBody extends UpdateDocumentInputBody {
  @override
  final String? folderId;
  @override
  final String? title;

  factory _$UpdateDocumentInputBody(
          [void Function(UpdateDocumentInputBodyBuilder)? updates]) =>
      (UpdateDocumentInputBodyBuilder()..update(updates))._build();

  _$UpdateDocumentInputBody._({this.folderId, this.title}) : super._();
  @override
  UpdateDocumentInputBody rebuild(
          void Function(UpdateDocumentInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  UpdateDocumentInputBodyBuilder toBuilder() =>
      UpdateDocumentInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is UpdateDocumentInputBody &&
        folderId == other.folderId &&
        title == other.title;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, folderId.hashCode);
    _$hash = $jc(_$hash, title.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'UpdateDocumentInputBody')
          ..add('folderId', folderId)
          ..add('title', title))
        .toString();
  }
}

class UpdateDocumentInputBodyBuilder
    implements
        Builder<UpdateDocumentInputBody, UpdateDocumentInputBodyBuilder> {
  _$UpdateDocumentInputBody? _$v;

  String? _folderId;
  String? get folderId => _$this._folderId;
  set folderId(String? folderId) => _$this._folderId = folderId;

  String? _title;
  String? get title => _$this._title;
  set title(String? title) => _$this._title = title;

  UpdateDocumentInputBodyBuilder() {
    UpdateDocumentInputBody._defaults(this);
  }

  UpdateDocumentInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _folderId = $v.folderId;
      _title = $v.title;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(UpdateDocumentInputBody other) {
    _$v = other as _$UpdateDocumentInputBody;
  }

  @override
  void update(void Function(UpdateDocumentInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  UpdateDocumentInputBody build() => _build();

  _$UpdateDocumentInputBody _build() {
    final _$result = _$v ??
        _$UpdateDocumentInputBody._(
          folderId: folderId,
          title: title,
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
