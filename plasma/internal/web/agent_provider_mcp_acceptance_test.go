package web

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	plasmamcp "github.com/c86j224s/liquid2/plasma/internal/mcp"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
	"github.com/c86j224s/liquid2/plasma/internal/reportprompt"
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

type combinedProviderWebExecutor struct {
	mu             sync.Mutex
	provider       AgentExecutor
	writes         []AgentResult
	providerErrors []string
	providerLogs   []string
	forkSequence   int
}

func (executor *combinedProviderWebExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	if req.ReportPlan != nil || req.ReportRequirements != nil || req.PartAssembly != nil || req.PartEdit != nil || req.FinalEditStage != nil || req.LongFormFinalize != nil {
		result, err := executor.provider.Run(ctx, req)
		executor.mu.Lock()
		if err != nil {
			executor.providerErrors = append(executor.providerErrors, err.Error())
		} else {
			executor.providerErrors = append(executor.providerErrors, "")
		}
		executor.providerLogs = append(executor.providerLogs, result.Log)
		executor.mu.Unlock()
		return result, err
	}
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if len(executor.writes) == 0 {
		return AgentResult{}, context.Canceled
	}
	result := executor.writes[0]
	executor.writes = executor.writes[1:]
	return result, nil
}

func (executor *combinedProviderWebExecutor) snapshotProviderObservations() ([]string, []string) {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	return append([]string(nil), executor.providerErrors...), append([]string(nil), executor.providerLogs...)
}

func (executor *combinedProviderWebExecutor) CheckForkSession(_ context.Context, sourceSessionID string) error {
	if strings.TrimSpace(sourceSessionID) == "" {
		return fmt.Errorf("source session id is required")
	}
	return nil
}

func (executor *combinedProviderWebExecutor) ForkSession(_ context.Context, sourceSessionID string) (AgentSessionForkResult, error) {
	if err := executor.CheckForkSession(context.Background(), sourceSessionID); err != nil {
		return AgentSessionForkResult{}, err
	}
	executor.mu.Lock()
	defer executor.mu.Unlock()
	executor.forkSequence++
	sessionID := fmt.Sprintf("33333333-3333-4333-8333-%012d", executor.forkSequence)
	return AgentSessionForkResult{SessionID: sessionID, SourceSessionID: strings.TrimSpace(sourceSessionID)}, nil
}

func TestRealProviderExecutorsSpawnBoundPlasmaMCP(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	binary := filepath.Join(t.TempDir(), "plasma")
	build := exec.Command("go", "build", "-o", binary, "./cmd/plasma")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build plasma: %v: %s", err, output)
	}
	shim := writeProviderMCPShim(t)
	for _, tc := range []struct {
		name, mode, executorName string
		makeExecutor             func(string, string, string) AgentExecutor
	}{
		{name: "codex", mode: reportModePlanned, executorName: "codex", makeExecutor: func(shim, binary, database string) AgentExecutor {
			return CodexExecutor{Command: shim, WorkDir: t.TempDir(), Timeout: 10 * time.Second, Env: os.Environ(), MCPServer: CodexMCPServer{Name: "plasma", Command: binary, Args: []string{"mcp", "-db", database}, Required: true}}
		}},
		{name: "claude", mode: reportModeLongForm, executorName: "claude", makeExecutor: func(shim, binary, database string) AgentExecutor {
			return ClaudeExecutor{Command: shim, WorkDir: t.TempDir(), Timeout: 10 * time.Second, Env: os.Environ(), MCPServer: ClaudeMCPServer{Name: "plasma", Command: binary, Args: []string{"mcp", "-db", database}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			database := filepath.Join(t.TempDir(), "plasma.db")
			store, err := sqlite.Open(ctx, database)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			service := app.NewService(store)
			if _, err := service.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_acceptance", Title: "Acceptance"}); err != nil {
				t.Fatal(err)
			}
			if _, err := service.AppendEvent(ctx, app.BuildMissionCreatedAppendRequest(app.MissionCreatedEventRequest{
				EventID: "evt_mission", MissionID: "mis_acceptance", Title: "Acceptance", Objective: "Verify provider path", Producer: app.Producer{Type: "user", ID: "test"},
			})); err != nil {
				t.Fatal(err)
			}
			pendingPayload, _ := json.Marshal(map[string]any{"kind": "markdown_report_artifact_pending", "report_mode": tc.mode, "agent_executor": tc.executorName})
			if _, err := service.AppendEvent(ctx, app.AppendEventRequest{EventID: "evt_pending", MissionID: "mis_acceptance", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: pendingPayload}); err != nil {
				t.Fatal(err)
			}
			req := AgentRequest{
				Prompt: "Submit the bound plan.", Model: "test-model", ReasoningEffort: "high", MissionID: "mis_acceptance",
				ToolSessionID: "ses_tool", AgentExecutor: tc.executorName, ExtraMCPTools: []string{plasmamcp.ToolReportPlanSubmit}, ReplaceMCPTools: true,
				ReportPlan: &AgentReportPlanContext{PendingEventID: "evt_pending", ReportMode: tc.mode, IdempotencyKey: "key_acceptance", AgentModel: "test-model", AgentReasoningEffort: "high"},
			}
			result, err := tc.makeExecutor(shim, binary, database).Run(ctx, req)
			if err != nil {
				t.Fatalf("%v; log:\n%s", err, result.Log)
			}
			if result.Text != reporting.ReportPlanSubmittedSentinel || strings.TrimSpace(result.SessionID) == "" {
				t.Fatalf("provider shim did not transmit sentinel/session: %#v", result)
			}
			events, err := service.ListEvents(ctx, "mis_acceptance")
			if err != nil {
				t.Fatal(err)
			}
			if countLedgerEventType(events, "report.plan.submitted") != 1 {
				t.Fatalf("real executor path did not persist one submission: %#v", events)
			}
		})
	}
}

