package finalstore

import (
	"context"

	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

// ReportingGateReader는 reporting canonical finalization replay 계약으로 gate 결과를 다시 읽는다.
type ReportingGateReader struct {
	Store reporting.LongFormFinalizationStore
}

// ReadFinalGate는 gate stage가 저장한 canonical artifact/event를 재조회한다.
func (reader ReportingGateReader) ReadFinalGate(ctx context.Context, input GateReadRequest) (GateRecord, error) {
	result, ok, err := reporting.LoadLongFormFinalization(ctx, reader.Store, input.Binding)
	if err != nil || !ok {
		return GateRecord{}, err
	}
	return GateRecord{
		Artifact: result.Artifact, Event: result.Event, Markdown: string(result.Artifact.Content),
		ReportSessionID: input.Binding.ProviderSessionID, CanonicalLineageCount: 1,
	}, nil
}
