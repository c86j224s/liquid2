// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'export_output_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$ExportOutputBody extends ExportOutputBody {
  @override
  final ExportArtifact export_;

  factory _$ExportOutputBody(
          [void Function(ExportOutputBodyBuilder)? updates]) =>
      (ExportOutputBodyBuilder()..update(updates))._build();

  _$ExportOutputBody._({required this.export_}) : super._();
  @override
  ExportOutputBody rebuild(void Function(ExportOutputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  ExportOutputBodyBuilder toBuilder() =>
      ExportOutputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is ExportOutputBody && export_ == other.export_;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, export_.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'ExportOutputBody')
          ..add('export_', export_))
        .toString();
  }
}

class ExportOutputBodyBuilder
    implements Builder<ExportOutputBody, ExportOutputBodyBuilder> {
  _$ExportOutputBody? _$v;

  ExportArtifactBuilder? _export_;
  ExportArtifactBuilder get export_ =>
      _$this._export_ ??= ExportArtifactBuilder();
  set export_(ExportArtifactBuilder? export_) => _$this._export_ = export_;

  ExportOutputBodyBuilder() {
    ExportOutputBody._defaults(this);
  }

  ExportOutputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _export_ = $v.export_.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(ExportOutputBody other) {
    _$v = other as _$ExportOutputBody;
  }

  @override
  void update(void Function(ExportOutputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  ExportOutputBody build() => _build();

  _$ExportOutputBody _build() {
    _$ExportOutputBody _$result;
    try {
      _$result = _$v ??
          _$ExportOutputBody._(
            export_: export_.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'export_';
        export_.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'ExportOutputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
