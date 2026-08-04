package workflow

import (
	"context"
	"strings"
	"sync"
)

type runRegistry struct {
	mu   sync.Mutex
	runs map[string]ownedRun
}

type ownedRun struct {
	id     string
	cancel context.CancelFunc
}

func (registry *runRegistry) start(workflowRunID string, cancel context.CancelFunc, newID func(string) string) (string, bool) {
	workflowRunID = strings.TrimSpace(workflowRunID)
	if workflowRunID == "" || newID == nil {
		return "", false
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if registry.runs == nil {
		registry.runs = map[string]ownedRun{}
	}
	if _, ok := registry.runs[workflowRunID]; ok {
		return "", false
	}
	id := newID("run")
	registry.runs[workflowRunID] = ownedRun{id: id, cancel: cancel}
	return id, true
}

func (registry *runRegistry) finish(workflowRunID string, id string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if current, ok := registry.runs[workflowRunID]; ok && current.id == id {
		delete(registry.runs, workflowRunID)
	}
}

func (registry *runRegistry) cancel(workflowRunID string) bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	current, ok := registry.runs[strings.TrimSpace(workflowRunID)]
	if !ok || current.cancel == nil {
		return false
	}
	current.cancel()
	return true
}

func (registry *runRegistry) has(workflowRunID string) bool {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	_, ok := registry.runs[strings.TrimSpace(workflowRunID)]
	return ok
}

type missionLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func (locks *missionLocks) lock(missionID string) func() {
	locks.mu.Lock()
	if locks.locks == nil {
		locks.locks = map[string]*sync.Mutex{}
	}
	lock := locks.locks[missionID]
	if lock == nil {
		lock = &sync.Mutex{}
		locks.locks[missionID] = lock
	}
	locks.mu.Unlock()
	lock.Lock()
	return lock.Unlock
}
