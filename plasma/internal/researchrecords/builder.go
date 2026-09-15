package researchrecords

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

var allowedEvidenceTypes = map[string]bool{
	"quote": true, "fact": true, "table_row": true, "statistic": true,
	"observation": true, "interpretation": true, "reaction": true, "rumor": true,
	"controversy": true, "market_signal": true, "code": true, "formula": true,
	"benchmark": true, "open_question": true, "user_assertion": true,
}

var allowedConfidenceLevels = map[string]bool{"low": true, "medium": true, "high": true, "unknown": true}
var allowedProposedLifecycleStates = map[string]bool{"draft": true, "proposed": true, "needs_review": true}

// BuildEvidenceRecord validates and normalizes evidence in the historical validation order.
// Snapshot reads occur only after all request, producer, event, assertion, and ref checks.
func BuildEvidenceRecord(ctx context.Context, snapshots SnapshotReader, req CreateEvidenceRecordRequest, createdEvent ledger.Event) (EvidenceRecord, error) {
	evidenceID := strings.TrimSpace(req.EvidenceID)
	missionID := strings.TrimSpace(req.MissionID)
	if err := validateID("evd_", evidenceID); err != nil {
		return EvidenceRecord{}, err
	}
	if err := validateID("mis_", missionID); err != nil {
		return EvidenceRecord{}, err
	}
	if strings.TrimSpace(req.Summary) == "" {
		return EvidenceRecord{}, fmt.Errorf("%w: evidence summary is required", producterror.ErrInvalidInput)
	}
	evidenceType := strings.TrimSpace(req.EvidenceType)
	if !allowedEvidenceTypes[evidenceType] {
		return EvidenceRecord{}, fmt.Errorf("%w: unsupported evidence type", producterror.ErrInvalidInput)
	}
	state, err := NormalizeProposedLifecycleState(req.State, "proposed")
	if err != nil {
		return EvidenceRecord{}, err
	}
	if err := validateProducer(req.Producer); err != nil {
		return EvidenceRecord{}, err
	}
	if createdEvent.MissionID != missionID || strings.TrimSpace(createdEvent.EventID) != strings.TrimSpace(req.CreatedEventID) {
		return EvidenceRecord{}, fmt.Errorf("%w: evidence creation event mismatch", producterror.ErrInvalidInput)
	}
	if evidenceType == "user_assertion" && !isApprovalProducer(createdEvent.Producer) {
		return EvidenceRecord{}, fmt.Errorf("%w: user assertion evidence requires a user or steering_chat event", producterror.ErrInvalidInput)
	}
	if evidenceType != "user_assertion" && len(req.SnapshotRefs) == 0 {
		return EvidenceRecord{}, fmt.Errorf("%w: evidence requires snapshot refs unless it is a user assertion", producterror.ErrInvalidInput)
	}
	snapshotRefs, err := normalizeSnapshotRefs(ctx, snapshots, missionID, req.SnapshotRefs)
	if err != nil {
		return EvidenceRecord{}, err
	}
	confidence, err := NormalizeConfidence(req.Confidence)
	if err != nil {
		return EvidenceRecord{}, err
	}
	return EvidenceRecord{
		SchemaVersion: EvidenceRecordSchemaVersion, ObjectKind: EvidenceRecordObjectKind,
		EvidenceID: evidenceID, MissionID: missionID, State: state,
		Summary: strings.TrimSpace(req.Summary), EvidenceType: evidenceType,
		SnapshotRefs: snapshotRefs, Confidence: confidence,
		Producer: normalizeProducer(req.Producer), CreatedEventID: strings.TrimSpace(req.CreatedEventID), CreatedAt: time.Now().UTC(),
	}, nil
}