func TestRealProviderExecutorsSpawnBoundRequirementMCP(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	binary := filepath.Join(t.TempDir(), "plasma")
	build := exec.Command("go", "build", "-o", binary, "./cmd/plasma")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build plasma: %v: %s", err, output)
	}
	shim := writeProviderMCPShim(t)
	for _, tc := range []struct {
		name, executorName string
		makeExecutor       func(string, string, string) AgentExecutor
	}{
		{name: "codex", executorName: "codex", makeExecutor: func(shim, binary, database string) AgentExecutor {
			return CodexExecutor{Command: shim, WorkDir: t.TempDir(), Timeout: 10 * time.Second, Env: os.Environ(), MCPServer: CodexMCPServer{Name: "plasma", Command: binary, Args: []string{"mcp", "-db", database}, Required: true}}
		}},
		{name: "claude", executorName: "claude", makeExecutor: func(shim, binary, database string) AgentExecutor {
			return ClaudeExecutor{Command: shim, WorkDir: t.TempDir(), Timeout: 10 * time.Second, Env: os.Environ(), MCPServer: ClaudeMCPServer{Name: "plasma", Command: binary, Args: []string{"mcp", "-db", database}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			database := filepath.Join(t.TempDir(), "plasma.db")
			store, err := sqlite.Open(ctx, database)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			service := app.NewService(store)
			if _, err := service.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_acceptance", Title: "Acceptance"}); err != nil {
				t.Fatal(err)
			}
			if _, err := service.AppendEvent(ctx, app.BuildMissionCreatedAppendRequest(app.MissionCreatedEventRequest{
				EventID: "evt_mission", MissionID: "mis_acceptance", Title: "Acceptance", Objective: "Verify requirement provider path", Producer: app.Producer{Type: "user", ID: "test"},
			})); err != nil {
				t.Fatal(err)
			}
			pendingPayload, _ := json.Marshal(map[string]any{"kind": "markdown_report_artifact_pending", "report_mode": reportModeLongForm, "agent_executor": tc.executorName})
			if _, err := service.AppendEvent(ctx, app.AppendEventRequest{EventID: "evt_pending", MissionID: "mis_acceptance", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: pendingPayload}); err != nil {
				t.Fatal(err)
			}
			planPayload := json.RawMessage(`{"pending_event_id":"evt_pending","report_mode":"long_form","plan":{"parts":[{"title":"Part","sections":[{"title":"Section"}]}]}}`)
			if _, err := service.AppendEvent(ctx, app.AppendEventRequest{EventID: "evt_plan", MissionID: "mis_acceptance", EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "ses_plan"}, Payload: planPayload}); err != nil {
				t.Fatal(err)
			}
			binding := reporting.ReportRequirementMapBinding{
				MissionID: "mis_acceptance", PendingEventID: "evt_pending", PlanEventID: "evt_plan", ToolSessionID: "ses_tool",
				PreviousProviderSessionID: "ses_plan", IdempotencyKey: "rrk_acceptance", AgentExecutor: tc.executorName,
				AgentModel: "test-model", AgentReasoningEffort: "high", Producer: app.Producer{Type: "agent_session", ID: "ses_tool"},
			}
			req := AgentRequest{
				Prompt: "Submit the bound requirements.", Model: "test-model", ReasoningEffort: "high", MissionID: "mis_acceptance",
				ToolSessionID: "ses_tool", AgentExecutor: tc.executorName, ExtraMCPTools: []string{plasmamcp.ToolReportRequirementsSubmit},
				ReplaceMCPTools: true, ReportRequirements: &binding,
			}
			result, err := tc.makeExecutor(shim, binary, database).Run(ctx, req)
			if err != nil {
				t.Fatalf("%v; log:\n%s", err, result.Log)
			}
			if result.Text != reporting.ReportRequirementsMappedSentinel || strings.TrimSpace(result.SessionID) == "" {
				t.Fatalf("provider shim did not transmit requirement sentinel/session: %#v", result)
			}
			events, err := service.ListEvents(ctx, "mis_acceptance")
			if err != nil {
				t.Fatal(err)
			}
			if countLedgerEventType(events, reporting.ReportRequirementsMappedEventType) != 1 {
				t.Fatalf("real executor path did not persist one requirement map: %#v", events)
			}
		})
	}
}

