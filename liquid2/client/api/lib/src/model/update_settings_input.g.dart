// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'update_settings_input.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$UpdateSettingsInput extends UpdateSettingsInput {
  @override
  final int? feedPollIntervalSeconds;
  @override
  final bool? feedSchedulerEnabled;

  factory _$UpdateSettingsInput(
          [void Function(UpdateSettingsInputBuilder)? updates]) =>
      (UpdateSettingsInputBuilder()..update(updates))._build();

  _$UpdateSettingsInput._(
      {this.feedPollIntervalSeconds, this.feedSchedulerEnabled})
      : super._();
  @override
  UpdateSettingsInput rebuild(
          void Function(UpdateSettingsInputBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  UpdateSettingsInputBuilder toBuilder() =>
      UpdateSettingsInputBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is UpdateSettingsInput &&
        feedPollIntervalSeconds == other.feedPollIntervalSeconds &&
        feedSchedulerEnabled == other.feedSchedulerEnabled;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, feedPollIntervalSeconds.hashCode);
    _$hash = $jc(_$hash, feedSchedulerEnabled.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'UpdateSettingsInput')
          ..add('feedPollIntervalSeconds', feedPollIntervalSeconds)
          ..add('feedSchedulerEnabled', feedSchedulerEnabled))
        .toString();
  }
}

class UpdateSettingsInputBuilder
    implements Builder<UpdateSettingsInput, UpdateSettingsInputBuilder> {
  _$UpdateSettingsInput? _$v;

  int? _feedPollIntervalSeconds;
  int? get feedPollIntervalSeconds => _$this._feedPollIntervalSeconds;
  set feedPollIntervalSeconds(int? feedPollIntervalSeconds) =>
      _$this._feedPollIntervalSeconds = feedPollIntervalSeconds;

  bool? _feedSchedulerEnabled;
  bool? get feedSchedulerEnabled => _$this._feedSchedulerEnabled;
  set feedSchedulerEnabled(bool? feedSchedulerEnabled) =>
      _$this._feedSchedulerEnabled = feedSchedulerEnabled;

  UpdateSettingsInputBuilder() {
    UpdateSettingsInput._defaults(this);
  }

  UpdateSettingsInputBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _feedPollIntervalSeconds = $v.feedPollIntervalSeconds;
      _feedSchedulerEnabled = $v.feedSchedulerEnabled;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(UpdateSettingsInput other) {
    _$v = other as _$UpdateSettingsInput;
  }

  @override
  void update(void Function(UpdateSettingsInputBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  UpdateSettingsInput build() => _build();

  _$UpdateSettingsInput _build() {
    final _$result = _$v ??
        _$UpdateSettingsInput._(
          feedPollIntervalSeconds: feedPollIntervalSeconds,
          feedSchedulerEnabled: feedSchedulerEnabled,
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
