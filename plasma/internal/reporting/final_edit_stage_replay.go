package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type FinalEditStageStartContract struct {
	FinalBinding LongFormFinalizeBinding
	Stage        string
}

type FinalEditStageStartResult struct {
	Binding        FinalEditStageBinding
	SourceArtifact app.RawArtifact
	Event          app.LedgerEvent
}

type FinalEditStageProgress struct {
	Binding        FinalEditStageBinding
	SourceArtifact app.RawArtifact
	StartEvent     app.LedgerEvent
	Submission     *FinalEditStageResult
}

func LoadFinalEditStageProgress(ctx context.Context, store FinalEditStageStore, contract FinalEditStageStartContract) (FinalEditStageProgress, bool, error) {
	finalBinding := normalizeLongFormFinalizeBinding(contract.FinalBinding)
	if err := validateLongFormFinalizeBinding(finalBinding); err != nil {
		return FinalEditStageProgress{}, false, err
	}
	stage := normalizeFinalEditStageBinding(FinalEditStageBinding{Stage: contract.Stage}).Stage
	if finalEditStartedEventType(stage) == "" {
		return FinalEditStageProgress{}, false, fmt.Errorf("%w: unsupported final edit stage", app.ErrInvalidInput)
	}
	events, err := store.ListEvents(ctx, finalBinding.MissionID)
	if err != nil {
		return FinalEditStageProgress{}, false, err
	}
	acceptedPending, err := longFormPendingLineage(events, finalBinding.PendingEventID)
	if err != nil {
		return FinalEditStageProgress{}, false, err
	}
	plan, ok, err := longFormPlanPipeline(events, finalBinding.PendingEventID, finalBinding.PlanEventID)
	if err != nil {
		return FinalEditStageProgress{}, false, err
	}
	if !ok || !isSupportedFinalEditPipeline(plan.Pipeline) {
		return FinalEditStageProgress{}, false, fmt.Errorf("%w: final edit plan is not active", app.ErrConflict)
	}
	finalBinding.PostReportHumanize = plan.PostReportHumanize
	var progress FinalEditStageProgress
	count := 0
	for _, event := range events {
		if event.EventType != finalEditStartedEventType(stage) ||
			event.MissionID != finalBinding.MissionID ||
			!acceptedPending[payloadString(eventPayload(event), "pending_event_id")] {
			continue
		}
		binding, ok := finalEditStageBindingFromStartEventForPipeline(event, plan.Pipeline)
		if !ok {
			return FinalEditStageProgress{}, false, fmt.Errorf("%w: stored final edit start is invalid", app.ErrConflict)
		}
		if binding.PendingEventID != finalBinding.PendingEventID ||
			binding.PlanEventID != finalBinding.PlanEventID ||
			binding.Stage != stage {
			continue
		}
		submitted, submittedOK, err := finalEditStageSubmittedEvent(events, binding)
		if err != nil {
			return FinalEditStageProgress{}, false, err
		}
		if err := validateFinalEditStageLineage(ctx, store, events, binding, submittedOK); err != nil {
			return FinalEditStageProgress{}, false, err
		}
		if err := finalEditStageStartMatchesFinalBinding(binding, finalBinding, plan); err != nil {
			return FinalEditStageProgress{}, false, err
		}
		source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
		if err != nil {
			return FinalEditStageProgress{}, false, err
		}
		if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.Filename != binding.Filename || source.SHA256 != contentSHA256(source.Content) {
			return FinalEditStageProgress{}, false, fmt.Errorf("%w: final edit start source artifact is invalid", app.ErrConflict)
		}
		progress = FinalEditStageProgress{Binding: binding, SourceArtifact: source, StartEvent: event}
		if submittedOK {
			result, err := finalEditStageResultFromEvent(ctx, store, progress.Binding, submitted, true)
			if err != nil {
				return FinalEditStageProgress{}, false, err
			}
			progress.Submission = &result
		}
		count++
	}
	if count > 1 {
		return FinalEditStageProgress{}, false, fmt.Errorf("%w: multiple final edit starts match current pending", app.ErrConflict)
	}
	if count == 0 {
		return FinalEditStageProgress{}, false, nil
	}
	return progress, true, nil
}

