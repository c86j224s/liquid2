package reporting_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestNormalizeFinalEditGateFindingsHashesStatementsAndRedactsRawText(t *testing.T) {
	ctx := context.Background()
	store := finalEditGateEvidenceStore{
		"evd_ok": {EvidenceID: "evd_ok", MissionID: "mis_1", State: "approved"},
	}

	got, err := reporting.NormalizeFinalEditGateFindings(ctx, store, "mis_1", []reporting.FinalEditGateFinding{
		{Statement: " Supported statement. ", Classification: reporting.FinalEditGateClassMissionSourceGrounded},
		{
			Statement:      "External claim.",
			Classification: reporting.FinalEditGateClassUnverifiedExternalFact,
			RepairAction:   reporting.FinalEditRepairAttachApprovedEvidence,
			EvidenceIDs:    []string{" evd_ok ", "evd_ok"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("finding count=%d, want 2", len(got))
	}
	if got[0].StatementSHA256 != sha256Hex("Supported statement.") || got[0].RepairAction != "" || len(got[0].EvidenceIDs) != 0 {
		t.Fatalf("grounded finding not normalized: %#v", got[0])
	}
	if got[1].StatementSHA256 != sha256Hex("External claim.") || got[1].RepairAction != reporting.FinalEditRepairAttachApprovedEvidence || !equalTestStrings(got[1].EvidenceIDs, []string{"evd_ok"}) {
		t.Fatalf("attach repair not normalized: %#v", got[1])
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if text := string(encoded); strings.Contains(text, "Supported statement") || strings.Contains(text, "External claim") {
		t.Fatalf("stored findings leaked raw statement text: %s", text)
	}
}

func TestNormalizeFinalEditGateFindingsRejectsUnsafeMetadataAndUnapprovedEvidence(t *testing.T) {
	ctx := context.Background()
	store := finalEditGateEvidenceStore{
		"evd_ok":      {EvidenceID: "evd_ok", MissionID: "mis_1", State: "approved"},
		"evd_pending": {EvidenceID: "evd_pending", MissionID: "mis_1", State: "proposed"},
		"evd_foreign": {EvidenceID: "evd_foreign", MissionID: "mis_other", State: "approved"},
	}
	tests := []struct {
		name    string
		finding reporting.FinalEditGateFinding
	}{
		{
			name:    "raw passage",
			finding: reporting.FinalEditGateFinding{Statement: "A", Classification: reporting.FinalEditGateClassMissionSourceGrounded, RawPassage: "quoted source body"},
		},
		{
			name:    "unapproved source ref",
			finding: reporting.FinalEditGateFinding{Statement: "A", Classification: reporting.FinalEditGateClassMissionSourceGrounded, UnapprovedSourceIDs: []string{"src_1"}},
		},
		{
			name:    "unsupported class",
			finding: reporting.FinalEditGateFinding{Statement: "A", Classification: "source_grounded"},
		},
		{
			name:    "grounded action",
			finding: reporting.FinalEditGateFinding{Statement: "A", Classification: reporting.FinalEditGateClassMissionSourceGrounded, RepairAction: reporting.FinalEditRepairRemove},
		},
		{
			name:    "unverified missing action",
			finding: reporting.FinalEditGateFinding{Statement: "A", Classification: reporting.FinalEditGateClassUnverifiedExternalFact},
		},
		{
			name:    "retain cannot carry evidence",
			finding: reporting.FinalEditGateFinding{Statement: "A", Classification: reporting.FinalEditGateClassUnverifiedExternalFact, RepairAction: reporting.FinalEditRepairRetainWithFootnote, EvidenceIDs: []string{"evd_ok"}},
		},
		{
			name:    "attach missing evidence",
			finding: reporting.FinalEditGateFinding{Statement: "A", Classification: reporting.FinalEditGateClassUnverifiedExternalFact, RepairAction: reporting.FinalEditRepairAttachApprovedEvidence},
		},
		{
			name:    "pending evidence",
			finding: reporting.FinalEditGateFinding{Statement: "A", Classification: reporting.FinalEditGateClassUnverifiedExternalFact, RepairAction: reporting.FinalEditRepairAttachApprovedEvidence, EvidenceIDs: []string{"evd_pending"}},
		},
		{
			name:    "foreign evidence",
			finding: reporting.FinalEditGateFinding{Statement: "A", Classification: reporting.FinalEditGateClassUnverifiedExternalFact, RepairAction: reporting.FinalEditRepairAttachApprovedEvidence, EvidenceIDs: []string{"evd_foreign"}},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			if _, err := reporting.NormalizeFinalEditGateFindings(ctx, store, "mis_1", []reporting.FinalEditGateFinding{test.finding}); !errors.Is(err, app.ErrInvalidInput) {
				t.Fatalf("error=%v, want invalid input", err)
			}
		})
	}
}

func TestNormalizeFinalEditGateFindingsRejectsDuplicateStatementHashes(t *testing.T) {
	_, err := reporting.NormalizeFinalEditGateFindings(context.Background(), nil, "mis_1", []reporting.FinalEditGateFinding{
		{Statement: " Same claim. ", Classification: reporting.FinalEditGateClassSessionGrounded},
		{Statement: "Same claim.", Classification: reporting.FinalEditGateClassDerivedSynthesis},
	})
	if !errors.Is(err, app.ErrConflict) {
		t.Fatalf("error=%v, want conflict", err)
	}
}

func TestFinalEditRepairActionOrderIsStable(t *testing.T) {
	want := []string{
		reporting.FinalEditRepairAttachApprovedEvidence,
		reporting.FinalEditRepairQualifyInferenceUncertainty,
		reporting.FinalEditRepairRetainWithFootnote,
		reporting.FinalEditRepairRemove,
	}
	if got := reporting.FinalEditRepairActionsInOrder(); !equalTestStrings(got, want) {
		t.Fatalf("repair order=%#v, want %#v", got, want)
	}
}

type finalEditGateEvidenceStore map[string]app.EvidenceRecord

func (s finalEditGateEvidenceStore) GetEvidenceRecord(_ context.Context, evidenceID string) (app.EvidenceRecord, error) {
	record, ok := s[evidenceID]
	if !ok {
		return app.EvidenceRecord{}, errors.New("missing evidence")
	}
	return record, nil
}

func sha256Hex(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func equalTestStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
