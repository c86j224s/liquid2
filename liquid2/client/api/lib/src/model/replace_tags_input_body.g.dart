// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'replace_tags_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$ReplaceTagsInputBody extends ReplaceTagsInputBody {
  @override
  final BuiltList<String>? tagIds;

  factory _$ReplaceTagsInputBody(
          [void Function(ReplaceTagsInputBodyBuilder)? updates]) =>
      (ReplaceTagsInputBodyBuilder()..update(updates))._build();

  _$ReplaceTagsInputBody._({this.tagIds}) : super._();
  @override
  ReplaceTagsInputBody rebuild(
          void Function(ReplaceTagsInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  ReplaceTagsInputBodyBuilder toBuilder() =>
      ReplaceTagsInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is ReplaceTagsInputBody && tagIds == other.tagIds;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, tagIds.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'ReplaceTagsInputBody')
          ..add('tagIds', tagIds))
        .toString();
  }
}

class ReplaceTagsInputBodyBuilder
    implements Builder<ReplaceTagsInputBody, ReplaceTagsInputBodyBuilder> {
  _$ReplaceTagsInputBody? _$v;

  ListBuilder<String>? _tagIds;
  ListBuilder<String> get tagIds => _$this._tagIds ??= ListBuilder<String>();
  set tagIds(ListBuilder<String>? tagIds) => _$this._tagIds = tagIds;

  ReplaceTagsInputBodyBuilder() {
    ReplaceTagsInputBody._defaults(this);
  }

  ReplaceTagsInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _tagIds = $v.tagIds?.toBuilder();
      _$v = null;
    }
    return this;
  }

  @override
  void replace(ReplaceTagsInputBody other) {
    _$v = other as _$ReplaceTagsInputBody;
  }

  @override
  void update(void Function(ReplaceTagsInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  ReplaceTagsInputBody build() => _build();

  _$ReplaceTagsInputBody _build() {
    _$ReplaceTagsInputBody _$result;
    try {
      _$result = _$v ??
          _$ReplaceTagsInputBody._(
            tagIds: _tagIds?.build(),
          );
    } catch (_) {
      late String _$failedField;
      try {
        _$failedField = 'tagIds';
        _tagIds?.build();
      } catch (e) {
        throw BuiltValueNestedFieldError(
            r'ReplaceTagsInputBody', _$failedField, e.toString());
      }
      rethrow;
    }
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
