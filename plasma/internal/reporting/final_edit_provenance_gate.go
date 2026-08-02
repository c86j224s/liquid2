package reporting

import (
	"context"
	"fmt"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

const (
	FinalEditGateClassMissionSourceGrounded  = "mission_source_grounded"
	FinalEditGateClassSessionGrounded        = "session_grounded"
	FinalEditGateClassDerivedSynthesis       = "derived_synthesis"
	FinalEditGateClassRhetoricalConstruction = "rhetorical_construction"
	FinalEditGateClassUnverifiedExternalFact = "unverified_external_fact"

	FinalEditRepairAttachApprovedEvidence      = "attach_approved_evidence"
	FinalEditRepairQualifyInferenceUncertainty = "qualify_inference_or_uncertainty"
	FinalEditRepairRetainWithFootnote          = "retain_with_footnote"
	FinalEditRepairRemove                      = "remove"
)

var finalEditGateClasses = map[string]bool{
	FinalEditGateClassMissionSourceGrounded:  true,
	FinalEditGateClassSessionGrounded:        true,
	FinalEditGateClassDerivedSynthesis:       true,
	FinalEditGateClassRhetoricalConstruction: true,
	FinalEditGateClassUnverifiedExternalFact: true,
}

func FinalEditRepairActionsInOrder() []string {
	return []string{
		FinalEditRepairAttachApprovedEvidence,
		FinalEditRepairQualifyInferenceUncertainty,
		FinalEditRepairRetainWithFootnote,
		FinalEditRepairRemove,
	}
}

type FinalEditGateFinding struct {
	Statement              string
	StatementSHA256        string
	Classification         string
	RepairAction           string
	EvidenceIDs            []string
	RawPassage             string
	UnapprovedSourceIDs    []string
	UnapprovedCandidateIDs []string
}

type StoredFinalEditGateFinding struct {
	StatementSHA256 string   `json:"statement_sha256"`
	Classification  string   `json:"classification"`
	RepairAction    string   `json:"repair_action,omitempty"`
	EvidenceIDs     []string `json:"evidence_ids,omitempty"`
}

type finalEditEvidenceStore interface {
	GetEvidenceRecord(context.Context, string) (app.EvidenceRecord, error)
}

func NormalizeFinalEditGateFindings(ctx context.Context, store finalEditEvidenceStore, missionID string, findings []FinalEditGateFinding) ([]StoredFinalEditGateFinding, error) {
	missionID = strings.TrimSpace(missionID)
	if missionID == "" {
		return nil, fmt.Errorf("%w: mission id is required", app.ErrInvalidInput)
	}
	out := make([]StoredFinalEditGateFinding, 0, len(findings))
	seen := map[string]bool{}
	for _, finding := range findings {
		normalized, err := normalizeFinalEditGateFinding(ctx, store, missionID, finding)
		if err != nil {
			return nil, err
		}
		if seen[normalized.StatementSHA256] {
			return nil, fmt.Errorf("%w: duplicate gate statement hash", app.ErrConflict)
		}
		seen[normalized.StatementSHA256] = true
		out = append(out, normalized)
	}
	return out, nil
}

func NormalizeFinalEditEvidenceGateFindings(ctx context.Context, store finalEditEvidenceStore, missionID string, findings []FinalEditGateFinding) ([]StoredFinalEditGateFinding, error) {
	missionID = strings.TrimSpace(missionID)
	if missionID == "" {
		return nil, fmt.Errorf("%w: mission id is required", app.ErrInvalidInput)
	}
	out := make([]StoredFinalEditGateFinding, 0, len(findings))
	seen := map[string]bool{}
	for _, finding := range findings {
		normalized, err := normalizeFinalEditEvidenceGateFinding(ctx, store, missionID, finding)
		if err != nil {
			return nil, err
		}
		if seen[normalized.StatementSHA256] {
			return nil, fmt.Errorf("%w: duplicate evidence gate statement hash", app.ErrConflict)
		}
		seen[normalized.StatementSHA256] = true
		out = append(out, normalized)
	}
	return out, nil
}

func normalizeFinalEditEvidenceGateFinding(ctx context.Context, store finalEditEvidenceStore, missionID string, finding FinalEditGateFinding) (StoredFinalEditGateFinding, error) {
	if strings.TrimSpace(finding.Statement) != "" ||
		strings.TrimSpace(finding.RepairAction) != "" ||
		strings.TrimSpace(finding.RawPassage) != "" ||
		len(finding.UnapprovedSourceIDs) > 0 ||
		len(finding.UnapprovedCandidateIDs) > 0 {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: evidence gate findings cannot carry prose, repair actions, raw passages, or unapproved refs", app.ErrInvalidInput)
	}
	statementSHA := strings.TrimSpace(finding.StatementSHA256)
	if !validStoredFinalEditStatementSHA256(statementSHA) {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: evidence gate statement hash is invalid", app.ErrInvalidInput)
	}
	classification := strings.TrimSpace(finding.Classification)
	if !finalEditGateClasses[classification] {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: unsupported gate classification", app.ErrInvalidInput)
	}
	evidenceIDs := normalizeReportingIDs(finding.EvidenceIDs)
	if len(evidenceIDs) > 0 && store == nil {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: evidence gate store is required", app.ErrInvalidInput)
	}
	for _, evidenceID := range evidenceIDs {
		record, err := store.GetEvidenceRecord(ctx, evidenceID)
		if err != nil || record.MissionID != missionID || strings.TrimSpace(record.State) != "approved" {
			return StoredFinalEditGateFinding{}, fmt.Errorf("%w: evidence gate evidence ref is not approved", app.ErrInvalidInput)
		}
	}
	return StoredFinalEditGateFinding{
		StatementSHA256: statementSHA,
		Classification:  classification,
		EvidenceIDs:     evidenceIDs,
	}, nil
}

func normalizeFinalEditGateFinding(ctx context.Context, store finalEditEvidenceStore, missionID string, finding FinalEditGateFinding) (StoredFinalEditGateFinding, error) {
	statement := strings.TrimSpace(finding.Statement)
	if statement == "" {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: gate statement text is required for server-side hashing", app.ErrInvalidInput)
	}
	if strings.TrimSpace(finding.RawPassage) != "" || len(finding.UnapprovedSourceIDs) > 0 || len(finding.UnapprovedCandidateIDs) > 0 {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: gate metadata cannot persist raw passages or unapproved source refs", app.ErrInvalidInput)
	}
	classification := strings.TrimSpace(finding.Classification)
	if !finalEditGateClasses[classification] {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: unsupported gate classification", app.ErrInvalidInput)
	}
	action := strings.TrimSpace(finding.RepairAction)
	if classification == FinalEditGateClassUnverifiedExternalFact {
		if !validFinalEditRepairAction(action) {
			return StoredFinalEditGateFinding{}, fmt.Errorf("%w: unverified external fact requires an ordered repair action", app.ErrInvalidInput)
		}
	} else if action != "" {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: only unverified external facts may carry repair actions", app.ErrInvalidInput)
	}
	evidenceIDs := normalizeReportingIDs(finding.EvidenceIDs)
	if len(evidenceIDs) > 0 && action != FinalEditRepairAttachApprovedEvidence {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: evidence refs are only valid for attach_approved_evidence repairs", app.ErrInvalidInput)
	}
	if action == FinalEditRepairAttachApprovedEvidence && len(evidenceIDs) == 0 {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: attach_approved_evidence requires approved evidence refs", app.ErrInvalidInput)
	}
	if len(evidenceIDs) > 0 && store == nil {
		return StoredFinalEditGateFinding{}, fmt.Errorf("%w: gate repair evidence store is required", app.ErrInvalidInput)
	}
	for _, evidenceID := range evidenceIDs {
		record, err := store.GetEvidenceRecord(ctx, evidenceID)
		if err != nil || record.MissionID != missionID || strings.TrimSpace(record.State) != "approved" {
			return StoredFinalEditGateFinding{}, fmt.Errorf("%w: gate repair evidence ref is not approved", app.ErrInvalidInput)
		}
	}
	return StoredFinalEditGateFinding{
		StatementSHA256: contentSHA256([]byte(statement)),
		Classification:  classification,
		RepairAction:    action,
		EvidenceIDs:     evidenceIDs,
	}, nil
}

func validFinalEditRepairAction(action string) bool {
	for _, allowed := range FinalEditRepairActionsInOrder() {
		if action == allowed {
			return true
		}
	}
	return false
}

func validStoredFinalEditStatementSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func equalStoredFinalEditGateFindings(left, right []StoredFinalEditGateFinding) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].StatementSHA256 != right[i].StatementSHA256 ||
			left[i].Classification != right[i].Classification ||
			left[i].RepairAction != right[i].RepairAction ||
			!equalStrings(left[i].EvidenceIDs, right[i].EvidenceIDs) {
			return false
		}
	}
	return true
}

func validateCanonicalArtifactEnvelope(artifact app.RawArtifact, binding LongFormFinalizeBinding, payload map[string]any) error {
	if artifact.MissionID != binding.MissionID ||
		artifact.MediaType != "text/markdown; charset=utf-8" ||
		artifact.Filename != binding.Filename ||
		artifact.SHA256 != contentSHA256(artifact.Content) ||
		payloadString(payload, "artifact_sha256") != artifact.SHA256 {
		return fmt.Errorf("%w: canonical long-form final artifact differs", app.ErrConflict)
	}
	changed, ok := payloadBoolStrict(payload, "final_edit_gate_changed")
	if !ok {
		return fmt.Errorf("%w: canonical final edit gate changed flag is invalid", app.ErrConflict)
	}
	if changed && artifact.Producer != binding.Producer {
		return fmt.Errorf("%w: changed canonical long-form final artifact producer differs", app.ErrConflict)
	}
	return nil
}

func normalizeReportingIDs(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}
