package web

import (
	"sync"
	"time"

	confluenceconnector "github.com/c86j224s/liquid2/plasma/internal/connectors/confluence"
)

type confluenceOAuthStates struct {
	mu     sync.Mutex
	states map[string]confluenceOAuthState
}

type confluenceOAuthState struct {
	MissionID    string
	ConnectionID string
	DisplayName  string
	AccountID    string
	AccountName  string
	DiscoveryURL string
	Config       confluenceconnector.OAuthConfig
	ExpiresAt    time.Time
}

func (states *confluenceOAuthStates) put(state string, entry confluenceOAuthState) {
	states.mu.Lock()
	defer states.mu.Unlock()
	if states.states == nil {
		states.states = map[string]confluenceOAuthState{}
	}
	states.pruneLocked(time.Now().UTC())
	states.states[state] = entry
}

func (states *confluenceOAuthStates) consume(state string) (confluenceOAuthState, bool) {
	states.mu.Lock()
	defer states.mu.Unlock()
	if states.states == nil {
		return confluenceOAuthState{}, false
	}
	states.pruneLocked(time.Now().UTC())
	entry, ok := states.states[state]
	if ok {
		delete(states.states, state)
	}
	return entry, ok
}

func (states *confluenceOAuthStates) pruneLocked(now time.Time) {
	for state, entry := range states.states {
		if !entry.ExpiresAt.IsZero() && !entry.ExpiresAt.After(now) {
			delete(states.states, state)
		}
	}
}
