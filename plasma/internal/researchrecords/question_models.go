package researchrecords

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

const (
	QuestionRecordSchemaVersion = "plasma.question_record.v1"
	QuestionRecordObjectKind    = "question_record"
)

// QuestionRecord는 조사 중 남겨 둔 질문 또는 미해결 쟁점이다.
type QuestionRecord struct {
	SchemaVersion      string    `json:"schema_version"`
	ObjectKind         string    `json:"object_kind"`
	QuestionID         string    `json:"question_id"`
	MissionID          string    `json:"mission_id"`
	State              string    `json:"state"`
	Text               string    `json:"text"`
	Priority           string    `json:"priority"`
	Blocking           bool      `json:"blocking"`
	RelatedEvidenceIDs []string  `json:"related_evidence_ids"`
	RelatedClaimIDs    []string  `json:"related_claim_ids"`
	Resolution         string    `json:"resolution,omitempty"`
	CreatedEventID     string    `json:"created_event_id"`
	CreatedAt          time.Time `json:"created_at"`
}

// CreateQuestionRecordRequest는 question record 생성 입력이다.
type CreateQuestionRecordRequest struct {
	QuestionID         string
	MissionID          string
	State              string
	Text               string
	Priority           string
	Blocking           bool
	RelatedEvidenceIDs []string
	RelatedClaimIDs    []string
	Resolution         string
	CreatedEventID     string
}

// QuestionRequirements는 question 생성에 필요한 관련 record 조회 계약이다.
// 두 callback은 ID 목록이 비어 있어도 lookup 단계에 도달하면 호출해야 한다.
type QuestionRequirements struct {
	RequireEvidenceRecords func(context.Context, string, []string) error
	RequireClaimRecords    func(context.Context, string, []string) error
}

// BuildQuestionRecord validates and normalizes a question in the historical order.
// Callers must provide both requirement callbacks; the builder deliberately does not
// add nil guards because callback invocation is part of the lookup contract.
func BuildQuestionRecord(ctx context.Context, requirements QuestionRequirements, req CreateQuestionRecordRequest, createdEvent ledger.Event) (QuestionRecord, error) {
	questionID := strings.TrimSpace(req.QuestionID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("qst_", questionID); err != nil {
		return QuestionRecord{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return QuestionRecord{}, err
	}
	if strings.TrimSpace(req.Text) == "" {
		return QuestionRecord{}, fmt.Errorf("%w: question text is required", producterror.ErrInvalidInput)
	}
	if createdEvent.MissionID != missionID || strings.TrimSpace(createdEvent.EventID) != strings.TrimSpace(req.CreatedEventID) {
		return QuestionRecord{}, fmt.Errorf("%w: question creation event mismatch", producterror.ErrInvalidInput)
	}
	state, err := normalizeQuestionCreateState(req.State)
	if err != nil {
		return QuestionRecord{}, err
	}
	if state == "answered" && strings.TrimSpace(req.Resolution) == "" {
		return QuestionRecord{}, fmt.Errorf("%w: answered question requires a resolution", producterror.ErrInvalidInput)
	}
	priority := strings.TrimSpace(req.Priority)
	if priority == "" {
		priority = "medium"
	}
	if !allowedPriorities[priority] {
		return QuestionRecord{}, fmt.Errorf("%w: unsupported question priority", producterror.ErrInvalidInput)
	}
	evidenceIDs, err := normalizeIDList("evd_", req.RelatedEvidenceIDs)
	if err != nil {
		return QuestionRecord{}, err
	}
	if err := requirements.RequireEvidenceRecords(ctx, missionID, evidenceIDs); err != nil {
		return QuestionRecord{}, err
	}
	claimIDs, err := normalizeIDList("clm_", req.RelatedClaimIDs)
	if err != nil {
		return QuestionRecord{}, err
	}
	if err := requirements.RequireClaimRecords(ctx, missionID, claimIDs); err != nil {
		return QuestionRecord{}, err
	}
	return QuestionRecord{
		SchemaVersion:      QuestionRecordSchemaVersion,
		ObjectKind:         QuestionRecordObjectKind,
		QuestionID:         questionID,
		MissionID:          missionID,
		State:              state,
		Text:               strings.TrimSpace(req.Text),
		Priority:           priority,
		Blocking:           req.Blocking,
		RelatedEvidenceIDs: evidenceIDs,
		RelatedClaimIDs:    claimIDs,
		Resolution:         strings.TrimSpace(req.Resolution),
		CreatedEventID:     strings.TrimSpace(req.CreatedEventID),
		CreatedAt:          time.Now().UTC(),
	}, nil
}

func normalizeQuestionState(state string) (string, error) {
	trimmed := strings.TrimSpace(state)
	if trimmed == "" {
		trimmed = "open"
	}
	if !allowedQuestionStates[trimmed] {
		return "", fmt.Errorf("%w: unsupported question state", producterror.ErrInvalidInput)
	}
	return trimmed, nil
}

func normalizeQuestionCreateState(state string) (string, error) {
	trimmed, err := normalizeQuestionState(state)
	if err != nil {
		return "", err
	}
	if !allowedQuestionCreateStates[trimmed] {
		return "", fmt.Errorf("%w: terminal question state requires a transition event", producterror.ErrInvalidInput)
	}
	return trimmed, nil
}

var allowedQuestionStates = map[string]bool{
	"open": true, "in_progress": true, "answered": true,
	"rejected": true, "superseded": true, "reopened": true,
}

var allowedQuestionCreateStates = map[string]bool{"open": true, "in_progress": true}
var allowedPriorities = map[string]bool{"low": true, "medium": true, "high": true}
