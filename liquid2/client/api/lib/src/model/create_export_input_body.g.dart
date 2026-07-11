// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'create_export_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$CreateExportInputBody extends CreateExportInputBody {
  @override
  final BuiltList<String>? documentIds;
  @override
  final bool? includeBlobs;

  factory _$CreateExportInputBody(
          [void Function(CreateExportInputBodyBuilder)? updates]) =>
      (CreateExportInputBodyBuilder()..update(updates))._build();

  _$CreateExportInputBody._({this.documentIds, this.includeBlobs}) : super._();
  @override
  CreateExportInputBody rebuild(
          void Function(CreateExportInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  CreateExportInputBodyBuilder toBuilder() =>
      CreateExportInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is CreateExportInputBody &&
        documentIds == other.documentIds &&
        includeBlobs == other.includeBlobs;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, documentIds.hashCode);
    _$hash = $jc(_$hash, includeBlobs.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'CreateExportInputBody')
          ..add('documentIds', documentIds)
          ..add('includeBlobs', includeBlobs))
        .toString();
  }
}

class CreateExportInputBodyBuilder
    implements Builder<CreateExportInputBody, CreateExportInputBodyBuilder> {
  _$CreateExportInputBody? _$v;

  ListBuilder<String>? _documentIds;
  ListBuilder<String> get documentIds =>
      _$this._documentIds ??= ListBuilder<String>();
  set documentIds(ListBuilder<String>? documentIds) =>
      _$this._documentIds = documentIds;

  bool? _includeBlobs;
  bool? get includeBlobs => _$this._includeBlobs;
  set includeBlobs(bool? includeBlobs) => _$this._includeBlobs = includeBlobs;

  CreateExportInputBodyBuilder() {
    CreateExportInputBody._defaults(this);
  }

  CreateExportInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _documentIds = $v.documentIds?.toBuilder();
      _includeBlobs = $v.includeBlobs;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(CreateExportInputBody other) {
    _$v = other as _$CreateExportInputBody;
  }

  @override
  void update(void Function(CreateExportInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  CreateExportInputBody build() => _build();

  _$CreateExportInputBody _build() {
    _$CreateExportInputBody _$result;
    try {
      _$result = _$v ??
          _$CreateExportInputBody._(
            documentIds: _documentIds?.build(),
            includeBlobs: includeBlobs,
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'documentIds';
        _documentIds?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'CreateExportInputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
