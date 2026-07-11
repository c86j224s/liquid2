package app

const (
	DefaultFeedPollIntervalSeconds = 2 * 60 * 60
	MinFeedPollIntervalSeconds     = 60
	MaxFeedPollIntervalSeconds     = 24 * 60 * 60
)

// AppSettings contains user-managed application behavior toggles.
type AppSettings struct {
	FeedSchedulerEnabled    bool   `json:"feedSchedulerEnabled"`
	FeedPollIntervalSeconds int    `json:"feedPollIntervalSeconds"`
	FeedNextPollAt          *int64 `json:"feedNextPollAt"`
	UpdatedAt               int64  `json:"updatedAt"`
}

// UpdateSettingsInput contains partial settings updates.
type UpdateSettingsInput struct {
	FeedSchedulerEnabled    *bool `json:"feedSchedulerEnabled,omitempty"`
	FeedPollIntervalSeconds *int  `json:"feedPollIntervalSeconds,omitempty"`
}

func DefaultAppSettings() AppSettings {
	return AppSettings{FeedPollIntervalSeconds: DefaultFeedPollIntervalSeconds}
}

func cloneAppSettings(settings AppSettings) AppSettings {
	settings.FeedNextPollAt = cloneInt64Ptr(settings.FeedNextPollAt)
	return settings
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