func finalEditStageStartMatchesFinalBinding(binding FinalEditStageBinding, finalBinding LongFormFinalizeBinding, plan FinalEditPipelinePlanState) error {
	if binding.MissionID != finalBinding.MissionID ||
		binding.PendingEventID != finalBinding.PendingEventID ||
		binding.PlanEventID != finalBinding.PlanEventID ||
		binding.Filename != finalBinding.Filename ||
		binding.Title != finalBinding.Title ||
		binding.AgentExecutor != finalBinding.AgentExecutor ||
		binding.AgentModel != finalBinding.AgentModel ||
		binding.AgentReasoningEffort != finalBinding.AgentReasoningEffort ||
		binding.AgentSelectionSource != finalBinding.AgentSelectionSource ||
		binding.MCPMode != finalBinding.MCPMode ||
		binding.RigorLevel != finalBinding.RigorLevel ||
		binding.RigorLabel != finalBinding.RigorLabel ||
		binding.ReportSessionPolicy != finalBinding.ReportSessionPolicy ||
		binding.ReportSessionPolicySelection != finalBinding.ReportSessionPolicySelection ||
		binding.PostReportHumanize != plan.PostReportHumanize ||
		binding.GenerationGuidanceProfile != finalBinding.GenerationGuidanceProfile ||
		binding.GenerationGuidanceSHA256 != finalBinding.GenerationGuidanceSHA256 ||
		binding.SessionChainKind != finalBinding.SessionChainKind ||
		binding.PreReportResearchSessionID != finalBinding.PreReportResearchSessionID ||
		binding.ReportPlanSessionID != finalBinding.ReportPlanSessionID {
		return fmt.Errorf("%w: final edit start binding differs from final binding", app.ErrConflict)
	}
	return nil
}

func loadLongFormCanonicalResult(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding, event app.LedgerEvent) (LongFormFinalizeResult, error) {
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	artifact, err := loadCanonicalArtifactForBinding(ctx, store, events, binding, event)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	return LongFormFinalizeResult{Artifact: artifact, Event: event, Replay: true}, nil
}

func replayLongFormFinalizeForRequest(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding, event app.LedgerEvent, req LongFormFinalizeRequest) (LongFormFinalizeResult, error) {
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if err := validateLongFormCanonicalReplayRequest(ctx, store, events, binding, event, req); err != nil {
		return LongFormFinalizeResult{}, err
	}
	artifact, err := loadCanonicalArtifactForBinding(ctx, store, events, binding, event)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	expected, err := longFormMarkdownForRequest(ctx, store, binding, req)
	if err != nil {
		return LongFormFinalizeResult{}, err
	}
	if artifact.SHA256 != contentSHA256([]byte(expected)) || string(artifact.Content) != expected {
		return LongFormFinalizeResult{}, fmt.Errorf("%w: canonical long-form finalization content differs", app.ErrConflict)
	}
	return LongFormFinalizeResult{Artifact: artifact, Event: event, Replay: true}, nil
}

func finalEditStageStartedEvent(events []app.LedgerEvent, binding FinalEditStageBinding) (app.LedgerEvent, bool, error) {
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	var found app.LedgerEvent
	count := 0
	for _, event := range events {
		if event.EventType != finalEditStartedEventType(binding.Stage) || event.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !finalEditStartedEventMatchesPipeline(event, binding, plan.Pipeline) {
			return app.LedgerEvent{}, false, fmt.Errorf("%w: final edit stage start binding differs", app.ErrConflict)
		}
		found, count = event, count+1
	}
	if count > 1 {
		return app.LedgerEvent{}, false, fmt.Errorf("%w: multiple final edit stage starts match binding", app.ErrConflict)
	}
	return found, count == 1, nil
}

