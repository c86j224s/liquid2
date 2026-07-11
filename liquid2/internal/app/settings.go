package app

import "context"

func (s *Service) GetSettings(ctx context.Context) (AppSettings, error) {
	return withView(ctx, s, func(tx RepositoryReader) (AppSettings, error) {
		return tx.Settings(), nil
	})
}

func (s *Service) UpdateSettings(ctx context.Context, input UpdateSettingsInput) (AppSettings, error) {
	return withUpdate(ctx, s, func(tx RepositoryTx) (AppSettings, error) {
		settings := tx.Settings()
		if input.FeedSchedulerEnabled != nil {
			settings.FeedSchedulerEnabled = *input.FeedSchedulerEnabled
		}
		if input.FeedPollIntervalSeconds != nil {
			interval := *input.FeedPollIntervalSeconds
			if interval < MinFeedPollIntervalSeconds || interval > MaxFeedPollIntervalSeconds {
				return AppSettings{}, validation("feed poll interval must be between 60 and 86400 seconds")
			}
			settings.FeedPollIntervalSeconds = interval
		}
		settings.UpdatedAt = tx.Now()
		if !settings.FeedSchedulerEnabled {
			settings.FeedNextPollAt = nil
		} else if input.FeedSchedulerEnabled != nil || input.FeedPollIntervalSeconds != nil {
			nextAt := settings.UpdatedAt + int64(settings.FeedPollIntervalSeconds)*1000
			settings.FeedNextPollAt = &nextAt
		}
		tx.PutSettings(settings)
		return settings, nil
	})
}

func (s *Service) SetFeedNextPollAt(ctx context.Context, nextAt *int64) error {
	_, err := withUpdate(ctx, s, func(tx RepositoryTx) (struct{}, error) {
		settings := tx.Settings()
		if sameInt64Ptr(settings.FeedNextPollAt, nextAt) {
			return struct{}{}, nil
		}
		settings.FeedNextPollAt = cloneInt64Ptr(nextAt)
		tx.PutSettings(settings)
		return struct{}{}, nil
	})
	return err
}

func sameInt64Ptr(left *int64, right *int64) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}
