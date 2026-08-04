package web

import (
	"context"
	"sync"
)

type missionTurnLocks struct {
	mu    sync.Mutex
	locks map[string]*sync.Mutex
}

func (locks *missionTurnLocks) lock(missionID string) func() {
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

type runningAgentTurns struct {
	mu    sync.Mutex
	turns map[string]runningAgentTurn
}

type runningAgentTurn struct {
	id       string
	executor string
	cancel   context.CancelFunc
}

func (turns *runningAgentTurns) start(missionID string, executor string, cancel context.CancelFunc) string {
	turns.mu.Lock()
	defer turns.mu.Unlock()
	if turns.turns == nil {
		turns.turns = map[string]runningAgentTurn{}
	}
	id := newID("run")
	turns.turns[missionID] = runningAgentTurn{id: id, executor: executor, cancel: cancel}
	return id
}

func (turns *runningAgentTurns) finish(missionID string, id string) {
	turns.mu.Lock()
	defer turns.mu.Unlock()
	if turns.turns == nil {
		return
	}
	if current, ok := turns.turns[missionID]; ok && current.id == id {
		delete(turns.turns, missionID)
	}
}

func (turns *runningAgentTurns) cancel(missionID string, executor string) bool {
	turns.mu.Lock()
	defer turns.mu.Unlock()
	if turns.turns == nil {
		return false
	}
	current, ok := turns.turns[missionID]
	if !ok {
		return false
	}
	if executor != "" && current.executor != executor {
		return false
	}
	current.cancel()
	return true
}

func (turns *runningAgentTurns) has(missionID string) bool {
	turns.mu.Lock()
	defer turns.mu.Unlock()
	if turns.turns == nil {
		return false
	}
	_, ok := turns.turns[missionID]
	return ok
}