func finalEditStageSubmittedEvent(events []app.LedgerEvent, binding FinalEditStageBinding) (app.LedgerEvent, bool, error) {
	plan, err := finalEditStagePlanForBinding(events, binding)
	if err != nil {
		return app.LedgerEvent{}, false, err
	}
	binding = finalEditStageBindingForPlan(binding, plan)
	var found app.LedgerEvent
	count := 0
	for _, event := range events {
		if event.EventType != finalEditSubmittedEventType(binding.Stage) || event.CorrelationID != binding.IdempotencyKey {
			continue
		}
		if !finalEditSubmittedEventMatchesPipeline(event, binding, plan.Pipeline) {
			return app.LedgerEvent{}, false, fmt.Errorf("%w: final edit stage submission binding differs", app.ErrConflict)
		}
		found, count = event, count+1
	}
	if count > 1 {
		return app.LedgerEvent{}, false, fmt.Errorf("%w: multiple final edit stage submissions match binding", app.ErrConflict)
	}
	return found, count == 1, nil
}

func finalEditStageResultFromEvent(ctx context.Context, store LongFormFinalizationStore, binding FinalEditStageBinding, event app.LedgerEvent, replay bool) (FinalEditStageResult, error) {
	payload := eventPayload(event)
	if !finalEditSubmittedEventMatchesPipeline(event, binding, finalEditStagePayloadPipeline(binding)) {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage submission binding differs", app.ErrConflict)
	}
	artifactID := payloadString(payload, "artifact_id")
	changed, ok := payloadBoolStrict(payload, "changed")
	if artifactID == "" || !ok {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage submission artifact fields are invalid", app.ErrConflict)
	}
	operationCount, ok := payloadIntStrict(payload, "operation_count")
	if !ok || operationCount < 0 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage operation count is invalid", app.ErrConflict)
	}
	diagnoses, diagnosesPresent, err := decodeFinalEditStyleOperationDiagnosesForStage(payload, binding.Stage, operationCount)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if binding.Stage == FinalEditStageStyle && diagnosesPresent && !changed && (operationCount != 0 || len(diagnoses) != 0) {
		return FinalEditStageResult{}, fmt.Errorf("%w: unchanged style edit replay must have empty operation diagnoses", app.ErrConflict)
	}
	if binding.Stage == FinalEditStageStyle && diagnosesPresent && changed && operationCount <= 0 {
		return FinalEditStageResult{}, fmt.Errorf("%w: changed style edit replay requires operation diagnoses", app.ErrConflict)
	}
	pipeline := finalEditStagePayloadPipeline(binding)
	findings, err := decodeStoredFinalEditGateFindingsPayloadForStage(payload["gate_findings"], pipeline, binding.Stage)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	semanticReview, err := decodeFinalEditSemanticAcceptancePayload(payload)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if err := validateStoredFinalEditGateFindingEvidence(ctx, store, binding.MissionID, findings); err != nil {
		return FinalEditStageResult{}, err
	}
	source, err := store.GetRawArtifact(ctx, binding.SourceArtifactID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if source.MissionID != binding.MissionID || source.MediaType != "text/markdown; charset=utf-8" || source.Filename != binding.Filename || source.SHA256 != contentSHA256(source.Content) {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage source artifact differs from binding", app.ErrConflict)
	}
	if payloadString(payload, "source_sha256") != source.SHA256 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage source sha differs", app.ErrConflict)
	}
	artifact, err := store.GetRawArtifact(ctx, artifactID)
	if err != nil {
		return FinalEditStageResult{}, err
	}
	if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" || artifact.Filename != binding.Filename || artifact.SHA256 != contentSHA256(artifact.Content) {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage artifact differs from binding", app.ErrConflict)
	}
	if !changed && artifact.ArtifactID != binding.SourceArtifactID {
		return FinalEditStageResult{}, fmt.Errorf("%w: no-op final edit stage must reuse source artifact", app.ErrConflict)
	}
	if changed && binding.Stage == FinalEditStageStyleSemanticValidation && artifact.ArtifactID == binding.SourceArtifactID {
		return FinalEditStageResult{}, fmt.Errorf("%w: edited final edit stage artifact differs from binding", app.ErrConflict)
	}
	if changed && binding.Stage != FinalEditStageStyleSemanticValidation && (artifact.ArtifactID != binding.EditedArtifactID || artifact.Producer != (app.Producer{Type: "agent_session", ID: binding.ProviderSessionID})) {
		return FinalEditStageResult{}, fmt.Errorf("%w: edited final edit stage artifact differs from binding", app.ErrConflict)
	}
	if changed && (artifact.SHA256 == source.SHA256 || string(artifact.Content) == string(source.Content)) {
		return FinalEditStageResult{}, fmt.Errorf("%w: changed final edit stage artifact must differ from source", app.ErrConflict)
	}
	if payloadString(payload, "artifact_sha256") != artifact.SHA256 {
		return FinalEditStageResult{}, fmt.Errorf("%w: final edit stage artifact sha differs", app.ErrConflict)
	}
	if pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		switch binding.Stage {
		case FinalEditStageStyleSemanticValidation:
			stageStore, ok := store.(FinalEditStageStore)
			if !ok {
				return FinalEditStageResult{}, fmt.Errorf("%w: semantic validation replay requires final edit stage store", app.ErrConflict)
			}
			if err := validateV3StyleSemanticValidationReplay(ctx, stageStore, binding, source, artifact, operationCount, changed, findings, semanticReview); err != nil {
				return FinalEditStageResult{}, err
			}
		case FinalEditStageEvidenceGate:
			if err := validateV3EvidenceGateReplay(source, artifact, operationCount, changed, findings, semanticReview); err != nil {
				return FinalEditStageResult{}, err
			}
		}
	}
	if semanticReview.Count > 0 || len(semanticReview.Records) > 0 || strings.TrimSpace(semanticReview.Digest) != "" {
		stageStore, ok := store.(FinalEditStageStore)
		if !ok {
			return FinalEditStageResult{}, fmt.Errorf("%w: semantic acceptance replay requires final edit stage store", app.ErrConflict)
		}
		if err := validateStoredFinalEditSemanticAcceptanceAgainstLineage(ctx, stageStore, binding, string(artifact.Content), semanticReview); err != nil {
			return FinalEditStageResult{}, err
		}
	}
	return FinalEditStageResult{
		Binding:                        finalEditStageBindingForCompare(binding),
		Artifact:                       artifact,
		Event:                          event,
		Replay:                         replay,
		OperationCount:                 operationCount,
		Changed:                        changed,
		GateFindings:                   findings,
		SemanticReview:                 semanticReview,
		StyleOperationDiagnoses:        diagnoses,
		StyleOperationDiagnosesPresent: diagnosesPresent,
	}, nil
}

