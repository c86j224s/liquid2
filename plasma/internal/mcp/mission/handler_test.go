package mission

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
	canonicalmission "github.com/c86j224s/liquid2/plasma/internal/mission"
	"github.com/c86j224s/liquid2/plasma/internal/researchrecords"
	"github.com/c86j224s/liquid2/plasma/internal/source"
)

type fakeReader struct {
	projection canonicalmission.Projection
	sources    []source.Snapshot
	evidence   []researchrecords.EvidenceRecord
	claims     []researchrecords.ClaimRecord
	questions  []researchrecords.QuestionRecord
	calls      []string
	err        error
}

func (f *fakeReader) GetProjection(context.Context, string) (canonicalmission.Projection, error) {
	f.calls = append(f.calls, "projection")
	return f.projection, f.err
}
func (f *fakeReader) ListSourceSnapshotsWithState(context.Context, source.ListRequest) ([]source.Snapshot, error) {
	f.calls = append(f.calls, "sources")
	return f.sources, f.err
}
func (f *fakeReader) ListEvidenceRecords(context.Context, string) ([]researchrecords.EvidenceRecord, error) {
	f.calls = append(f.calls, "evidence")
	return f.evidence, f.err
}
func (f *fakeReader) ListClaimRecords(context.Context, string) ([]researchrecords.ClaimRecord, error) {
	f.calls = append(f.calls, "claims")
	return f.claims, f.err
}
func (f *fakeReader) ListQuestionRecords(context.Context, string) ([]researchrecords.QuestionRecord, error) {
	f.calls = append(f.calls, "questions")
	return f.questions, f.err
}

type fakeUpdater struct {
	requests []canonicalmission.UpdateMissionMetadataRequest
}

func (f *fakeUpdater) UpdateMissionMetadata(_ context.Context, req canonicalmission.UpdateMissionMetadataRequest) (canonicalmission.UpdateMissionMetadataResult, error) {
	f.requests = append(f.requests, req)
	return canonicalmission.UpdateMissionMetadataResult{Event: ledger.Event{EventID: req.EventID}}, nil
}

func testHandler(reader Reader, updater MetadataUpdater, bound func(string) error) *Handler {
	return NewHandler(reader, updater, bound, func(string) string { return "evt_new" }, func(snapshot source.Snapshot) any { return snapshot.SnapshotID }, func(name, mission, kind, message string, retryable bool, related []string) wire.ToolResult {
		return wire.ToolResult{ToolName: name, MissionID: mission, Error: &wire.ToolError{ErrorKind: kind, Message: message, Retryable: retryable, RelatedObjectIDs: related}}
	}, func(name, mission string, err error, related []string) wire.ToolResult {
		return wire.ToolResult{ToolName: name, MissionID: mission, Error: &wire.ToolError{ErrorKind: "internal", Message: err.Error(), RelatedObjectIDs: related}}
	})
}

func TestCallGetPreservesProjectionAndIncludeOrder(t *testing.T) {
	reader := &fakeReader{projection: canonicalmission.Projection{MissionID: "mis_1"}, sources: []source.Snapshot{{SnapshotID: "src_1"}}, evidence: []researchrecords.EvidenceRecord{{EvidenceID: "evd_1"}}, claims: []researchrecords.ClaimRecord{{ClaimID: "clm_1"}}, questions: []researchrecords.QuestionRecord{{QuestionID: "qst_1"}}}
	handler := testHandler(reader, nil, func(string) error { return nil })
	result := handler.CallGet(context.Background(), wire.ToolCall{Name: "plasma.mission.get", Arguments: json.RawMessage(`{"mission_id":"mis_1","include":["all"]}`)})
	if result.Error != nil {
		t.Fatalf("unexpected error: %#v", result.Error)
	}
	if !reflect.DeepEqual(reader.calls, []string{"projection", "sources", "evidence", "claims", "questions"}) {
		t.Fatalf("calls = %#v", reader.calls)
	}
	output := result.Content.(GetOutput)
	if output.ActiveReportVersion != nil || len(output.OpenQuestions) != 1 || output.Sources[0] != "src_1" {
		t.Fatalf("output = %#v", output)
	}
}

func TestCallGetStopsBeforeReaderOnBindingAndPreservesErrors(t *testing.T) {
	reader := &fakeReader{}
	handler := testHandler(reader, nil, func(string) error { return errors.New("outside binding") })
	result := handler.CallGet(context.Background(), wire.ToolCall{Name: "get", Arguments: json.RawMessage(`{"mission_id":"mis_1"}`)})
	if result.Error == nil || result.Error.Message != "outside binding" {
		t.Fatalf("result = %#v", result)
	}
	if len(reader.calls) != 0 {
		t.Fatalf("reader calls = %#v", reader.calls)
	}
}

func TestCallUpdateOptionalUpdaterAndEmptyFields(t *testing.T) {
	args := json.RawMessage(`{"mission_id":"mis_1","title":"","objective":"","scope":{"included":[],"excluded":[]}}`)
	withoutUpdater := testHandler(&fakeReader{}, nil, func(string) error { return nil }).CallUpdate(context.Background(), wire.ToolCall{Name: "update", Arguments: args})
	if withoutUpdater.Error == nil || withoutUpdater.Error.ErrorKind != "internal" {
		t.Fatalf("without updater = %#v", withoutUpdater)
	}
	updater := &fakeUpdater{}
	result := testHandler(&fakeReader{}, updater, func(string) error { return nil }).CallUpdate(context.Background(), wire.ToolCall{Name: "update", Arguments: args})
	if result.Error != nil || len(updater.requests) != 1 {
		t.Fatalf("result = %#v requests = %#v", result, updater.requests)
	}
	request := updater.requests[0]
	if request.EventID != "evt_new" || request.Title == nil || *request.Title != "" || request.Objective == nil || *request.Objective != "" || request.Scope == nil || len(request.Scope.Included) != 0 {
		t.Fatalf("request = %#v", request)
	}
}

func TestCallUpdateRequiresAFieldBeforeUpdater(t *testing.T) {
	updater := &fakeUpdater{}
	result := testHandler(&fakeReader{}, updater, func(string) error { return nil }).CallUpdate(context.Background(), wire.ToolCall{Name: "update", Arguments: json.RawMessage(`{"mission_id":"mis_1"}`)})
	if result.Error == nil || result.Error.Message != "at least one metadata field is required" || len(updater.requests) != 0 {
		t.Fatalf("result = %#v requests = %#v", result, updater.requests)
	}
}
