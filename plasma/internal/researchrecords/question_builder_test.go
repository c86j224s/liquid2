package researchrecords

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func TestBuildQuestionRecordOrdersEvidenceThenClaimCallbacksIncludingEmptyLists(t *testing.T) {
	var calls []string
	requirements := QuestionRequirements{
		RequireEvidenceRecords: func(_ context.Context, missionID string, ids []string) error {
			calls = append(calls, "evidence:"+missionID+":"+strings.Join(ids, ","))
			return nil
		},
		RequireClaimRecords: func(_ context.Context, missionID string, ids []string) error {
			calls = append(calls, "claim:"+missionID+":"+strings.Join(ids, ","))
			return nil
		},
	}
	record, err := BuildQuestionRecord(context.Background(), requirements, CreateQuestionRecordRequest{
		QuestionID: " qst_1 ", MissionID: " mis_1 ", Text: "  question  ", CreatedEventID: " evt_1 ",
	}, ledger.Event{MissionID: "mis_1", EventID: "evt_1"})
	if err != nil {
		t.Fatalf("BuildQuestionRecord returned error: %v", err)
	}
	if want := []string{"evidence:mis_1:", "claim:mis_1:"}; !reflect.DeepEqual(calls, want) {
		t.Fatalf("callback order mismatch: got=%v want=%v", calls, want)
	}
	if record.Priority != "medium" || record.Text != "question" || record.CreatedAt.Location() != time.UTC {
		t.Fatalf("unexpected normalized question: %#v", record)
	}
}

func TestBuildQuestionRecordRejectsEarlyFieldsAndEventMismatchWithoutCallbacks(t *testing.T) {
	calls := 0
	requirements := QuestionRequirements{
		RequireEvidenceRecords: func(context.Context, string, []string) error { calls++; return nil },
		RequireClaimRecords:    func(context.Context, string, []string) error { calls++; return nil },
	}
	for _, req := range []CreateQuestionRecordRequest{
		{QuestionID: "bad", MissionID: "mis_1", Text: "question", CreatedEventID: "evt_1"},
		{QuestionID: "qst_1", MissionID: "mis_1", Text: "", CreatedEventID: "evt_1"},
	} {
		if _, err := BuildQuestionRecord(context.Background(), requirements, req, ledger.Event{MissionID: "mis_1", EventID: "evt_1"}); err == nil {
			t.Fatal("expected early validation error")
		}
	}
	if _, err := BuildQuestionRecord(context.Background(), requirements, CreateQuestionRecordRequest{
		QuestionID: "qst_1", MissionID: "mis_1", Text: "question", CreatedEventID: "evt_other",
	}, ledger.Event{MissionID: "mis_1", EventID: "evt_1"}); err == nil {
		t.Fatal("expected created-event mismatch")
	}
	if calls != 0 {
		t.Fatalf("early validation called requirements %d times", calls)
	}
}

func TestBuildQuestionRecordRejectsDuplicateIDsAndResolutionGuard(t *testing.T) {
	requirements := QuestionRequirements{
		RequireEvidenceRecords: func(context.Context, string, []string) error { return nil },
		RequireClaimRecords:    func(context.Context, string, []string) error { return nil },
	}
	base := CreateQuestionRecordRequest{QuestionID: "qst_1", MissionID: "mis_1", Text: "question", CreatedEventID: "evt_1"}
	base.RelatedEvidenceIDs = []string{"evd_1", " evd_1"}
	if _, err := BuildQuestionRecord(context.Background(), requirements, base, ledger.Event{MissionID: "mis_1", EventID: "evt_1"}); err == nil {
		t.Fatal("expected duplicate ID error")
	}
	base.RelatedEvidenceIDs = nil
	base.State = "answered"
	if _, err := BuildQuestionRecord(context.Background(), requirements, base, ledger.Event{MissionID: "mis_1", EventID: "evt_1"}); err == nil {
		t.Fatal("expected terminal create-state error")
	}
	base.State = "open"
	base.Resolution = "  resolution  "
	if _, err := BuildQuestionRecord(context.Background(), requirements, base, ledger.Event{MissionID: "mis_1", EventID: "evt_1"}); err != nil {
		t.Fatalf("unexpected resolved open question error: %v", err)
	}
}
