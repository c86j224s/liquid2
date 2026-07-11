// GENERATED CODE - DO NOT MODIFY BY HAND

part of 'app_settings.dart';

// **************************************************************************
// BuiltValueGenerator
// **************************************************************************

class _$AppSettings extends AppSettings {
  @override
  final int? feedNextPollAt;
  @override
  final int feedPollIntervalSeconds;
  @override
  final bool feedSchedulerEnabled;
  @override
  final int updatedAt;

  factory _$AppSettings([void Function(AppSettingsBuilder)? updates]) =>
      (AppSettingsBuilder()..update(updates))._build();

  _$AppSettings._(
      {this.feedNextPollAt,
      required this.feedPollIntervalSeconds,
      required this.feedSchedulerEnabled,
      required this.updatedAt})
      : super._();
  @override
  AppSettings rebuild(void Function(AppSettingsBuilder) updates) =>
      (toBuilder()..update(updates)).build();

  @override
  AppSettingsBuilder toBuilder() => AppSettingsBuilder()..replace(this);

  @override
  bool operator ==(Object other) {
    if (identical(other, this)) return true;
    return other is AppSettings &&
        feedNextPollAt == other.feedNextPollAt &&
        feedPollIntervalSeconds == other.feedPollIntervalSeconds &&
        feedSchedulerEnabled == other.feedSchedulerEnabled &&
        updatedAt == other.updatedAt;
  }

  @override
  int get hashCode {
    var _$hash = 0;
    _$hash = $jc(_$hash, feedNextPollAt.hashCode);
    _$hash = $jc(_$hash, feedPollIntervalSeconds.hashCode);
    _$hash = $jc(_$hash, feedSchedulerEnabled.hashCode);
    _$hash = $jc(_$hash, updatedAt.hashCode);
    _$hash = $jf(_$hash);
    return _$hash;
  }

  @override
  String toString() {
    return (newBuiltValueToStringHelper(r'AppSettings')
          ..add('feedNextPollAt', feedNextPollAt)
          ..add('feedPollIntervalSeconds', feedPollIntervalSeconds)
          ..add('feedSchedulerEnabled', feedSchedulerEnabled)
          ..add('updatedAt', updatedAt))
        .toString();
  }
}

class AppSettingsBuilder implements Builder<AppSettings, AppSettingsBuilder> {
  _$AppSettings? _$v;

  int? _feedNextPollAt;
  int? get feedNextPollAt => _$this._feedNextPollAt;
  set feedNextPollAt(int? feedNextPollAt) =>
      _$this._feedNextPollAt = feedNextPollAt;

  int? _feedPollIntervalSeconds;
  int? get feedPollIntervalSeconds => _$this._feedPollIntervalSeconds;
  set feedPollIntervalSeconds(int? feedPollIntervalSeconds) =>
      _$this._feedPollIntervalSeconds = feedPollIntervalSeconds;

  bool? _feedSchedulerEnabled;
  bool? get feedSchedulerEnabled => _$this._feedSchedulerEnabled;
  set feedSchedulerEnabled(bool? feedSchedulerEnabled) =>
      _$this._feedSchedulerEnabled = feedSchedulerEnabled;

  int? _updatedAt;
  int? get updatedAt => _$this._updatedAt;
  set updatedAt(int? updatedAt) => _$this._updatedAt = updatedAt;

  AppSettingsBuilder() {
    AppSettings._defaults(this);
  }

  AppSettingsBuilder get _$this {
    final $v = _$v;
    if ($v != null) {
      _feedNextPollAt = $v.feedNextPollAt;
      _feedPollIntervalSeconds = $v.feedPollIntervalSeconds;
      _feedSchedulerEnabled = $v.feedSchedulerEnabled;
      _updatedAt = $v.updatedAt;
      _$v = null;
    }
    return this;
  }

  @override
  void replace(AppSettings other) {
    _$v = other as _$AppSettings;
  }

  @override
  void update(void Function(AppSettingsBuilder)? updates) {
    if (updates != null) updates(this);
  }

  @override
  AppSettings build() => _build();

  _$AppSettings _build() {
    final _$result = _$v ??
        _$AppSettings._(
          feedNextPollAt: feedNextPollAt,
          feedPollIntervalSeconds: BuiltValueNullFieldError.checkNotNull(
              feedPollIntervalSeconds,
              r'AppSettings',
              'feedPollIntervalSeconds'),
          feedSchedulerEnabled: BuiltValueNullFieldError.checkNotNull(
              feedSchedulerEnabled, r'AppSettings', 'feedSchedulerEnabled'),
          updatedAt: BuiltValueNullFieldError.checkNotNull(
              updatedAt, r'AppSettings', 'updatedAt'),
        );
    replace(_$result);
    return _$result;
  }
}

// ignore_for_file: deprecated_member_use_from_same_package,type=lint
