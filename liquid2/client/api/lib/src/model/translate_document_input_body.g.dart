// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'translate_document_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$TranslateDocumentInputBody extends TranslateDocumentInputBody {
  @override
  final String sourceContentId;
  @override
  final String targetLanguage;

  factory _$TranslateDocumentInputBody(
          [void Function(TranslateDocumentInputBodyBuilder)? updates]) =>
      (TranslateDocumentInputBodyBuilder()..update(updates))._build();

  _$TranslateDocumentInputBody._(
      {required this.sourceContentId, required this.targetLanguage})
      : super._();
  @override
  TranslateDocumentInputBody rebuild(
          void Function(TranslateDocumentInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  TranslateDocumentInputBodyBuilder toBuilder() =>
      TranslateDocumentInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is TranslateDocumentInputBody &&
        sourceContentId == other.sourceContentId &&
        targetLanguage == other.targetLanguage;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, sourceContentId.hashCode);
    _$hash = $jc(_$hash, targetLanguage.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'TranslateDocumentInputBody')
          ..add('sourceContentId', sourceContentId)
          ..add('targetLanguage', targetLanguage))
        .toString();
  }
}

class TranslateDocumentInputBodyBuilder
    implements
        Builder<TranslateDocumentInputBody, TranslateDocumentInputBodyBuilder> {
  _$TranslateDocumentInputBody? _$v;

  String? _sourceContentId;
  String? get sourceContentId => _$this._sourceContentId;
  set sourceContentId(String? sourceContentId) =>
      _$this._sourceContentId = sourceContentId;

  String? _targetLanguage;
  String? get targetLanguage => _$this._targetLanguage;
  set targetLanguage(String? targetLanguage) =>
      _$this._targetLanguage = targetLanguage;

  TranslateDocumentInputBodyBuilder() {
    TranslateDocumentInputBody._defaults(this);
  }

  TranslateDocumentInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _sourceContentId = $v.sourceContentId;
      _targetLanguage = $v.targetLanguage;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(TranslateDocumentInputBody other) {
    _$v = other as _$TranslateDocumentInputBody;
  }

  @override
  void update(void Function(TranslateDocumentInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  TranslateDocumentInputBody build() => _build();

  _$TranslateDocumentInputBody _build() {
    final _$result = _$v ??
        _$TranslateDocumentInputBody._(
          sourceContentId: BuiltValueNullFieldError.checkNotNull(
              sourceContentId,
              r'TranslateDocumentInputBody',
              'sourceContentId'),
          targetLanguage: BuiltValueNullFieldError.checkNotNull(
              targetLanguage, r'TranslateDocumentInputBody', 'targetLanguage'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
