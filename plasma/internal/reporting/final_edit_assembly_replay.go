package reporting

import (
	"fmt"
	artifactcontract "github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
	"strings"
)

func finalEditAssemblyCreatedEvent(events []ledger.Event, binding FinalEditStageBinding, request artifactcontract.CreateRequest, partArtifactIDs []string) (ledger.Event, bool, error) {
	key := FinalEditAssemblyIdempotencyKey(binding.PlanEventID, partArtifactIDs)
	var found ledger.Event
	count := 0
	for _, event := range events {
		if event.EventType != FinalEditAssemblyCreatedEventType || event.CorrelationID != key {
			continue
		}
		if err := validateFinalEditAssemblyCreatedEvent(events, event, binding, request, partArtifactIDs); err != nil {
			return ledger.Event{}, false, err
		}
		found, count = event, count+1
	}
	if count > 1 {
		return ledger.Event{}, false, fmt.Errorf("%w: multiple final assembly events match binding", producterror.ErrConflict)
	}
	return found, count == 1, nil
}

func validateFinalEditAssemblyCreatedEvent(events []ledger.Event, event ledger.Event, binding FinalEditStageBinding, request artifactcontract.CreateRequest, partArtifactIDs []string) error {
	acceptedPending, err := longFormPendingLineage(events, binding.PendingEventID)
	if err != nil {
		return err
	}
	payload := eventPayload(event)
	payloadParts, err := stringSlicePayload(payload["part_artifact_ids"])
	if err != nil {
		return err
	}
	sourceWordCount, ok := payloadIntStrict(payload, "source_word_count")
	expectedSourceWordCount := len(strings.Fields(string(request.Content)))
	if event.MissionID != binding.MissionID ||
		event.Producer != finalEditAssemblyProducer ||
		event.CausationEventID != binding.PlanEventID ||
		payloadString(payload, "kind") != FinalEditAssemblyKind ||
		payloadString(payload, "schema") != FinalEditAssemblySchema ||
		!acceptedPending[payloadString(payload, "pending_event_id")] ||
		payloadString(payload, "plan_event_id") != binding.PlanEventID ||
		payloadString(payload, "final_edit_pipeline") != binding.FinalEditPipeline ||
		payloadString(payload, "title") != binding.Title ||
		payloadString(payload, "artifact_id") != request.ArtifactID ||
		payloadString(payload, "filename") != binding.Filename ||
		payloadString(payload, "producer_id") != FinalEditAssemblyProducerID ||
		payloadString(payload, "artifact_sha256") != contentSHA256(request.Content) ||
		!ok || sourceWordCount != expectedSourceWordCount ||
		!equalStrings(payloadParts, partArtifactIDs) {
		return fmt.Errorf("%w: final assembly event differs from deterministic contract", producterror.ErrConflict)
	}
	return nil
}

func isFinalEditAssemblyPipeline(pipeline string) bool {
	return pipeline == FinalEditPipelineAssemblyWriterReaderStyleGateV2 || pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
}

func validateFinalEditAssemblyArtifact(artifact artifactcontract.Raw, request artifactcontract.CreateRequest) error {
	expectedSHA := contentSHA256(request.Content)
	if artifact.ArtifactID != request.ArtifactID ||
		artifact.MissionID != request.MissionID ||
		artifact.MediaType != request.MediaType ||
		artifact.Filename != request.Filename ||
		artifact.Producer != request.Producer ||
		artifact.SHA256 != expectedSHA {
		return fmt.Errorf("%w: existing final assembly artifact differs from deterministic contract", producterror.ErrConflict)
	}
	return nil
}