func decodeFinalEditStyleOperationDiagnosesForStage(payload map[string]any, stage string, operationCount int) ([]FinalEditStyleOperationDiagnosis, bool, error) {
	value, present := payload[FinalEditStyleOperationDiagnosesField]
	_, versionPresent := payload[FinalEditStyleOperationDiagnosesVersionField]
	if stage != FinalEditStageStyle {
		if present || versionPresent {
			return nil, false, fmt.Errorf("%w: style operation diagnoses are only valid for style edit", app.ErrConflict)
		}
		return nil, false, nil
	}
	if versionPresent {
		version, ok := payloadIntStrict(payload, FinalEditStyleOperationDiagnosesVersionField)
		if !ok || version != FinalEditStyleOperationDiagnosesVersion {
			return nil, false, fmt.Errorf("%w: style operation diagnoses version is invalid", app.ErrConflict)
		}
		if !present {
			return nil, false, fmt.Errorf("%w: style operation diagnoses payload is invalid", app.ErrConflict)
		}
		return decodeFinalEditStyleOperationDiagnosesPayload(value, operationCount, true)
	}
	if !present {
		return nil, false, nil
	}
	return decodeLegacyFinalEditStyleOperationDiagnosesPayload(value, operationCount)
}

func validateV3EvidenceGateReplay(source app.RawArtifact, artifact app.RawArtifact, operationCount int, changed bool, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) error {
	if operationCount != 0 || changed {
		return fmt.Errorf("%w: evidence gate replay must be an unchanged zero-operation submission", app.ErrConflict)
	}
	if artifact.ArtifactID != source.ArtifactID || artifact.SHA256 != source.SHA256 || string(artifact.Content) != string(source.Content) {
		return fmt.Errorf("%w: evidence gate replay must reuse the exact source artifact", app.ErrConflict)
	}
	if semanticReview.Count != 0 || len(semanticReview.Records) != 0 || strings.TrimSpace(semanticReview.Digest) != "" {
		return fmt.Errorf("%w: evidence gate replay cannot carry semantic acceptance", app.ErrConflict)
	}
	return validateFinalEditEvidenceGateFindingStatementsInSource(string(source.Content), findings)
}

