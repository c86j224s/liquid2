package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/c86j224s/liquid2/internal/app"
	feedrefresh "github.com/c86j224s/liquid2/internal/feeds"
	jobruntime "github.com/c86j224s/liquid2/internal/jobs"
)

type jobRuntimeConfig struct {
	jobsEnabled         bool
	translationProvider string
}

type jobRuntimeStatus struct {
	jobsEnabled        bool
	translationEnabled bool
}

func startJobRuntime(ctx context.Context, logger *slog.Logger, service *app.Service, queue jobruntime.Queue) (jobRuntimeStatus, func(), error) {
	config, err := readJobRuntimeConfig()
	if err != nil {
		logger.Error("job runtime config failed", slog.String("component", "api"), slog.Any("error", err))
		return jobRuntimeStatus{}, nil, err
	}
	if queue == nil {
		if config.jobsEnabled || config.translationProvider != "" {
			err := errors.New("job runtime requires LIQUID2_DB_PATH")
			logger.Error("job runtime unavailable", slog.String("component", "api"), slog.Any("error", err))
			return jobRuntimeStatus{}, nil, err
		}
		return jobRuntimeStatus{}, func() {}, nil
	}
	if !config.jobsEnabled {
		if config.translationProvider != "" {
			err := errors.New("translation worker requires LIQUID2_JOBS_ENABLED=1")
			logger.Error("translation worker unavailable", slog.String("component", "api"), slog.Any("error", err))
			return jobRuntimeStatus{}, nil, err
		}
		return jobRuntimeStatus{}, func() {}, nil
	}
	pipeline := feedrefresh.NewPipeline(service, nil, nil, feedrefresh.WithLogger(logger))
	runnerOptions := []jobruntime.RunnerOption{
		jobruntime.WithLogger(logger),
		jobruntime.WithHandler(jobruntime.KindPollFeed, pipeline.Handle),
	}
	translationHandler, translationEnabled, err := newTranslationHandler(logger, service, config.translationProvider)
	if err != nil {
		logger.Error("translation worker config failed", slog.String("component", "api"), slog.Any("error", err))
		return jobRuntimeStatus{}, nil, err
	}
	if translationEnabled {
		runnerOptions = append(runnerOptions, jobruntime.WithHandler(jobruntime.KindTranslateDocument, translationHandler))
		logger.Info("translation worker enabled",
			slog.String("component", "api"),
			slog.String("operation", "translation_worker_start"),
			slog.String("provider", config.translationProvider),
		)
	}
	runner := jobruntime.NewRunner(queue, runnerOptions...)
	if err := runner.RecoverRunning(ctx); err != nil {
		logger.Error("job recovery failed", slog.String("component", "api"), slog.String("operation", "job_recovery"), slog.Any("error", err))
		return jobRuntimeStatus{}, nil, err
	}
	runnerDone := make(chan error, 1)
	go func() {
		runnerDone <- runRuntime(logger, "job runner", func() error {
			return runner.Run(ctx)
		})
	}()
	logger.Info("job runner enabled", slog.String("component", "api"), slog.String("operation", "job_runner_start"))
	scheduler, schedulerDone := startFeedScheduler(ctx, logger, service, queue)
	return jobRuntimeStatus{jobsEnabled: true, translationEnabled: translationEnabled}, func() {
		stopFeedScheduler(logger, scheduler, schedulerDone)
		if err := runner.Close(); err != nil {
			logger.Error("close job runner failed", slog.String("component", "api"), slog.Any("error", err))
		}
		<-runnerDone
	}, nil
}

func readJobRuntimeConfig() (jobRuntimeConfig, error) {
	return jobRuntimeConfig{
		jobsEnabled:         getenv("LIQUID2_JOBS_ENABLED", "") == "1",
		translationProvider: normalizeTranslationProviderName(getenv("LIQUID2_TRANSLATION_PROVIDER", "")),
	}, nil
}

func startFeedScheduler(
	ctx context.Context,
	logger *slog.Logger,
	service *app.Service,
	queue jobruntime.Queue,
) (*feedrefresh.Scheduler, <-chan error) {
	scheduler := feedrefresh.NewScheduler(service, queue,
		feedrefresh.WithSchedulerLogger(logger),
	)
	done := make(chan error, 1)
	go func() {
		done <- runRuntime(logger, "feed scheduler", func() error {
			return scheduler.Run(ctx)
		})
	}()
	logger.Info("feed scheduler started",
		slog.String("component", "api"),
		slog.String("operation", "feed_scheduler_start"),
	)
	return scheduler, done
}

func stopFeedScheduler(logger *slog.Logger, scheduler *feedrefresh.Scheduler, done <-chan error) {
	if scheduler == nil {
		return
	}
	if err := scheduler.Close(); err != nil {
		logger.Error("close feed scheduler failed", slog.String("component", "api"), slog.Any("error", err))
	}
	<-done
}

func logRuntimeError(logger *slog.Logger, message string, err error) {
	if err != nil && !errors.Is(err, context.Canceled) {
		logger.Error(message, slog.String("component", "api"), slog.Any("error", err))
	}
}

func runRuntime(logger *slog.Logger, name string, run func() error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%s panicked", name)
			logger.Error(err.Error(),
				slog.String("component", "api"),
				slog.Any("panic", recovered),
				slog.String("stack", string(debug.Stack())),
			)
		}
	}()
	err = run()
	logRuntimeError(logger, name+" stopped with error", err)
	return err
}
