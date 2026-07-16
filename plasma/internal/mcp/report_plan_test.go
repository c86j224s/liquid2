package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type reportPlanMCPService struct {
	*fakeMCPService
	submission  app.ReportPlanSubmissionRequest
	submissions int
	refs        []app.ReportBlockSourceRefs
	refErr      error
}

func (service *reportPlanMCPService) SubmitReportPlan(_ context.Context, req app.ReportPlanSubmissionRequest) (app.ReportPlanSubmission, error) {
	service.submission = req
	service.submissions++
	return app.ReportPlanSubmission{Event: app.LedgerEvent{EventID: req.EventID, MissionID: req.MissionID, EventType: "report.plan.submitted"}}, nil
}

func (service *reportPlanMCPService) ValidateReportPlanRefs(_ context.Context, _ string, refs []app.ReportBlockSourceRefs) error {
	service.refs = refs
	return service.refErr
}

func testReportPlanBinding() ReportPlanBinding {
	return ReportPlanBinding{PendingEventID: "evt_pending", ReportMode: "planned", IdempotencyKey: "key_1", ToolSessionID: "ses_tool", PreviousProviderSessionID: "ses_previous", AgentExecutor: "codex", AgentModel: "gpt-test", AgentReasoningEffort: "high"}
}

func TestReportPlanToolRequiresBindingAndExplicitEnablement(t *testing.T) {
	service := &reportPlanMCPService{fakeMCPService: &fakeMCPService{}}
	binding := Binding{MissionID: "mis_1", AgentSessionID: "ses_tool", AgentExecutor: "codex"}
	cases := []struct {
		name    string
		options []Option
		want    bool
	}{
		{"default", []Option{WithBinding(binding)}, false},
		{"binding only", []Option{WithBinding(binding), WithReportPlanBinding(testReportPlanBinding())}, false},
		{"enable only", []Option{WithBinding(binding), WithEnabledTools([]string{ToolReportPlanSubmit})}, false},
		{"bound planning session", []Option{WithBinding(binding), WithReportPlanBinding(testReportPlanBinding()), WithEnabledTools([]string{ToolMissionGet, ToolReportPlanSubmit})}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tools := NewServer(service, tc.options...).ListTools()
			foundPlan, foundResearch := false, false
			for _, tool := range tools {
				foundPlan = foundPlan || tool.Name == ToolReportPlanSubmit
				foundResearch = foundResearch || tool.Name == ToolMissionGet
			}
			if foundPlan != tc.want {
				t.Fatalf("plan tool visibility=%v, want %v: %#v", foundPlan, tc.want, tools)
			}
			if tc.want && !foundResearch {
				t.Fatal("existing research tool was not retained")
			}
		})
	}
}

func TestReportPlanToolSubmitsNormalizedPlanAndAllRefKinds(t *testing.T) {
	service := &reportPlanMCPService{fakeMCPService: &fakeMCPService{}}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_tool", AgentExecutor: "codex"}), WithReportPlanBinding(testReportPlanBinding()), WithEnabledTools([]string{ToolReportPlanSubmit}))
	args := json.RawMessage(`{"mission_id":"mis_1","session_id":"ses_tool","pending_event_id":"evt_pending","report_mode":"planned","idempotency_key":"key_1","producer":{"type":"agent_session","id":"ses_tool"},"plan":{"summary":"summary","sections":[{"title":"section","purpose":"purpose","target_refs":{"claim_ids":["clm_1"],"evidence_ids":["evd_1"],"snapshot_ids":["src_1"],"question_ids":["qst_1"],"option_ids":["opt_1"]}}]}}`)
	result := server.Call(context.Background(), ToolCall{Name: ToolReportPlanSubmit, Arguments: args})
	if result.Error != nil {
		t.Fatalf("submit failed: %#v", result.Error)
	}
	if service.submission.PlanHash == "" || service.submission.ArgumentsHash == "" || service.submission.Attempt != 1 {
		t.Fatalf("missing durable metadata: %#v", service.submission)
	}
	if service.submission.PreviousProviderSessionID != "ses_previous" || service.submission.ToolProducer.ID != "ses_tool" || service.submission.AgentModel != "gpt-test" {
		t.Fatalf("truthful binding metadata was not stored: %#v", service.submission)
	}
	if len(service.refs) != 1 || len(service.refs[0].ClaimIDs) != 1 || len(service.refs[0].EvidenceIDs) != 1 || len(service.refs[0].SnapshotIDs) != 1 || len(service.refs[0].QuestionIDs) != 1 || len(service.refs[0].OptionIDs) != 1 {
		t.Fatalf("missing refs: %#v", service.refs)
	}
	content, ok := result.Content.(map[string]any)
	if !ok || content["plan_hash"] == "" || content["submission_event_id"] == "" {
		t.Fatalf("unexpected result: %#v", result.Content)
	}
	if _, containsPlan := content["plan"]; containsPlan {
		t.Fatal("success response leaked plan")
	}
}