func validateV3StyleSemanticValidationReplay(ctx context.Context, store FinalEditStageStore, binding FinalEditStageBinding, source app.RawArtifact, artifact app.RawArtifact, operationCount int, changed bool, findings []StoredFinalEditGateFinding, semanticReview FinalEditSemanticAttestation) error {
	if operationCount != 0 || len(findings) != 0 {
		return fmt.Errorf("%w: style semantic validation replay must be a zero-operation verdict submission", app.ErrConflict)
	}
	records := semanticReview.Records
	for _, record := range records {
		if record.Verdict != FinalEditSemanticAcceptedEquivalent && record.Verdict != FinalEditSemanticRejectedRevertToReader {
			return fmt.Errorf("%w: style semantic validation replay verdict is invalid", app.ErrConflict)
		}
	}
	comparison, err := FinalEditSemanticComparison(ctx, store, binding, "")
	if err != nil {
		return err
	}
	if len(comparison) == 0 {
		if changed || artifact.ArtifactID != source.ArtifactID || artifact.SHA256 != source.SHA256 || string(artifact.Content) != string(source.Content) {
			return fmt.Errorf("%w: unchanged style semantic validation replay must reuse source artifact", app.ErrConflict)
		}
		if semanticReview.Count != 0 || len(records) != 0 || strings.TrimSpace(semanticReview.Digest) != "" {
			return fmt.Errorf("%w: unchanged style semantic validation replay cannot carry semantic acceptance", app.ErrConflict)
		}
		return nil
	}
	if semanticReview.Count != len(comparison) || len(records) != len(comparison) || strings.TrimSpace(semanticReview.Digest) == "" {
		return fmt.Errorf("%w: style semantic validation replay requires complete semantic acceptance", app.ErrConflict)
	}
	reviews := make([]FinalEditSemanticAcceptance, 0, len(records))
	for _, record := range records {
		reviews = append(reviews, FinalEditSemanticAcceptance{ParagraphOrdinal: record.ParagraphOrdinal, Verdict: record.Verdict})
	}
	resolved, attestation, err := BuildFinalEditStyleSemanticValidation(ctx, store, binding, reviews)
	if err != nil {
		return err
	}
	events, err := store.ListEvents(ctx, binding.MissionID)
	if err != nil {
		return err
	}
	style, ok, err := finalEditStyleSubmissionForGate(ctx, store, events, binding)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: style semantic validation replay requires style lineage", app.ErrConflict)
	}
	if string(artifact.Content) != resolved || artifact.SHA256 != contentSHA256([]byte(resolved)) {
		return fmt.Errorf("%w: style semantic validation replay artifact differs from deterministic resolution", app.ErrConflict)
	}
	if !equalStoredFinalEditSemanticAcceptance(attestation.Records, records) || attestation.Digest != semanticReview.Digest || attestation.Count != semanticReview.Count {
		return fmt.Errorf("%w: style semantic validation replay semantic acceptance differs", app.ErrConflict)
	}
	if changed != (artifact.ArtifactID != source.ArtifactID) {
		return fmt.Errorf("%w: style semantic validation replay changed flag differs", app.ErrConflict)
	}
	switch {
	case resolved == string(source.Content):
		if changed || artifact.ArtifactID != source.ArtifactID {
			return fmt.Errorf("%w: style semantic validation replay must reuse source artifact", app.ErrConflict)
		}
	case resolved == string(style.SourceArtifact.Content):
		if !changed || artifact.ArtifactID != style.SourceArtifact.ArtifactID {
			return fmt.Errorf("%w: style semantic validation replay must reuse reader artifact", app.ErrConflict)
		}
	case !changed || artifact.ArtifactID != binding.EditedArtifactID:
		return fmt.Errorf("%w: style semantic validation replay artifact identity differs", app.ErrConflict)
	case artifact.Producer != (app.Producer{Type: "agent_session", ID: binding.ProviderSessionID}):
		return fmt.Errorf("%w: style semantic validation replay artifact producer differs", app.ErrConflict)
	}
	return nil
}

