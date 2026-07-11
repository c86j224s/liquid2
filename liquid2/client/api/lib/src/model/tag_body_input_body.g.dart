// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'tag_body_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$TagBodyInputBody extends TagBodyInputBody {
  @override
  final String name;

  factory _$TagBodyInputBody(
          [void Function(TagBodyInputBodyBuilder)? updates]) =>
      (TagBodyInputBodyBuilder()..update(updates))._build();

  _$TagBodyInputBody._({required this.name}) : super._();
  @override
  TagBodyInputBody rebuild(void Function(TagBodyInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  TagBodyInputBodyBuilder toBuilder() =>
      TagBodyInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is TagBodyInputBody && name == other.name;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, name.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'TagBodyInputBody')..add('name', name))
        .toString();
  }
}

class TagBodyInputBodyBuilder
    implements Builder<TagBodyInputBody, TagBodyInputBodyBuilder> {
  _$TagBodyInputBody? _$v;

  String? _name;
  String? get name => _$this._name;
  set name(String? name) => _$this._name = name;

  TagBodyInputBodyBuilder() {
    TagBodyInputBody._defaults(this);
  }

  TagBodyInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _name = $v.name;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(TagBodyInputBody other) {
    _$v = other as _$TagBodyInputBody;
  }

  @override
  void update(void Function(TagBodyInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  TagBodyInputBody build() => _build();

  _$TagBodyInputBody _build() {
    final _$result = _$v ??
        _$TagBodyInputBody._(
          name: BuiltValueNullFieldError.checkNotNull(
              name, r'TagBodyInputBody', 'name'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
