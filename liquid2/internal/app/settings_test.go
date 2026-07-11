package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestSettingsDefaultAndUpdate(t *testing.T) {
	service := NewService(WithClock(func() int64 { return 1760000000000 }))
	t.Cleanup(func() { _ = service.Close() })
	ctx := context.Background()

	settings, err := service.GetSettings(ctx)
	if err != nil {
		t.Fatalf("get settings: %v", err)
	}
	if settings.FeedSchedulerEnabled || settings.FeedPollIntervalSeconds != 7200 || settings.FeedNextPollAt != nil {
		t.Fatalf("unexpected defaults %#v", settings)
	}

	settings, err = service.UpdateSettings(ctx, UpdateSettingsInput{
		FeedSchedulerEnabled:    settingsBool(true),
		FeedPollIntervalSeconds: settingsInt(300),
	})
	if err != nil {
		t.Fatalf("update settings: %v", err)
	}
	if !settings.FeedSchedulerEnabled || settings.FeedPollIntervalSeconds != 300 || settings.UpdatedAt != 1760000000000 {
		t.Fatalf("unexpected updated settings %#v", settings)
	}
	if settings.FeedNextPollAt == nil || *settings.FeedNextPollAt != 1760000300000 {
		t.Fatalf("unexpected updated settings %#v", settings)
	}

	settings, err = service.UpdateSettings(ctx, UpdateSettingsInput{
		FeedSchedulerEnabled: settingsBool(false),
	})
	if err != nil {
		t.Fatalf("disable scheduler: %v", err)
	}
	if settings.FeedNextPollAt != nil {
		t.Fatalf("expected next poll cleared, got %#v", settings)
	}
}

func TestSettingsRejectInvalidPollInterval(t *testing.T) {
	service := NewService()
	t.Cleanup(func() { _ = service.Close() })

	_, err := service.UpdateSettings(context.Background(), UpdateSettingsInput{
		FeedPollIntervalSeconds: settingsInt(30),
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestSQLiteRepositoryPersistsSettings(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "liquid2.db")
	service, closeService := newSQLiteService(t, ctx, dbPath)

	if _, err := service.UpdateSettings(ctx, UpdateSettingsInput{
		FeedSchedulerEnabled:    settingsBool(true),
		FeedPollIntervalSeconds: settingsInt(900),
	}); err != nil {
		t.Fatalf("update settings: %v", err)
	}
	closeService()

	service, closeService = newSQLiteService(t, ctx, dbPath)
	defer closeService()
	settings, err := service.GetSettings(ctx)
	if err != nil {
		t.Fatalf("get persisted settings: %v", err)
	}
	if !settings.FeedSchedulerEnabled || settings.FeedPollIntervalSeconds != 900 || settings.FeedNextPollAt == nil {
		t.Fatalf("unexpected persisted settings %#v", settings)
	}
}

func settingsBool(value bool) *bool {
	return &value
}

func settingsInt(value int) *int {
	return &value
}
