package finalstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// AdoptGate는 이미 canonical으로 저장된 long-form gate 결과를 durable reader에서 다시 읽어 채택한다.
//
// 이 함수는 runtime long-form finalization graph에 연결되어 있으며 terminal event를
// 새로 쓰지 않는다. gate stage가 이미 기록한 canonical 결과만 재조회·검증한다.
func (runner Runner) AdoptGate(ctx context.Context, input GateInput) (Output, error) {
	if runner.GateReader == nil {
		return Output{}, fmt.Errorf("%w: final gate durable reader is required", producterror.ErrInvalidInput)
	}
	record, err := runner.GateReader.ReadFinalGate(ctx, input.GateReadRequest)
	if err != nil {
		return Output{}, err
	}
	if record.CanonicalLineageCount != 1 {
		return Output{}, fmt.Errorf("%w: final gate canonical lineage is not unique", producterror.ErrConflict)
	}
	if strings.TrimSpace(record.ReportSessionID) == "" {
		return Output{}, fmt.Errorf("%w: final gate report session is required", producterror.ErrInvalidInput)
	}
	canonicalArtifactID := strings.TrimSpace(record.Artifact.ArtifactID)
	if err := validateStoredArtifact(record.Artifact, input.MissionID, canonicalArtifactID, record.Markdown); err != nil {
		return Output{}, err
	}
	if err := validateStoredEvent(record.Event, input.MissionID, input.PendingEventID, canonicalArtifactID, input.PlanEventID); err != nil {
		return Output{}, err
	}
	return Output{Artifact: record.Artifact, Event: record.Event, Markdown: record.Markdown, ReportSessionID: record.ReportSessionID}, nil
}
