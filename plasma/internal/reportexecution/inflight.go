package reportexecution

import (
	"context"
	"strings"
	"sync"
)

type FailurePayloadProvider interface {
	FailurePayload() map[string]any
}

// InFlight는 프로세스 안에서 중복 report 실행을 막고 취소 함수를 보관하는 보조 lock이다.
type InFlight struct {
	mu    sync.Mutex
	runs  map[string]inFlightRun
	newID func(string) string
}

type inFlightRun struct {
	id             string
	pendingEventID string
	cancel         context.CancelFunc
}

// SetNewID는 테스트가 보고서 runner의 ID 생성 함수를 고정할 수 있게 하는 hook이다.
func (runs *InFlight) SetNewID(newID func(string) string) {
	runs.mu.Lock()
	defer runs.mu.Unlock()
	runs.newID = newID
}

// Start는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runs *InFlight) Start(missionID string, pendingEventID string, cancel context.CancelFunc) (string, bool) {
	runs.mu.Lock()
	defer runs.mu.Unlock()
	if runs.runs == nil {
		runs.runs = map[string]inFlightRun{}
	}
	if _, ok := runs.runs[missionID]; ok {
		return "", false
	}
	newID := runs.newID
	if newID == nil {
		newID = func(prefix string) string { return prefix + "_report" }
	}
	id := newID("run")
	runs.runs[missionID] = inFlightRun{id: id, pendingEventID: pendingEventID, cancel: cancel}
	return id, true
}

// Finish는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runs *InFlight) Finish(missionID string, id string) {
	runs.mu.Lock()
	defer runs.mu.Unlock()
	if runs.runs == nil {
		return
	}
	if current, ok := runs.runs[missionID]; ok && current.id == id {
		delete(runs.runs, missionID)
	}
}

// Owns는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runs *InFlight) Owns(missionID string, pendingEventID string) bool {
	runs.mu.Lock()
	defer runs.mu.Unlock()
	if runs.runs == nil {
		return false
	}
	current, ok := runs.runs[missionID]
	return ok && current.pendingEventID == pendingEventID
}

// PendingEventID는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runs *InFlight) PendingEventID(missionID string) (string, bool) {
	runs.mu.Lock()
	defer runs.mu.Unlock()
	if runs.runs == nil {
		return "", false
	}
	current, ok := runs.runs[missionID]
	if !ok {
		return "", false
	}
	return current.pendingEventID, true
}

// Cancel는 보고서 생성 파이프라인 실행 lifecycle을 다룬다. 중복 실행과 취소는 저장된 pending/terminal 이벤트 기준으로 판정한다.
func (runs *InFlight) Cancel(missionID string, pendingEventID string) bool {
	var cancel context.CancelFunc
	runs.mu.Lock()
	if runs.runs != nil {
		if current, ok := runs.runs[missionID]; ok && (strings.TrimSpace(pendingEventID) == "" || current.pendingEventID == pendingEventID) {
			cancel = current.cancel
			delete(runs.runs, missionID)
		}
	}
	runs.mu.Unlock()
	if cancel == nil {
		return false
	}
	cancel()
	return true
}