func TestAgentProviderMCPRealProviderExecutorsSpawnBoundPartEdit(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	binary := filepath.Join(t.TempDir(), "plasma")
	build := exec.Command("go", "build", "-o", binary, "./cmd/plasma")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build plasma: %v: %s", err, output)
	}
	shim := writeProviderMCPShim(t)
	for _, tc := range []struct {
		name, executorName string
		makeExecutor       func(string, string, string) AgentExecutor
	}{
		{name: "codex", executorName: "codex", makeExecutor: func(shim, binary, database string) AgentExecutor {
			return CodexExecutor{Command: shim, WorkDir: t.TempDir(), Timeout: 10 * time.Second, Env: os.Environ(), MCPServer: CodexMCPServer{Name: "plasma", Command: binary, Args: []string{"mcp", "-db", database}, Required: true}}
		}},
		{name: "claude", executorName: "claude", makeExecutor: func(shim, binary, database string) AgentExecutor {
			return ClaudeExecutor{Command: shim, WorkDir: t.TempDir(), Timeout: 10 * time.Second, Env: os.Environ(), MCPServer: ClaudeMCPServer{Name: "plasma", Command: binary, Args: []string{"mcp", "-db", database}}}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			database := filepath.Join(t.TempDir(), "plasma.db")
			store, err := sqlite.Open(ctx, database)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			service := app.NewService(store)
			seedProviderPartEditAcceptance(t, ctx, service)
			binding := reporting.PartEditBinding{
				MissionID: "mis_acceptance", PendingEventID: "evt_pending", PlanEventID: "evt_plan",
				SourcePartEventID: "evt_part", SourceArtifactID: "art_part", EditedArtifactID: "art_part_edit",
				Filename: "part-edited.md", ToolSessionID: "ses_tool", ProviderSessionID: "provider-editor",
				PreviousProviderSessionID: "provider-editor", IdempotencyKey: "report-part-edit:evt_pending:evt_plan:1", PartIndex: 1,
				AgentExecutor: tc.executorName, AgentModel: "test-model", AgentReasoningEffort: "high",
				MCPMode: "auto", ReportSessionPolicy: reportSessionPolicyIsolatedFork,
				GenerationGuidanceProfile: reportprompt.ProfileNarrativeContract,
				SessionChainKind:          "section_fanout_report", ReportPlanSessionID: "provider-plan",
				ForkSourceAgentSessionID: "provider-plan",
			}
			req := AgentRequest{
				Prompt: "Submit the bound Part edit.", Model: "test-model", ReasoningEffort: "high", MissionID: "mis_acceptance",
				ToolSessionID: "ses_tool", PreviousSessionID: "provider-editor", AgentExecutor: tc.executorName,
				ExtraMCPTools: []string{
					plasmamcp.ToolReportPartEditStart,
					plasmamcp.ToolReportPartEditRead,
					plasmamcp.ToolReportPartEditPatch,
					plasmamcp.ToolReportPartEditSubmit,
				},
				ReplaceMCPTools: true, PartEdit: &binding,
			}
			result, err := tc.makeExecutor(shim, binary, database).Run(ctx, req)
			if err != nil {
				t.Fatalf("%v; log:\n%s", err, result.Log)
			}
			if result.Text != reporting.PartEditSubmittedSentinel || strings.TrimSpace(result.SessionID) == "" {
				t.Fatalf("provider shim did not transmit Part edit sentinel/session: %#v", result)
			}
			events, err := service.ListEvents(ctx, "mis_acceptance")
			if err != nil {
				t.Fatal(err)
			}
			if countLedgerEventType(events, reporting.PartEditedEventType) != 1 {
				t.Fatalf("real executor path did not persist one Part edit: %#v", events)
			}
			payload := ledgerEventPayload(t, events, reporting.PartEditedEventType)
			if payload["source_part_event_id"] != "evt_part" ||
				payload["source_artifact_id"] != "art_part" ||
				payload["artifact_id"] != "art_part_edit" ||
				payload["changed"] != true ||
				payload["agent_executor"] != tc.executorName {
				t.Fatalf("Part edit binding identity was not preserved: %#v", payload)
			}
		})
	}
}

