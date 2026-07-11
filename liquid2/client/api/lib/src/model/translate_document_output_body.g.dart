// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'translate_document_output_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$TranslateDocumentOutputBody extends TranslateDocumentOutputBody {
  @override
  final Job job;

  factory _$TranslateDocumentOutputBody(
          [void Function(TranslateDocumentOutputBodyBuilder)? updates]) =>
      (TranslateDocumentOutputBodyBuilder()..update(updates))._build();

  _$TranslateDocumentOutputBody._({required this.job}) : super._();
  @override
  TranslateDocumentOutputBody rebuild(
          void Function(TranslateDocumentOutputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  TranslateDocumentOutputBodyBuilder toBuilder() =>
      TranslateDocumentOutputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is TranslateDocumentOutputBody && job == other.job;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, job.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'TranslateDocumentOutputBody')
          ..add('job', job))
        .toString();
  }
}

class TranslateDocumentOutputBodyBuilder
    implements
        Builder<TranslateDocumentOutputBody,
            TranslateDocumentOutputBodyBuilder> {
  _$TranslateDocumentOutputBody? _$v;

  JobBuilder? _job;
  JobBuilder get job => _$this._job ??= JobBuilder();
  set job(JobBuilder? job) => _$this._job = job;

  TranslateDocumentOutputBodyBuilder() {
    TranslateDocumentOutputBody._defaults(this);
  }

  TranslateDocumentOutputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _job = $v.job.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(TranslateDocumentOutputBody other) {
    _$v = other as _$TranslateDocumentOutputBody;
  }

  @override
  void update(void Function(TranslateDocumentOutputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  TranslateDocumentOutputBody build() => _build();

  _$TranslateDocumentOutputBody _build() {
    _$TranslateDocumentOutputBody _$result;
    try {
      _$result = _$v ??
          _$TranslateDocumentOutputBody._(
            job: job.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'job';
        job.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'TranslateDocumentOutputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
