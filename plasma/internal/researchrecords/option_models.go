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
	OptionRecordSchemaVersion = "plasma.option_record.v1"
	OptionRecordObjectKind    = "option_record"
)

// OptionRecord는 비교/의사결정형 조사에서 검토하는 선택지다.
type OptionRecord struct {
	SchemaVersion      string    `json:"schema_version"`
	ObjectKind         string    `json:"object_kind"`
	OptionID           string    `json:"option_id"`
	MissionID          string    `json:"mission_id"`
	State              string    `json:"state"`
	Title              string    `json:"title"`
	Description        string    `json:"description"`
	Pros               []string  `json:"pros"`
	Cons               []string  `json:"cons"`
	SupportingClaimIDs []string  `json:"supporting_claim_ids"`
	RiskLevel          string    `json:"risk_level"`
	CreatedEventID     string    `json:"created_event_id"`
	CreatedAt          time.Time `json:"created_at"`
}

// CreateOptionRecordRequest는 option record 생성 입력이다.
type CreateOptionRecordRequest struct {
	OptionID           string
	MissionID          string
	State              string
	Title              string
	Description        string
	Pros               []string
	Cons               []string
	SupportingClaimIDs []string
	RiskLevel          string
	CreatedEventID     string
}

// OptionRequirements는 option 생성에 필요한 외부 조회 계약이다.
// callback은 claim ID 목록이 비어 있어도 lookup 단계에 도달하면 호출해야 한다.
type OptionRequirements struct {
	RequireMissionEvent func(context.Context, string, string) (ledger.Event, error)
	RequireClaimRecords func(context.Context, string, []string) error
}

// BuildOptionRecord validates and normalizes an option in the historical order.
// The mission event is intentionally looked up after early ID/title validation and
// before lifecycle validation; callers must provide both requirement callbacks.
func BuildOptionRecord(ctx context.Context, requirements OptionRequirements, req CreateOptionRecordRequest) (OptionRecord, error) {
	optionID := strings.TrimSpace(req.OptionID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("opt_", optionID); err != nil {
		return OptionRecord{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return OptionRecord{}, err
	}
	if strings.TrimSpace(req.Title) == "" {
		return OptionRecord{}, fmt.Errorf("%w: option title is required", producterror.ErrInvalidInput)
	}
	if _, err := requirements.RequireMissionEvent(ctx, missionID, req.CreatedEventID); err != nil {
		return OptionRecord{}, err
	}
	state, err := NormalizeProposedLifecycleState(req.State, "proposed")
	if err != nil {
		return OptionRecord{}, err
	}
	claimIDs, err := normalizeIDList("clm_", req.SupportingClaimIDs)
	if err != nil {
		return OptionRecord{}, err
	}
	if err := requirements.RequireClaimRecords(ctx, missionID, claimIDs); err != nil {
		return OptionRecord{}, err
	}
	riskLevel := strings.TrimSpace(req.RiskLevel)
	if riskLevel == "" {
		riskLevel = "unknown"
	}
	if !allowedRiskLevels[riskLevel] {
		return OptionRecord{}, fmt.Errorf("%w: unsupported option risk level", producterror.ErrInvalidInput)
	}
	return OptionRecord{
		SchemaVersion:      OptionRecordSchemaVersion,
		ObjectKind:         OptionRecordObjectKind,
		OptionID:           optionID,
		MissionID:          missionID,
		State:              state,
		Title:              strings.TrimSpace(req.Title),
		Description:        strings.TrimSpace(req.Description),
		Pros:               normalizeStringList(req.Pros),
		Cons:               normalizeStringList(req.Cons),
		SupportingClaimIDs: claimIDs,
		RiskLevel:          riskLevel,
		CreatedEventID:     strings.TrimSpace(req.CreatedEventID),
		CreatedAt:          time.Now().UTC(),
	}, nil
}

var allowedRiskLevels = map[string]bool{"low": true, "medium": true, "high": true, "unknown": true}
