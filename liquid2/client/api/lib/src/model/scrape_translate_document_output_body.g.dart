// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'scrape_translate_document_output_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$ScrapeTranslateDocumentOutputBody
    extends ScrapeTranslateDocumentOutputBody {
  @override
  final DocumentDetail document;
  @override
  final Job job;

  factory _$ScrapeTranslateDocumentOutputBody(
          [void Function(ScrapeTranslateDocumentOutputBodyBuilder)? updates]) =>
      (ScrapeTranslateDocumentOutputBodyBuilder()..update(updates))._build();

  _$ScrapeTranslateDocumentOutputBody._(
      {required this.document, required this.job})
      : super._();
  @override
  ScrapeTranslateDocumentOutputBody rebuild(
          void Function(ScrapeTranslateDocumentOutputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  ScrapeTranslateDocumentOutputBodyBuilder toBuilder() =>
      ScrapeTranslateDocumentOutputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is ScrapeTranslateDocumentOutputBody &&
        document == other.document &&
        job == other.job;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, document.hashCode);
    _$hash = $jc(_$hash, job.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'ScrapeTranslateDocumentOutputBody')
          ..add('document', document)
          ..add('job', job))
        .toString();
  }
}

class ScrapeTranslateDocumentOutputBodyBuilder
    implements
        Builder<ScrapeTranslateDocumentOutputBody,
            ScrapeTranslateDocumentOutputBodyBuilder> {
  _$ScrapeTranslateDocumentOutputBody? _$v;

  DocumentDetailBuilder? _document;
  DocumentDetailBuilder get document =>
      _$this._document ??= DocumentDetailBuilder();
  set document(DocumentDetailBuilder? document) => _$this._document = document;

  JobBuilder? _job;
  JobBuilder get job => _$this._job ??= JobBuilder();
  set job(JobBuilder? job) => _$this._job = job;

  ScrapeTranslateDocumentOutputBodyBuilder() {
    ScrapeTranslateDocumentOutputBody._defaults(this);
  }

  ScrapeTranslateDocumentOutputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _document = $v.document.toBuilder();
      _job = $v.job.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(ScrapeTranslateDocumentOutputBody other) {
    _$v = other as _$ScrapeTranslateDocumentOutputBody;
  }

  @override
  void update(
      void Function(ScrapeTranslateDocumentOutputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  ScrapeTranslateDocumentOutputBody build() => _build();

  _$ScrapeTranslateDocumentOutputBody _build() {
    _$ScrapeTranslateDocumentOutputBody _$result;
    try {
      _$result = _$v ??
          _$ScrapeTranslateDocumentOutputBody._(
            document: document.build(),
            job: job.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'document';
        document.build();
        _$failedField = 'job';
        job.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'ScrapeTranslateDocumentOutputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