func TestReportPlanToolValidationBudgetAndBindingFailures(t *testing.T) {
	service := &reportPlanMCPService{fakeMCPService: &fakeMCPService{}}
	options := []Option{WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_tool", AgentExecutor: "codex"}), WithReportPlanBinding(testReportPlanBinding()), WithEnabledTools([]string{ToolReportPlanSubmit})}
	server := NewServer(service, options...)
	base := `{"mission_id":"mis_1","session_id":"ses_tool","pending_event_id":"evt_pending","report_mode":"%s","idempotency_key":"key_1","producer":{"type":"agent_session","id":"ses_tool"},"plan":{"summary":"summary","sections":[]}}`
	bindingFailure := server.Call(context.Background(), ToolCall{Name: ToolReportPlanSubmit, Arguments: json.RawMessage(`{"mission_id":"mis_other"}`)})
	if bindingFailure.Error == nil || bindingFailure.Error.ErrorKind != "binding" || bindingFailure.Error.Retryable {
		t.Fatalf("unexpected binding failure: %#v", bindingFailure.Error)
	}
	if server.reportPlanAttemptCount() != 1 {
		t.Fatalf("parsed binding failure was not charged: %d", server.reportPlanAttemptCount())
	}
	server = NewServer(service, options...)
	for attempt := 1; attempt <= 3; attempt++ {
		result := server.Call(context.Background(), ToolCall{Name: ToolReportPlanSubmit, Arguments: json.RawMessage(sprintf(base, "mystery"))})
		if result.Error == nil || result.Error.ErrorKind != "validation" || result.Error.Retryable != (attempt < 3) {
			t.Fatalf("attempt %d: %#v", attempt, result.Error)
		}
	}
	fourth := server.Call(context.Background(), ToolCall{Name: ToolReportPlanSubmit, Arguments: json.RawMessage(sprintf(base, "planned"))})
	if fourth.Error == nil || fourth.Error.Retryable || service.submission.EventID != "" {
		t.Fatalf("valid fourth call was not blocked before storage: %#v %#v", fourth.Error, service.submission)
	}
}

func TestReportPlanToolSuccessReplayAndStrictDecodingConsumeParsedBudget(t *testing.T) {
	service := &reportPlanMCPService{fakeMCPService: &fakeMCPService{}}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_tool", AgentExecutor: "codex"}), WithReportPlanBinding(testReportPlanBinding()), WithEnabledTools([]string{ToolReportPlanSubmit}))
	valid := json.RawMessage(`{"mission_id":"mis_1","session_id":"ses_tool","pending_event_id":"evt_pending","report_mode":"planned","idempotency_key":"key_1","producer":{"type":"agent_session","id":"ses_tool"},"plan":{"summary":"summary"}}`)
	for call := 0; call < 2; call++ {
		if result := server.Call(context.Background(), ToolCall{Name: ToolReportPlanSubmit, Arguments: valid}); result.Error != nil {
			t.Fatalf("valid replay failed: %#v", result.Error)
		}
	}
	if server.reportPlanAttemptCount() != 2 || service.submissions != 2 {
		t.Fatalf("success/replay parsed calls were not charged: calls=%d storage=%d", server.reportPlanAttemptCount(), service.submissions)
	}
	unknown := json.RawMessage(`{"mission_id":"mis_1","session_id":"ses_tool","pending_event_id":"evt_pending","report_mode":"planned","idempotency_key":"key_1","producer":{"type":"agent_session","id":"ses_tool"},"plan":{"summary":"summary","unknown":true}}`)
	if result := server.Call(context.Background(), ToolCall{Name: ToolReportPlanSubmit, Arguments: unknown}); result.Error == nil || result.Error.ErrorKind != "validation" || result.Error.Retryable {
		t.Fatalf("unknown nested field bypassed strict decoding: %#v", result.Error)
	}
	if fourth := server.Call(context.Background(), ToolCall{Name: ToolReportPlanSubmit, Arguments: valid}); fourth.Error == nil || fourth.Error.Retryable || service.submissions != 2 {
		t.Fatalf("fourth parsed call reached storage: %#v storage=%d", fourth.Error, service.submissions)
	}
}

func TestReportPlanSchemaClosesEveryObjectBoundary(t *testing.T) {
	text := string(schemaReportPlanSubmit)
	if strings.Count(text, `"additionalProperties":false`) != 8 || !strings.Contains(text, `"type":"object",
  "additionalProperties":false`) || !strings.Contains(text, `"const":"planned"`) || !strings.Contains(text, `"const":"long_form"`) {
		t.Fatalf("report plan schema is not closed and mode-discriminated: %s", text)
	}
}

func TestValidateReportPlanBindingRejectsPartialAndConflictingBindings(t *testing.T) {
	binding := Binding{MissionID: "mis_1", AgentSessionID: "ses_tool", AgentExecutor: "codex"}
	if err := ValidateReportPlanBinding(binding, ReportPlanBinding{PendingEventID: "evt_pending"}); err == nil {
		t.Fatal("expected partial binding to fail")
	}
	plan := testReportPlanBinding()
	plan.ToolSessionID = "ses_other"
	if err := ValidateReportPlanBinding(binding, plan); err == nil {
		t.Fatal("expected conflicting binding to fail")
	}
	if err := ValidateReportPlanBinding(binding, testReportPlanBinding()); err != nil {
		t.Fatalf("complete binding failed: %v", err)
	}
}

func TestReportPlanToolStdioExposureSubmissionAndProtocolBudget(t *testing.T) {
	service := &reportPlanMCPService{fakeMCPService: &fakeMCPService{}}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_tool", AgentExecutor: "codex"}), WithReportPlanBinding(testReportPlanBinding()), WithEnabledTools([]string{ToolMissionGet, ToolReportPlanSubmit}))
	validArgs := `{"mission_id":"mis_1","session_id":"ses_tool","pending_event_id":"evt_pending","report_mode":"planned","idempotency_key":"key_1","producer":{"type":"agent_session","id":"ses_tool"},"plan":{"summary":"summary","sections":[]}}`
	invalidEnvelopeInput := strings.Join([]string{
		`{"id":91,"method":"tools/call","params":{"name":"plasma.report.plan.submit","arguments":` + validArgs + `}}`,
		`{"jsonrpc":"1.0","id":92,"method":"tools/call","params":{"name":"plasma.report.plan.submit","arguments":` + validArgs + `}}`,
		`{"jsonrpc":"2.0","id":{"invalid":true},"method":"tools/call","params":{"name":"plasma.report.plan.submit","arguments":` + validArgs + `}}`,
	}, "\n") + "\n"
	var invalidOutput bytes.Buffer
	if err := ServeStdio(context.Background(), strings.NewReader(invalidEnvelopeInput), &invalidOutput, server); err != nil {
		t.Fatal(err)
	}
	invalidLines := strings.Split(strings.TrimSpace(invalidOutput.String()), "\n")
	if len(invalidLines) != 3 || server.reportPlanAttemptCount() != 0 || service.submissions != 0 {
		t.Fatalf("invalid envelopes reached parsed-call accounting: output=%s calls=%d storage=%d", invalidOutput.String(), server.reportPlanAttemptCount(), service.submissions)
	}
	for index, line := range invalidLines {
		if !strings.Contains(line, `"code":-32600`) || !strings.Contains(line, `"message":"invalid request"`) {
			t.Fatalf("invalid envelope %d did not return invalid-request: %s", index, line)
		}
	}
	if !strings.Contains(invalidLines[2], `"id":null`) {
		t.Fatalf("invalid request ID was reflected in the response: %s", invalidLines[2])
	}
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":"invalid"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"plasma.report.plan.submit","arguments":{"mission_id":"mis_1","session_id":"ses_tool","pending_event_id":"evt_pending","report_mode":"mystery","idempotency_key":"key_1","producer":{"type":"agent_session","id":"ses_tool"},"plan":{}}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"plasma.report.plan.submit","arguments":` + validArgs + `}}`,
		`{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"plasma.report.plan.submit","arguments":` + validArgs + `}}`,
		`{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"plasma.report.plan.submit","arguments":` + validArgs + `}}`,
	}, "\n") + "\n"
	var output bytes.Buffer
	if err := ServeStdio(context.Background(), strings.NewReader(input), &output, server); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 6 {
		t.Fatalf("unexpected stdio output: %s", output.String())
	}
	if !strings.Contains(lines[0], ToolMissionGet) || !strings.Contains(lines[0], ToolReportPlanSubmit) {
		t.Fatalf("tool list missing bound tools: %s", lines[0])
	}
	if !strings.Contains(lines[1], `\"error_kind\":\"protocol\"`) || !strings.Contains(lines[1], `\"retryable\":false`) {
		t.Fatalf("params failure was not protocol error: %s", lines[1])
	}
	if !strings.Contains(lines[2], `\"error_kind\":\"validation\"`) || !strings.Contains(lines[2], `\"retryable\":true`) {
		t.Fatalf("first parsed validation did not retain retry: %s", lines[2])
	}
	if !strings.Contains(lines[3], `\"submission_event_id\"`) {
		t.Fatalf("valid call did not submit after one charged attempt: %s %#v", lines[3], service.submission)
	}
	if !strings.Contains(lines[4], `\"submission_event_id\"`) || !strings.Contains(lines[5], `\"retryable\":false`) || service.submissions != 2 || service.submission.Attempt != 3 {
		t.Fatalf("third replay/fourth rejection contract failed: %s %s storage=%d", lines[4], lines[5], service.submissions)
	}
}

func sprintf(format, value string) string {
	encoded, _ := json.Marshal(value)
	_ = encoded
	return stringReplaceOnce(format, "%s", value)
}

func stringReplaceOnce(value, old, replacement string) string {
	for i := 0; i+len(old) <= len(value); i++ {
		if value[i:i+len(old)] == old {
			return value[:i] + replacement + value[i+len(old):]
		}
	}
	return value
}
