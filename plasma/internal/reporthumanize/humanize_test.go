package reporthumanize

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/agentexec"
	"github.com/c86j224s/liquid2/plasma/internal/artifact"
	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/reportexecution"
	"github.com/c86j224s/liquid2/plasma/internal/reportpatch"
)

type testService struct {
	artifacts map[string]artifact.Raw
	events    []ledger.Event
	seq       int64
}

func (svc *testService) GetRawArtifact(_ context.Context, artifactID string) (artifact.Raw, error) {
	return svc.artifacts[artifactID], nil
}

func (svc *testService) AppendEvent(_ context.Context, req ledger.AppendRequest) (ledger.Event, error) {
	return svc.append(req), nil
}

func (svc *testService) AppendReportTerminalIfOpen(_ context.Context, missionID string, pendingEventID string, reqs []ledger.AppendRequest) ([]ledger.Event, bool, error) {
	if _, ok := reportexecution.CompletedPendingEventIDs(svc.events)[pendingEventID]; ok {
		return nil, false, nil
	}
	appended := make([]ledger.Event, 0, len(reqs))
	for _, req := range reqs {
		appended = append(appended, svc.append(req))
	}
	return appended, true, nil
}

func (svc *testService) ListEvents(_ context.Context, missionID string) ([]ledger.Event, error) {
	var events []ledger.Event
	for _, event := range svc.events {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (svc *testService) append(req ledger.AppendRequest) ledger.Event {
	svc.seq++
	event := ledger.Event{
		EventID:          req.EventID,
		MissionID:        req.MissionID,
		Sequence:         svc.seq,
		EventType:        req.EventType,
		Producer:         req.Producer,
		CausationEventID: req.CausationEventID,
		CorrelationID:    req.CorrelationID,
		Payload:          req.Payload,
	}
	svc.events = append(svc.events, event)
	return event
}

type testExecutor struct {
	result  agentexec.AgentResult
	request agentexec.AgentRequest
}

func (executor *testExecutor) Run(_ context.Context, req agentexec.AgentRequest) (agentexec.AgentResult, error) {
	executor.request = req
	return executor.result, nil
}

func TestHumanizeNoChangesSkipsAndUsesPatchToolSurface(t *testing.T) {
	svc := &testService{}
	executor := &testExecutor{result: agentexec.AgentResult{Text: "NO_H5_CHANGES", SessionID: "ses_report"}}
	result, err := HumanizeMarkdownReport(context.Background(), svc, testID, "mis_1", Input{
		Title:             "Report",
		Markdown:          "# Report\n\n수행되어야 한다.",
		SourceArtifact:    artifact.Raw{ArtifactID: "art_source", MissionID: "mis_1", MediaType: "text/markdown", SHA256: "sha"},
		ExecutorName:      "codex",
		MCPMode:           "auto",
		PreviousSessionID: "ses_report",
		ReportMode:        reportexecution.ModePlanned,
		PendingEventID:    "evt_report",
	}, executor)
	if err != nil {
		t.Fatalf("HumanizeMarkdownReport returned error: %v", err)
	}
	if result.Applied {
		t.Fatalf("expected no-op H5 to skip, got %#v", result)
	}
	if got := eventTypes(svc.events); !reflect.DeepEqual(got, []string{"report.humanize.pending", "report.humanize.skipped"}) {
		t.Fatalf("event types = %#v", got)
	}
	if executor.request.UserText != "humanize finalized markdown report tone" {
		t.Fatalf("unexpected user text: %q", executor.request.UserText)
	}
	if !reflect.DeepEqual(executor.request.ExtraMCPTools, reportpatch.MCPTools()) {
		t.Fatalf("patch tools = %#v", executor.request.ExtraMCPTools)
	}
	if !executor.request.ReplaceMCPTools {
		t.Fatalf("expected H5 to replace MCP tool surface")
	}
	if executor.request.ReportPatch == nil ||
		executor.request.ReportPatch.ReportSessionPolicy != reportexecution.SessionPolicySameSession ||
		executor.request.ReportPatch.ReportSessionPolicySelection != "auto_same_report_session_h5" ||
		executor.request.ReportPatch.SessionChainKind != "same_report_session_h5_humanize_patch" {
		t.Fatalf("unexpected report patch context: %#v", executor.request.ReportPatch)
	}
}

func TestHumanizeSessionMismatchFailsTerminally(t *testing.T) {
	svc := &testService{}
	executor := &testExecutor{result: agentexec.AgentResult{Text: "done", SessionID: "ses_other"}}
	result, err := HumanizeMarkdownReport(context.Background(), svc, testID, "mis_1", Input{
		Title:             "Report",
		Markdown:          "# Report\n\n수행되어야 한다.",
		SourceArtifact:    artifact.Raw{ArtifactID: "art_source", MissionID: "mis_1", MediaType: "text/markdown", SHA256: "sha"},
		ExecutorName:      "codex",
		PreviousSessionID: "ses_report",
		ReportMode:        reportexecution.ModePlanned,
		PendingEventID:    "evt_report",
	}, executor)
	if err != nil {
		t.Fatalf("HumanizeMarkdownReport returned error: %v", err)
	}
	if result.Applied {
		t.Fatalf("expected mismatch to preserve original artifact, got %#v", result)
	}
	if got := eventTypes(svc.events); !reflect.DeepEqual(got, []string{"report.humanize.pending", "report.humanize.failed"}) {
		t.Fatalf("event types = %#v", got)
	}
	payload := eventPayload(t, svc.events[len(svc.events)-1])
	if !strings.Contains(payload["error"].(string), "agent returned a different session id") ||
		payload["preserved_original_markdown"] != true {
		t.Fatalf("unexpected failure payload: %#v", payload)
	}
}

func testID(prefix string) string {
	return prefix + "_test"
}

func eventTypes(events []ledger.Event) []string {
	types := make([]string, len(events))
	for i, event := range events {
		types[i] = event.EventType
	}
	return types
}

func eventPayload(t *testing.T, event ledger.Event) map[string]any {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	return payload
}