// NormalizeConfidence applies the evidence confidence defaults and allowlist.
func NormalizeConfidence(confidence Confidence) (Confidence, error) {
	level := strings.TrimSpace(confidence.Level)
	if level == "" {
		level = "unknown"
	}
	if !allowedConfidenceLevels[level] {
		return Confidence{}, fmt.Errorf("%w: unsupported confidence level", producterror.ErrInvalidInput)
	}
	return Confidence{Level: level, Rationale: strings.TrimSpace(confidence.Rationale), OpenRisks: normalizeStringList(confidence.OpenRisks), NeedsVerification: confidence.NeedsVerification}, nil
}

// NormalizeProposedLifecycleState normalizes states allowed at evidence creation.
func NormalizeProposedLifecycleState(state, defaultState string) (string, error) {
	trimmed := strings.TrimSpace(state)
	if trimmed == "" {
		trimmed = defaultState
	}
	if !map[string]bool{"draft": true, "proposed": true, "needs_review": true, "approved": true, "rejected": true, "superseded": true, "archived": true}[trimmed] {
		return "", fmt.Errorf("%w: unsupported lifecycle state", producterror.ErrInvalidInput)
	}
	if !allowedProposedLifecycleStates[trimmed] {
		return "", fmt.Errorf("%w: terminal lifecycle state requires a transition event", producterror.ErrInvalidInput)
	}
	return trimmed, nil
}

func normalizeSnapshotRefs(ctx context.Context, snapshots SnapshotReader, missionID string, refs []SnapshotRef) ([]SnapshotRef, error) {
	normalized := make([]SnapshotRef, 0, len(refs))
	seen := map[string]struct{}{}
	for _, ref := range refs {
		snapshotID, artifactID := strings.TrimSpace(ref.SnapshotID), strings.TrimSpace(ref.ArtifactID)
		if err := validateID("src_", snapshotID); err != nil {
			return nil, err
		}
		if err := validateID("art_", artifactID); err != nil {
			return nil, err
		}
		key := snapshotID + "\x00" + artifactID
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("%w: duplicate snapshot ref", producterror.ErrInvalidInput)
		}
		seen[key] = struct{}{}
		snapshot, err := snapshots.GetSourceSnapshot(ctx, snapshotID)
		if err != nil {
			return nil, err
		}
		if snapshot.MissionID != missionID {
			return nil, fmt.Errorf("%w: evidence snapshot belongs to another mission", producterror.ErrInvalidInput)
		}
		if !containsString(snapshot.ArtifactIDs, artifactID) {
			return nil, fmt.Errorf("%w: evidence artifact is not linked to snapshot", producterror.ErrInvalidInput)
		}
		locator := append(json.RawMessage(nil), ref.Locator...)
		if len(locator) == 0 {
			locator = json.RawMessage(`{}`)
		}
		if !json.Valid(locator) {
			return nil, fmt.Errorf("%w: evidence locator must be valid JSON", producterror.ErrInvalidInput)
		}
		normalized = append(normalized, SnapshotRef{SnapshotID: snapshotID, ArtifactID: artifactID, Locator: locator})
	}
	return normalized, nil
}

func validateID(prefix, id string) error {
	if !strings.HasPrefix(strings.TrimSpace(id), prefix) || len(strings.TrimSpace(id)) <= len(prefix) {
		return fmt.Errorf("%w: id must start with %s", producterror.ErrInvalidInput, prefix)
	}
	return nil
}
func validateProducer(producer ledger.Producer) error {
	if strings.TrimSpace(producer.Type) == "" || strings.TrimSpace(producer.ID) == "" {
		return fmt.Errorf("%w: producer type and id are required", producterror.ErrInvalidInput)
	}
	return nil
}
func normalizeProducer(producer ledger.Producer) ledger.Producer {
	return ledger.Producer{Type: strings.TrimSpace(producer.Type), ID: strings.TrimSpace(producer.ID)}
}
func isApprovalProducer(producer ledger.Producer) bool {
	return producer.Type == "user" || producer.Type == "steering_chat"
}
func normalizeStringList(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}
func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}