func decodeFinalEditSemanticAcceptancePayload(payload map[string]any) (FinalEditSemanticAttestation, error) {
	records, err := decodeStoredFinalEditSemanticAcceptancePayload(payload["semantic_acceptance"])
	if err != nil {
		return FinalEditSemanticAttestation{}, err
	}
	count, ok := payloadIntStrict(payload, "semantic_acceptance_count")
	if !ok {
		if len(records) == 0 && payload["semantic_acceptance_count"] == nil && payloadString(payload, "semantic_acceptance_digest") == "" {
			return FinalEditSemanticAttestation{}, nil
		}
		return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance count is invalid", app.ErrConflict)
	}
	digest := payloadString(payload, "semantic_acceptance_digest")
	expected, err := finalEditSemanticAcceptanceDigest(records)
	if err != nil {
		return FinalEditSemanticAttestation{}, err
	}
	if count != len(records) || digest != expected {
		return FinalEditSemanticAttestation{}, fmt.Errorf("%w: semantic acceptance digest differs", app.ErrConflict)
	}
	return FinalEditSemanticAttestation{Records: records, Digest: digest, Count: count}, nil
}

func decodeStoredFinalEditSemanticAcceptancePayload(value any) ([]StoredFinalEditSemanticAcceptance, error) {
	if value == nil {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: semantic acceptance payload is invalid", app.ErrConflict)
	}
	var records []StoredFinalEditSemanticAcceptance
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&records); err != nil {
		return nil, fmt.Errorf("%w: semantic acceptance payload is invalid", app.ErrConflict)
	}
	if decoder.More() {
		return nil, fmt.Errorf("%w: semantic acceptance payload is invalid", app.ErrConflict)
	}
	out := make([]StoredFinalEditSemanticAcceptance, 0, len(records))
	seen := map[int]bool{}
	seenFinal := map[int]bool{}
	for _, record := range records {
		normalized, err := normalizeStoredFinalEditSemanticAcceptance(record)
		if err != nil {
			return nil, fmt.Errorf("%w: semantic acceptance payload is invalid", app.ErrConflict)
		}
		if seen[normalized.ParagraphOrdinal] {
			return nil, fmt.Errorf("%w: duplicate semantic acceptance payload", app.ErrConflict)
		}
		seen[normalized.ParagraphOrdinal] = true
		if seenFinal[normalized.FinalParagraphOrdinal] {
			return nil, fmt.Errorf("%w: duplicate semantic acceptance final paragraph payload", app.ErrConflict)
		}
		seenFinal[normalized.FinalParagraphOrdinal] = true
		if !semanticVerdictMatchesFinal(normalized) {
			return nil, fmt.Errorf("%w: semantic acceptance payload is unresolved", app.ErrConflict)
		}
		out = append(out, normalized)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ParagraphOrdinal < out[j].ParagraphOrdinal })
	return out, nil
}

func finalEditStartedEventMatchesPipeline(event app.LedgerEvent, binding FinalEditStageBinding, pipeline string) bool {
	stored, ok := finalEditStageBindingFromStartEventForPipeline(event, pipeline)
	return ok && finalEditStageBindingsEqual(stored, binding)
}

func finalEditSubmittedEventMatchesPipeline(event app.LedgerEvent, binding FinalEditStageBinding, pipeline string) bool {
	stored, ok := finalEditStageBindingFromSubmittedEventForPipeline(event, pipeline)
	if !ok || !finalEditStageBindingsEqual(stored, binding) {
		return false
	}
	payload := eventPayload(event)
	artifactID := payloadString(payload, "artifact_id")
	changed, ok := payloadBoolStrict(payload, "changed")
	if ok && changed && binding.Stage == FinalEditStageStyleSemanticValidation {
		return artifactID != "" && artifactID != binding.SourceArtifactID
	}
	return ok && ((!changed && artifactID == binding.SourceArtifactID) || (changed && artifactID == binding.EditedArtifactID))
}

func finalEditStageBindingFromStartEventForPipeline(event app.LedgerEvent, pipeline string) (FinalEditStageBinding, bool) {
	binding, err := decodeFinalEditStageBindingFromEventForPipeline(event, false, pipeline)
	return binding, err == nil
}

