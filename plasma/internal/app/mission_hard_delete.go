package app

import (
	"context"
	"fmt"
	"strings"
)

const (
	// MissionHardDeleteBlocker* 값은 hard delete가 왜 막혔는지 설명하는 안정적인 코드다.
	MissionHardDeleteBlockerNotArchived = "mission_not_archived"
	MissionHardDeleteBlockerActiveWork  = "active_work"
)

// MissionHardDeleteImpact는 hard delete가 삭제할 저장 객체 수와 artifact byte 수다.
type MissionHardDeleteImpact struct {
	LedgerEvents                int64 `json:"ledger_events"`
	RawArtifacts                int64 `json:"raw_artifacts"`
	RawArtifactBytes            int64 `json:"raw_artifact_bytes"`
	SourceSnapshots             int64 `json:"source_snapshots"`
	SourceSnapshotArtifactLinks int64 `json:"source_snapshot_artifact_links"`
	EvidenceRecords             int64 `json:"evidence_records"`
	ClaimRecords                int64 `json:"claim_records"`
	QuestionRecords             int64 `json:"question_records"`
	OptionRecords               int64 `json:"option_records"`
	ProposalBundles             int64 `json:"proposal_bundles"`
	Reports                     int64 `json:"reports"`
	ReportVersions              int64 `json:"report_versions"`
	ReportBlocks                int64 `json:"report_blocks"`
}

// MissionHardDeleteBlocker는 hard delete를 막는 제품 조건 하나다.
type MissionHardDeleteBlocker struct {
	ReasonCode string `json:"reason_code"`
	Message    string `json:"message"`
}

// MissionHardDeletePreview는 삭제 가능 여부와 예상 영향 범위를 사용자에게 보여 주는 view다.
type MissionHardDeletePreview struct {
	MissionID       string                     `json:"mission_id"`
	Title           string                     `json:"title"`
	LifecycleState  string                     `json:"lifecycle_state"`
	Eligible        bool                       `json:"eligible"`
	BlockingReasons []MissionHardDeleteBlocker `json:"blocking_reasons"`
	Impact          MissionHardDeleteImpact    `json:"impact"`
}

// MissionHardDeleteRequest는 사용자가 mission ID를 다시 확인한 hard delete 입력이다.
type MissionHardDeleteRequest struct {
	MissionID        string
	ConfirmMissionID string
	Producer         Producer
}

// MissionHardDeleteResult는 hard delete 실행 결과와 삭제된 범위를 반환한다.
type MissionHardDeleteResult struct {
	MissionID string                  `json:"mission_id"`
	Deleted   bool                    `json:"deleted"`
	Impact    MissionHardDeleteImpact `json:"impact"`
}

// MissionHardDeleteStore는 hard delete preview와 실제 삭제를 제공하는 저장소 port다.
type MissionHardDeleteStore interface {
	PreviewMissionHardDelete(context.Context, string) (MissionHardDeleteImpact, error)
	HardDeleteMission(context.Context, string, func([]LedgerEvent) error) (MissionHardDeleteImpact, error)
}

// PreviewMissionHardDelete는 archived 상태와 active work 여부를 확인해 삭제 가능 여부를
// 계산한다.
func (s *Service) PreviewMissionHardDelete(ctx context.Context, missionID string) (MissionHardDeletePreview, error) {
	trimmed := strings.TrimSpace(missionID)
	if err := validateID("mis_", trimmed); err != nil {
		return MissionHardDeletePreview{}, err
	}
	store, ok := s.store.(MissionHardDeleteStore)
	if !ok {
		return MissionHardDeletePreview{}, fmt.Errorf("%w: mission hard delete store is required", ErrInvalidInput)
	}
	projection, err := s.GetProjection(ctx, trimmed)
	if err != nil {
		return MissionHardDeletePreview{}, err
	}
	events, err := s.ListEvents(ctx, trimmed)
	if err != nil {
		return MissionHardDeletePreview{}, err
	}
	impact, err := store.PreviewMissionHardDelete(ctx, trimmed)
	if err != nil {
		return MissionHardDeletePreview{}, err
	}
	blockers := missionHardDeleteBlockers(projection, events)
	return MissionHardDeletePreview{
		MissionID:       trimmed,
		Title:           projection.Title,
		LifecycleState:  normalizeMissionLifecycleState(projection.LifecycleState),
		Eligible:        len(blockers) == 0,
		BlockingReasons: blockers,
		Impact:          impact,
	}, nil
}

// HardDeleteMission는 애플리케이션 서비스 계층의 명시적 상태 전이를 수행한다. 결과는 장부나 저장소 기록으로 확인한다.
func (s *Service) HardDeleteMission(ctx context.Context, req MissionHardDeleteRequest) (MissionHardDeleteResult, error) {
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("mis_", missionID); err != nil {
		return MissionHardDeleteResult{}, err
	}
	if strings.TrimSpace(req.ConfirmMissionID) != missionID {
		return MissionHardDeleteResult{}, fmt.Errorf("%w: confirmation mission_id does not match", ErrInvalidInput)
	}
	if req.Producer.Type != "user" {
		return MissionHardDeleteResult{}, fmt.Errorf("%w: mission hard delete requires a user producer", ErrInvalidInput)
	}
	store, ok := s.store.(MissionHardDeleteStore)
	if !ok {
		return MissionHardDeleteResult{}, fmt.Errorf("%w: mission hard delete store is required", ErrInvalidInput)
	}
	preview, err := s.PreviewMissionHardDelete(ctx, missionID)
	if err != nil {
		return MissionHardDeleteResult{}, err
	}
	if !preview.Eligible {
		return MissionHardDeleteResult{}, fmt.Errorf("%w: mission is not eligible for hard delete", ErrConflict)
	}
	impact, err := store.HardDeleteMission(ctx, missionID, func(events []LedgerEvent) error {
		if len(events) == 0 {
			return fmt.Errorf("%w: mission does not exist", ErrInvalidInput)
		}
		projection, err := BuildProjection(missionID, events)
		if err != nil {
			return err
		}
		if len(missionHardDeleteBlockers(projection, events)) > 0 {
			return fmt.Errorf("%w: mission is not eligible for hard delete", ErrConflict)
		}
		return nil
	})
	if err != nil {
		return MissionHardDeleteResult{}, err
	}
	return MissionHardDeleteResult{MissionID: missionID, Deleted: true, Impact: impact}, nil
}

func missionHardDeleteBlockers(projection MissionProjection, events []LedgerEvent) []MissionHardDeleteBlocker {
	blockers := []MissionHardDeleteBlocker{}
	if normalizeMissionLifecycleState(projection.LifecycleState) != MissionLifecycleArchived {
		blockers = append(blockers, MissionHardDeleteBlocker{
			ReasonCode: MissionHardDeleteBlockerNotArchived,
			Message:    "미션을 먼저 보관해야 완전 삭제할 수 있습니다.",
		})
	}
	if err := validateNoActiveAgentWork(events); err != nil {
		blockers = append(blockers, MissionHardDeleteBlocker{
			ReasonCode: MissionHardDeleteBlockerActiveWork,
			Message:    "진행 중인 작업이 있어 완전 삭제할 수 없습니다.",
		})
	}
	return blockers
}
