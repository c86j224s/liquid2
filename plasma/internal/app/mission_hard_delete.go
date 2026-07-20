package app

import (
	"context"
	"fmt"
	"strings"
)

const (
	MissionHardDeleteBlockerNotArchived = "mission_not_archived"
	MissionHardDeleteBlockerActiveWork  = "active_work"
)

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

type MissionHardDeleteBlocker struct {
	ReasonCode string `json:"reason_code"`
	Message    string `json:"message"`
}

type MissionHardDeletePreview struct {
	MissionID       string                     `json:"mission_id"`
	Title           string                     `json:"title"`
	LifecycleState  string                     `json:"lifecycle_state"`
	Eligible        bool                       `json:"eligible"`
	BlockingReasons []MissionHardDeleteBlocker `json:"blocking_reasons"`
	Impact          MissionHardDeleteImpact    `json:"impact"`
}

type MissionHardDeleteRequest struct {
	MissionID        string
	ConfirmMissionID string
	Producer         Producer
}

type MissionHardDeleteResult struct {
	MissionID string                  `json:"mission_id"`
	Deleted   bool                    `json:"deleted"`
	Impact    MissionHardDeleteImpact `json:"impact"`
}

type MissionHardDeleteStore interface {
	PreviewMissionHardDelete(context.Context, string) (MissionHardDeleteImpact, error)
	HardDeleteMission(context.Context, string, func([]LedgerEvent) error) (MissionHardDeleteImpact, error)
}

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
