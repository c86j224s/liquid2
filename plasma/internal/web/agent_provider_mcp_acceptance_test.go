package web

import (
	"context"
	"encoding/json"
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
	"github.com/c86j224s/liquid2/plasma/internal/storage/sqlite"
)

type combinedProviderWebExecutor struct {
	mu       sync.Mutex
	provider AgentExecutor
	writes   []AgentResult
}

func (executor *combinedProviderWebExecutor) Run(ctx context.Context, req AgentRequest) (AgentResult, error) {
	if req.ReportPlan != nil || req.PartAssembly != nil || req.LongFormFinalize != nil {
		return executor.provider.Run(ctx, req)
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
				t.Fatal(err)
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
				{Text: "Section body.", SessionID: "22222222-2222-4222-8222-222222222222"},
				{Text: `{"intro":"Part intro","transitions":[],"closing":"Part close"}`, SessionID: "22222222-2222-4222-8222-222222222222"},
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
			postJSON(t, server.URL+"/api/missions/"+missionID+"/reports", reportRequest)
			detail := waitForEventType(t, server.URL, missionID, "report.artifact.created")
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
def emit(sentinel):
  if kind == "codex":
    open(out,"w").write(sentinel)
    print(json.dumps({"type":"thread.started","thread_id":"provider-codex-session"}))
  else:
    print(json.dumps({"type":"result","session_id":"22222222-2222-4222-8222-222222222222","result":sentinel}))
final="-report-long-form-finalize-binding-json" in args
narrative_final="plasma.report.long_form.final_edit.start" in args
part="-report-part-assembly-binding-json" in args
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