func finalEditStageBindingFromSubmittedEventForPipeline(event app.LedgerEvent, pipeline string) (FinalEditStageBinding, bool) {
	binding, err := decodeFinalEditStageBindingFromEventForPipeline(event, true, pipeline)
	return binding, err == nil
}

func decodeFinalEditStageBindingFromEventForPipeline(event app.LedgerEvent, submitted bool, pipeline string) (FinalEditStageBinding, error) {
	pipeline = strings.TrimSpace(pipeline)
	if !isSupportedFinalEditPipeline(pipeline) {
		return FinalEditStageBinding{}, fmt.Errorf("%w: unsupported final edit pipeline", app.ErrConflict)
	}
	payload := eventPayload(event)
	binding := finalEditStageBindingFromPayload(event, payload)
	if pipeline == FinalEditPipelineAssemblyWriterReaderStyleGateV2 || pipeline == FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3 {
		binding.FinalEditPipeline = pipeline
	}
	if err := validateFinalEditStageBinding(binding); err != nil {
		return FinalEditStageBinding{}, err
	}
	if binding.RigorLevel == "" || binding.RigorLabel == "" {
		return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage event binding is incomplete", app.ErrConflict)
	}
	expectedType := finalEditStartedEventType(binding.Stage)
	expectedKind := "long_form_final_edit_" + binding.Stage + "_started"
	if submitted {
		expectedType = finalEditSubmittedEventType(binding.Stage)
		expectedKind = "long_form_final_edit_" + binding.Stage + "_submitted"
	}
	if event.EventType != expectedType ||
		event.CorrelationID != binding.IdempotencyKey ||
		event.CausationEventID != binding.PlanEventID ||
		event.Producer != binding.Producer ||
		payloadString(payload, "final_edit_pipeline") != pipeline ||
		payloadString(payload, "kind") != expectedKind ||
		payloadString(payload, "stage_id") != finalEditStageID(binding.Stage) {
		return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage event envelope differs", app.ErrConflict)
	}
	if submitted {
		if payloadString(payload, "artifact_id") == "" || payloadString(payload, "edited_artifact_id") != binding.EditedArtifactID {
			return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage submitted artifact envelope differs", app.ErrConflict)
		}
		if _, ok := payloadIntStrict(payload, "operation_count"); !ok {
			return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage operation count is invalid", app.ErrConflict)
		}
		if _, ok := payloadBoolStrict(payload, "changed"); !ok {
			return FinalEditStageBinding{}, fmt.Errorf("%w: final edit stage changed flag is invalid", app.ErrConflict)
		}
	}
	return finalEditStageBindingForCompare(binding), nil
}

func finalEditStageBindingsEqual(left, right FinalEditStageBinding) bool {
	return finalEditStageBindingForCompare(left) == finalEditStageBindingForCompare(right)
}

func finalEditStageBindingForCompare(binding FinalEditStageBinding) FinalEditStageBinding {
	binding = normalizeFinalEditStageBinding(binding)
	if binding.FinalEditPipeline == FinalEditPipelineReaderStyleGateV1 {
		binding.FinalEditPipeline = ""
	}
	return binding
}

func validateStoredFinalEditGateFindingEvidence(ctx context.Context, store LongFormFinalizationStore, missionID string, findings []StoredFinalEditGateFinding) error {
	requiresEvidence := false
	for _, finding := range findings {
		requiresEvidence = requiresEvidence || len(finding.EvidenceIDs) > 0
	}
	if !requiresEvidence {
		return nil
	}
	validator, ok := store.(finalEditEvidenceStore)
	if !ok || validator == nil {
		return fmt.Errorf("%w: final edit gate evidence validator is required", app.ErrConflict)
	}
	missionID = strings.TrimSpace(missionID)
	for _, finding := range findings {
		for _, evidenceID := range finding.EvidenceIDs {
			record, err := validator.GetEvidenceRecord(ctx, evidenceID)
			if err != nil || record.MissionID != missionID || strings.TrimSpace(record.State) != "approved" {
				return fmt.Errorf("%w: final edit gate evidence ref is not approved", app.ErrConflict)
			}
		}
	}
	return nil
}
