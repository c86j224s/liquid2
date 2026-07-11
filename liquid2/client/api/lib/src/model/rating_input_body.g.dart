// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'rating_input_body.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$RatingInputBody extends RatingInputBody {
  @override
  final int? rating;

  factory _$RatingInputBody([void Function(RatingInputBodyBuilder)? updates]) =>
      (RatingInputBodyBuilder()..update(updates))._build();

  _$RatingInputBody._({this.rating}) : super._();
  @override
  RatingInputBody rebuild(void Function(RatingInputBodyBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  RatingInputBodyBuilder toBuilder() => RatingInputBodyBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is RatingInputBody && rating == other.rating;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, rating.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'RatingInputBody')
          ..add('rating', rating))
        .toString();
  }
}

class RatingInputBodyBuilder
    implements Builder<RatingInputBody, RatingInputBodyBuilder> {
  _$RatingInputBody? _$v;

  int? _rating;
  int? get rating => _$this._rating;
  set rating(int? rating) => _$this._rating = rating;

  RatingInputBodyBuilder() {
    RatingInputBody._defaults(this);
  }

  RatingInputBodyBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _rating = $v.rating;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(RatingInputBody other) {
    _$v = other as _$RatingInputBody;
  }

  @override
  void update(void Function(RatingInputBodyBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  RatingInputBody build() => _build();

  _$RatingInputBody _build() {
    final _$result = _$v ??
        _$RatingInputBody._(
          rating: rating,
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
