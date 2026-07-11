package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

var errRepositoryClosed = errors.New("repository closed")

type repositorySeed func(RepositoryTx)

type memoryRepositoryConfig struct {
	logger *slog.Logger
	now    func() int64
	seeds  []repositorySeed
}

type memoryRepository struct {
	logger    *slog.Logger
	ops       chan memoryRepositoryOp
	stop      chan chan struct{}
	closed    chan struct{}
	mu        sync.RWMutex
	closing   bool
	closeOnce sync.Once
}

type repositoryState struct {
	next     int64
	now      func() int64
	docs     map[string]*documentRecord
	versions map[string][]*DocumentVersion
	notes    map[string]map[string]*DocumentNote
	folders  map[string]*Folder
	tags     map[string]*Tag
	tagSlugs map[string]string
	feeds    map[string]*Feed
	feedURLs map[string]string
	items    map[string]map[string]*FeedItem
	jobs     map[string]*Job
	settings AppSettings
}

type memoryRepositoryOp struct {
	run  func(*repositoryState) error
	done chan error
}

type memoryAbort struct {
	err error
}

func newMemoryRepository(config memoryRepositoryConfig) *memoryRepository {
	state := newEmptyRepositoryState(config.now)
	tx := memoryTx{memoryReader: memoryReader{state: state}}
	for _, seed := range config.seeds {
		seed(tx)
	}
	repo := &memoryRepository{
		logger: config.logger.With("repository", "memory"),
		ops:    make(chan memoryRepositoryOp),
		stop:   make(chan chan struct{}),
		closed: make(chan struct{}),
	}
	go repo.run(state)
	return repo
}

func newEmptyRepositoryState(now func() int64) *repositoryState {
	return &repositoryState{
		now:      now,
		docs:     map[string]*documentRecord{},
		versions: map[string][]*DocumentVersion{},
		notes:    map[string]map[string]*DocumentNote{},
		folders:  map[string]*Folder{},
		tags:     map[string]*Tag{},
		tagSlugs: map[string]string{},
		feeds:    map[string]*Feed{},
		feedURLs: map[string]string{},
		items:    map[string]map[string]*FeedItem{},
		jobs:     map[string]*Job{},
		settings: cloneAppSettings(DefaultAppSettings()),
	}
}

func (repo *memoryRepository) View(ctx context.Context, fn func(RepositoryReader) error) error {
	return repo.do(ctx, func(state *repositoryState) error {
		return fn(memoryReader{state: state})
	})
}

func (repo *memoryRepository) Update(ctx context.Context, fn func(RepositoryTx) error) error {
	return repo.do(ctx, func(state *repositoryState) error {
		pending := cloneRepositoryState(state)
		if err := fn(memoryTx{memoryReader: memoryReader{state: pending}}); err != nil {
			return err
		}
		*state = *pending
		return nil
	})
}

func (repo *memoryRepository) do(ctx context.Context, fn func(*repositoryState) error) error {
	op := memoryRepositoryOp{run: fn, done: make(chan error, 1)}
	repo.mu.RLock()
	if repo.closing {
		repo.mu.RUnlock()
		return errRepositoryClosed
	}
	select {
	case repo.ops <- op:
		repo.mu.RUnlock()
	case <-ctx.Done():
		repo.mu.RUnlock()
		return ctx.Err()
	}
	return <-op.done
}

func (repo *memoryRepository) run(state *repositoryState) {
	for {
		select {
		case op := <-repo.ops:
			err := runMemoryRepositoryOp(state, op.run)
			if err != nil && !expectedRepositoryError(err) {
				repo.logger.Error("memory repository operation failed",
					slog.String("operation", "memory_repository"),
					slog.Any("error", err),
				)
			}
			op.done <- err
		case done := <-repo.stop:
			close(repo.closed)
			close(done)
			return
		}
	}
}

func (repo *memoryRepository) Close() error {
	var done chan struct{}
	repo.closeOnce.Do(func() {
		repo.mu.Lock()
		repo.closing = true
		repo.mu.Unlock()
		done = make(chan struct{})
		repo.stop <- done
		<-done
	})
	if done == nil {
		<-repo.closed
	}
	return nil
}

func runMemoryRepositoryOp(state *repositoryState, run func(*repositoryState) error) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			if abort, ok := recovered.(memoryAbort); ok {
				err = abort.err
				return
			}
			err = fmt.Errorf("memory repository operation panic: %v", recovered)
		}
	}()
	return run(state)
}

func expectedRepositoryError(err error) bool {
	return errors.Is(err, ErrConflict) ||
		errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrValidation) ||
		errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded)
}

func formatSeq(seq int64) string {
	return strconv.FormatInt(seq, 10)
}

func (state *repositoryState) nextID(prefix string) string {
	state.next++
	return prefix + "_" + time.UnixMilli(state.now()).Format("20060102150405") + "_" + formatSeq(state.next)
}