func TestWebReportAPIUsesRealProviderExecutorsAndBuiltMCP(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	binary := filepath.Join(t.TempDir(), "plasma")
	build := exec.Command("go", "build", "-o", binary, "./cmd/plasma")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build plasma: %v: %s", err, output)
	}
	shim := writeProviderMCPShim(t)
	for _, tc := range []struct {
		name, mode, executorName, model, providerSession string
		makeProvider                                     func(string, string, string) AgentExecutor
		writes                                           []AgentResult
	}{
		{
			name: "planned codex", mode: reportModePlanned, executorName: "codex", model: "codex-model-id", providerSession: "provider-codex-session",
			makeProvider: func(shim, binary, database string) AgentExecutor {
				return CodexExecutor{Command: shim, WorkDir: t.TempDir(), Timeout: 10 * time.Second, Env: os.Environ(), MCPServer: CodexMCPServer{Name: "plasma", Command: binary, Args: []string{"mcp", "-db", database}, Required: true}}
			},
			writes: []AgentResult{{Text: "# Planned report\n\nBody.", SessionID: "provider-codex-session"}},
		},
		{
			name: "long form claude", mode: reportModeLongForm, executorName: "claude", model: "claude-model-id", providerSession: "22222222-2222-4222-8222-222222222222",
			makeProvider: func(shim, binary, database string) AgentExecutor {
				return ClaudeExecutor{Command: shim, WorkDir: t.TempDir(), Timeout: 10 * time.Second, Env: append(os.Environ(), "PLASMA_TEST_FINAL_ACK=ACK_NOT_EXACT"), MCPServer: ClaudeMCPServer{Name: "plasma", Command: binary, Args: []string{"mcp", "-db", database}}}
			},
			writes: []AgentResult{
				{Text: "Part owner brief.", SessionID: "33333333-3333-4333-8333-000000000001", Resumed: true},
				{Text: "Section body.", SessionID: "33333333-3333-4333-8333-000000000002", Resumed: true},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			database := filepath.Join(t.TempDir(), "plasma.db")
			store, err := sqlite.Open(ctx, database)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			service := app.NewService(store)
			executor := &combinedProviderWebExecutor{provider: tc.makeProvider(shim, binary, database), writes: append([]AgentResult(nil), tc.writes...)}
			server := httptest.NewServer(NewServer(service, Options{AgentExecutors: map[string]AgentExecutor{tc.executorName: executor}}))
			defer server.Close()
			mission := postJSON(t, server.URL+"/api/missions", map[string]any{"title": "Combined provider acceptance"})
			missionID := nestedString(t, mission, "projection", "mission_id")
			reportRequest := map[string]any{
				"title": "Report", "report_mode": tc.mode, "agent_executor": tc.executorName, "agent_model": tc.model,
			}
			if tc.executorName == "codex" {
				reportRequest["agent_reasoning_effort"] = "high"
			}
			if tc.mode == reportModeLongForm {
				reportRequest["execution_strategy"] = reportExecutionStrategySectionFanout
				reportRequest["generation_guidance_profile"] = reportprompt.ProfilePartConnectiveEconomyVoice
				reportRequest["post_report_humanize"] = "disabled"
			}
			postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", reportRequest)
			var detail map[string]any
			deadline := time.Now().Add(3 * time.Second)
			for time.Now().Before(deadline) {
				detail = getJSON(t, server.URL+"/api/missions/"+missionID)
				if countEvents(detail, "report.artifact.created") >= 1 {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			if countEvents(detail, "report.artifact.created") < 1 {
				errors, logs := executor.snapshotProviderObservations()
				t.Fatalf("timed out waiting for report.artifact.created: provider_errors=%#v provider_logs=%#v events=%#v", errors, logs, detail["events"])
			}
			if tc.mode == reportModeLongForm {
				time.Sleep(100 * time.Millisecond)
				detail = getJSON(t, server.URL+"/api/missions/"+missionID)
				if countEvents(detail, "report.artifact.created") != 1 || countEvents(detail, "report.draft.failed") != 0 || countEvents(detail, "turn.agent.response") != 0 {
					t.Fatalf("built provider acknowledgement anomaly contradicted canonical success: %#v", detail["events"])
				}
			}
			if countEvents(detail, "report.plan.submitted") != 1 || countEvents(detail, "report.plan.created") != 1 {
				t.Fatalf("combined provider path did not create exactly one submission and canonical: %#v", detail["events"])
			}
			if tc.mode == reportModeLongForm {
				if countEvents(detail, reporting.PartPlanCreatedEventType) != 1 ||
					countEvents(detail, reporting.PartEditStartedEventType) != 1 ||
					countEvents(detail, reporting.PartEditedEventType) != 1 {
					t.Fatalf("combined provider W4 path did not use Part owner authoring: %#v", detail["events"])
				}
				planPayload := lastEventPayload(t, detail, "report.plan.created")
				if planPayload["part_planning_enabled"] != true || planPayload["part_edit_enabled"] != true {
					t.Fatalf("combined provider W4 plan payload missing flags: %#v", planPayload)
				}
			}
			for _, raw := range detail["events"].([]any) {
				event := raw.(map[string]any)
				if event["EventType"] != "report.plan.created" {
					continue
				}
				payload := nestedMap(t, event, "Payload")
				if payload["agent_session_id"] != tc.providerSession || payload["returned_agent_session_id"] != tc.providerSession || payload["agent_model"] != tc.model {
					t.Fatalf("canonical provider lineage/model is not truthful: %#v", payload)
				}
			}
		})
	}
}

func countLedgerEventType(events []app.LedgerEvent, eventType string) int {
	count := 0
	for _, event := range events {
		if event.EventType == eventType {
			count++
		}
	}
	return count
}

func seedProviderPartEditAcceptance(t *testing.T, ctx context.Context, service *app.Service) {
	t.Helper()
	if _, err := service.CreateMission(ctx, app.CreateMissionRequest{MissionID: "mis_acceptance", Title: "Acceptance"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AppendEvent(ctx, app.BuildMissionCreatedAppendRequest(app.MissionCreatedEventRequest{
		EventID: "evt_mission", MissionID: "mis_acceptance", Title: "Acceptance", Objective: "Verify Part edit provider path", Producer: app.Producer{Type: "user", ID: "test"},
	})); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateRawArtifact(ctx, app.CreateRawArtifactRequest{
		ArtifactID: "art_part", MissionID: "mis_acceptance", MediaType: "text/markdown; charset=utf-8",
		Filename: "part.md", Producer: app.Producer{Type: "agent_session", ID: "provider-part"},
		Content: []byte("# Part 1\n\nSource body.\n"),
	}); err != nil {
		t.Fatal(err)
	}
	for _, req := range []app.AppendEventRequest{
		{EventID: "evt_pending", MissionID: "mis_acceptance", EventType: "report.draft.pending", Producer: app.Producer{Type: "user", ID: "test"}, Payload: mustJSON(map[string]any{"report_mode": reportModeLongForm, "agent_executor": "codex"})},
		{EventID: "evt_plan", MissionID: "mis_acceptance", EventType: "report.plan.created", Producer: app.Producer{Type: "agent_session", ID: "provider-plan"}, Payload: mustJSON(map[string]any{
			"pending_event_id": "evt_pending", "report_mode": reportModeLongForm, "artifact_id": "art_final", "part_edit_enabled": true,
			"plan": narrativeContractTestPlan(),
		})},
		{EventID: "evt_part", MissionID: "mis_acceptance", EventType: "report.part.created", Producer: app.Producer{Type: "agent_session", ID: "provider-part"}, Payload: mustJSON(map[string]any{
			"pending_event_id": "evt_pending", "plan_event_id": "evt_plan", "artifact_id": "art_part", "part_index": 1,
		})},
	} {
		if _, err := service.AppendEvent(ctx, req); err != nil {
			t.Fatal(err)
		}
	}
}

func ledgerEventPayload(t *testing.T, events []app.LedgerEvent, eventType string) map[string]any {
	t.Helper()
	for _, event := range events {
		if event.EventType != eventType {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal(event.Payload, &payload); err != nil {
			t.Fatal(err)
		}
		return payload
	}
	t.Fatalf("missing event type %s in %#v", eventType, events)
	return nil
}

func writeProviderMCPShim(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "provider-shim.py")
	script := `#!/usr/bin/env python3
import json, os, subprocess, sys
argv=sys.argv[1:]; command=None; args=None; out=None; kind="claude"
for i,value in enumerate(argv):
  if value == "--output-last-message": out=argv[i+1]; kind="codex"
  if value == "--mcp-config":
    cfg=json.load(open(argv[i+1])); server=next(iter(cfg["mcpServers"].values())); command=server["command"]; args=server["args"]
for value in argv:
  if value.startswith("mcp_servers.plasma.command="): command=json.loads(value.split("=",1)[1])
  if value.startswith("mcp_servers.plasma.args="): args=json.loads(value.split("=",1)[1])
if not command or not args: raise SystemExit("missing generated MCP config")
def flag(name): return args[args.index(name)+1]
def resumed_session(default_session):
  return argv[argv.index("--resume")+1] if "--resume" in argv else default_session
def emit(sentinel):
  if kind == "codex":
    open(out,"w").write(sentinel)
    print(json.dumps({"type":"thread.started","thread_id":resumed_session("provider-codex-session")}))
  else:
    print(json.dumps({"type":"result","session_id":resumed_session("22222222-2222-4222-8222-222222222222"),"result":sentinel}))
final="-report-long-form-finalize-binding-json" in args
stage="-report-final-edit-stage-binding-json" in args
narrative_final="plasma.report.long_form.final_edit.start" in args
part="-report-part-assembly-binding-json" in args
part_edit="-report-part-edit-binding-json" in args
requirements="-report-requirements-binding-json" in args
if part:
  binding=json.loads(flag("-report-part-assembly-binding-json"))
  draft_id="rpa_testshim"
  producer={"type":"agent_session","id":binding["tool_session_id"]}
  base={"mission_id":binding["mission_id"],"session_id":binding["tool_session_id"],"producer":producer}
  start={**base,"pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"draft_id":draft_id,"part_index":binding["part_index"],"section_count":binding["section_count"],"idempotency_key":"part_start_key"}
  patch_intro={**base,"draft_id":draft_id,"field":"intro","markdown":"Part intro","summary":"intro","idempotency_key":"part_intro_key"}
  patch_close={**base,"draft_id":draft_id,"field":"closing","markdown":"Part close","summary":"closing","idempotency_key":"part_close_key"}
  submit={**base,"pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"draft_id":draft_id,"idempotency_key":"part_submit_key"}
  calls=[
    ("plasma.report.part_assembly.start", start, "draft_id"),
    ("plasma.report.part_assembly.patch", patch_intro, "draft_id"),
    ("plasma.report.part_assembly.patch", patch_close, "draft_id"),
    ("plasma.report.part_assembly.submit", submit, "event_id"),
  ]
  messages=[{"jsonrpc":"2.0","id":i+1,"method":"tools/call","params":{"name":tool,"arguments":arguments}} for i,(tool,arguments,_) in enumerate(calls)]
  proc=subprocess.run([command]+args,input="".join(json.dumps(x)+"\n" for x in messages),text=True,capture_output=True)
  if proc.returncode or '"isError":true' in proc.stdout or "event_id" not in proc.stdout: raise SystemExit(proc.stderr+proc.stdout)
  emit("PART_ASSEMBLY_SUBMITTED")
  raise SystemExit(0)
elif part_edit:
  binding=json.loads(flag("-report-part-edit-binding-json"))
  draft_id="rpe_testshim"
  producer={"type":"agent_session","id":binding["tool_session_id"]}
  common={"mission_id":binding["mission_id"],"session_id":binding["tool_session_id"],"producer":producer}
  bad={**common,"session_id":"ses_wrong","pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"draft_id":"rpe_bad","part_index":binding["part_index"],"source_artifact_id":binding["source_artifact_id"],"idempotency_key":"part_edit_bad_identity"}
  start={**common,"pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"draft_id":draft_id,"part_index":binding["part_index"],"source_artifact_id":binding["source_artifact_id"],"idempotency_key":"part_edit_start_key"}
  read={"mission_id":binding["mission_id"],"session_id":binding["tool_session_id"],"draft_id":draft_id,"offset":0,"max_bytes":65536}
  patch={**common,"draft_id":draft_id,"operation":"append","replacement":"\n\nEdited body.","summary":"acceptance edit","idempotency_key":"part_edit_patch_key"}
  submit={**common,"pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"draft_id":draft_id,"idempotency_key":"part_edit_submit_key"}
  calls=[
    ("tools/list",None),
    ("plasma.report.part_edit.start",bad),
    ("plasma.report.part_edit.start",start),
    ("plasma.report.part_edit.read",read),
    ("plasma.report.part_edit.patch",patch),
    ("plasma.report.part_edit.submit",submit),
  ]
  messages=[]
  for i,(tool,arguments) in enumerate(calls):
    message={"jsonrpc":"2.0","id":i+1,"method":"tools/list","params":{}} if tool=="tools/list" else {"jsonrpc":"2.0","id":i+1,"method":"tools/call","params":{"name":tool,"arguments":arguments}}
    messages.append(message)
  proc=subprocess.run([command]+args,input="".join(json.dumps(x)+"\n" for x in messages),text=True,capture_output=True)
  if proc.returncode: raise SystemExit(proc.stderr+proc.stdout)
  responses=[json.loads(line) for line in proc.stdout.splitlines() if line.strip()]
  listed=json.dumps(responses[0])
  for tool in ["plasma.report.part_edit.start","plasma.report.part_edit.read","plasma.report.part_edit.patch","plasma.report.part_edit.submit"]:
    if tool not in listed: raise SystemExit("missing Part edit tool "+tool+"\n"+proc.stdout)
  for forbidden in ["plasma.research.read","plasma.sources.read","plasma.report.part_assembly.start","plasma.report.long_form.finalize"]:
    if forbidden in listed: raise SystemExit("forbidden tool leaked "+forbidden+"\n"+proc.stdout)
  if not responses[1].get("result",{}).get("isError"): raise SystemExit("bad Part edit identity was not rejected\n"+proc.stdout)
  for response in responses[2:]:
    if response.get("result",{}).get("isError"): raise SystemExit("valid Part edit call failed\n"+proc.stdout)
  if "event_id" not in proc.stdout: raise SystemExit("Part edit submit did not create an event\n"+proc.stdout)
  emit("PART_EDIT_SUBMITTED")
  raise SystemExit(0)
elif stage:
  binding=json.loads(flag("-report-final-edit-stage-binding-json"))
  draft_id="rfe_testshim"
  producer={"type":"agent_session","id":binding["tool_session_id"]}
  common={"mission_id":binding["mission_id"],"session_id":binding["tool_session_id"],"producer":producer}
  start={**common,"pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"draft_id":draft_id,"idempotency_key":"stage_start_key"}
  read={"mission_id":binding["mission_id"],"session_id":binding["tool_session_id"],"draft_id":draft_id,"offset":0,"max_bytes":65536}
  submit={**common,"pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"draft_id":draft_id,"idempotency_key":"stage_submit_key"}
  if binding["stage"] == "final_write":
    prefix="plasma.report.long_form.final_write"; sentinel="FINAL_EDIT_STAGE_SUBMITTED"
  elif binding["stage"] == "reader_edit":
    prefix="plasma.report.long_form.reader_edit"; sentinel="FINAL_EDIT_STAGE_SUBMITTED"
  elif binding["stage"] == "style_edit":
    prefix="plasma.report.long_form.style_edit"; sentinel="FINAL_EDIT_STAGE_SUBMITTED"
  elif binding["stage"] == "style_semantic_validation":
    read_pages=[]
    offset=0
    proc=subprocess.Popen([command]+args,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
    msg_id=[0]
    def call_read_only(tool,arguments):
      msg_id[0]+=1
      proc.stdin.write(json.dumps({"jsonrpc":"2.0","id":msg_id[0],"method":"tools/call","params":{"name":tool,"arguments":arguments}})+"\n")
      proc.stdin.flush()
      line=proc.stdout.readline()
      if not line: raise SystemExit(proc.stderr.read())
      return json.loads(line)
    while True:
      page_read={**read,"offset":offset,"max_bytes":128}
      response=call_read_only("plasma.report.long_form.style_semantic_validation.read",page_read)
      if response.get("result",{}).get("isError"): raise SystemExit(json.dumps(response))
      content=response["result"]["content"][0]["text"]
      review_output=json.loads(content)
      tool_content=review_output.get("content",{})
      page_content=tool_content.get("content","") if isinstance(tool_content, dict) else tool_content
      read_pages.append(page_content)
      truncated=tool_content.get("truncated",False) if isinstance(tool_content, dict) else review_output.get("truncated",False)
      if not truncated: break
      offset=tool_content["next_offset"] if isinstance(tool_content, dict) else review_output["next_offset"]
    semantic=[]
    items=json.loads("".join(read_pages) or "[]")
    semantic=[{"paragraph_ordinal":item["paragraph_ordinal"],"verdict":"accepted_equivalent"} for item in items]
    submit={**submit,"semantic_acceptance":semantic}
    response=call_read_only("plasma.report.long_form.style_semantic_validation.submit",submit)
    proc.stdin.close()
    stderr=proc.stderr.read()
    code=proc.wait()
    if code or response.get("result",{}).get("isError") or "event_id" not in json.dumps(response): raise SystemExit(stderr+json.dumps(response))
    emit("FINAL_EDIT_STAGE_SUBMITTED")
    raise SystemExit(0)
  elif binding["stage"] == "evidence_gate":
    read_pages=[]
    offset=0
    proc=subprocess.Popen([command]+args,stdin=subprocess.PIPE,stdout=subprocess.PIPE,stderr=subprocess.PIPE,text=True)
    msg_id=[0]
    def call_read_only(tool,arguments):
      msg_id[0]+=1
      proc.stdin.write(json.dumps({"jsonrpc":"2.0","id":msg_id[0],"method":"tools/call","params":{"name":tool,"arguments":arguments}})+"\n")
      proc.stdin.flush()
      line=proc.stdout.readline()
      if not line: raise SystemExit(proc.stderr.read())
      return json.loads(line)
    while True:
      page_read={**read,"offset":offset,"max_bytes":128}
      response=call_read_only("plasma.report.long_form.evidence_gate.read",page_read)
      if response.get("result",{}).get("isError"): raise SystemExit(json.dumps(response))
      read_output=json.loads(response["result"]["content"][0]["text"])
      tool_content=read_output.get("content",{})
      page_content=tool_content.get("content","") if isinstance(tool_content, dict) else tool_content
      read_pages.append(page_content)
      truncated=tool_content.get("truncated",False) if isinstance(tool_content, dict) else read_output.get("truncated",False)
      if not truncated: break
      offset=tool_content["next_offset"] if isinstance(tool_content, dict) else read_output["next_offset"]
    tool_content="".join(read_pages)
    packet_content=tool_content.get("content",tool_content) if isinstance(tool_content, dict) else tool_content
    packet=json.loads(packet_content) if isinstance(packet_content, str) else packet_content
    passages=packet.get("passages") or []
    if not passages: raise SystemExit("evidence read returned no passages")
    submit={**submit,"gate_findings":[{"statement_sha256":passages[0]["statement_sha256"],"classification":"derived_synthesis"}]}
    response=call_read_only("plasma.report.long_form.evidence_gate.submit",submit)
    proc.stdin.close()
    stderr=proc.stderr.read()
    code=proc.wait()
    if code or response.get("result",{}).get("isError") or "event_id" not in json.dumps(response): raise SystemExit(stderr+json.dumps(response))
    emit(os.environ.get("PLASMA_TEST_FINAL_ACK", "REPORT_FINALIZED"))
    raise SystemExit(0)
  else:
    prefix="plasma.report.long_form.final_edit"; sentinel=os.environ.get("PLASMA_TEST_FINAL_ACK", "REPORT_FINALIZED")
    submit={**submit,"gate_findings":[]}
  calls=[
    (prefix+".start",start),
    (prefix+".read",read),
  ]
  if binding["stage"] == "corrective_gate" and binding.get("post_report_humanize") == "enabled":
    review={**read,"max_bytes":65536}
    calls.append(("plasma.report.long_form.style_review.read",review))
  calls.append((prefix+".submit",submit))
  messages=[{"jsonrpc":"2.0","id":i+1,"method":"tools/call","params":{"name":tool,"arguments":arguments}} for i,(tool,arguments) in enumerate(calls)]
  proc=subprocess.run([command]+args,input="".join(json.dumps(x)+"\n" for x in messages),text=True,capture_output=True)
  responses=[json.loads(line) for line in proc.stdout.splitlines() if line.strip()]
  if binding["stage"] == "corrective_gate" and binding.get("post_report_humanize") == "enabled" and len(responses) >= 3 and not responses[2].get("result",{}).get("isError"):
    content=responses[2]["result"]["content"][0]["text"]
    review_output=json.loads(content)
    items=json.loads(review_output.get("content","[]"))
    semantic=[{"paragraph_ordinal":item["paragraph_ordinal"],"final_paragraph_ordinal":item["paragraph_ordinal"],"verdict":"accepted_equivalent"} for item in items]
    if semantic:
      submit={**submit,"semantic_acceptance":semantic}
    calls[-1]=(prefix+".submit",submit)
    messages=[{"jsonrpc":"2.0","id":i+1,"method":"tools/call","params":{"name":tool,"arguments":arguments}} for i,(tool,arguments) in enumerate(calls)]
    proc=subprocess.run([command]+args,input="".join(json.dumps(x)+"\n" for x in messages),text=True,capture_output=True)
  if proc.returncode or '"isError":true' in proc.stdout or "event_id" not in proc.stdout: raise SystemExit(proc.stderr+proc.stdout)
  emit(sentinel)
  raise SystemExit(0)
elif final:
  binding=json.loads(flag("-report-long-form-finalize-binding-json"))
  producer={"type":"agent_session","id":binding["tool_session_id"]}
  if narrative_final:
    draft_id="rfe_testshim"
    common={"mission_id":binding["mission_id"],"session_id":binding["tool_session_id"]}
    start={**common,"pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"draft_id":draft_id,"idempotency_key":"final_start_key","producer":producer}
    read={**common,"draft_id":draft_id,"offset":0,"max_bytes":65536}
    patch={**common,"draft_id":draft_id,"operation":"append","replacement":"\n\nEditorial pass complete.","summary":"acceptance edit","idempotency_key":"final_patch_key","producer":producer}
    submit={**common,"pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"draft_id":draft_id,"idempotency_key":"final_submit_key","producer":producer}
    calls=[
      ("plasma.report.long_form.final_edit.start",start),
      ("plasma.report.long_form.final_edit.read",read),
      ("plasma.report.long_form.final_edit.patch",patch),
      ("plasma.report.long_form.final_edit.submit",submit),
    ]
    messages=[{"jsonrpc":"2.0","id":i+1,"method":"tools/call","params":{"name":tool,"arguments":arguments}} for i,(tool,arguments) in enumerate(calls)]
    proc=subprocess.run([command]+args,input="".join(json.dumps(x)+"\n" for x in messages),text=True,capture_output=True)
    if proc.returncode or '"isError":true' in proc.stdout or "event_id" not in proc.stdout: raise SystemExit(proc.stderr+proc.stdout)
    sentinel=os.environ.get("PLASMA_TEST_FINAL_ACK", "REPORT_FINALIZED")
    emit(sentinel)
    raise SystemExit(0)
  arguments={"mission_id":binding["mission_id"],"session_id":binding["tool_session_id"],"pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"idempotency_key":binding["idempotency_key"],"producer":producer,"opening_markdown":"# Long report","closing_markdown":"## Close"}
  tool="plasma.report.long_form.finalize"; expected="artifact_sha256"; sentinel="REPORT_FINALIZED"
elif requirements:
  binding=json.loads(flag("-report-requirements-binding-json"))
  producer={"type":"agent_session","id":binding["tool_session_id"]}
  requirement_map={"reviewed_event_ids":[binding["pending_event_id"]],"requirements":[{"requirement_id":"req_acceptance","instruction":"include the accepted requirement","source_event_ids":[binding["pending_event_id"]],"owner":{"part_index":1,"section_index":1}}]}
  arguments={"mission_id":binding["mission_id"],"session_id":binding["tool_session_id"],"pending_event_id":binding["pending_event_id"],"plan_event_id":binding["plan_event_id"],"idempotency_key":binding["idempotency_key"],"producer":producer,"requirement_map":requirement_map}
  tool="plasma.report.requirements.submit"; expected="requirement_map_event_id"; sentinel="REQUIREMENTS_MAPPED"
else:
  mode=flag("-report-plan-mode")
  plan={"summary":"Plan","sections":[{"title":"Section","purpose":"Verify"}]} if mode == "planned" else {"summary":"Plan","parts":[{"title":"Part","purpose":"Verify","sections":[{"title":"Section","purpose":"Verify"}]}]}
  if "-report-plan-require-writing-contract" in args:
    plan["writing_contract"]={"central_question":"What must the reader understand?","reader_takeaway":"The verified result and its limit.","reading_path":["state the result","explain the evidence","close with the limit"],"must_keep":["the verified result"],"visual_role":"none needed","tone_and_shape":"direct edited explanation"}
  arguments={"mission_id":flag("-mission-id"),"session_id":flag("-report-plan-tool-session-id"),"pending_event_id":flag("-report-plan-pending-event-id"),"report_mode":mode,"idempotency_key":flag("-report-plan-idempotency-key"),"producer":{"type":"agent_session","id":flag("-report-plan-tool-session-id")},"plan":plan}
  tool="plasma.report.plan.submit"; expected="submission_event_id"; sentinel="PLAN_SUBMITTED"
messages=[{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}},{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":tool,"arguments":arguments}}]
proc=subprocess.run([command]+args,input="".join(json.dumps(x)+"\n" for x in messages),text=True,capture_output=True)
if proc.returncode or expected not in proc.stdout or '"isError":true' in proc.stdout: raise SystemExit(proc.stderr+proc.stdout)
if final: sentinel=os.environ.get("PLASMA_TEST_FINAL_ACK", sentinel)
emit(sentinel)
`
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
