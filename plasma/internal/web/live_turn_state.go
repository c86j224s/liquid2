package web

import "sync"

const liveTurnSchema = "plasma-live-turn/v1"

type liveTurnSnapshot struct {
	Schema       string `json:"schema"`
	MissionID    string `json:"mission_id"`
	UserEventID  string `json:"user_event_id"`
	State        string `json:"state"`
	Phase        string `json:"phase,omitempty"`
	ToolCategory string `json:"tool_category,omitempty"`
	Status       string `json:"status,omitempty"`
	Preview      string `json:"preview,omitempty"`
	Sequence     int64  `json:"sequence"`
	Terminal     bool   `json:"terminal,omitempty"`
}

type liveTurnKey struct {
	missionID   string
	userEventID string
}

type liveTurnEntry struct {
	snapshot    liveTurnSnapshot
	subscribers map[chan liveTurnSnapshot]struct{}
}

// liveTurnStore owns process-local transient status for one ordinary or
// workflow agent turn.
//
// It deliberately stores only mission/user-event scoped snapshots and closes
// subscribers on terminal state. Durable conversation state remains in the
// ledger.
type liveTurnStore struct {
	mu    sync.Mutex
	turns map[liveTurnKey]*liveTurnEntry
}

func (store *liveTurnStore) start(missionID, userEventID string) {
	if missionID == "" || userEventID == "" {
		return
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if store.turns == nil {
		store.turns = map[liveTurnKey]*liveTurnEntry{}
	}
	key := liveTurnKey{missionID: missionID, userEventID: userEventID}
	store.turns[key] = &liveTurnEntry{
		snapshot: liveTurnSnapshot{
			Schema:      liveTurnSchema,
			MissionID:   missionID,
			UserEventID: userEventID,
			State:       "activity",
			Phase:       string(AgentPhaseThinking),
			Status:      liveStatusForPhase(AgentPhaseThinking),
			Sequence:    1,
		},
		subscribers: map[chan liveTurnSnapshot]struct{}{},
	}
}

func (store *liveTurnStore) applyObservation(missionID, userEventID string, event AgentObservation) {
	switch event.Type {
	case AgentObservationPhase:
		store.updateActivity(missionID, userEventID, event.Phase, "")
	case AgentObservationTool:
		store.updateActivity(missionID, userEventID, AgentPhaseThinking, string(event.ToolCategory))
	case AgentObservationAnswer:
		store.updateAnswer(missionID, userEventID, event.Text)
	}
}

func (store *liveTurnStore) updateActivity(missionID, userEventID string, phase AgentPhase, toolCategory string) {
	if phase == "" {
		phase = AgentPhaseThinking
	}
	store.withEntry(missionID, userEventID, func(entry *liveTurnEntry) (liveTurnSnapshot, bool, bool) {
		if entry.snapshot.Terminal {
			return liveTurnSnapshot{}, false, false
		}
		if entry.snapshot.Preview != "" {
			entry.snapshot.State = "answer"
		} else {
			entry.snapshot.State = "activity"
		}
		entry.snapshot.Phase = string(phase)
		entry.snapshot.ToolCategory = toolCategory
		entry.snapshot.Status = liveStatusForActivity(phase, AgentToolCategory(toolCategory))
		entry.snapshot.Sequence++
		return entry.snapshot, true, false
	})
}

func (store *liveTurnStore) updateAnswer(missionID, userEventID, preview string) {
	if preview == "" {
		return
	}
	store.withEntry(missionID, userEventID, func(entry *liveTurnEntry) (liveTurnSnapshot, bool, bool) {
		if entry.snapshot.Terminal {
			return liveTurnSnapshot{}, false, false
		}
		entry.snapshot.State = "answer"
		entry.snapshot.Status = ""
		entry.snapshot.ToolCategory = ""
		entry.snapshot.Preview = preview
		entry.snapshot.Sequence++
		return entry.snapshot, true, false
	})
}

func (store *liveTurnStore) finish(missionID, userEventID, state string) {
	if state == "" {
		state = "completed"
	}
	store.withEntry(missionID, userEventID, func(entry *liveTurnEntry) (liveTurnSnapshot, bool, bool) {
		entry.snapshot.State = state
		entry.snapshot.Status = ""
		entry.snapshot.Terminal = true
		entry.snapshot.Sequence++
		return entry.snapshot, true, true
	})
}

func liveStatusForActivity(phase AgentPhase, category AgentToolCategory) string {
	if category != "" {
		return liveStatusForToolCategory(category)
	}
	return liveStatusForPhase(phase)
}

func liveStatusForPhase(phase AgentPhase) string {
	switch phase {
	default:
		return "생각 중..."
	}
}

func liveStatusForToolCategory(category AgentToolCategory) string {
	switch category {
	case AgentToolCategoryWebSearch:
		return "웹에서 조사하는 중..."
	case AgentToolCategoryWebRead:
		return "웹 자료를 읽는 중..."
	case AgentToolCategoryMissionRead:
		return "미션 자료를 살펴보는 중..."
	case AgentToolCategorySourcePropose:
		return "자료를 제안하는 중..."
	case AgentToolCategoryOrganize:
		return "조사 내용을 정리하는 중..."
	case AgentToolCategoryValidate:
		return "답변을 확인하는 중..."
	default:
		return "작업 중..."
	}
}
