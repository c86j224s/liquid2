// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'form_file.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$FormFile extends FormFile {
  @override
  final String contentType;
  @override
  final String filename;
  @override
  final bool isSet;
  @override
  final int size;

  factory _$FormFile([void Function(FormFileBuilder)? updates]) =>
      (FormFileBuilder()..update(updates))._build();

  _$FormFile._(
      {required this.contentType,
      required this.filename,
      required this.isSet,
      required this.size})
      : super._();
  @override
  FormFile rebuild(void Function(FormFileBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  FormFileBuilder toBuilder() => FormFileBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is FormFile &&
        contentType == other.contentType &&
        filename == other.filename &&
        isSet == other.isSet &&
        size == other.size;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, contentType.hashCode);
    _$hash = $jc(_$hash, filename.hashCode);
    _$hash = $jc(_$hash, isSet.hashCode);
    _$hash = $jc(_$hash, size.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'FormFile')
          ..add('contentType', contentType)
          ..add('filename', filename)
          ..add('isSet', isSet)
          ..add('size', size))
        .toString();
  }
}

class FormFileBuilder implements Builder<FormFile, FormFileBuilder> {
  _$FormFile? _$v;

  String? _contentType;
  String? get contentType => _$this._contentType;
  set contentType(String? contentType) => _$this._contentType = contentType;

  String? _filename;
  String? get filename => _$this._filename;
  set filename(String? filename) => _$this._filename = filename;

  bool? _isSet;
  bool? get isSet => _$this._isSet;
  set isSet(bool? isSet) => _$this._isSet = isSet;

  int? _size;
  int? get size => _$this._size;
  set size(int? size) => _$this._size = size;

  FormFileBuilder() {
    FormFile._defaults(this);
  }

  FormFileBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _contentType = $v.contentType;
      _filename = $v.filename;
      _isSet = $v.isSet;
      _size = $v.size;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(FormFile other) {
    _$v = other as _$FormFile;
  }

  @override
  void update(void Function(FormFileBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  FormFile build() => _build();

  _$FormFile _build() {
    final _$result = _$v ??
        _$FormFile._(
          contentType: BuiltValueNullFieldError.checkNotNull(
              contentType, r'FormFile', 'contentType'),
          filename: BuiltValueNullFieldError.checkNotNull(
              filename, r'FormFile', 'filename'),
          isSet: BuiltValueNullFieldError.checkNotNull(
              isSet, r'FormFile', 'isSet'),
          size:
              BuiltValueNullFieldError.checkNotNull(size, r'FormFile', 'size'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
