package mcp

import (
	"bytes"
	"compress/zlib"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	mermaidpkg "github.com/c86j224s/liquid2/plasma/internal/mermaid"
	"github.com/c86j224s/liquid2/plasma/internal/sources/localpath"
	"github.com/c86j224s/liquid2/plasma/internal/sources/urlsource"
)

func TestListToolsSchemasAreValid(t *testing.T) {
	server := NewServer(&fakeMCPService{})

	tools := server.ListTools()
	expected := map[string]bool{
		ToolMissionGet:              false,
		ToolMissionUpdate:           false,
		ToolSourcesList:             false,
		ToolSourcesRead:             false,
		ToolSourcesTree:             false,
		ToolSourcesGrep:             false,
		ToolSourcesSearch:           false,
		ToolSourceCandidatesPropose: false,
		ToolSourceCandidatesRead:    false,
		ToolLocalPathRoots:          false,
		ToolLocalPathTree:           false,
		ToolResearchOutline:         false,
		ToolResearchList:            false,
		ToolResearchRead:            false,
		ToolResearchGrep:            false,
		ToolResearchRefs:            false,
		ToolMermaidValidate:         false,
		ToolWorkflowStart:           false,
		ToolWorkflowStatus:          false,
		ToolWorkflowStop:            false,
	}
	for _, tool := range tools {
		if _, ok := expected[tool.Name]; !ok {
			t.Fatalf("unexpected tool exposed: %s", tool.Name)
		}
		expected[tool.Name] = true
		if !json.Valid(tool.InputSchema) {
			t.Fatalf("tool %s has invalid input schema: %s", tool.Name, string(tool.InputSchema))
		}
		if strings.Contains(tool.Name, "approve") || tool.Name == "plasma.report.draft" {
			t.Fatalf("tool should not be exposed in Wave 8: %s", tool.Name)
		}
	}
	for name, seen := range expected {
		if !seen {
			t.Fatalf("expected tool missing: %s", name)
		}
	}
}

func TestMermaidValidateToolPreflightsKnownParseRisks(t *testing.T) {
	server := NewServer(&fakeMCPService{}, WithBinding(Binding{MissionID: "mis_1"}))
	result := server.dispatchCall(context.Background(), ToolCall{
		Name: ToolMermaidValidate,
		Arguments: mustArgs(t, map[string]any{
			"mission_id": "mis_1",
			"source": `requirementDiagram

requirement root_access {
  id: AUTH-ROOT
  text: Access decisions must combine identity, policy, and auditability
  risk: high
  verifymethod: inspection
}`,
		}),
	})
	if result.Error != nil {
		t.Fatalf("validate returned tool error: %#v", result.Error)
	}
	content, ok := result.Content.(mermaidpkg.Result)
	if !ok {
		t.Fatalf("unexpected content type: %#v", result.Content)
	}
	if content.OK || content.DiagramType != "requirementDiagram" {
		t.Fatalf("expected failed requirementDiagram preflight, got %#v", content)
	}
	if !containsMermaidIssue(content.Errors, "requirement_id_token") || !containsMermaidIssue(content.Errors, "requirement_text_needs_quotes") {
		t.Fatalf("expected id and text issues, got %#v", content.Errors)
	}
}

func TestMermaidValidateToolAcceptsQuotedRequirementText(t *testing.T) {
	server := NewServer(&fakeMCPService{}, WithBinding(Binding{MissionID: "mis_1"}))
	result := server.dispatchCall(context.Background(), ToolCall{
		Name: ToolMermaidValidate,
		Arguments: mustArgs(t, map[string]any{
			"mission_id": "mis_1",
			"source": `requirementDiagram

requirement root_access {
  id: AUTH_ROOT
  text: "Access decisions must combine identity, policy, and auditability"
  risk: high
  verifymethod: inspection
}`,
		}),
	})
	if result.Error != nil {
		t.Fatalf("validate returned tool error: %#v", result.Error)
	}
	content, ok := result.Content.(mermaidpkg.Result)
	if !ok {
		t.Fatalf("unexpected content type: %#v", result.Content)
	}
	if !content.OK || content.CanConfirmRender {
		t.Fatalf("expected static preflight pass without render guarantee, got %#v", content)
	}
}

func TestWorkflowStartSchemaAllowsUnlimitedDurationAndCapsSteps(t *testing.T) {
	tool := toolByName(t, NewServer(&fakeMCPService{}).ListTools(), ToolWorkflowStart)
	var schema struct {
		Properties map[string]struct {
			Minimum *float64 `json:"minimum"`
			Maximum *float64 `json:"maximum"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(tool.InputSchema, &schema); err != nil {
		t.Fatal(err)
	}
	maxSteps := schema.Properties["max_steps"]
	maxDuration := schema.Properties["max_duration_ms"]
	if maxSteps.Maximum == nil || *maxSteps.Maximum != 20 {
		t.Fatalf("expected max_steps maximum 20, got %#v", maxSteps.Maximum)
	}
	if maxDuration.Minimum == nil || *maxDuration.Minimum != 0 {
		t.Fatalf("expected max_duration_ms minimum 0, got %#v", maxDuration.Minimum)
	}
}

func TestMissionUpdateToolUsesSharedServiceAndIsIdempotent(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))
	call := ToolCall{Name: ToolMissionUpdate, Arguments: json.RawMessage(`{"mission_id":"mis_1","session_id":"ses_1","idempotency_key":"once","producer":{"type":"user","id":"reviewer"},"title":" New ","scope":{"included":[" A ",""],"excluded":[]}}`)}
	first := server.dispatchCall(context.Background(), call)
	second := server.dispatchCall(context.Background(), call)
	if first.Error != nil || second.Error != nil {
		t.Fatalf("update errors: %#v %#v", first.Error, second.Error)
	}
	if len(service.metadataRequests) != 1 {
		t.Fatalf("expected one service call, got %d", len(service.metadataRequests))
	}
	req := service.metadataRequests[0]
	if req.Title == nil || *req.Title != " New " || req.Producer.Type != "user" || len(first.CreatedEventIDs) != 1 {
		t.Fatalf("unexpected request/result: %#v %#v", req, first)
	}

	wrongMission := ToolCall{Name: ToolMissionUpdate, Arguments: json.RawMessage(`{"mission_id":"mis_2","session_id":"ses_1","idempotency_key":"wrong","producer":{"type":"user","id":"reviewer"},"title":"X"}`)}
	if result := server.dispatchCall(context.Background(), wrongMission); result.Error == nil {
		t.Fatal("expected bound mission rejection")
	}
	nonUser := ToolCall{Name: ToolMissionUpdate, Arguments: json.RawMessage(`{"mission_id":"mis_1","session_id":"ses_1","idempotency_key":"agent","producer":{"type":"agent_session","id":"ses_1"},"title":"X"}`)}
	if result := server.dispatchCall(context.Background(), nonUser); result.Error == nil {
		t.Fatal("expected non-user rejection")
	}
}

func TestEnabledToolsFiltersListedAndCallableTools(t *testing.T) {
	server := NewServer(&fakeMCPService{}, WithEnabledTools([]string{
		ToolResearchOutline,
		ToolSourcesRead,
	}))

	tools := toolNames(server.ListTools())
	if !reflect.DeepEqual(tools, []string{ToolSourcesRead, ToolResearchOutline}) {
		t.Fatalf("expected enabled tools only, got %#v", tools)
	}
	result := server.dispatchCall(context.Background(), ToolCall{Name: ToolMissionGet})
	if result.Error == nil || !strings.Contains(result.Error.Message, "not enabled") {
		t.Fatalf("expected disabled tool call to fail, got %#v", result)
	}
}

func TestLegacyResearchLoopToolsRequireExplicitOption(t *testing.T) {
	defaultServerTools := NewServer(&fakeMCPService{}).ListTools()
	defaultTools := toolNames(defaultServerTools)
	for _, legacy := range []string{
		ToolEvidencePropose,
		ToolQuestionsPropose,
		ToolClaimsPropose,
		ToolClaimConfidence,
		ToolProposalsSubmit,
	} {
		if containsString(defaultTools, legacy) {
			t.Fatalf("default MCP tools must not expose legacy mutation tool %q: %#v", legacy, defaultTools)
		}
	}
	for _, toolName := range []string{ToolResearchList, ToolResearchRead, ToolResearchRefs} {
		defaultKinds := schemaObjectKindEnum(t, toolByName(t, defaultServerTools, toolName).InputSchema)
		if containsString(defaultKinds, app.ResearchIDEObjectEvidenceRecord) {
			t.Fatalf("default research schema must not expose legacy object kinds for %s: %#v", toolName, defaultKinds)
		}
	}

	legacyServerTools := NewServer(&fakeMCPService{}, WithLegacyResearchLoop()).ListTools()
	legacyTools := toolNames(legacyServerTools)
	for _, legacy := range []string{
		ToolEvidencePropose,
		ToolQuestionsPropose,
		ToolClaimsPropose,
		ToolClaimConfidence,
		ToolProposalsSubmit,
	} {
		if !containsString(legacyTools, legacy) {
			t.Fatalf("legacy option should expose %q: %#v", legacy, legacyTools)
		}
	}
	for _, toolName := range []string{ToolResearchList, ToolResearchRead, ToolResearchRefs} {
		legacyKinds := schemaObjectKindEnum(t, toolByName(t, legacyServerTools, toolName).InputSchema)
		for _, kind := range []string{
			app.ResearchIDEObjectEvidenceRecord,
			app.ResearchIDEObjectClaimRecord,
			app.ResearchIDEObjectQuestionRecord,
			app.ResearchIDEObjectProposalBundle,
			app.ResearchIDEObjectReportVersion,
			app.ResearchIDEObjectReportBlock,
		} {
			if !containsString(legacyKinds, kind) {
				t.Fatalf("legacy research schema for %s must expose %s: %#v", toolName, kind, legacyKinds)
			}
		}
	}
}

func TestLegacyMutationCallsRequireExplicitOption(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service)

	result := server.Call(context.Background(), ToolCall{
		Name:      ToolEvidencePropose,
		Arguments: evidenceProposalArgs("evd_disabled", "legacy-disabled"),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" || !strings.Contains(result.Error.Message, "default C1 loop") {
		t.Fatalf("expected disabled legacy mutation error, got %#v", result)
	}
	if len(service.events) != 0 || len(service.evidenceRequests) != 0 || len(service.proposalRequests) != 0 {
		t.Fatalf("disabled legacy mutation reached storage: events=%#v evidence=%#v proposals=%#v", service.events, service.evidenceRequests, service.proposalRequests)
	}
}

func TestOperatorSourceMutationToolsRequireExplicitOption(t *testing.T) {
	defaultServer := NewServer(&fakeMCPService{})
	defaultTools := toolNames(defaultServer.ListTools())
	for _, toolName := range []string{ToolLocalPathAttach, ToolSourcesRemove, ToolSourcesRestore} {
		if containsString(defaultTools, toolName) {
			t.Fatalf("default MCP tools must not expose source mutation tool %q: %#v", toolName, defaultTools)
		}
		result := defaultServer.Call(context.Background(), ToolCall{
			Name: toolName,
			Arguments: mustArgs(t, map[string]any{
				"mission_id":      "mis_1",
				"session_id":      "ses_1",
				"idempotency_key": "mutation-disabled",
				"producer":        map[string]any{"type": "agent_session", "id": "ses_1"},
				"snapshot_id":     "src_1",
			}),
		})
		if result.Error == nil || !strings.Contains(result.Error.Message, "explicit operator surface") {
			t.Fatalf("expected source mutation disabled error for %s, got %#v", toolName, result)
		}
	}

	operatorTools := toolNames(NewServer(&fakeMCPService{}, WithOperatorSourceMutation()).ListTools())
	for _, toolName := range []string{ToolLocalPathAttach, ToolSourcesRemove, ToolSourcesRestore} {
		if !containsString(operatorTools, toolName) {
			t.Fatalf("operator option should expose %q: %#v", toolName, operatorTools)
		}
	}
}

func TestReservedInspectImageToolIsNotExposed(t *testing.T) {
	tools := toolNames(NewServer(&fakeMCPService{}).ListTools())
	if containsString(tools, ReservedInspectImageToolName) {
		t.Fatalf("reserved image inspect tool must not be exposed without a real vision engine: %#v", tools)
	}
}

func TestExperimentalReportCompositionToolsRequireExplicitOption(t *testing.T) {
	defaultTools := toolNames(NewServer(&fakeMCPService{}).ListTools())
	for _, experimental := range []string{
		ToolExperimentReportCreate,
		ToolExperimentReportAppend,
		ToolExperimentReportRead,
		ToolExperimentReportFinalize,
	} {
		if containsString(defaultTools, experimental) {
			t.Fatalf("default MCP tools must not expose experimental report tool %q: %#v", experimental, defaultTools)
		}
	}

	service := &fakeMCPService{}
	defaultServer := NewServer(service)
	result := defaultServer.Call(context.Background(), ToolCall{
		Name: ToolExperimentReportCreate,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":      "mis_1",
			"session_id":      "ses_1",
			"idempotency_key": "draft-disabled",
			"producer":        map[string]any{"type": "agent_session", "id": "ses_1"},
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" || !strings.Contains(result.Error.Message, "disabled") {
		t.Fatalf("expected disabled experimental report error, got %#v", result)
	}
	if len(service.events) != 0 || len(service.artifacts) != 0 {
		t.Fatalf("disabled experimental report tool wrote state: events=%#v artifacts=%#v", service.events, service.artifacts)
	}

	experimentTools := NewServer(&fakeMCPService{}, WithExperimentalReportComposition()).ListTools()
	toolNames := toolNames(experimentTools)
	for _, experimental := range []string{
		ToolExperimentReportCreate,
		ToolExperimentReportAppend,
		ToolExperimentReportRead,
		ToolExperimentReportFinalize,
	} {
		if !containsString(toolNames, experimental) {
			t.Fatalf("experimental option should expose %q: %#v", experimental, toolNames)
		}
		if !json.Valid(toolByName(t, experimentTools, experimental).InputSchema) {
			t.Fatalf("experimental tool %s has invalid schema", experimental)
		}
	}
}

func TestExperimentalReportCompositionCreatesFinalArtifact(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service,
		WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1", AgentExecutor: "codex"}),
		WithExperimentalReportComposition(),
	)
	ctx := context.Background()
	common := map[string]any{
		"mission_id": "mis_1",
		"session_id": "ses_1",
		"producer":   map[string]any{"type": "agent_session", "id": "ses_1"},
	}

	createArgs := cloneMap(common)
	createArgs["idempotency_key"] = "draft-create"
	createArgs["draft_id"] = "rpd_report"
	createArgs["title"] = "Plasma Report"
	created := server.Call(ctx, ToolCall{Name: ToolExperimentReportCreate, Arguments: mustArgs(t, createArgs)})
	if created.Error != nil {
		t.Fatalf("report create returned error: %#v", created.Error)
	}
	createdOutput := created.Content.(experimentReportDraftOutput)
	if createdOutput.DraftID != "rpd_report" || createdOutput.State != "open" {
		t.Fatalf("unexpected create output: %#v", createdOutput)
	}

	appendArgs := cloneMap(common)
	appendArgs["idempotency_key"] = "draft-append"
	appendArgs["draft_id"] = "rpd_report"
	appendArgs["content"] = "# Plasma Report\n\n조사 결과입니다.\n"
	appended := server.Call(ctx, ToolCall{Name: ToolExperimentReportAppend, Arguments: mustArgs(t, appendArgs)})
	if appended.Error != nil {
		t.Fatalf("report append returned error: %#v", appended.Error)
	}
	appendedOutput := appended.Content.(experimentReportDraftOutput)
	if appendedOutput.ContentLength == 0 || appendedOutput.ChunkCount != 1 {
		t.Fatalf("unexpected append output: %#v", appendedOutput)
	}

	read := server.Call(ctx, ToolCall{
		Name: ToolExperimentReportRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id": "mis_1",
			"session_id": "ses_1",
			"draft_id":   "rpd_report",
			"max_bytes":  10,
		}),
	})
	if read.Error != nil {
		t.Fatalf("report read returned error: %#v", read.Error)
	}
	readOutput := read.Content.(experimentReportReadOutput)
	if !readOutput.Truncated || readOutput.NextOffset == 0 || !strings.Contains(readOutput.Content, "# Plasma") {
		t.Fatalf("unexpected read output: %#v", readOutput)
	}

	finalizeArgs := cloneMap(common)
	finalizeArgs["idempotency_key"] = "draft-finalize"
	finalizeArgs["draft_id"] = "rpd_report"
	finalizeArgs["artifact_id"] = "art_report"
	finalizeArgs["filename"] = "plasma-report.md"
	finalized := server.Call(ctx, ToolCall{Name: ToolExperimentReportFinalize, Arguments: mustArgs(t, finalizeArgs)})
	if finalized.Error != nil {
		t.Fatalf("report finalize returned error: %#v", finalized.Error)
	}
	output := finalized.Content.(experimentReportFinalizeOutput)
	if output.Artifact.ArtifactID != "art_report" || output.Artifact.MediaType != "text/markdown; charset=utf-8" || output.EventID == "" {
		t.Fatalf("unexpected finalize output: %#v", output)
	}
	if output.HumanizeReady == nil ||
		output.HumanizeReady.Profile != experimentReportHumanizeProfile ||
		output.HumanizeReady.Target != experimentReportHumanizeTarget ||
		output.HumanizeReady.SourceArtifactID != "art_report" ||
		output.HumanizeReady.Reason != experimentReportHumanizeReason {
		t.Fatalf("expected MCP finalize to return H5-ready metadata without creating a humanized artifact, got %#v", output.HumanizeReady)
	}
	if !fakeMCPHasEventType(service.events, "experiment.report.artifact.created") ||
		!fakeMCPHasEventType(service.events, "experiment.report.humanize.ready") {
		t.Fatalf("expected experiment report artifact and H5-ready marker events, got %#v", service.events)
	}
	if stored, ok := service.artifacts["art_report"]; !ok || string(stored.Content) != "# Plasma Report\n\n조사 결과입니다.\n" {
		t.Fatalf("expected stored report artifact, got %#v", service.artifacts)
	}
}

func TestReportPatchToolsCreatePatchedMarkdownArtifact(t *testing.T) {
	service := &fakeMCPService{artifacts: map[string]app.RawArtifact{
		"art_report_base": {
			ArtifactID: "art_report_base",
			MissionID:  "mis_patch",
			MediaType:  "text/markdown; charset=utf-8",
			Filename:   "base.md",
			Content:    []byte("# Report\n\nOld wording.\n"),
		},
	}}
	server := NewServer(service,
		WithReportPatch(),
		WithBinding(Binding{MissionID: "mis_patch", AgentSessionID: "ses_patch_tool", AgentExecutor: "codex"}),
		WithReportPatchBinding(ReportPatchBinding{
			BaseArtifactID:               "art_report_base",
			PendingEventID:               "evt_patch_pending",
			AgentExecutor:                "codex",
			AgentModel:                   "gpt-5.5",
			AgentReasoningEffort:         "medium",
			MCPMode:                      "auto",
			AgentSessionID:               "provider-report-session",
			PreviousAgentSessionID:       "provider-report-session",
			ReturnedAgentSessionID:       "provider-report-session",
			ReportSessionID:              "provider-report-session",
			ReportSessionPolicy:          "same_session",
			ReportSessionPolicySelection: "explicit_same_session",
			SessionChainKind:             "same_report_session_patch",
		}),
	)
	tools := toolNames(server.ListTools())
	for _, toolName := range []string{ToolReportPatchStart, ToolReportPatchRead, ToolReportPatchApply, ToolReportPatchFinalize} {
		if !containsString(tools, toolName) {
			t.Fatalf("report patch option should expose %s: %#v", toolName, tools)
		}
	}

	ctx := context.Background()
	common := map[string]any{
		"mission_id": "mis_patch",
		"session_id": "ses_patch_tool",
		"producer":   map[string]any{"type": "agent_session", "id": "ses_patch_tool"},
	}
	startArgs := cloneMap(common)
	startArgs["idempotency_key"] = "patch-start"
	startArgs["base_artifact_id"] = "art_report_base"
	startArgs["instruction"] = "Replace old wording."
	startArgs["title"] = "Patched Report"
	start := server.Call(ctx, ToolCall{Name: ToolReportPatchStart, Arguments: mustArgs(t, startArgs)})
	if start.Error != nil {
		t.Fatalf("start failed: %#v", start.Error)
	}
	startOutput := start.Content.(reportPatchOutput)

	wrongStartArgs := cloneMap(common)
	wrongStartArgs["idempotency_key"] = "patch-start-wrong-base"
	wrongStartArgs["base_artifact_id"] = "art_other"
	wrongStartArgs["instruction"] = "Patch another artifact."
	wrongStart := server.Call(ctx, ToolCall{Name: ToolReportPatchStart, Arguments: mustArgs(t, wrongStartArgs)})
	if wrongStart.Error == nil {
		t.Fatalf("expected wrong base artifact to be rejected")
	}

	applyArgs := cloneMap(common)
	applyArgs["idempotency_key"] = "patch-apply"
	applyArgs["patch_id"] = startOutput.PatchID
	applyArgs["operation"] = "replace"
	applyArgs["match_text"] = "Old wording."
	applyArgs["replacement"] = "New wording."
	applyArgs["summary"] = "Updated the stale sentence."
	apply := server.Call(ctx, ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, applyArgs)})
	if apply.Error != nil {
		t.Fatalf("apply failed: %#v", apply.Error)
	}

	read := server.Call(ctx, ToolCall{Name: ToolReportPatchRead, Arguments: mustArgs(t, map[string]any{
		"mission_id": "mis_patch",
		"session_id": "ses_patch_tool",
		"patch_id":   startOutput.PatchID,
		"max_bytes":  1024,
	})})
	if read.Error != nil {
		t.Fatalf("read failed: %#v", read.Error)
	}
	if content := read.Content.(reportPatchReadOutput).Content; !strings.Contains(content, "New wording.") {
		t.Fatalf("expected patched content, got %q", content)
	}

	finalizeArgs := cloneMap(common)
	finalizeArgs["idempotency_key"] = "patch-finalize"
	finalizeArgs["patch_id"] = startOutput.PatchID
	finalizeArgs["artifact_id"] = "art_report_patched"
	finalizeArgs["pending_event_id"] = "evt_stale_pending"
	finalizeArgs["agent_executor"] = "claude"
	finalizeArgs["agent_model"] = "stale-model"
	finalizeArgs["agent_reasoning_effort"] = "high"
	finalizeArgs["mcp_mode"] = "manual"
	finalizeArgs["agent_session_id"] = "stale-report-session"
	finalizeArgs["previous_agent_session_id"] = "stale-previous-session"
	finalizeArgs["returned_agent_session_id"] = "stale-returned-session"
	finalizeArgs["report_session_id"] = "stale-report-session"
	finalizeArgs["report_session_policy"] = "isolated_fork"
	finalizeArgs["report_session_policy_selection"] = "stale_selection"
	finalizeArgs["session_chain_kind"] = "stale_patch_chain"
	finalized := server.Call(ctx, ToolCall{Name: ToolReportPatchFinalize, Arguments: mustArgs(t, finalizeArgs)})
	if finalized.Error != nil {
		t.Fatalf("finalize failed: %#v", finalized.Error)
	}
	if got := string(service.artifacts["art_report_patched"].Content); !strings.Contains(got, "New wording.") {
		t.Fatalf("expected finalized artifact to contain patch, got %q", got)
	}
	var reportEvent app.AppendEventRequest
	for _, event := range service.events {
		if event.EventType == "report.patch.finalized" {
			reportEvent = event
			break
		}
	}
	if reportEvent.EventID == "" {
		t.Fatalf("expected report patch finalized event, got %#v", service.events)
	}
	var payload map[string]any
	if err := json.Unmarshal(reportEvent.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["kind"] != "markdown_report_patch_finalized" {
		t.Fatalf("expected provisional patch finalized payload, got %#v", payload)
	}
	if payload["pending_event_id"] != "evt_patch_pending" ||
		payload["base_report_artifact_id"] != "art_report_base" ||
		payload["agent_executor"] != "codex" ||
		payload["agent_model"] != "gpt-5.5" ||
		payload["agent_session_id"] != "provider-report-session" ||
		payload["report_session_id"] != "provider-report-session" ||
		payload["report_session_policy"] != "same_session" ||
		payload["report_session_policy_selection"] != "explicit_same_session" ||
		payload["tool_session_id"] != "ses_patch_tool" ||
		payload["composition_strategy"] != "mcp_patch_markdown" {
		t.Fatalf("expected patch lineage payload, got %#v", payload)
	}

	finalizeAgainArgs := cloneMap(common)
	finalizeAgainArgs["idempotency_key"] = "patch-finalize-again"
	finalizeAgainArgs["patch_id"] = startOutput.PatchID
	finalizeAgainArgs["pending_event_id"] = "evt_patch_pending"
	finalizeAgainArgs["agent_executor"] = "codex"
	finalizeAgainArgs["report_session_id"] = "provider-report-session"
	finalizeAgainArgs["report_session_policy"] = "same_session"
	finalizeAgainArgs["report_session_policy_selection"] = "explicit_same_session"
	finalizedAgain := server.Call(ctx, ToolCall{Name: ToolReportPatchFinalize, Arguments: mustArgs(t, finalizeAgainArgs)})
	if finalizedAgain.Error != nil {
		t.Fatalf("finalize retry should return existing artifact: %#v", finalizedAgain.Error)
	}
}

func TestReportPatchFinalizeRollsBackArtifactWhenEventAppendFails(t *testing.T) {
	service := &fakeMCPService{
		artifacts: map[string]app.RawArtifact{
			"art_report_base": {
				ArtifactID: "art_report_base",
				MissionID:  "mis_patch",
				MediaType:  "text/markdown; charset=utf-8",
				Filename:   "report.md",
				Content:    []byte("# Report\n\nOld wording.\n"),
			},
		},
		appendEventErrByType: map[string]error{
			"report.patch.finalized": errors.New("append failed"),
		},
	}
	server := NewServer(service,
		WithReportPatch(),
		WithBinding(Binding{MissionID: "mis_patch", AgentSessionID: "ses_patch_tool", AgentExecutor: "codex"}),
		WithReportPatchBinding(ReportPatchBinding{
			BaseArtifactID:               "art_report_base",
			PendingEventID:               "evt_patch_pending",
			AgentExecutor:                "codex",
			AgentSessionID:               "provider-report-session",
			PreviousAgentSessionID:       "provider-report-session",
			ReturnedAgentSessionID:       "provider-report-session",
			ReportSessionID:              "provider-report-session",
			ReportSessionPolicy:          "same_session",
			ReportSessionPolicySelection: "explicit_same_session",
			SessionChainKind:             "same_report_session_patch",
		}),
	)
	common := map[string]any{
		"mission_id": "mis_patch",
		"session_id": "ses_patch_tool",
		"producer":   map[string]any{"type": "agent_session", "id": "ses_patch_tool"},
	}
	startArgs := cloneMap(common)
	startArgs["idempotency_key"] = "patch-start"
	startArgs["base_artifact_id"] = "art_report_base"
	startArgs["instruction"] = "Patch wording."
	start := server.Call(context.Background(), ToolCall{Name: ToolReportPatchStart, Arguments: mustArgs(t, startArgs)})
	if start.Error != nil {
		t.Fatalf("start failed: %#v", start.Error)
	}

	applyArgs := cloneMap(common)
	applyArgs["idempotency_key"] = "patch-apply"
	applyArgs["patch_id"] = start.Content.(reportPatchOutput).PatchID
	applyArgs["operation"] = "replace"
	applyArgs["match_text"] = "Old wording."
	applyArgs["replacement"] = "New wording."
	applyArgs["summary"] = "Update wording."
	apply := server.Call(context.Background(), ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, applyArgs)})
	if apply.Error != nil {
		t.Fatalf("apply failed: %#v", apply.Error)
	}

	finalizeArgs := cloneMap(common)
	finalizeArgs["idempotency_key"] = "patch-finalize"
	finalizeArgs["patch_id"] = start.Content.(reportPatchOutput).PatchID
	finalizeArgs["artifact_id"] = "art_report_patched_orphan"
	finalizeArgs["pending_event_id"] = "evt_patch_pending"
	finalizeArgs["agent_executor"] = "codex"
	finalizeArgs["report_session_id"] = "provider-report-session"
	finalizeArgs["report_session_policy"] = "same_session"
	finalizeArgs["report_session_policy_selection"] = "explicit_same_session"
	finalized := server.Call(context.Background(), ToolCall{Name: ToolReportPatchFinalize, Arguments: mustArgs(t, finalizeArgs)})
	if finalized.Error == nil {
		t.Fatal("expected finalize event append failure")
	}
	if _, exists := service.artifacts["art_report_patched_orphan"]; exists {
		t.Fatalf("failed finalize must not leave a raw artifact")
	}
	if fakeMCPHasEventType(service.events, "report.patch.finalized") {
		t.Fatalf("failed finalize must not append a finalized event")
	}
}

func TestReportPatchHumanizeSessionRejectsFidelityDriftAtApply(t *testing.T) {
	service := &fakeMCPService{artifacts: map[string]app.RawArtifact{
		"art_report_base": {
			ArtifactID: "art_report_base",
			MissionID:  "mis_patch_h5",
			MediaType:  "text/markdown; charset=utf-8",
			Filename:   "report.md",
			Content:    []byte("# Report\n\n핵심 판단은 반드시 원문에 근거해야 한다.\n\n이 접근은 가능하다.\n\n따라서 “문장이 더 자연스러운가”만이 아니라 출처 참조 보존도 중요하다.\n\n출처: https://example.com/report\n"),
		},
	}}
	server := NewServer(service,
		WithReportPatch(),
		WithBinding(Binding{MissionID: "mis_patch_h5", AgentSessionID: "ses_patch_tool", AgentExecutor: "codex"}),
		WithReportPatchBinding(ReportPatchBinding{
			BaseArtifactID:               "art_report_base",
			PendingEventID:               "evt_patch_pending",
			AgentExecutor:                "codex",
			AgentSessionID:               "provider-report-session",
			PreviousAgentSessionID:       "provider-report-session",
			ReturnedAgentSessionID:       "provider-report-session",
			ReportSessionID:              "provider-report-session",
			ReportSessionPolicy:          "same_session",
			ReportSessionPolicySelection: "explicit_same_session",
			SessionChainKind:             "same_report_session_h5_humanize_patch",
		}),
	)
	common := map[string]any{
		"mission_id": "mis_patch_h5",
		"session_id": "ses_patch_tool",
		"producer":   map[string]any{"type": "agent_session", "id": "ses_patch_tool"},
	}
	startArgs := cloneMap(common)
	startArgs["idempotency_key"] = "patch-start"
	startArgs["base_artifact_id"] = "art_report_base"
	startArgs["instruction"] = "Conservative H5 tone pass."
	start := server.Call(context.Background(), ToolCall{Name: ToolReportPatchStart, Arguments: mustArgs(t, startArgs)})
	if start.Error != nil {
		t.Fatalf("start failed: %#v", start.Error)
	}
	patchID := start.Content.(reportPatchOutput).PatchID

	badApplyArgs := cloneMap(common)
	badApplyArgs["idempotency_key"] = "patch-bad-quote"
	badApplyArgs["patch_id"] = patchID
	badApplyArgs["operation"] = "replace"
	badApplyArgs["match_text"] = "“문장이 더 자연스러운가”"
	badApplyArgs["replacement"] = "“문장이 더 자연스러워졌는가”"
	badApplyArgs["summary"] = "Changed quoted text."
	badApply := server.Call(context.Background(), ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, badApplyArgs)})
	if badApply.Error == nil || !strings.Contains(badApply.Error.Message, "fidelity guard") {
		t.Fatalf("expected H5 quoted text drift to fail at apply, got %#v", badApply.Error)
	}

	badMeaningApplyArgs := cloneMap(common)
	badMeaningApplyArgs["idempotency_key"] = "patch-bad-meaning"
	badMeaningApplyArgs["patch_id"] = patchID
	badMeaningApplyArgs["operation"] = "replace"
	badMeaningApplyArgs["match_text"] = "원문"
	badMeaningApplyArgs["replacement"] = "인상"
	badMeaningApplyArgs["summary"] = "Changed a core meaning marker."
	badMeaningApply := server.Call(context.Background(), ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, badMeaningApplyArgs)})
	if badMeaningApply.Error == nil || !strings.Contains(badMeaningApply.Error.Message, "fidelity guard") {
		t.Fatalf("expected H5 core meaning drift to fail at apply, got %#v", badMeaningApply.Error)
	}

	badNegationApplyArgs := cloneMap(common)
	badNegationApplyArgs["idempotency_key"] = "patch-bad-negation"
	badNegationApplyArgs["patch_id"] = patchID
	badNegationApplyArgs["operation"] = "replace"
	badNegationApplyArgs["match_text"] = "가능하다"
	badNegationApplyArgs["replacement"] = "가능하지 않다"
	badNegationApplyArgs["summary"] = "Changed polarity."
	badNegationApply := server.Call(context.Background(), ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, badNegationApplyArgs)})
	if badNegationApply.Error == nil || !strings.Contains(badNegationApply.Error.Message, "fidelity guard") {
		t.Fatalf("expected H5 polarity drift to fail at apply, got %#v", badNegationApply.Error)
	}

	badStandaloneNegationApplyArgs := cloneMap(common)
	badStandaloneNegationApplyArgs["idempotency_key"] = "patch-bad-standalone-negation"
	badStandaloneNegationApplyArgs["patch_id"] = patchID
	badStandaloneNegationApplyArgs["operation"] = "replace"
	badStandaloneNegationApplyArgs["match_text"] = "가능하다"
	badStandaloneNegationApplyArgs["replacement"] = "안 된다"
	badStandaloneNegationApplyArgs["summary"] = "Changed polarity with standalone negation."
	badStandaloneNegationApply := server.Call(context.Background(), ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, badStandaloneNegationApplyArgs)})
	if badStandaloneNegationApply.Error == nil || !strings.Contains(badStandaloneNegationApply.Error.Message, "fidelity guard") {
		t.Fatalf("expected H5 standalone negation drift to fail at apply, got %#v", badStandaloneNegationApply.Error)
	}

	badJoinedNegationApplyArgs := cloneMap(common)
	badJoinedNegationApplyArgs["idempotency_key"] = "patch-bad-joined-negation"
	badJoinedNegationApplyArgs["patch_id"] = patchID
	badJoinedNegationApplyArgs["operation"] = "replace"
	badJoinedNegationApplyArgs["match_text"] = "가능하다"
	badJoinedNegationApplyArgs["replacement"] = "안된다"
	badJoinedNegationApplyArgs["summary"] = "Changed polarity with joined negation."
	badJoinedNegationApply := server.Call(context.Background(), ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, badJoinedNegationApplyArgs)})
	if badJoinedNegationApply.Error == nil || !strings.Contains(badJoinedNegationApply.Error.Message, "fidelity guard") {
		t.Fatalf("expected H5 joined negation drift to fail at apply, got %#v", badJoinedNegationApply.Error)
	}

	badMemoNegationApplyArgs := cloneMap(common)
	badMemoNegationApplyArgs["idempotency_key"] = "patch-bad-memo-negation"
	badMemoNegationApplyArgs["patch_id"] = patchID
	badMemoNegationApplyArgs["operation"] = "replace"
	badMemoNegationApplyArgs["match_text"] = "가능하다"
	badMemoNegationApplyArgs["replacement"] = "안됨"
	badMemoNegationApplyArgs["summary"] = "Changed polarity with memo-style negation."
	badMemoNegationApply := server.Call(context.Background(), ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, badMemoNegationApplyArgs)})
	if badMemoNegationApply.Error == nil || !strings.Contains(badMemoNegationApply.Error.Message, "fidelity guard") {
		t.Fatalf("expected H5 memo-style negation drift to fail at apply, got %#v", badMemoNegationApply.Error)
	}

	appendArgs := cloneMap(common)
	appendArgs["idempotency_key"] = "patch-bad-append"
	appendArgs["patch_id"] = patchID
	appendArgs["operation"] = "append"
	appendArgs["replacement"] = "\n\n새 문단입니다."
	appendArgs["summary"] = "Append a new paragraph."
	appended := server.Call(context.Background(), ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, appendArgs)})
	if appended.Error == nil || !strings.Contains(appended.Error.Message, "only supports small replace operations") {
		t.Fatalf("expected H5 append to fail at apply, got %#v", appended.Error)
	}

	goodApplyArgs := cloneMap(common)
	goodApplyArgs["idempotency_key"] = "patch-good-prose"
	goodApplyArgs["patch_id"] = patchID
	goodApplyArgs["operation"] = "replace"
	goodApplyArgs["match_text"] = "중요하다."
	goodApplyArgs["replacement"] = "중요한 조건이다."
	goodApplyArgs["summary"] = "Smoothed prose without changing source line."
	goodApply := server.Call(context.Background(), ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, goodApplyArgs)})
	if goodApply.Error != nil {
		t.Fatalf("expected safe H5 prose edit to pass, got %#v", goodApply.Error)
	}
}

func TestReportPatchHumanizeSessionRejectsCumulativeFidelityDriftAtFinalize(t *testing.T) {
	baseLines := []string{"# Report", ""}
	for i := 1; i <= 24; i++ {
		baseLines = append(baseLines, fmt.Sprintf("문장 %02d은 다듬을 대상이다.", i))
	}
	baseContent := strings.Join(baseLines, "\n") + "\n"
	service := &fakeMCPService{artifacts: map[string]app.RawArtifact{
		"art_report_base": {
			ArtifactID: "art_report_base",
			MissionID:  "mis_patch_h5",
			MediaType:  "text/markdown; charset=utf-8",
			Filename:   "report.md",
			Content:    []byte(baseContent),
		},
	}}
	server := NewServer(service,
		WithReportPatch(),
		WithBinding(Binding{MissionID: "mis_patch_h5", AgentSessionID: "ses_patch_tool", AgentExecutor: "codex"}),
		WithReportPatchBinding(ReportPatchBinding{
			BaseArtifactID:               "art_report_base",
			PendingEventID:               "evt_patch_pending",
			AgentExecutor:                "codex",
			AgentSessionID:               "provider-report-session",
			PreviousAgentSessionID:       "provider-report-session",
			ReturnedAgentSessionID:       "provider-report-session",
			ReportSessionID:              "provider-report-session",
			ReportSessionPolicy:          "same_session",
			ReportSessionPolicySelection: "explicit_same_session",
			SessionChainKind:             "same_report_session_h5_humanize_patch",
		}),
	)
	common := map[string]any{
		"mission_id": "mis_patch_h5",
		"session_id": "ses_patch_tool",
		"producer":   map[string]any{"type": "agent_session", "id": "ses_patch_tool"},
	}
	startArgs := cloneMap(common)
	startArgs["idempotency_key"] = "patch-start"
	startArgs["base_artifact_id"] = "art_report_base"
	startArgs["instruction"] = "Conservative H5 tone pass."
	start := server.Call(context.Background(), ToolCall{Name: ToolReportPatchStart, Arguments: mustArgs(t, startArgs)})
	if start.Error != nil {
		t.Fatalf("start failed: %#v", start.Error)
	}
	patchID := start.Content.(reportPatchOutput).PatchID

	for i := 1; i <= 9; i++ {
		applyArgs := cloneMap(common)
		applyArgs["idempotency_key"] = fmt.Sprintf("patch-line-%02d", i)
		applyArgs["patch_id"] = patchID
		applyArgs["operation"] = "replace"
		applyArgs["match_text"] = fmt.Sprintf("문장 %02d은 다듬을 대상이다.", i)
		applyArgs["replacement"] = fmt.Sprintf("문장 %02d은 다듬을 대상입니다.", i)
		applyArgs["summary"] = "Small tone edit."
		apply := server.Call(context.Background(), ToolCall{Name: ToolReportPatchApply, Arguments: mustArgs(t, applyArgs)})
		if apply.Error != nil {
			t.Fatalf("apply %d should pass per-step guard: %#v", i, apply.Error)
		}
	}

	finalizeArgs := cloneMap(common)
	finalizeArgs["idempotency_key"] = "patch-finalize"
	finalizeArgs["patch_id"] = patchID
	finalizeArgs["artifact_id"] = "art_report_cumulative_rewrite"
	finalizeArgs["pending_event_id"] = "evt_patch_pending"
	finalizeArgs["agent_executor"] = "codex"
	finalizeArgs["report_session_id"] = "provider-report-session"
	finalizeArgs["report_session_policy"] = "same_session"
	finalizeArgs["report_session_policy_selection"] = "explicit_same_session"
	finalized := server.Call(context.Background(), ToolCall{Name: ToolReportPatchFinalize, Arguments: mustArgs(t, finalizeArgs)})
	if finalized.Error == nil || !strings.Contains(finalized.Error.Message, "fidelity guard") {
		t.Fatalf("expected cumulative H5 drift to fail at finalize, got %#v", finalized.Error)
	}
	if _, exists := service.artifacts["art_report_cumulative_rewrite"]; exists {
		t.Fatalf("finalize guard must reject before writing a raw artifact")
	}
	if fakeMCPHasEventType(service.events, "report.patch.finalized") {
		t.Fatalf("finalize guard must reject before writing a finalized event")
	}
}

func TestReportPatchHumanizeSessionRejectsNoopFinalizeBeforeArtifactWrite(t *testing.T) {
	service := &fakeMCPService{artifacts: map[string]app.RawArtifact{
		"art_report_base": {
			ArtifactID: "art_report_base",
			MissionID:  "mis_patch_h5",
			MediaType:  "text/markdown; charset=utf-8",
			Filename:   "report.md",
			Content:    []byte("# Report\n\n이미 충분히 자연스럽다.\n"),
		},
	}}
	server := NewServer(service,
		WithReportPatch(),
		WithBinding(Binding{MissionID: "mis_patch_h5", AgentSessionID: "ses_patch_tool", AgentExecutor: "codex"}),
		WithReportPatchBinding(ReportPatchBinding{
			BaseArtifactID:               "art_report_base",
			PendingEventID:               "evt_patch_pending",
			AgentExecutor:                "codex",
			AgentSessionID:               "provider-report-session",
			PreviousAgentSessionID:       "provider-report-session",
			ReturnedAgentSessionID:       "provider-report-session",
			ReportSessionID:              "provider-report-session",
			ReportSessionPolicy:          "same_session",
			ReportSessionPolicySelection: "explicit_same_session",
			SessionChainKind:             "same_report_session_h5_humanize_patch",
		}),
	)
	common := map[string]any{
		"mission_id": "mis_patch_h5",
		"session_id": "ses_patch_tool",
		"producer":   map[string]any{"type": "agent_session", "id": "ses_patch_tool"},
	}
	startArgs := cloneMap(common)
	startArgs["idempotency_key"] = "patch-start"
	startArgs["base_artifact_id"] = "art_report_base"
	startArgs["instruction"] = "Conservative H5 tone pass."
	start := server.Call(context.Background(), ToolCall{Name: ToolReportPatchStart, Arguments: mustArgs(t, startArgs)})
	if start.Error != nil {
		t.Fatalf("start failed: %#v", start.Error)
	}

	finalizeArgs := cloneMap(common)
	finalizeArgs["idempotency_key"] = "patch-finalize"
	finalizeArgs["patch_id"] = start.Content.(reportPatchOutput).PatchID
	finalizeArgs["artifact_id"] = "art_report_noop_h5"
	finalizeArgs["pending_event_id"] = "evt_patch_pending"
	finalizeArgs["agent_executor"] = "codex"
	finalizeArgs["report_session_id"] = "provider-report-session"
	finalizeArgs["report_session_policy"] = "same_session"
	finalizeArgs["report_session_policy_selection"] = "explicit_same_session"
	finalized := server.Call(context.Background(), ToolCall{Name: ToolReportPatchFinalize, Arguments: mustArgs(t, finalizeArgs)})
	if finalized.Error == nil || !strings.Contains(finalized.Error.Message, "NO_H5_CHANGES") {
		t.Fatalf("expected no-op H5 finalize to fail before artifact write, got %#v", finalized.Error)
	}
	if _, exists := service.artifacts["art_report_noop_h5"]; exists {
		t.Fatalf("no-op finalize guard must reject before writing a raw artifact")
	}
	if fakeMCPHasEventType(service.events, "report.patch.finalized") {
		t.Fatalf("no-op finalize guard must reject before writing a finalized event")
	}
}

func TestExperimentalReportFinalizeSucceedsWhenHumanizeReadyMarkerFails(t *testing.T) {
	service := &fakeMCPService{
		appendEventErrByType: map[string]error{
			"experiment.report.humanize.ready": errors.New("ready marker unavailable"),
		},
	}
	server := NewServer(service,
		WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1", AgentExecutor: "codex"}),
		WithExperimentalReportComposition(),
	)
	ctx := context.Background()
	common := map[string]any{
		"mission_id": "mis_1",
		"session_id": "ses_1",
		"producer":   map[string]any{"type": "agent_session", "id": "ses_1"},
	}

	createArgs := cloneMap(common)
	createArgs["idempotency_key"] = "draft-create"
	createArgs["draft_id"] = "rpd_report"
	createArgs["title"] = "Plasma Report"
	if result := server.Call(ctx, ToolCall{Name: ToolExperimentReportCreate, Arguments: mustArgs(t, createArgs)}); result.Error != nil {
		t.Fatalf("report create returned error: %#v", result.Error)
	}

	appendArgs := cloneMap(common)
	appendArgs["idempotency_key"] = "draft-append"
	appendArgs["draft_id"] = "rpd_report"
	appendArgs["content"] = "# Plasma Report\n\n조사 결과입니다.\n"
	if result := server.Call(ctx, ToolCall{Name: ToolExperimentReportAppend, Arguments: mustArgs(t, appendArgs)}); result.Error != nil {
		t.Fatalf("report append returned error: %#v", result.Error)
	}

	finalizeArgs := cloneMap(common)
	finalizeArgs["idempotency_key"] = "draft-finalize"
	finalizeArgs["draft_id"] = "rpd_report"
	finalizeArgs["artifact_id"] = "art_report"
	finalizeArgs["filename"] = "plasma-report.md"
	finalized := server.Call(ctx, ToolCall{Name: ToolExperimentReportFinalize, Arguments: mustArgs(t, finalizeArgs)})
	if finalized.Error != nil {
		t.Fatalf("ready marker failure must not fail original finalize: %#v", finalized.Error)
	}
	output := finalized.Content.(experimentReportFinalizeOutput)
	if output.Artifact.ArtifactID != "art_report" || output.HumanizeReady != nil {
		t.Fatalf("expected finalized source artifact without H5-ready output, got %#v", output)
	}
	if len(finalized.CreatedEventIDs) != 1 || finalized.CreatedEventIDs[0] == "" {
		t.Fatalf("expected only original artifact event id after ready marker failure, got %#v", finalized.CreatedEventIDs)
	}
	if !fakeMCPHasEventType(service.events, "experiment.report.artifact.created") ||
		fakeMCPHasEventType(service.events, "experiment.report.humanize.ready") {
		t.Fatalf("expected original artifact event and no ready marker event, got %#v", service.events)
	}
	server.mu.Lock()
	draft := server.reportDrafts["rpd_report"]
	server.mu.Unlock()
	if !draft.Finalized || draft.ArtifactID != "art_report" || draft.HumanizeReadyEventID != "" {
		t.Fatalf("expected draft finalized despite ready marker failure, got %#v", draft)
	}
}

func TestResearchToolsDelegateToReaderAndEnforceBinding(t *testing.T) {
	service := &fakeMCPService{
		outline: app.ResearchIDEOutline{MissionID: "mis_1", Title: "Mission"},
		page: app.ResearchIDEPage{
			MissionID:  "mis_1",
			ObjectKind: app.ResearchIDEObjectRawArtifact,
			Items: []app.ResearchIDEObjectSummary{{
				ObjectKind: app.ResearchIDEObjectRawArtifact,
				ObjectID:   "art_1",
				MissionID:  "mis_1",
				Summary:    "Artifact",
			}},
			Limit: 10,
		},
		read: app.ResearchIDEObjectRead{
			ObjectKind: app.ResearchIDEObjectRawArtifact,
			ObjectID:   "art_1",
			MissionID:  "mis_1",
			Summary:    "artifact",
			Data:       "hello",
			Truncated:  true,
			NextOffset: 5,
		},
		grep: app.ResearchIDEGrepResult{
			MissionID: "mis_1",
			Query:     "hello",
			Matches: []app.ResearchIDEGrepMatch{{
				ObjectKind: app.ResearchIDEObjectRawArtifact,
				ObjectID:   "art_1",
				MissionID:  "mis_1",
				Snippet:    "hello",
			}},
		},
		refs: app.ResearchIDEReferences{
			MissionID:  "mis_1",
			ObjectKind: app.ResearchIDEObjectSourceSnapshot,
			ObjectID:   "src_1",
			Forward:    []app.ResearchIDEObjectRef{{ObjectKind: app.ResearchIDEObjectRawArtifact, ObjectID: "art_1"}},
		},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1"}))
	ctx := context.Background()

	calls := []ToolCall{
		{Name: ToolResearchOutline, Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1"})},
		{Name: ToolResearchList, Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1", "object_kind": "raw_artifact", "limit": 10})},
		{Name: ToolResearchRead, Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1", "object_kind": "raw_artifact", "object_id": "art_1", "max_bytes": 5})},
		{Name: ToolResearchGrep, Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1", "query": "hello", "limit": 10})},
		{Name: ToolResearchRefs, Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1", "object_kind": "source_snapshot", "object_id": "src_1", "limit": 10})},
	}
	for _, call := range calls {
		result := server.Call(ctx, call)
		if result.Error != nil {
			t.Fatalf("%s returned error: %#v", call.Name, result.Error)
		}
		if result.MissionID != "mis_1" || result.Content == nil {
			t.Fatalf("%s returned unexpected result: %#v", call.Name, result)
		}
	}
	if service.lastRead.MaxBytes != 5 || service.lastRead.ObjectID != "art_1" {
		t.Fatalf("research read request was not forwarded: %#v", service.lastRead)
	}
	var readTracePayload map[string]any
	for _, event := range service.events {
		var payload map[string]any
		if event.EventType != "mcp.tool.called" || json.Unmarshal(event.Payload, &payload) != nil {
			continue
		}
		if payload["tool_name"] == ToolResearchRead {
			readTracePayload = payload
			break
		}
	}
	if readTracePayload == nil {
		t.Fatalf("expected research read trace event, got %#v", service.events)
	}
	metrics := readTracePayload["io_metrics"].(map[string]any)
	if metrics["read_kind"] != "research_object" || metrics["object_kind"] != app.ResearchIDEObjectRawArtifact || metrics["object_id"] != "art_1" {
		t.Fatalf("unexpected research read metrics: %#v", metrics)
	}
	if metrics["requested_max_bytes"] != float64(5) || metrics["returned_content_bytes"] != float64(5) || metrics["next_offset"] != float64(5) || metrics["response_truncated"] != true {
		t.Fatalf("unexpected research read byte metrics: %#v", metrics)
	}

	rejected := server.Call(ctx, ToolCall{
		Name:      ToolResearchOutline,
		Arguments: mustArgs(t, map[string]any{"mission_id": "mis_other"}),
	})
	if rejected.Error == nil || rejected.Error.ErrorKind != "validation" {
		t.Fatalf("expected bound mission rejection, got %#v", rejected)
	}
}

func TestWorkflowToolsUseSharedProjectionAndBinding(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1", CurrentUserEventID: "evt_user_1", AgentExecutor: "codex"}))
	ctx := context.Background()

	start := server.Call(ctx, ToolCall{
		Name: ToolWorkflowStart,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":      "mis_1",
			"workflow_run_id": "wfr_mcp",
			"instruction":     "Run bounded workflow",
			"agent_executor":  "codex",
			"mcp_mode":        "auto",
			"max_steps":       2,
			"max_duration_ms": 60000,
		}),
	})
	if start.Error != nil {
		t.Fatalf("workflow start returned error: %#v", start.Error)
	}
	content, ok := start.Content.(map[string]any)
	if !ok || content["provider_invoked"] != false {
		t.Fatalf("workflow start should not invoke provider: %#v", start.Content)
	}
	if len(service.workflowRuns) != 1 || service.workflowRuns[0].Status != app.WorkflowStatusQueued {
		t.Fatalf("expected queued workflow run, got %#v", service.workflowRuns)
	}
	if service.workflowRuns[0].StartAfterEventID != "evt_user_1" {
		t.Fatalf("expected workflow start to inherit bound current user event, got %#v", service.workflowRuns[0])
	}

	rejectedExecutor := server.Call(ctx, ToolCall{
		Name: ToolWorkflowStart,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":     "mis_1",
			"instruction":    "Run with mismatched executor",
			"agent_executor": "claude",
		}),
	})
	if rejectedExecutor.Error == nil || rejectedExecutor.Error.ErrorKind != "validation" {
		t.Fatalf("expected executor binding rejection, got %#v", rejectedExecutor)
	}
	rejectedEvent := server.Call(ctx, ToolCall{
		Name: ToolWorkflowStart,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":             "mis_1",
			"instruction":            "Run against another turn",
			"start_after_event_id":   "evt_user_other",
			"agent_executor":         "codex",
			"requested_by_session":   "ses_1",
			"requested_by_surface":   "mcp",
			"requested_tool_session": "ses_1",
		}),
	})
	if rejectedEvent.Error == nil || rejectedEvent.Error.ErrorKind != "validation" {
		t.Fatalf("expected current user event binding rejection, got %#v", rejectedEvent)
	}

	status := server.Call(ctx, ToolCall{
		Name:      ToolWorkflowStatus,
		Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1", "workflow_run_id": "wfr_mcp"}),
	})
	if status.Error != nil || status.MissionID != "mis_1" {
		t.Fatalf("workflow status returned unexpected result: %#v", status)
	}

	stop := server.Call(ctx, ToolCall{
		Name:      ToolWorkflowStop,
		Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1", "workflow_run_id": "wfr_mcp", "reason": "user stop"}),
	})
	if stop.Error != nil {
		t.Fatalf("workflow stop returned error: %#v", stop.Error)
	}
	if service.workflowRuns[0].Status != app.WorkflowStatusStopping {
		t.Fatalf("expected stopping workflow, got %#v", service.workflowRuns[0])
	}
	if len(service.events) == 0 || service.events[len(service.events)-1].EventType != "mcp.tool.called" {
		t.Fatalf("expected mcp.tool.called trace event, got %#v", service.events)
	}

	rejected := server.Call(ctx, ToolCall{
		Name:      ToolWorkflowStatus,
		Arguments: mustArgs(t, map[string]any{"mission_id": "mis_other"}),
	})
	if rejected.Error == nil || rejected.Error.ErrorKind != "validation" {
		t.Fatalf("expected bound mission rejection, got %#v", rejected)
	}
}

func TestWorkflowStartAcceptsOmittedAndZeroDuration(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1", CurrentUserEventID: "evt_user_1", AgentExecutor: "codex"}))
	for _, tc := range []struct {
		name string
		args map[string]any
	}{
		{name: "omitted", args: map[string]any{"mission_id": "mis_1", "workflow_run_id": "wfr_omitted", "instruction": "run", "agent_executor": "codex"}},
		{name: "zero", args: map[string]any{"mission_id": "mis_1", "workflow_run_id": "wfr_zero", "instruction": "run", "agent_executor": "codex", "max_duration_ms": 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := server.Call(context.Background(), ToolCall{Name: ToolWorkflowStart, Arguments: mustArgs(t, tc.args)})
			if result.Error != nil {
				t.Fatalf("workflow start returned error: %#v", result.Error)
			}
		})
	}
	if len(service.workflowRuns) != 2 {
		t.Fatalf("expected two workflow runs, got %#v", service.workflowRuns)
	}
	for _, run := range service.workflowRuns {
		if run.MaxDurationMS != 0 {
			t.Fatalf("expected unlimited duration, got %#v", run)
		}
	}
}

func TestWorkflowStartRequiresDeferredTurnBinding(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1", AgentExecutor: "codex"}))
	ctx := context.Background()

	result := server.Call(ctx, ToolCall{
		Name: ToolWorkflowStart,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":     "mis_1",
			"instruction":    "Run bounded workflow",
			"agent_executor": "codex",
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("expected validation error without current user event binding, got %#v", result)
	}
	if len(service.workflowRuns) != 0 {
		t.Fatalf("workflow start without deferred turn should not create a run, got %#v", service.workflowRuns)
	}
}

func TestResearchLegacyReadsRequireLegacyMode(t *testing.T) {
	service := &fakeMCPService{
		page: app.ResearchIDEPage{
			MissionID:  "mis_1",
			ObjectKind: app.ResearchIDEObjectEvidenceRecord,
			Items: []app.ResearchIDEObjectSummary{{
				ObjectKind: app.ResearchIDEObjectEvidenceRecord,
				ObjectID:   "evd_1",
				MissionID:  "mis_1",
				Summary:    "Historical evidence",
			}},
		},
	}
	defaultServer := NewServer(service, WithBinding(Binding{MissionID: "mis_1"}))
	rejected := defaultServer.Call(context.Background(), ToolCall{
		Name: ToolResearchList,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"object_kind": "evidence_record",
			"legacy":      true,
			"limit":       10,
		}),
	})
	if rejected.Error == nil || rejected.Error.ErrorKind != "validation" {
		t.Fatalf("expected legacy read rejection, got %#v", rejected)
	}

	legacyServer := NewServer(service, WithBinding(Binding{MissionID: "mis_1"}), WithLegacyResearchLoop())
	result := legacyServer.Call(context.Background(), ToolCall{
		Name: ToolResearchList,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"object_kind": "evidence_record",
			"legacy":      true,
			"limit":       10,
		}),
	})
	if result.Error != nil {
		t.Fatalf("legacy read returned error: %#v", result.Error)
	}
}

func TestMissionBoundToolCallsAreLogged(t *testing.T) {
	service := &fakeMCPService{
		outline: app.ResearchIDEOutline{MissionID: "mis_1", Title: "Mission"},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_trace"}))

	result := server.Call(context.Background(), ToolCall{
		Name:      ToolResearchOutline,
		Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1"}),
	})
	if result.Error != nil {
		t.Fatalf("research outline returned error: %#v", result.Error)
	}
	if result.TraceEventID == "" {
		t.Fatalf("expected trace event id in result: %#v", result)
	}
	if len(service.events) != 1 {
		t.Fatalf("expected one trace event, got %#v", service.events)
	}
	event := service.events[0]
	if event.EventType != "mcp.tool.called" || event.Producer.Type != "agent_session" || event.Producer.ID != "ses_trace" {
		t.Fatalf("unexpected trace event: %#v", event)
	}
	var payload map[string]any
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["tool_name"] != ToolResearchOutline || payload["success"] != true || payload["tool_session_id"] != "ses_trace" {
		t.Fatalf("unexpected trace payload: %#v", payload)
	}
	metrics, ok := payload["io_metrics"].(map[string]any)
	if !ok {
		t.Fatalf("expected io_metrics in trace payload: %#v", payload)
	}
	for _, key := range []string{"argument_raw_bytes", "argument_summary_bytes", "result_raw_bytes", "result_summary_bytes", "content_raw_bytes"} {
		if _, ok := metrics[key]; !ok {
			t.Fatalf("expected io_metrics.%s in %#v", key, metrics)
		}
	}
}

func TestMissionGetIsReadOnly(t *testing.T) {
	service := &fakeMCPService{
		projection: app.MissionProjection{MissionID: "mis_1", Title: "Research mission"},
	}
	server := NewServer(service)

	result := server.Call(context.Background(), ToolCall{
		Name:      ToolMissionGet,
		Arguments: mustArgs(t, map[string]any{"mission_id": " mis_1 "}),
	})
	if result.Error != nil {
		t.Fatalf("mission get returned error: %#v", result.Error)
	}
	if result.MissionID != "mis_1" {
		t.Fatalf("unexpected mission id: %#v", result)
	}
	output, ok := result.Content.(missionGetOutput)
	if !ok {
		t.Fatalf("unexpected mission get content type: %T", result.Content)
	}
	if output.MissionProjection.Title != "Research mission" {
		t.Fatalf("unexpected projection: %#v", output.MissionProjection)
	}
	if len(service.events) != 0 {
		t.Fatalf("mission get must not write events: %#v", service.events)
	}
}

func TestMissionGetCanIncludeStoredSources(t *testing.T) {
	service := &fakeMCPService{
		projection: app.MissionProjection{MissionID: "mis_1", Title: "Research mission"},
		sources: []app.SourceSnapshot{{
			SnapshotID:  "src_1",
			MissionID:   "mis_1",
			Title:       "Pinned source",
			ArtifactIDs: []string{"art_1"},
		}},
	}
	server := NewServer(service)

	result := server.Call(context.Background(), ToolCall{
		Name: ToolMissionGet,
		Arguments: mustArgs(t, map[string]any{
			"mission_id": "mis_1",
			"include":    []string{"sources"},
		}),
	})
	if result.Error != nil {
		t.Fatalf("mission get returned error: %#v", result.Error)
	}
	output := result.Content.(missionGetOutput)
	if len(output.Sources) != 1 || output.Sources[0].SnapshotID != "src_1" {
		t.Fatalf("expected source snapshot in mission output, got %#v", output.Sources)
	}
	if len(service.events) != 0 {
		t.Fatalf("mission get with sources must not write events: %#v", service.events)
	}
}

func TestMissionGetCanIncludeResearchRecords(t *testing.T) {
	service := &fakeMCPService{
		projection: app.MissionProjection{MissionID: "mis_1", Title: "Research mission"},
		evidence: []app.EvidenceRecord{{
			EvidenceID: "evd_1",
			MissionID:  "mis_1",
			Summary:    "Pinned evidence",
		}},
		claims: []app.ClaimRecord{{
			ClaimID:   "clm_1",
			MissionID: "mis_1",
			Text:      "Existing claim",
		}},
		questions: []app.QuestionRecord{{
			QuestionID: "qst_1",
			MissionID:  "mis_1",
			Text:       "Open question",
		}},
	}
	server := NewServer(service)

	result := server.Call(context.Background(), ToolCall{
		Name: ToolMissionGet,
		Arguments: mustArgs(t, map[string]any{
			"mission_id": "mis_1",
			"include":    []string{"records"},
		}),
	})
	if result.Error != nil {
		t.Fatalf("mission get returned error: %#v", result.Error)
	}
	output := result.Content.(missionGetOutput)
	if len(output.Evidence) != 1 || output.Evidence[0].EvidenceID != "evd_1" {
		t.Fatalf("expected evidence in mission output, got %#v", output.Evidence)
	}
	if len(output.Claims) != 1 || output.Claims[0].ClaimID != "clm_1" {
		t.Fatalf("expected claims in mission output, got %#v", output.Claims)
	}
	if len(output.OpenQuestions) != 1 || output.OpenQuestions[0].QuestionID != "qst_1" {
		t.Fatalf("expected questions in mission output, got %#v", output.OpenQuestions)
	}
	if len(service.events) != 0 {
		t.Fatalf("mission get with records must not write events: %#v", service.events)
	}
}

func TestSourcesListReturnsStoredSourceSnapshots(t *testing.T) {
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{
			{
				SnapshotID:  "src_1",
				MissionID:   "mis_1",
				Title:       "Pinned source",
				ArtifactIDs: []string{"art_1"},
				State: app.SourceState{ConfluenceUpdate: &app.ConfluenceUpdateState{
					Status:         app.ConfluenceUpdateStatusAvailable,
					CheckedAt:      time.Date(2026, 7, 14, 1, 2, 3, 0, time.UTC),
					CurrentVersion: 7,
					LatestVersion:  8,
				}},
			},
			{
				SnapshotID:  "src_removed",
				MissionID:   "mis_1",
				Title:       "Removed source",
				ArtifactIDs: []string{"art_removed"},
				State:       app.SourceState{State: app.SourceStateRemoved, Removed: true},
			},
			{
				SnapshotID:  "src_superseded",
				MissionID:   "mis_1",
				Title:       "Superseded source",
				ArtifactIDs: []string{"art_superseded"},
				State:       app.SourceState{Superseded: true, SupersededBy: "src_1"},
			},
		},
	}
	server := NewServer(service)

	result := server.Call(context.Background(), ToolCall{
		Name:      ToolSourcesList,
		Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1"}),
	})
	if result.Error != nil {
		t.Fatalf("sources list returned error: %#v", result.Error)
	}
	output := result.Content.(sourcesListOutput)
	if len(output.Sources) != 1 || output.Sources[0].SnapshotID != "src_1" {
		t.Fatalf("unexpected sources list output: %#v", output)
	}
	if output.Sources[0].RetrievalPolicy != app.SourceRetrievalPolicySnapshotOnly || output.Sources[0].State.State != app.SourceStateActive {
		t.Fatalf("expected explicit policy and active state, got %#v", output.Sources[0])
	}
	if update := output.Sources[0].State.ConfluenceUpdate; update == nil || update.Status != app.ConfluenceUpdateStatusAvailable || update.LatestVersion != 8 {
		t.Fatalf("expected Confluence update state in MCP source list, got %#v", output.Sources[0])
	}

	withRemoved := server.Call(context.Background(), ToolCall{
		Name:      ToolSourcesList,
		Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1", "include_removed": true}),
	})
	if withRemoved.Error != nil {
		t.Fatalf("sources list include_removed returned error: %#v", withRemoved.Error)
	}
	withRemovedOutput := withRemoved.Content.(sourcesListOutput)
	if len(withRemovedOutput.Sources) != 2 || !withRemovedOutput.Sources[1].State.Removed {
		t.Fatalf("expected removed source when include_removed=true, got %#v", withRemovedOutput)
	}

	withSuperseded := server.Call(context.Background(), ToolCall{
		Name:      ToolSourcesList,
		Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1", "include_superseded": true}),
	})
	if withSuperseded.Error != nil {
		t.Fatalf("sources list include_superseded returned error: %#v", withSuperseded.Error)
	}
	withSupersededOutput := withSuperseded.Content.(sourcesListOutput)
	if len(withSupersededOutput.Sources) != 2 || !withSupersededOutput.Sources[1].State.Superseded {
		t.Fatalf("expected superseded source when include_superseded=true, got %#v", withSupersededOutput)
	}
}

func TestSourcesReadReturnsBoundedArtifactContent(t *testing.T) {
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID:  "src_1",
			MissionID:   "mis_1",
			Title:       "Pinned source",
			ArtifactIDs: []string{"art_1"},
		}},
		artifacts: map[string]app.RawArtifact{
			"art_1": {
				ArtifactID: "art_1",
				MissionID:  "mis_1",
				MediaType:  "text/plain; charset=utf-8",
				ByteSize:   11,
				SHA256:     "sha",
				Content:    []byte("hello world"),
			},
		},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_trace"}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_1",
			"max_bytes":   5,
		}),
	})
	if result.Error != nil {
		t.Fatalf("sources read returned error: %#v", result.Error)
	}
	output := result.Content.(sourcesReadOutput)
	if output.Content != "hello" || output.Offset != 0 || output.NextOffset != 5 || output.ContentLength != 11 || !output.Truncated || output.Artifact.ArtifactID != "art_1" {
		t.Fatalf("unexpected sources read output: %#v", output)
	}
	var tracePayload map[string]any
	if err := json.Unmarshal(service.events[len(service.events)-1].Payload, &tracePayload); err != nil {
		t.Fatal(err)
	}
	metrics := tracePayload["io_metrics"].(map[string]any)
	if metrics["read_kind"] != "source_text" || metrics["source_snapshot_id"] != "src_1" || metrics["source_artifact_id"] != "art_1" || metrics["source_media_type"] != "text/plain; charset=utf-8" {
		t.Fatalf("unexpected source read metrics: %#v", metrics)
	}
	if metrics["requested_max_bytes"] != float64(5) || metrics["returned_content_bytes"] != float64(5) || metrics["content_length"] != float64(11) || metrics["next_offset"] != float64(5) || metrics["response_truncated"] != true {
		t.Fatalf("unexpected source read byte metrics: %#v", metrics)
	}

	next := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_1",
			"offset":      output.NextOffset,
			"max_bytes":   20,
		}),
	})
	if next.Error != nil {
		t.Fatalf("sources read next chunk returned error: %#v", next.Error)
	}
	nextOutput := next.Content.(sourcesReadOutput)
	if nextOutput.Content != " world" || nextOutput.Offset != 5 || nextOutput.NextOffset != 0 || nextOutput.Truncated {
		t.Fatalf("unexpected next sources read output: %#v", nextOutput)
	}
}

func TestSourcesReadPDFReturnsExtractedText(t *testing.T) {
	pdfBytes := testPDFBytes(t, []string{"MCP PDF Source", "Alpha code is 67."})
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID:  "src_pdf",
			MissionID:   "mis_1",
			Title:       "PDF source",
			ArtifactIDs: []string{"art_pdf"},
			Connector: app.ConnectorRef{
				ConnectorID:      app.SourceConnectorTypePDFURL,
				ConnectorType:    app.SourceConnectorTypePDFURL,
				ExternalSourceID: "https://example.com/source.pdf",
				ExternalURI:      "https://example.com/source.pdf",
			},
		}},
		artifacts: map[string]app.RawArtifact{
			"art_pdf": {
				ArtifactID: "art_pdf",
				MissionID:  "mis_1",
				MediaType:  "application/pdf",
				ByteSize:   int64(len(pdfBytes)),
				SHA256:     "sha",
				Content:    pdfBytes,
			},
		},
	}
	server := NewServer(service)

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_pdf",
			"max_bytes":   10,
		}),
	})
	if result.Error != nil {
		t.Fatalf("sources read pdf returned error: %#v", result.Error)
	}
	output := result.Content.(sourcesReadOutput)
	if !strings.Contains(output.Content, "MCP PDF") || strings.Contains(output.Content, "%PDF-") {
		t.Fatalf("expected extracted PDF text without raw bytes, got %#v", output)
	}
	if output.Extraction == nil || output.Extraction.Type != "pdf_text" || output.Extraction.PageCount != 1 {
		t.Fatalf("expected pdf_text extraction metadata, got %#v", output)
	}
	if output.ContentLengthKnown || output.Extraction.TextLengthKnown {
		t.Fatalf("expected truncated PDF read to mark lengths unknown, got %#v", output)
	}
	encoded, err := json.Marshal(output)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"content_length_known":false`, `"text_length_known":false`} {
		if !strings.Contains(string(encoded), expected) {
			t.Fatalf("expected %s in JSON output: %s", expected, encoded)
		}
	}
}

func TestSourcesReadMediaSourceReturnsMetadataWithoutBytes(t *testing.T) {
	locators, err := json.Marshal([]app.MediaLocator{{
		LocatorType:       app.SourceLocatorTypeMedia,
		MediaKind:         app.MediaKindImage,
		Provider:          "media_url",
		CanonicalURL:      "https://example.com/image.png",
		DirectMediaURL:    "https://example.com/image.png",
		MIMEType:          "image/png",
		ByteSize:          12,
		Width:             640,
		Height:            480,
		License:           "CC-BY",
		InspectionSupport: "metadata_only_until_vision_engine_configured",
	}})
	if err != nil {
		t.Fatal(err)
	}
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID:  "src_media",
			MissionID:   "mis_1",
			Title:       "Image",
			ArtifactIDs: []string{"art_media"},
			Connector: app.ConnectorRef{
				ConnectorID:      app.SourceConnectorTypeMediaURL,
				ConnectorType:    app.SourceConnectorTypeMediaURL,
				ExternalSourceID: "https://example.com/image.png",
				ExternalURI:      "https://example.com/image.png",
			},
			Locators: locators,
		}},
		artifacts: map[string]app.RawArtifact{
			"art_media": {
				ArtifactID: "art_media",
				MissionID:  "mis_1",
				MediaType:  "image/png",
				ByteSize:   12,
				SHA256:     "sha",
				Content:    []byte{0x89, 0x50, 0x4e, 0x47},
			},
		},
	}
	server := NewServer(service)

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_media",
		}),
	})
	if result.Error != nil {
		t.Fatalf("sources read media returned error: %#v", result.Error)
	}
	output := result.Content.(mediaSourceReadOutput)
	if output.Media.MediaKind != app.MediaKindImage || output.Artifact.ArtifactID != "art_media" {
		t.Fatalf("unexpected media source read output: %#v", output)
	}
	if strings.Contains(output.InspectionNote, "not returned") == false || strings.Contains(output.InspectionNote, "vision engine") == false {
		t.Fatalf("expected boundary inspection note, got %q", output.InspectionNote)
	}
}

func TestSourcesReadLiveLocalPathObservesWithBoundSession(t *testing.T) {
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID: "src_live",
			MissionID:  "mis_1",
			Connector: app.ConnectorRef{
				ConnectorID:   app.SourceConnectorTypeLocalPath,
				ConnectorType: app.SourceConnectorTypeLocalPath,
			},
			Title:  "notes.md",
			Access: app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicyLiveReference},
			State:  app.SourceState{State: app.SourceStateActive},
		}},
		localReadResult: app.ReadLocalPathSourceResult{
			Snapshot: app.SourceSnapshot{
				SnapshotID: "src_live",
				MissionID:  "mis_1",
				Connector:  app.ConnectorRef{ConnectorType: app.SourceConnectorTypeLocalPath},
				Access:     app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicyLiveReference},
				State:      app.SourceState{State: app.SourceStateActive},
			},
			Read: localpath.ReadResult{
				Content: "live text",
				Metadata: localpath.PathMetadata{
					RootID:       "workspace",
					RelativePath: "docs/notes.md",
					Subpath:      "notes.md",
					PathKind:     "file",
					Size:         17,
					NextOffset:   9,
					Truncated:    true,
				},
			},
			ObservationEvent: &app.LedgerEvent{EventID: "evt_observed", MissionID: "mis_1", EventType: app.SourceObservedEvent},
		},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_live",
			"subpath":     "notes.md",
			"max_bytes":   9,
		}),
	})
	if result.Error != nil {
		t.Fatalf("live source read returned error: %#v", result.Error)
	}
	if service.localReadReq.Producer.Type != "agent_session" || service.localReadReq.Producer.ID != "ses_1" || service.localReadReq.ToolSessionID != "ses_1" || service.localReadReq.Subpath != "notes.md" {
		t.Fatalf("live source read did not use bound agent session: %#v", service.localReadReq)
	}
	output := result.Content.(sourcesReadOutput)
	if output.Content != "live text" || output.ObservationEventID != "evt_observed" || output.ObservationMetadata == nil || output.ObservationMetadata.RootID != "workspace" {
		t.Fatalf("unexpected live read output: %#v", output)
	}
	if len(result.CreatedEventIDs) != 1 || result.CreatedEventIDs[0] != "evt_observed" {
		t.Fatalf("expected observation event id in result, got %#v", result.CreatedEventIDs)
	}
	var tracePayload map[string]any
	if err := json.Unmarshal(service.events[len(service.events)-1].Payload, &tracePayload); err != nil {
		t.Fatal(err)
	}
	metrics := tracePayload["io_metrics"].(map[string]any)
	if metrics["read_kind"] != "source_live_reference" || metrics["source_snapshot_id"] != "src_live" || metrics["observation_event_id"] != "evt_observed" || metrics["relative_path"] != "docs/notes.md" || metrics["subpath"] != "notes.md" {
		t.Fatalf("unexpected live source read metrics: %#v", metrics)
	}
	if metrics["returned_content_bytes"] != float64(9) || metrics["content_length"] != float64(17) || metrics["next_offset"] != float64(9) || metrics["response_truncated"] != true {
		t.Fatalf("unexpected live source byte metrics: %#v", metrics)
	}

	unbound := NewServer(service)
	rejected := unbound.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_live",
		}),
	})
	if rejected.Error == nil || rejected.Error.ErrorKind != "validation" || !strings.Contains(rejected.Error.Message, "bound") {
		t.Fatalf("expected unbound live read rejection, got %#v", rejected)
	}
}

func TestSourcesTreeAndGrepLiveLocalPathUseSourceScope(t *testing.T) {
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID: "src_dir",
			MissionID:  "mis_1",
			Connector: app.ConnectorRef{
				ConnectorID:   app.SourceConnectorTypeLocalPath,
				ConnectorType: app.SourceConnectorTypeLocalPath,
			},
			Title:  "docs",
			Access: app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicyLiveReference},
			State:  app.SourceState{State: app.SourceStateActive},
		}},
		localTreeResult: app.TreeLocalPathSourceResult{
			Snapshot: app.SourceSnapshot{
				SnapshotID: "src_dir",
				MissionID:  "mis_1",
				Connector:  app.ConnectorRef{ConnectorType: app.SourceConnectorTypeLocalPath},
				Access:     app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicyLiveReference},
				State:      app.SourceState{State: app.SourceStateActive},
			},
			Tree: localpath.TreeResult{
				RootID:       "workspace",
				RelativePath: "docs/nested",
				Entries:      []localpath.TreeEntry{{Name: "notes.md", RelativePath: "docs/nested/notes.md", PathKind: "file"}},
				Metadata:     localpath.PathMetadata{RootID: "workspace", RelativePath: "docs/nested", Subpath: "nested", PathKind: "directory"},
			},
			ObservationEvent: &app.LedgerEvent{EventID: "evt_tree", MissionID: "mis_1", EventType: app.SourceObservedEvent},
		},
		localGrepResult: app.GrepLocalPathSourceResult{
			Snapshot: app.SourceSnapshot{
				SnapshotID: "src_dir",
				MissionID:  "mis_1",
				Connector:  app.ConnectorRef{ConnectorType: app.SourceConnectorTypeLocalPath},
				Access:     app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicyLiveReference},
				State:      app.SourceState{State: app.SourceStateActive},
			},
			Grep: localpath.GrepResult{
				RootID:       "workspace",
				RelativePath: "docs/nested",
				Query:        "needle",
				Matches:      []localpath.GrepMatch{{RelativePath: "docs/nested/notes.md", Line: 1, Column: 1, Snippet: "needle"}},
				Metadata:     localpath.PathMetadata{RootID: "workspace", RelativePath: "docs/nested", Subpath: "nested", PathKind: "directory"},
			},
			ObservationEvent: &app.LedgerEvent{EventID: "evt_grep", MissionID: "mis_1", EventType: app.SourceObservedEvent},
		},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))

	tree := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesTree,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_dir",
			"subpath":     "nested",
			"depth":       1,
			"limit":       10,
		}),
	})
	if tree.Error != nil {
		t.Fatalf("sources tree returned error: %#v", tree.Error)
	}
	if service.localTreeReq.Subpath != "nested" || service.localTreeReq.Producer.ID != "ses_1" || service.localTreeReq.ToolSessionID != "ses_1" {
		t.Fatalf("tree request was not source-scoped with bound session: %#v", service.localTreeReq)
	}
	treeOutput := tree.Content.(sourcesTreeOutput)
	if treeOutput.ObservationEventID != "evt_tree" || treeOutput.ObservationMetadata == nil || treeOutput.ObservationMetadata.Subpath != "nested" || len(tree.CreatedEventIDs) != 1 {
		t.Fatalf("unexpected tree output: %#v", tree)
	}

	grep := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesGrep,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":   "mis_1",
			"snapshot_id":  "src_dir",
			"subpath":      "nested",
			"query":        "needle",
			"max_snippets": 5,
		}),
	})
	if grep.Error != nil {
		t.Fatalf("sources grep returned error: %#v", grep.Error)
	}
	if service.localGrepReq.Subpath != "nested" || service.localGrepReq.Query != "needle" || service.localGrepReq.MaxSnippets != 5 || service.localGrepReq.Producer.ID != "ses_1" {
		t.Fatalf("grep request was not source-scoped with bound session: %#v", service.localGrepReq)
	}
	grepOutput := grep.Content.(sourcesGrepOutput)
	if grepOutput.ObservationEventID != "evt_grep" || grepOutput.ObservationMetadata == nil || grepOutput.ObservationMetadata.Subpath != "nested" || len(grepOutput.Grep.Matches) != 1 {
		t.Fatalf("unexpected grep output: %#v", grepOutput)
	}
}

func TestSourcesReadRejectsSubpathForSnapshotOnlySources(t *testing.T) {
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID:  "src_1",
			MissionID:   "mis_1",
			Title:       "Pinned source",
			ArtifactIDs: []string{"art_1"},
			Access:      app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicySnapshotOnly},
		}},
		artifacts: map[string]app.RawArtifact{
			"art_1": {
				ArtifactID: "art_1",
				MissionID:  "mis_1",
				MediaType:  "text/plain; charset=utf-8",
				ByteSize:   11,
				SHA256:     "sha",
				Content:    []byte("hello world"),
			},
		},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_1",
			"subpath":     "nested/file.txt",
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" || !strings.Contains(result.Error.Message, "subpath") {
		t.Fatalf("expected snapshot-only subpath validation error, got %#v", result)
	}
}

func TestSourcesReadRejectsSubpathForLiveNonLocalPathSources(t *testing.T) {
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID: "src_media",
			MissionID:  "mis_1",
			Connector: app.ConnectorRef{
				ConnectorID:   app.SourceConnectorTypeMediaURL,
				ConnectorType: app.SourceConnectorTypeMediaURL,
			},
			Access: app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicyLiveReference},
			State:  app.SourceState{State: app.SourceStateActive},
		}},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_media",
			"subpath":     "/path/to/private-key",
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" || !strings.Contains(result.Error.Message, "subpath") {
		t.Fatalf("expected live non-local subpath validation error, got %#v", result)
	}
	var tracePayload map[string]any
	if err := json.Unmarshal(service.events[len(service.events)-1].Payload, &tracePayload); err != nil {
		t.Fatal(err)
	}
	metrics := tracePayload["io_metrics"].(map[string]any)
	if _, ok := metrics["subpath"]; ok {
		t.Fatalf("rejected subpath should not be recorded in io_metrics: %#v", metrics)
	}
}

func TestSourcesReadImageReturnsMetadataOnly(t *testing.T) {
	imageBytes := []byte{0x89, 0x50, 0x4e, 0x47, 0x00, 0x01, 0x02}
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID:  "src_image",
			MissionID:   "mis_1",
			Title:       "Uploaded image",
			ArtifactIDs: []string{"art_image"},
			Connector: app.ConnectorRef{
				ConnectorID:      app.SourceConnectorTypeFileUpload,
				ConnectorType:    app.SourceConnectorTypeFileUpload,
				ExternalSourceID: "file_upload:sha",
			},
		}},
		artifacts: map[string]app.RawArtifact{
			"art_image": {
				ArtifactID: "art_image",
				MissionID:  "mis_1",
				MediaType:  "image/png",
				ByteSize:   int64(len(imageBytes)),
				SHA256:     "sha",
				Content:    imageBytes,
			},
		},
	}
	server := NewServer(service)

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_image",
		}),
	})
	if result.Error != nil {
		t.Fatalf("sources read image returned error: %#v", result.Error)
	}
	output := result.Content.(sourcesReadOutput)
	if !output.MetadataOnly || output.Content != "" || output.Artifact.ReadKind != "metadata" {
		t.Fatalf("expected metadata-only image read, got %#v", output)
	}
}

func TestSourcesReadLiveLocalPathPDFUsesExtractedTextLengthWhenEmpty(t *testing.T) {
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID: "src_pdf_live",
			MissionID:  "mis_1",
			Connector: app.ConnectorRef{
				ConnectorID:   app.SourceConnectorTypeLocalPath,
				ConnectorType: app.SourceConnectorTypeLocalPath,
			},
			Title:  "scan.pdf",
			Access: app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicyLiveReference},
			State:  app.SourceState{State: app.SourceStateActive},
		}},
		localReadResult: app.ReadLocalPathSourceResult{
			Snapshot: app.SourceSnapshot{
				SnapshotID: "src_pdf_live",
				MissionID:  "mis_1",
				Connector:  app.ConnectorRef{ConnectorType: app.SourceConnectorTypeLocalPath},
				Access:     app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicyLiveReference},
				State:      app.SourceState{State: app.SourceStateActive},
			},
			Read: localpath.ReadResult{
				Content: "",
				Metadata: localpath.PathMetadata{
					RootID:          "workspace",
					RelativePath:    "scan.pdf",
					PathKind:        "file",
					Size:            1048576,
					Extraction:      "pdf_text",
					PageCount:       3,
					TextLength:      0,
					TextLengthKnown: true,
					Cap:             "pdf_text",
				},
			},
			ObservationEvent: &app.LedgerEvent{EventID: "evt_pdf_observed", MissionID: "mis_1", EventType: app.SourceObservedEvent},
		},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_pdf_live",
		}),
	})
	if result.Error != nil {
		t.Fatalf("live PDF source read returned error: %#v", result.Error)
	}
	output := result.Content.(sourcesReadOutput)
	if output.Content != "" || output.ContentLength != 0 || !output.ContentLengthKnown {
		t.Fatalf("expected empty extracted PDF text length, got %#v", output)
	}
	if output.Extraction == nil || output.Extraction.Type != "pdf_text" || output.Extraction.TextLength != 0 || !output.Extraction.TextLengthKnown {
		t.Fatalf("expected empty pdf_text extraction metadata, got %#v", output.Extraction)
	}
}

func TestSourcesReadPaginatesUTF8ContentOnRuneBoundaries(t *testing.T) {
	content := []byte("가나다🙂xyz")
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID:  "src_1",
			MissionID:   "mis_1",
			Title:       "Pinned source",
			ArtifactIDs: []string{"art_1"},
		}},
		artifacts: map[string]app.RawArtifact{
			"art_1": {
				ArtifactID: "art_1",
				MissionID:  "mis_1",
				MediaType:  "text/plain; charset=utf-8",
				ByteSize:   int64(len(content)),
				SHA256:     "sha",
				Content:    content,
			},
		},
	}
	server := NewServer(service)

	first := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_1",
			"max_bytes":   4,
		}),
	})
	if first.Error != nil {
		t.Fatalf("sources read first UTF-8 chunk returned error: %#v", first.Error)
	}
	firstOutput := first.Content.(sourcesReadOutput)
	if firstOutput.Content != "가" || firstOutput.NextOffset != len([]byte("가")) || !firstOutput.Truncated {
		t.Fatalf("unexpected first UTF-8 chunk: %#v", firstOutput)
	}

	second := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_1",
			"offset":      firstOutput.NextOffset,
			"max_bytes":   4,
		}),
	})
	if second.Error != nil {
		t.Fatalf("sources read second UTF-8 chunk returned error: %#v", second.Error)
	}
	secondOutput := second.Content.(sourcesReadOutput)
	if secondOutput.Content != "나" || secondOutput.NextOffset != len([]byte("가나")) || !secondOutput.Truncated {
		t.Fatalf("unexpected second UTF-8 chunk: %#v", secondOutput)
	}

	midRune := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_1",
			"offset":      1,
			"max_bytes":   4,
		}),
	})
	if midRune.Error == nil || midRune.Error.ErrorKind != "validation" || !strings.Contains(midRune.Error.Message, "UTF-8 boundary") {
		t.Fatalf("expected mid-rune offset rejection, got %#v", midRune)
	}
}

func TestSourcesReadRejectsArtifactOutsideSnapshot(t *testing.T) {
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID:  "src_1",
			MissionID:   "mis_1",
			ArtifactIDs: []string{"art_1"},
		}},
	}
	server := NewServer(service)

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":  "mis_1",
			"snapshot_id": "src_1",
			"artifact_id": "art_2",
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("expected validation error, got %#v", result)
	}
}

func TestLocalPathMCPToolsUseSharedServices(t *testing.T) {
	service := &fakeMCPService{
		localRoots: []localpath.RootView{{RootID: "workspace", Alias: "Workspace"}},
		localTree: localpath.TreeResult{
			RootID:       "workspace",
			RootAlias:    "Workspace",
			RelativePath: "docs",
			Entries:      []localpath.TreeEntry{{Name: "notes.md", RelativePath: "docs/notes.md", PathKind: "file"}},
		},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))
	operatorServer := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}), WithOperatorSourceMutation())
	ctx := context.Background()

	roots := server.Call(ctx, ToolCall{
		Name:      ToolLocalPathRoots,
		Arguments: mustArgs(t, map[string]any{"mission_id": "mis_1"}),
	})
	if roots.Error != nil {
		t.Fatalf("local path roots returned error: %#v", roots.Error)
	}
	rootOutput := roots.Content.(localPathRootsOutput)
	if len(rootOutput.Roots) != 1 || rootOutput.Roots[0].RootID != "workspace" {
		t.Fatalf("unexpected roots output: %#v", rootOutput)
	}
	encodedRoots, _ := json.Marshal(rootOutput)
	if strings.Contains(string(encodedRoots), "/Users/") || strings.Contains(string(encodedRoots), "/tmp/") {
		t.Fatalf("roots output leaked absolute path: %s", string(encodedRoots))
	}

	tree := server.Call(ctx, ToolCall{
		Name: ToolLocalPathTree,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":    "mis_1",
			"root_id":       "workspace",
			"relative_path": "docs",
			"depth":         1,
			"limit":         10,
		}),
	})
	if tree.Error != nil {
		t.Fatalf("local path tree returned error: %#v", tree.Error)
	}
	treeOutput := tree.Content.(localPathTreeOutput)
	if len(treeOutput.Tree.Entries) != 1 || treeOutput.Tree.Entries[0].RelativePath != "docs/notes.md" {
		t.Fatalf("unexpected tree output: %#v", treeOutput)
	}

	absolute := server.Call(ctx, ToolCall{
		Name: ToolLocalPathTree,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":    "mis_1",
			"root_id":       "workspace",
			"relative_path": "/path/to/secret.txt",
		}),
	})
	if absolute.Error == nil || absolute.Error.ErrorKind != "validation" {
		t.Fatalf("expected absolute path rejection, got %#v", absolute)
	}

	attach := operatorServer.Call(ctx, ToolCall{
		Name: ToolLocalPathAttach,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":      "mis_1",
			"session_id":      "ses_1",
			"idempotency_key": "attach-1",
			"producer":        map[string]any{"type": "agent_session", "id": "ses_1"},
			"snapshot_id":     "src_live",
			"root_id":         "workspace",
			"relative_path":   "docs/notes.md",
			"title":           "Notes",
		}),
	})
	if attach.Error != nil {
		t.Fatalf("local path attach returned error: %#v", attach.Error)
	}
	if service.localAttachReq.RootID != "workspace" || service.localAttachReq.RelativePath != "docs/notes.md" || service.localAttachReq.Producer.ID != "ses_1" {
		t.Fatalf("attach request was not forwarded: %#v", service.localAttachReq)
	}
	attachOutput := attach.Content.(localPathAttachOutput)
	if attachOutput.Snapshot.RetrievalPolicy != app.SourceRetrievalPolicyLiveReference {
		t.Fatalf("expected live reference attach output, got %#v", attachOutput)
	}

	remove := operatorServer.Call(ctx, ToolCall{
		Name: ToolSourcesRemove,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":      "mis_1",
			"session_id":      "ses_1",
			"idempotency_key": "remove-1",
			"producer":        map[string]any{"type": "agent_session", "id": "ses_1"},
			"snapshot_id":     "src_live",
			"reason":          "No longer needed",
		}),
	})
	if remove.Error != nil {
		t.Fatalf("source remove returned error: %#v", remove.Error)
	}
	if service.sourceRemoveReq.SnapshotID != "src_live" || service.sourceRemoveReq.Reason != "No longer needed" {
		t.Fatalf("remove request was not forwarded: %#v", service.sourceRemoveReq)
	}
	removeOutput := remove.Content.(sourceStateChangeOutput)
	if !removeOutput.Snapshot.State.Removed {
		t.Fatalf("expected removed source state, got %#v", removeOutput)
	}

	restore := operatorServer.Call(ctx, ToolCall{
		Name: ToolSourcesRestore,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":      "mis_1",
			"session_id":      "ses_1",
			"idempotency_key": "restore-1",
			"producer":        map[string]any{"type": "agent_session", "id": "ses_1"},
			"snapshot_id":     "src_live",
		}),
	})
	if restore.Error != nil {
		t.Fatalf("source restore returned error: %#v", restore.Error)
	}
	if service.sourceRestoreReq.SnapshotID != "src_live" {
		t.Fatalf("restore request was not forwarded: %#v", service.sourceRestoreReq)
	}
}

func TestBoundServerRejectsDifferentMissionRead(t *testing.T) {
	service := &fakeMCPService{
		sources: []app.SourceSnapshot{{
			SnapshotID:  "src_1",
			MissionID:   "mis_1",
			ArtifactIDs: []string{"art_1"},
		}},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_bound"}))

	result := server.Call(context.Background(), ToolCall{
		Name:      ToolSourcesList,
		Arguments: mustArgs(t, map[string]any{"mission_id": "mis_2"}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("expected validation error for cross-mission read, got %#v", result)
	}
}

func TestBoundServerRejectsDifferentMutatingSession(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_bound"}), WithLegacyResearchLoop())

	result := server.Call(context.Background(), ToolCall{
		Name: ToolQuestionsPropose,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":        "mis_1",
			"session_id":        "ses_1",
			"idempotency_key":   "qst-bound",
			"producer":          map[string]any{"type": "agent_session", "id": "ses_1"},
			"question_id":       "qst_bound",
			"event_id":          "evt_bound",
			"proposal_id":       "prp_bound",
			"proposal_event_id": "evt_bound_proposal",
			"text":              "Should this write be rejected?",
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("expected validation error for cross-session write, got %#v", result)
	}
	if len(service.events) != 0 || len(service.questionRequests) != 0 {
		t.Fatalf("cross-session write reached storage: events=%#v questions=%#v", service.events, service.questionRequests)
	}
}

func TestSourcesSearchUsesMountedConnector(t *testing.T) {
	service := &fakeMCPService{
		searchResult: app.Liquid2SourceSearchResult{
			Candidates: []app.Liquid2SourceCandidate{{
				Connector:   app.ConnectorRef{ConnectorID: app.Liquid2ConnectorID, ExternalSourceID: "doc_1"},
				Title:       "Candidate",
				CanSnapshot: true,
			}},
			NextCursor: "cursor_2",
		},
	}
	server := NewServer(service, WithLiquid2Connector(fakeMCPConnector{}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesSearch,
		Arguments: mustArgs(t, map[string]any{
			"mission_id": "mis_1",
			"query":      "storage",
			"connectors": []string{"liquid2", "liquid2"},
			"limit":      5,
		}),
	})
	if result.Error != nil {
		t.Fatalf("sources search returned error: %#v", result.Error)
	}
	if !service.searchUsedConnector {
		t.Fatalf("search did not receive the mounted connector")
	}
	if service.searchRequest.MissionID != "mis_1" || service.searchRequest.Query != "storage" {
		t.Fatalf("unexpected search request: %#v", service.searchRequest)
	}
	output := result.Content.(sourcesSearchOutput)
	if len(output.Candidates) != 1 || output.NextCursors[app.Liquid2ConnectorID] != "cursor_2" {
		t.Fatalf("unexpected search output: %#v", output)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	if strings.Contains(string(encoded), "ConnectorID") || !strings.Contains(string(encoded), "connector_id") {
		t.Fatalf("search output leaked internal field names: %s", string(encoded))
	}
}

func TestSourcesSearchUsesConfluenceFactoryForDiscoveryOnly(t *testing.T) {
	service := &fakeMCPService{
		confluenceAccess: app.ConnectorAccessProjection{
			MissionID:    "mis_1",
			ConnectorID:  app.ConfluenceConnectorID,
			Enabled:      true,
			ConnectionID: "cnf_1",
			CloudID:      "cloud_1",
			SpaceKey:     "ENG",
			Status:       app.ConnectorAccessStatusEnabled,
		},
		confluenceSearchResult: app.ConfluenceSourceSearchResult{
			Candidates: []app.ConfluenceSourceCandidate{{
				Connector: app.ConnectorRef{
					ConnectorID:      app.ConfluenceConnectorID,
					ConnectorType:    app.ConfluenceConnectorType,
					ExternalSourceID: app.ConfluenceExternalSourceID("cloud_1", "123"),
					ExternalVersion:  "7",
				},
				Title:       "Roadmap",
				SourceURI:   "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap",
				Summary:     "secret body excerpt",
				Version:     7,
				CanSnapshot: true,
			}},
		},
	}
	var factoryReq ConfluenceConnectorRequest
	server := NewServer(service, WithConfluenceConnectorFactory(func(_ context.Context, req ConfluenceConnectorRequest) (app.ConfluenceSourceConnector, error) {
		factoryReq = req
		return fakeMCPConfluenceConnector{}, nil
	}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesSearch,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":    "mis_1",
			"query":         "roadmap",
			"connectors":    []string{"confluence"},
			"connection_id": "cnf_1",
			"cloud_id":      "cloud_1",
			"space_key":     "ENG",
		}),
	})
	if result.Error != nil {
		t.Fatalf("sources search returned error: %#v", result.Error)
	}
	if factoryReq.ConnectionID != "cnf_1" || factoryReq.CloudID != "cloud_1" {
		t.Fatalf("unexpected factory request: %#v", factoryReq)
	}
	if factoryReq.SpaceKey != "ENG" {
		t.Fatalf("expected factory request to use granted space, got %#v", factoryReq)
	}
	if service.confluenceSearchRequest.CloudID != "cloud_1" || service.confluenceSearchRequest.SpaceKey != "ENG" {
		t.Fatalf("unexpected confluence search request: %#v", service.confluenceSearchRequest)
	}
	output := result.Content.(sourcesSearchOutput)
	if len(output.Candidates) != 1 || output.Candidates[0].Connector.ConnectorID != app.ConfluenceConnectorID {
		t.Fatalf("unexpected search output: %#v", output)
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal result: %v", err)
	}
	for _, leaked := range []string{"body", "storage", "plain_text", "secret"} {
		if strings.Contains(string(encoded), leaked) {
			t.Fatalf("confluence MCP result leaked %q: %s", leaked, string(encoded))
		}
	}
}

func TestSourcesSearchDeniesConfluenceWithoutMissionGrant(t *testing.T) {
	service := &fakeMCPService{}
	factoryCalled := false
	server := NewServer(service, WithConfluenceConnectorFactory(func(context.Context, ConfluenceConnectorRequest) (app.ConfluenceSourceConnector, error) {
		factoryCalled = true
		return fakeMCPConfluenceConnector{}, nil
	}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesSearch,
		Arguments: mustArgs(t, map[string]any{
			"mission_id": "mis_1",
			"query":      "roadmap",
			"connectors": []string{"confluence"},
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "permission" {
		t.Fatalf("expected permission error, got %#v", result)
	}
	if factoryCalled {
		t.Fatal("confluence factory must not be called before a mission grant exists")
	}
}

func TestSourcesSearchRejectsConfluenceGrantMismatch(t *testing.T) {
	service := &fakeMCPService{
		confluenceAccess: app.ConnectorAccessProjection{
			MissionID:    "mis_1",
			ConnectorID:  app.ConfluenceConnectorID,
			Enabled:      true,
			ConnectionID: "cnf_granted",
			CloudID:      "cloud_1",
			Status:       app.ConnectorAccessStatusEnabled,
		},
	}
	factoryCalled := false
	server := NewServer(service, WithConfluenceConnectorFactory(func(context.Context, ConfluenceConnectorRequest) (app.ConfluenceSourceConnector, error) {
		factoryCalled = true
		return fakeMCPConfluenceConnector{}, nil
	}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesSearch,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":    "mis_1",
			"query":         "roadmap",
			"connectors":    []string{"confluence"},
			"connection_id": "cnf_other",
			"cloud_id":      "cloud_1",
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "permission" {
		t.Fatalf("expected permission mismatch error, got %#v", result)
	}
	if factoryCalled {
		t.Fatal("confluence factory must not be called for grant mismatch")
	}
}

func TestSourceCandidatesProposeRecordsReviewAndStartsStaging(t *testing.T) {
	service := &fakeMCPService{}
	fetchStarted := make(chan struct{})
	releaseFetch := make(chan struct{})
	server := NewServer(service, WithBinding(Binding{
		MissionID:          "mis_1",
		AgentSessionID:     "ses_1",
		CurrentUserEventID: "evt_user",
		AgentExecutor:      "codex",
	}), WithSourceCandidateFetcher(func(ctx context.Context, rawURL string) (urlsource.Fetched, error) {
		close(fetchStarted)
		select {
		case <-releaseFetch:
			return urlsource.Fetched{Content: []byte("candidate body"), MediaType: "text/plain; charset=utf-8", Title: "Roadmap"}, nil
		case <-ctx.Done():
			return urlsource.Fetched{}, ctx.Err()
		}
	}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourceCandidatesPropose,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":      "mis_1",
			"session_id":      "ses_1",
			"idempotency_key": "source-candidate-1",
			"producer":        map[string]any{"type": "agent_session", "id": "ses_1"},
			"candidates": []map[string]any{{
				"url":    "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap#section",
				"reason": "이 Confluence 문서는 조사 중 언급된 제품 로드맵의 원문이므로 사용자가 소스로 채택할지 검토할 필요가 있습니다.",
			}},
		}),
	})
	if result.Error != nil {
		t.Fatalf("source candidate propose returned error: %#v", result.Error)
	}
	if !result.RequiresUserApproval {
		t.Fatalf("source candidate proposal must remain user-review gated: %#v", result)
	}
	if len(result.CreatedEventIDs) != 2 {
		t.Fatalf("expected proposed and staging_started events, got %#v", result.CreatedEventIDs)
	}
	var candidateEvent app.AppendEventRequest
	for _, event := range service.events {
		if event.EventType == "source.candidate.proposed" {
			candidateEvent = event
			break
		}
	}
	if candidateEvent.EventType == "" {
		t.Fatalf("expected source.candidate.proposed event, got %#v", service.events)
	}
	var payload struct {
		Kind           string                         `json:"kind"`
		Source         string                         `json:"source"`
		UserEventID    string                         `json:"user_event_id"`
		AgentExecutor  string                         `json:"agent_executor"`
		ToolSessionID  string                         `json:"tool_session_id"`
		CandidateCount int                            `json:"candidate_count"`
		Candidates     []sourceCandidateProposalEvent `json:"candidates"`
	}
	if err := json.Unmarshal(candidateEvent.Payload, &payload); err != nil {
		t.Fatalf("unmarshal source candidate payload: %v", err)
	}
	if payload.Kind != "source_candidate_proposed" || payload.Source != "mcp" || payload.UserEventID != "evt_user" || payload.AgentExecutor != "codex" || payload.ToolSessionID != "ses_1" {
		t.Fatalf("unexpected source candidate payload metadata: %#v", payload)
	}
	if payload.CandidateCount != 1 || len(payload.Candidates) != 1 {
		t.Fatalf("unexpected source candidate count: %#v", payload)
	}
	if got := payload.Candidates[0].URL; got != "https://docs.atlassian.net/wiki/spaces/ENG/pages/123/Roadmap" {
		t.Fatalf("expected normalized candidate URL without fragment, got %q", got)
	}
	if got := payload.Candidates[0].Title; got != "Roadmap" {
		t.Fatalf("expected Confluence candidate title to be derived from URL, got %q", got)
	}
	if payload.Candidates[0].State != "proposed" || payload.Candidates[0].Reason == "" {
		t.Fatalf("unexpected candidate payload: %#v", payload.Candidates[0])
	}
	if !fakeMCPHasEventType(service.events, "source.candidate.staging_started") {
		t.Fatalf("expected staging_started event, got %#v", service.events)
	}
	output, ok := result.Content.(sourceCandidatesProposeOutput)
	if !ok {
		t.Fatalf("unexpected output type: %#v", result.Content)
	}
	if len(output.Staging) != 1 || output.Staging[0].StagingState != "fetching" {
		t.Fatalf("expected fetching staging output, got %#v", output.Staging)
	}
	select {
	case <-fetchStarted:
	case <-time.After(time.Second):
		t.Fatal("expected background source candidate fetch to start")
	}
	close(releaseFetch)
}

func TestSourceCandidatesReadReturnsStagedUnapprovedCandidate(t *testing.T) {
	content := []byte("candidate body")
	sum := sha256.Sum256(content)
	service := &fakeMCPService{
		ledgerEvents: []app.LedgerEvent{{
			EventID:   "evt_staged",
			MissionID: "mis_1",
			Sequence:  1,
			EventType: "source.candidate.staged",
			Payload: mustJSON(map[string]any{
				"url":                "https://example.com/source",
				"proposal_event_id":  "evt_proposed",
				"artifact_id":        "art_candidate",
				"approval_state":     "unapproved_candidate",
				"not_report_default": true,
			}),
		}},
		artifacts: map[string]app.RawArtifact{
			"art_candidate": {
				ArtifactID: "art_candidate",
				MissionID:  "mis_1",
				MediaType:  "text/plain; charset=utf-8",
				ByteSize:   int64(len(content)),
				SHA256:     hex.EncodeToString(sum[:]),
				Content:    content,
			},
		},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1"}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourceCandidatesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id": "mis_1",
			"url":        "https://example.com/source#ignored",
			"max_bytes":  64,
		}),
	})
	if result.Error != nil {
		t.Fatalf("source candidate read returned error: %#v", result.Error)
	}
	output, ok := result.Content.(sourceCandidatesReadOutput)
	if !ok {
		t.Fatalf("unexpected output type: %#v", result.Content)
	}
	if output.ApprovalState != "unapproved_candidate" || !output.NotReportDefault {
		t.Fatalf("candidate read must identify unapproved candidate boundary: %#v", output)
	}
	if output.StagingState != "staged" || output.Content != "candidate body" {
		t.Fatalf("unexpected staged candidate read output: %#v", output)
	}
}

func TestSourceCandidatesReadReturnsPDFTextForStagedCandidate(t *testing.T) {
	content := testPDFBytes(t, []string{"Candidate PDF Source", "Alpha code is 92."})
	sum := sha256.Sum256(content)
	service := &fakeMCPService{
		ledgerEvents: []app.LedgerEvent{{
			EventID:   "evt_staged",
			MissionID: "mis_1",
			Sequence:  1,
			EventType: "source.candidate.staged",
			Payload: mustJSON(map[string]any{
				"url":                "https://example.com/candidate.pdf",
				"proposal_event_id":  "evt_proposed",
				"artifact_id":        "art_candidate",
				"approval_state":     "unapproved_candidate",
				"not_report_default": true,
			}),
		}},
		artifacts: map[string]app.RawArtifact{
			"art_candidate": {
				ArtifactID: "art_candidate",
				MissionID:  "mis_1",
				MediaType:  "application/pdf",
				ByteSize:   int64(len(content)),
				SHA256:     hex.EncodeToString(sum[:]),
				Content:    content,
			},
		},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1"}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourceCandidatesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id": "mis_1",
			"url":        "https://example.com/candidate.pdf",
			"max_bytes":  20000,
		}),
	})
	if result.Error != nil {
		t.Fatalf("source candidate read returned error: %#v", result.Error)
	}
	output, ok := result.Content.(sourceCandidatesReadOutput)
	if !ok {
		t.Fatalf("unexpected output type: %#v", result.Content)
	}
	if output.Extraction == nil || output.Extraction.Type != "pdf_text" {
		t.Fatalf("expected PDF extraction metadata, got %#v", output)
	}
	if !strings.Contains(output.Content, "Candidate PDF Source") || strings.Contains(output.Content, "%PDF-") {
		t.Fatalf("expected extracted PDF text without raw PDF bytes, got %#v", output.Content)
	}
}

func TestSourceCandidatesReadRejectsAmbiguousProposalOnlySelector(t *testing.T) {
	service := &fakeMCPService{
		ledgerEvents: []app.LedgerEvent{
			{
				EventID:   "evt_staged_a",
				MissionID: "mis_1",
				Sequence:  1,
				EventType: "source.candidate.staged",
				Payload: mustJSON(map[string]any{
					"url":               "https://example.com/a",
					"proposal_event_id": "evt_proposed",
					"staging_event_id":  "evt_started_a",
					"artifact_id":       "art_a",
				}),
			},
			{
				EventID:   "evt_staged_b",
				MissionID: "mis_1",
				Sequence:  2,
				EventType: "source.candidate.staged",
				Payload: mustJSON(map[string]any{
					"url":               "https://example.com/b",
					"proposal_event_id": "evt_proposed",
					"staging_event_id":  "evt_started_b",
					"artifact_id":       "art_b",
				}),
			},
		},
	}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1"}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourceCandidatesRead,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":        "mis_1",
			"proposal_event_id": "evt_proposed",
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("expected ambiguous proposal selector validation error, got %#v", result)
	}
}

func TestSourceCandidatesProposeRequiresBoundSession(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithBinding(Binding{MissionID: "mis_1", AgentSessionID: "ses_1"}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourceCandidatesPropose,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":      "mis_1",
			"session_id":      "ses_other",
			"idempotency_key": "source-candidate-1",
			"producer":        map[string]any{"type": "agent_session", "id": "ses_other"},
			"candidates": []map[string]any{{
				"url":    "https://example.com/source",
				"reason": "원문 확인이 필요한 자료입니다.",
			}},
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("expected bound-session validation error, got %#v", result)
	}
	if fakeMCPHasEventType(service.events, "source.candidate.proposed") {
		t.Fatalf("source candidate event must not be created for an unbound session: %#v", service.events)
	}
}

func TestSourcesSearchRequiresMountedConnector(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service)

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesSearch,
		Arguments: mustArgs(t, map[string]any{
			"mission_id": "mis_1",
			"query":      "storage",
		}),
	})

	if result.Error == nil || result.Error.ErrorKind != "connector" {
		t.Fatalf("expected connector error for missing connector, got %#v", result)
	}
	if service.searchUsedConnector {
		t.Fatal("missing connector search reached storage")
	}
}

func TestSourcesSnapshotRequiresUserApprovalAndDoesNotWrite(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithLiquid2Connector(fakeMCPConnector{}))

	result := server.Call(context.Background(), ToolCall{
		Name: ToolSourcesSnapshot,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":      "mis_1",
			"session_id":      "ses_1",
			"idempotency_key": "snap-1",
			"producer":        map[string]any{"type": "agent_session", "id": "ses_1"},
			"connector":       map[string]any{"connector_id": "liquid2", "external_source_id": "doc_1"},
			"artifact_id":     "art_1",
			"snapshot_id":     "src_1",
			"event_id":        "evt_snapshot",
			"ranges":          []map[string]any{{"content_id": "content_1", "start": 0, "end": 10}},
			"reason":          "Support evidence proposal.",
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "approval_required" || !result.RequiresUserApproval {
		t.Fatalf("expected approval required snapshot result, got %#v", result)
	}
	if service.snapshotRequest.ExternalSourceID != "" || len(service.events) != 0 {
		t.Fatalf("snapshot approval guard wrote state: snapshot=%#v events=%#v", service.snapshotRequest, service.events)
	}
}

func TestEvidenceProposeCreatesPendingProposalAndReplaysIdempotently(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithLegacyResearchLoop())
	call := ToolCall{
		Name: ToolEvidencePropose,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":        "mis_1",
			"session_id":        "ses_1",
			"idempotency_key":   "evd-1",
			"producer":          map[string]any{"type": "agent_session", "id": "ses_1"},
			"evidence_id":       "evd_1",
			"event_id":          "evt_evidence",
			"proposal_id":       "prp_1",
			"proposal_event_id": "evt_proposal",
			"proposal_title":    "Review evidence",
			"summary":           "Pinned quote supports the storage decision.",
			"evidence_type":     "quote",
			"snapshot_refs": []map[string]any{{
				"snapshot_id": "src_1",
				"artifact_id": "art_1",
				"locator":     map[string]any{"locator_type": "text_quote", "exact": "storage"},
			}},
			"confidence": map[string]any{"level": "medium", "rationale": "Pinned source."},
		}),
	}

	first := server.Call(context.Background(), call)
	second := server.Call(context.Background(), call)
	if first.Error != nil {
		t.Fatalf("evidence propose returned error: %#v", first.Error)
	}
	if second.Error != nil {
		t.Fatalf("evidence propose replay returned error: %#v", second.Error)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("idempotent replay changed result:\nfirst=%#v\nsecond=%#v", first, second)
	}
	if len(service.events) != 2 || len(service.evidenceRequests) != 1 || len(service.proposalRequests) != 1 {
		t.Fatalf("idempotency did not suppress duplicate writes: events=%d evidence=%d proposals=%d",
			len(service.events), len(service.evidenceRequests), len(service.proposalRequests))
	}
	if service.evidenceRequests[0].State != "proposed" {
		t.Fatalf("evidence tool changed accepted state: %#v", service.evidenceRequests[0])
	}
	proposal := service.proposalRequests[0]
	if proposal.State != "pending_review" || proposal.RequestedDecision != "approve" {
		t.Fatalf("proposal was not submitted for user review: %#v", proposal)
	}
	if !first.RequiresUserApproval || first.ProposalID != "prp_1" {
		t.Fatalf("unexpected proposal result: %#v", first)
	}
}

func TestEvidenceProposeAcceptsResearchSignalType(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithLegacyResearchLoop())

	result := server.Call(context.Background(), ToolCall{
		Name: ToolEvidencePropose,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":        "mis_1",
			"session_id":        "ses_1",
			"idempotency_key":   "evd-reaction",
			"producer":          map[string]any{"type": "agent_session", "id": "ses_1"},
			"evidence_id":       "evd_reaction",
			"event_id":          "evt_evidence_reaction",
			"proposal_id":       "prp_reaction",
			"proposal_event_id": "evt_proposal_reaction",
			"proposal_title":    "Review reaction signal",
			"summary":           "Audience reactions repeatedly mention one expectation.",
			"evidence_type":     "reaction",
			"snapshot_refs": []map[string]any{{
				"snapshot_id": "src_1",
				"artifact_id": "art_1",
				"locator":     map[string]any{"locator_type": "text_quote", "exact": "expectation"},
			}},
			"confidence": map[string]any{"level": "low", "rationale": "Useful signal, not a strict fact."},
		}),
	})
	if result.Error != nil {
		t.Fatalf("evidence propose reaction returned error: %#v", result.Error)
	}
	if len(service.evidenceRequests) != 1 || service.evidenceRequests[0].EvidenceType != "reaction" {
		t.Fatalf("expected reaction evidence request, got %#v", service.evidenceRequests)
	}
}

func TestMutatingToolRejectsSameIdempotencyKeyWithDifferentPayload(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithLegacyResearchLoop())
	first := evidenceProposalArgs("evd_1", "idem-conflict")
	second := evidenceProposalArgs("evd_2", "idem-conflict")

	okResult := server.Call(context.Background(), ToolCall{Name: ToolEvidencePropose, Arguments: first})
	conflict := server.Call(context.Background(), ToolCall{Name: ToolEvidencePropose, Arguments: second})
	if okResult.Error != nil {
		t.Fatalf("first evidence proposal returned error: %#v", okResult.Error)
	}
	if conflict.Error == nil || conflict.Error.ErrorKind != "conflict" {
		t.Fatalf("expected idempotency conflict, got %#v", conflict)
	}
	if len(service.events) != 2 || len(service.evidenceRequests) != 1 || service.evidenceRequests[0].EvidenceID != "evd_1" {
		t.Fatalf("conflicting replay wrote state: events=%d evidence=%#v", len(service.events), service.evidenceRequests)
	}
}

func TestClaimProposeCannotApproveClaimState(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithLegacyResearchLoop())

	result := server.Call(context.Background(), ToolCall{
		Name: ToolClaimsPropose,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":              "mis_1",
			"session_id":              "ses_1",
			"idempotency_key":         "clm-1",
			"producer":                map[string]any{"type": "agent_session", "id": "ses_1"},
			"claim_id":                "clm_1",
			"event_id":                "evt_claim",
			"proposal_id":             "prp_claim",
			"proposal_event_id":       "evt_claim_proposal",
			"text":                    "Plasma must keep its DB separate from Liquid2.",
			"claim_type":              "decision",
			"supporting_evidence_ids": []string{"evd_1"},
			"confidence":              map[string]any{"level": "high"},
		}),
	})
	if result.Error != nil {
		t.Fatalf("claim propose returned error: %#v", result.Error)
	}
	if len(service.claimRequests) != 1 {
		t.Fatalf("expected one claim request, got %d", len(service.claimRequests))
	}
	claim := service.claimRequests[0]
	if claim.State != "proposed" || claim.Approval.State != "pending" || !claim.Approval.Required {
		t.Fatalf("claim tool bypassed approval boundary: %#v", claim)
	}
}

func TestClaimConfidenceUpdateIsAdvisoryTool(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithLegacyResearchLoop())

	result := server.Call(context.Background(), ToolCall{
		Name: ToolClaimConfidence,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":         "mis_1",
			"session_id":         "ses_1",
			"idempotency_key":    "clm-confidence-1",
			"producer":           map[string]any{"type": "agent_session", "id": "ses_1"},
			"claim_id":           "clm_1",
			"event_id":           "evt_confidence",
			"basis_evidence_ids": []string{"evd_1"},
			"confidence":         map[string]any{"level": "high", "rationale": "Pinned evidence now directly supports the claim."},
		}),
	})
	if result.Error != nil {
		t.Fatalf("confidence update returned error: %#v", result.Error)
	}
	if result.RequiresUserApproval {
		t.Fatalf("confidence update should not require approval: %#v", result)
	}
	if len(service.confidenceRequests) != 1 {
		t.Fatalf("expected one confidence request, got %d", len(service.confidenceRequests))
	}
	req := service.confidenceRequests[0]
	if req.ClaimID != "clm_1" || req.Confidence.Level != "high" || req.Origin != "agent" {
		t.Fatalf("unexpected confidence request: %#v", req)
	}
	if len(service.events) != 1 || service.events[0].EventType != app.ClaimConfidenceUpdatedEvent {
		t.Fatalf("unexpected confidence event writes: %#v", service.events)
	}
}

func TestMutatingToolRejectsNonAgentSessionProducerBeforeWrites(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithLegacyResearchLoop())

	result := server.Call(context.Background(), ToolCall{
		Name: ToolQuestionsPropose,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":        "mis_1",
			"session_id":        "ses_1",
			"idempotency_key":   "qst-1",
			"producer":          map[string]any{"type": "user", "id": "ses_1"},
			"question_id":       "qst_1",
			"event_id":          "evt_question",
			"proposal_id":       "prp_question",
			"proposal_event_id": "evt_question_proposal",
			"text":              "Which sources remain unverified?",
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("expected validation error, got %#v", result)
	}
	if len(service.events) != 0 || len(service.questionRequests) != 0 {
		t.Fatalf("invalid producer wrote state: events=%#v questions=%#v", service.events, service.questionRequests)
	}
}

func TestMutatingToolRequiresExplicitProducer(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithLegacyResearchLoop())

	result := server.Call(context.Background(), ToolCall{
		Name: ToolQuestionsPropose,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":        "mis_1",
			"session_id":        "ses_1",
			"idempotency_key":   "qst-missing-producer",
			"question_id":       "qst_missing_producer",
			"event_id":          "evt_missing_producer",
			"proposal_id":       "prp_missing_producer",
			"proposal_event_id": "evt_missing_producer_proposal",
			"text":              "Which producer made this proposal?",
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("expected validation error, got %#v", result)
	}
	if len(service.events) != 0 || len(service.questionRequests) != 0 {
		t.Fatalf("missing producer wrote state: events=%#v questions=%#v", service.events, service.questionRequests)
	}
}

func TestProposalValidationRejectsUnsupportedEnumBeforeWrites(t *testing.T) {
	service := &fakeMCPService{}
	server := NewServer(service, WithLegacyResearchLoop())

	result := server.Call(context.Background(), ToolCall{
		Name: ToolEvidencePropose,
		Arguments: mustArgs(t, map[string]any{
			"mission_id":        "mis_1",
			"session_id":        "ses_1",
			"idempotency_key":   "evd-bad-type",
			"producer":          map[string]any{"type": "agent_session", "id": "ses_1"},
			"evidence_id":       "evd_bad",
			"event_id":          "evt_bad_evidence",
			"proposal_id":       "prp_bad",
			"proposal_event_id": "evt_bad_proposal",
			"summary":           "This should not be written.",
			"evidence_type":     "user_assertion",
			"snapshot_refs": []map[string]any{{
				"snapshot_id": "src_1",
				"artifact_id": "art_1",
			}},
		}),
	})
	if result.Error == nil || result.Error.ErrorKind != "validation" {
		t.Fatalf("expected validation error, got %#v", result)
	}
	if len(service.events) != 0 || len(service.evidenceRequests) != 0 || len(service.proposalRequests) != 0 {
		t.Fatalf("unsupported evidence type wrote state: events=%#v evidence=%#v proposals=%#v",
			service.events, service.evidenceRequests, service.proposalRequests)
	}
}

func mustArgs(t *testing.T, value any) json.RawMessage {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	return encoded
}

func testPDFBytes(t *testing.T, lines []string) []byte {
	t.Helper()
	var stream bytes.Buffer
	stream.WriteString("BT\n/F1 12 Tf\n72 720 Td\n")
	for i, line := range lines {
		if i > 0 {
			stream.WriteString("0 -18 Td\n")
		}
		fmt.Fprintf(&stream, "(%s) Tj\n", escapeTestPDFString(line))
	}
	stream.WriteString("ET\n")
	var compressed bytes.Buffer
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(stream.Bytes()); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d /Filter /FlateDecode >>\nstream\n%s\nendstream", compressed.Len(), compressed.String()),
	}
	var out bytes.Buffer
	out.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for i, obj := range objects {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj\n%s\nendobj\n", i+1, obj)
	}
	xref := out.Len()
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i <= len(objects); i++ {
		fmt.Fprintf(&out, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&out, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	return out.Bytes()
}

func escapeTestPDFString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `(`, `\(`)
	value = strings.ReplaceAll(value, `)`, `\)`)
	return value
}

func toolNames(tools []ToolDefinition) []string {
	names := make([]string, 0, len(tools))
	for _, tool := range tools {
		names = append(names, tool.Name)
	}
	return names
}

func cloneMap(values map[string]any) map[string]any {
	cloned := make(map[string]any, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func fakeMCPHasEventType(events []app.AppendEventRequest, eventType string) bool {
	for _, event := range events {
		if event.EventType == eventType {
			return true
		}
	}
	return false
}

func containsMermaidIssue(issues []mermaidpkg.Issue, kind string) bool {
	for _, issue := range issues {
		if issue.Kind == kind {
			return true
		}
	}
	return false
}

func toolByName(t *testing.T, tools []ToolDefinition, name string) ToolDefinition {
	t.Helper()
	for _, tool := range tools {
		if tool.Name == name {
			return tool
		}
	}
	t.Fatalf("tool %s not found in %#v", name, toolNames(tools))
	return ToolDefinition{}
}

func schemaObjectKindEnum(t *testing.T, schema json.RawMessage) []string {
	t.Helper()
	var decoded map[string]any
	if err := json.Unmarshal(schema, &decoded); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	properties, ok := decoded["properties"].(map[string]any)
	if !ok {
		t.Fatalf("schema has no properties: %#v", decoded)
	}
	objectKind, ok := properties["object_kind"].(map[string]any)
	if !ok {
		t.Fatalf("schema has no object_kind property: %#v", properties)
	}
	rawEnum, ok := objectKind["enum"].([]any)
	if !ok {
		t.Fatalf("object_kind has no enum: %#v", objectKind)
	}
	values := make([]string, 0, len(rawEnum))
	for _, value := range rawEnum {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("object_kind enum contains non-string value: %#v", rawEnum)
		}
		values = append(values, text)
	}
	return values
}

func evidenceProposalArgs(evidenceID, idempotencyKey string) json.RawMessage {
	encoded, err := json.Marshal(map[string]any{
		"mission_id":        "mis_1",
		"session_id":        "ses_1",
		"idempotency_key":   idempotencyKey,
		"producer":          map[string]any{"type": "agent_session", "id": "ses_1"},
		"evidence_id":       evidenceID,
		"event_id":          "evt_" + evidenceID,
		"proposal_id":       "prp_" + evidenceID,
		"proposal_event_id": "evt_prp_" + evidenceID,
		"proposal_title":    "Review evidence",
		"summary":           "Pinned quote supports the storage decision.",
		"evidence_type":     "quote",
		"snapshot_refs": []map[string]any{{
			"snapshot_id": "src_1",
			"artifact_id": "art_1",
			"locator":     map[string]any{"locator_type": "text_quote", "exact": "storage"},
		}},
		"confidence": map[string]any{"level": "medium"},
	})
	if err != nil {
		panic(err)
	}
	return encoded
}

type fakeMCPService struct {
	projection   app.MissionProjection
	ledgerEvents []app.LedgerEvent
	sources      []app.SourceSnapshot
	artifacts    map[string]app.RawArtifact
	evidence     []app.EvidenceRecord
	claims       []app.ClaimRecord
	questions    []app.QuestionRecord
	outline      app.ResearchIDEOutline
	page         app.ResearchIDEPage
	read         app.ResearchIDEObjectRead
	grep         app.ResearchIDEGrepResult
	refs         app.ResearchIDEReferences
	lastRead     app.ResearchIDEReadRequest
	workflowRuns []app.WorkflowRunView

	searchUsedConnector     bool
	searchRequest           app.Liquid2SourceSearchRequest
	searchResult            app.Liquid2SourceSearchResult
	confluenceSearchRequest app.ConfluenceSourceSearchRequest
	confluenceSearchResult  app.ConfluenceSourceSearchResult
	confluenceAccess        app.ConnectorAccessProjection

	snapshotRequest app.SnapshotLiquid2SourceRequest
	snapshotResult  app.Liquid2SnapshotResult

	localRoots          []localpath.RootView
	localTree           localpath.TreeResult
	localAttachReq      app.AttachLocalPathSourceRequest
	localAttachResult   app.LocalPathSourceResult
	localReadReq        app.ReadLocalPathSourceRequest
	localReadResult     app.ReadLocalPathSourceResult
	localTreeReq        app.TreeLocalPathSourceRequest
	localTreeResult     app.TreeLocalPathSourceResult
	localGrepReq        app.GrepLocalPathSourceRequest
	localGrepResult     app.GrepLocalPathSourceResult
	sourceRemoveReq     app.RemoveSourceRequest
	sourceRemoveResult  app.SourceStateChangeResult
	sourceRestoreReq    app.RestoreSourceRequest
	sourceRestoreResult app.SourceStateChangeResult

	events               []app.AppendEventRequest
	appendEventErrByType map[string]error
	evidenceRequests     []app.CreateEvidenceRecordRequest
	claimRequests        []app.CreateClaimRecordRequest
	confidenceRequests   []app.UpdateClaimConfidenceRequest
	questionRequests     []app.CreateQuestionRecordRequest
	proposalRequests     []app.CreateProposalBundleRequest
	metadataRequests     []app.UpdateMissionMetadataRequest
}

func (f *fakeMCPService) UpdateMissionMetadata(_ context.Context, req app.UpdateMissionMetadataRequest) (app.UpdateMissionMetadataResult, error) {
	f.metadataRequests = append(f.metadataRequests, req)
	projection := app.MissionProjection{MissionID: req.MissionID}
	if req.Title != nil {
		projection.Title = strings.TrimSpace(*req.Title)
	}
	event := app.LedgerEvent{EventID: req.EventID, MissionID: req.MissionID, EventType: "mission.metadata.updated", Producer: req.Producer}
	return app.UpdateMissionMetadataResult{Event: event, Projection: projection}, nil
}

func (f *fakeMCPService) GetProjection(_ context.Context, missionID string) (app.MissionProjection, error) {
	projection := f.projection
	if projection.MissionID == "" {
		projection.MissionID = missionID
	}
	return projection, nil
}

func (f *fakeMCPService) ListEvents(_ context.Context, missionID string) ([]app.LedgerEvent, error) {
	events := []app.LedgerEvent{}
	for _, event := range f.ledgerEvents {
		if event.MissionID == missionID {
			events = append(events, event)
		}
	}
	return events, nil
}

func (f *fakeMCPService) ListSourceSnapshots(_ context.Context, missionID string) ([]app.SourceSnapshot, error) {
	return f.ListSourceSnapshotsWithState(context.Background(), app.ListSourceSnapshotsRequest{MissionID: missionID})
}

func (f *fakeMCPService) ListSourceSnapshotsWithState(_ context.Context, req app.ListSourceSnapshotsRequest) ([]app.SourceSnapshot, error) {
	sources := []app.SourceSnapshot{}
	for _, source := range f.sources {
		if source.MissionID != req.MissionID {
			continue
		}
		if source.State.State == "" {
			source.State.State = app.SourceStateActive
		}
		source.State.Removed = source.State.Removed || source.State.State == app.SourceStateRemoved
		if source.State.Removed && !req.IncludeRemoved {
			continue
		}
		if source.State.Superseded && !req.IncludeSuperseded {
			continue
		}
		sources = append(sources, source)
	}
	return sources, nil
}

func (f *fakeMCPService) GetSourceSnapshot(_ context.Context, snapshotID string) (app.SourceSnapshot, error) {
	for _, source := range f.sources {
		if source.SnapshotID == snapshotID {
			return source, nil
		}
	}
	return app.SourceSnapshot{}, errors.New("missing source snapshot")
}

func (f *fakeMCPService) GetRawArtifact(_ context.Context, artifactID string) (app.RawArtifact, error) {
	if f.artifacts != nil {
		if artifact, ok := f.artifacts[artifactID]; ok {
			return artifact, nil
		}
	}
	return app.RawArtifact{}, errors.New("missing artifact")
}

func (f *fakeMCPService) ListRawArtifacts(_ context.Context, missionID string) ([]app.RawArtifact, error) {
	var artifacts []app.RawArtifact
	for _, artifact := range f.artifacts {
		if artifact.MissionID == missionID {
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts, nil
}

func (f *fakeMCPService) CreateRawArtifact(_ context.Context, req app.CreateRawArtifactRequest) (app.RawArtifact, error) {
	if f.artifacts == nil {
		f.artifacts = map[string]app.RawArtifact{}
	}
	if strings.TrimSpace(req.ExpectedSHA256) != "" {
		sum := sha256.Sum256(req.Content)
		if !strings.EqualFold(strings.TrimSpace(req.ExpectedSHA256), hex.EncodeToString(sum[:])) {
			return app.RawArtifact{}, app.ErrInvalidInput
		}
	}
	sum := sha256.Sum256(req.Content)
	artifact := app.RawArtifact{
		ArtifactID: req.ArtifactID,
		MissionID:  req.MissionID,
		MediaType:  req.MediaType,
		ByteSize:   int64(len(req.Content)),
		SHA256:     hex.EncodeToString(sum[:]),
		StorageURI: "memory://" + req.ArtifactID,
		Filename:   req.Filename,
		Producer:   req.Producer,
		CreatedAt:  time.Now().UTC(),
		Content:    append([]byte(nil), req.Content...),
	}
	f.artifacts[artifact.ArtifactID] = artifact
	return artifact, nil
}

func (f *fakeMCPService) CreateRawArtifactWithEvent(
	ctx context.Context,
	req app.CreateRawArtifactRequest,
	eventReqForArtifact func(app.RawArtifact) app.AppendEventRequest,
) (app.RawArtifact, app.LedgerEvent, error) {
	if eventReqForArtifact == nil {
		return app.RawArtifact{}, app.LedgerEvent{}, app.ErrInvalidInput
	}
	artifact, err := f.CreateRawArtifact(ctx, req)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, err
	}
	event, err := f.AppendEvent(ctx, eventReqForArtifact(artifact))
	if err != nil {
		delete(f.artifacts, artifact.ArtifactID)
		return app.RawArtifact{}, app.LedgerEvent{}, err
	}
	return artifact, event, nil
}

func (f *fakeMCPService) CreateRawArtifactWithEventConditionally(
	ctx context.Context,
	req app.CreateRawArtifactRequest,
	build func([]app.LedgerEvent, app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error),
) (app.RawArtifact, app.LedgerEvent, bool, error) {
	previous, hadPrevious := f.artifacts[req.ArtifactID]
	artifact, err := f.CreateRawArtifact(ctx, req)
	if err != nil {
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	eventReq, existing, create, err := build(f.ledgerEvents, artifact)
	if err != nil {
		delete(f.artifacts, artifact.ArtifactID)
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	if !create {
		if !hadPrevious {
			delete(f.artifacts, artifact.ArtifactID)
			return app.RawArtifact{}, app.LedgerEvent{}, false, errors.New("missing artifact")
		}
		f.artifacts[artifact.ArtifactID] = previous
		return previous, existing, false, nil
	}
	event, err := f.AppendEvent(ctx, eventReq)
	if err != nil {
		delete(f.artifacts, artifact.ArtifactID)
		return app.RawArtifact{}, app.LedgerEvent{}, false, err
	}
	return artifact, event, true, nil
}

func (f *fakeMCPService) ListLocalPathRoots(_ context.Context) ([]localpath.RootView, error) {
	return append([]localpath.RootView(nil), f.localRoots...), nil
}

func (f *fakeMCPService) BrowseLocalPathRoot(_ context.Context, req app.BrowseLocalPathRootRequest) (localpath.TreeResult, error) {
	tree := f.localTree
	if tree.RootID == "" {
		tree.RootID = req.RootID
	}
	if tree.RelativePath == "" {
		tree.RelativePath = req.RelativePath
	}
	return tree, nil
}

func (f *fakeMCPService) AttachLocalPathSource(_ context.Context, req app.AttachLocalPathSourceRequest) (app.LocalPathSourceResult, error) {
	f.localAttachReq = req
	result := f.localAttachResult
	if result.Snapshot.SnapshotID == "" {
		result.Snapshot = app.SourceSnapshot{
			SnapshotID: req.SnapshotID,
			MissionID:  req.MissionID,
			Connector: app.ConnectorRef{
				ConnectorID:      app.SourceConnectorTypeLocalPath,
				ConnectorType:    app.SourceConnectorTypeLocalPath,
				ExternalSourceID: req.RootID + ":" + req.RelativePath,
			},
			Title:  req.Title,
			Access: app.SourceAccess{RetrievalPolicy: app.SourceRetrievalPolicyLiveReference},
			State:  app.SourceState{State: app.SourceStateActive},
		}
	}
	if result.Event == nil {
		result.Event = &app.LedgerEvent{EventID: "evt_local_attach", MissionID: req.MissionID, EventType: app.SourceLocalPathAttachedEvent}
	}
	return result, nil
}

func (f *fakeMCPService) ReadLocalPathSource(_ context.Context, req app.ReadLocalPathSourceRequest) (app.ReadLocalPathSourceResult, error) {
	f.localReadReq = req
	result := f.localReadResult
	if result.Snapshot.SnapshotID == "" {
		for _, source := range f.sources {
			if source.SnapshotID == req.SnapshotID {
				result.Snapshot = source
				break
			}
		}
	}
	if result.Read.Metadata.RootID == "" {
		result.Read = localpath.ReadResult{
			Content: "live body",
			Metadata: localpath.PathMetadata{
				RootID:       "workspace",
				RelativePath: "notes.md",
				PathKind:     "file",
				Size:         9,
			},
		}
	}
	if result.ObservationEvent == nil {
		result.ObservationEvent = &app.LedgerEvent{EventID: "evt_observed", MissionID: req.MissionID, EventType: app.SourceObservedEvent}
	}
	return result, nil
}

func (f *fakeMCPService) TreeLocalPathSource(_ context.Context, req app.TreeLocalPathSourceRequest) (app.TreeLocalPathSourceResult, error) {
	f.localTreeReq = req
	result := f.localTreeResult
	if result.Snapshot.SnapshotID == "" {
		for _, source := range f.sources {
			if source.SnapshotID == req.SnapshotID {
				result.Snapshot = source
				break
			}
		}
	}
	if result.Tree.RootID == "" {
		result.Tree = localpath.TreeResult{
			RootID:       "workspace",
			RelativePath: "docs",
			Metadata:     localpath.PathMetadata{RootID: "workspace", RelativePath: "docs", Subpath: req.Subpath, PathKind: "directory"},
		}
	}
	if result.ObservationEvent == nil {
		result.ObservationEvent = &app.LedgerEvent{EventID: "evt_tree", MissionID: req.MissionID, EventType: app.SourceObservedEvent}
	}
	return result, nil
}

func (f *fakeMCPService) GrepLocalPathSource(_ context.Context, req app.GrepLocalPathSourceRequest) (app.GrepLocalPathSourceResult, error) {
	f.localGrepReq = req
	result := f.localGrepResult
	if result.Snapshot.SnapshotID == "" {
		for _, source := range f.sources {
			if source.SnapshotID == req.SnapshotID {
				result.Snapshot = source
				break
			}
		}
	}
	if result.Grep.RootID == "" {
		result.Grep = localpath.GrepResult{
			RootID:       "workspace",
			RelativePath: "docs",
			Query:        req.Query,
			Metadata:     localpath.PathMetadata{RootID: "workspace", RelativePath: "docs", Subpath: req.Subpath, PathKind: "directory"},
		}
	}
	if result.ObservationEvent == nil {
		result.ObservationEvent = &app.LedgerEvent{EventID: "evt_grep", MissionID: req.MissionID, EventType: app.SourceObservedEvent}
	}
	return result, nil
}

func (f *fakeMCPService) RemoveSource(_ context.Context, req app.RemoveSourceRequest) (app.SourceStateChangeResult, error) {
	f.sourceRemoveReq = req
	result := f.sourceRemoveResult
	if result.Snapshot.SnapshotID == "" {
		result.Snapshot = app.SourceSnapshot{SnapshotID: req.SnapshotID, MissionID: req.MissionID, State: app.SourceState{State: app.SourceStateRemoved, Removed: true}}
	}
	if result.Event == nil && !result.Idempotent {
		result.Event = &app.LedgerEvent{EventID: "evt_source_removed", MissionID: req.MissionID, EventType: app.SourceRemovedEvent}
	}
	return result, nil
}

func (f *fakeMCPService) RestoreSource(_ context.Context, req app.RestoreSourceRequest) (app.SourceStateChangeResult, error) {
	f.sourceRestoreReq = req
	result := f.sourceRestoreResult
	if result.Snapshot.SnapshotID == "" {
		result.Snapshot = app.SourceSnapshot{SnapshotID: req.SnapshotID, MissionID: req.MissionID, State: app.SourceState{State: app.SourceStateActive}}
	}
	if result.Event == nil && !result.Idempotent {
		result.Event = &app.LedgerEvent{EventID: "evt_source_restored", MissionID: req.MissionID, EventType: app.SourceRestoredEvent}
	}
	return result, nil
}

func (f *fakeMCPService) ListEvidenceRecords(_ context.Context, missionID string) ([]app.EvidenceRecord, error) {
	records := []app.EvidenceRecord{}
	for _, record := range f.evidence {
		if record.MissionID == missionID {
			records = append(records, record)
		}
	}
	return records, nil
}

func (f *fakeMCPService) ListClaimRecords(_ context.Context, missionID string) ([]app.ClaimRecord, error) {
	records := []app.ClaimRecord{}
	for _, record := range f.claims {
		if record.MissionID == missionID {
			records = append(records, record)
		}
	}
	return records, nil
}

func (f *fakeMCPService) ListQuestionRecords(_ context.Context, missionID string) ([]app.QuestionRecord, error) {
	records := []app.QuestionRecord{}
	for _, record := range f.questions {
		if record.MissionID == missionID {
			records = append(records, record)
		}
	}
	return records, nil
}

func (f *fakeMCPService) OutlineMission(_ context.Context, missionID string) (app.ResearchIDEOutline, error) {
	outline := f.outline
	if outline.MissionID == "" {
		outline.MissionID = missionID
	}
	return outline, nil
}

func (f *fakeMCPService) OutlineMissionLegacy(ctx context.Context, missionID string) (app.ResearchIDEOutline, error) {
	return f.OutlineMission(ctx, missionID)
}

func (f *fakeMCPService) ListMissionObjects(_ context.Context, missionID, objectKind string, limit int, cursor string) (app.ResearchIDEPage, error) {
	page := f.page
	if page.MissionID == "" {
		page.MissionID = missionID
	}
	if page.ObjectKind == "" {
		page.ObjectKind = objectKind
	}
	if page.Limit == 0 {
		page.Limit = limit
	}
	if page.NextCursor == "" {
		page.NextCursor = cursor
	}
	return page, nil
}

func (f *fakeMCPService) ListMissionObjectsLegacy(ctx context.Context, missionID, objectKind string, limit int, cursor string) (app.ResearchIDEPage, error) {
	return f.ListMissionObjects(ctx, missionID, objectKind, limit, cursor)
}

func (f *fakeMCPService) ReadMissionObject(_ context.Context, req app.ResearchIDEReadRequest) (app.ResearchIDEObjectRead, error) {
	f.lastRead = req
	read := f.read
	if read.MissionID == "" {
		read.MissionID = req.MissionID
	}
	if read.ObjectKind == "" {
		read.ObjectKind = req.ObjectKind
	}
	if read.ObjectID == "" {
		read.ObjectID = req.ObjectID
	}
	return read, nil
}

func (f *fakeMCPService) GrepMissionObjects(_ context.Context, missionID, query string, limit int, cursor string) (app.ResearchIDEGrepResult, error) {
	grep := f.grep
	if grep.MissionID == "" {
		grep.MissionID = missionID
	}
	if grep.Query == "" {
		grep.Query = query
	}
	if grep.Limit == 0 {
		grep.Limit = limit
	}
	if grep.NextCursor == "" {
		grep.NextCursor = cursor
	}
	return grep, nil
}

func (f *fakeMCPService) GrepMissionObjectsLegacy(ctx context.Context, missionID, query string, limit int, cursor string) (app.ResearchIDEGrepResult, error) {
	return f.GrepMissionObjects(ctx, missionID, query, limit, cursor)
}

func (f *fakeMCPService) ListObjectReferences(_ context.Context, missionID, objectKind, objectID string, limit int, cursor string) (app.ResearchIDEReferences, error) {
	refs := f.refs
	if refs.MissionID == "" {
		refs.MissionID = missionID
	}
	if refs.ObjectKind == "" {
		refs.ObjectKind = objectKind
	}
	if refs.ObjectID == "" {
		refs.ObjectID = objectID
	}
	if refs.Limit == 0 {
		refs.Limit = limit
	}
	if refs.NextCursor == "" {
		refs.NextCursor = cursor
	}
	return refs, nil
}

func (f *fakeMCPService) ListObjectReferencesLegacy(ctx context.Context, missionID, objectKind, objectID string, limit int, cursor string) (app.ResearchIDEReferences, error) {
	return f.ListObjectReferences(ctx, missionID, objectKind, objectID, limit, cursor)
}

func (f *fakeMCPService) RequestWorkflowRun(_ context.Context, req app.RequestWorkflowRunRequest) (app.WorkflowRunView, error) {
	runID := strings.TrimSpace(req.WorkflowRunID)
	if runID == "" {
		runID = "wfr_fake"
	}
	view := app.WorkflowRunView{
		WorkflowRunID:      runID,
		MissionID:          req.MissionID,
		Status:             app.WorkflowStatusQueued,
		RequestedBySurface: req.RequestedBySurface,
		AgentExecutor:      req.AgentExecutor,
		MCPMode:            req.MCPMode,
		Instruction:        req.Instruction,
		MaxSteps:           req.MaxSteps,
		MaxDurationMS:      req.MaxDurationMS,
		StartAfterEventID:  req.StartAfterEventID,
		StatusText:         "queued",
		LatestEventID:      "evt_workflow_requested",
	}
	f.workflowRuns = append(f.workflowRuns, view)
	return view, nil
}

func (f *fakeMCPService) GetWorkflowRun(_ context.Context, missionID string, workflowRunID string) (app.WorkflowRunView, error) {
	for _, run := range f.workflowRuns {
		if run.MissionID == missionID && run.WorkflowRunID == workflowRunID {
			return run, nil
		}
	}
	return app.WorkflowRunView{}, app.ErrInvalidInput
}

func (f *fakeMCPService) ListWorkflowRuns(_ context.Context, missionID string) ([]app.WorkflowRunView, error) {
	var runs []app.WorkflowRunView
	for _, run := range f.workflowRuns {
		if run.MissionID == missionID {
			runs = append(runs, run)
		}
	}
	return runs, nil
}

func (f *fakeMCPService) RequestWorkflowStop(_ context.Context, req app.RequestWorkflowStopRequest) (app.WorkflowRunView, error) {
	for i := range f.workflowRuns {
		if f.workflowRuns[i].MissionID == req.MissionID && f.workflowRuns[i].WorkflowRunID == req.WorkflowRunID {
			f.workflowRuns[i].Status = app.WorkflowStatusStopping
			f.workflowRuns[i].StopReason = req.Reason
			f.workflowRuns[i].LatestEventID = "evt_workflow_stop"
			return f.workflowRuns[i], nil
		}
	}
	return app.WorkflowRunView{}, app.ErrInvalidInput
}

func (f *fakeMCPService) SearchLiquid2Sources(
	_ context.Context,
	connector app.Liquid2SourceConnector,
	req app.Liquid2SourceSearchRequest,
) (app.Liquid2SourceSearchResult, error) {
	if connector == nil {
		return app.Liquid2SourceSearchResult{}, errors.New("connector missing")
	}
	f.searchUsedConnector = true
	f.searchRequest = req
	result := f.searchResult
	result.MissionID = req.MissionID
	return result, nil
}

func (f *fakeMCPService) SearchConfluenceSources(
	_ context.Context,
	connector app.ConfluenceSourceConnector,
	req app.ConfluenceSourceSearchRequest,
) (app.ConfluenceSourceSearchResult, error) {
	if connector == nil {
		return app.ConfluenceSourceSearchResult{}, errors.New("connector missing")
	}
	f.confluenceSearchRequest = req
	result := f.confluenceSearchResult
	result.MissionID = req.MissionID
	result.CloudID = req.CloudID
	return result, nil
}

func (f *fakeMCPService) GetMissionConnectorAccess(_ context.Context, missionID string, connectorID string) (app.ConnectorAccessProjection, error) {
	access := f.confluenceAccess
	if access.MissionID == "" {
		access.MissionID = missionID
	}
	if access.ConnectorID == "" {
		access.ConnectorID = connectorID
	}
	if access.Status == "" {
		access.Status = app.ConnectorAccessStatusDisabled
	}
	return access, nil
}

func (f *fakeMCPService) SnapshotLiquid2Source(
	_ context.Context,
	connector app.Liquid2SourceConnector,
	req app.SnapshotLiquid2SourceRequest,
) (app.Liquid2SnapshotResult, error) {
	if connector == nil {
		return app.Liquid2SnapshotResult{}, errors.New("connector missing")
	}
	f.snapshotRequest = req
	return f.snapshotResult, nil
}

func (f *fakeMCPService) SnapshotLiquid2SourceWithEvent(
	_ context.Context,
	connector app.Liquid2SourceConnector,
	req app.SnapshotLiquid2SourceWithEventRequest,
) (app.Liquid2SnapshotWithEventResult, error) {
	if connector == nil {
		return app.Liquid2SnapshotWithEventResult{}, errors.New("connector missing")
	}
	f.snapshotRequest = req.Snapshot
	eventReq := app.AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.Snapshot.MissionID,
		EventType: "source.snapshotted",
		Producer:  req.Producer,
		Payload: mustJSON(map[string]any{
			"snapshot_id":  f.snapshotResult.Snapshot.SnapshotID,
			"artifact_ids": f.snapshotResult.Snapshot.ArtifactIDs,
			"reason":       req.Snapshot.Reason,
		}),
	}
	f.events = append(f.events, eventReq)
	return app.Liquid2SnapshotWithEventResult{
		Artifact: f.snapshotResult.Artifact,
		Snapshot: f.snapshotResult.Snapshot,
		Event: app.LedgerEvent{
			EventID:   eventReq.EventID,
			MissionID: eventReq.MissionID,
			EventType: eventReq.EventType,
			Producer:  eventReq.Producer,
			Payload:   eventReq.Payload,
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

func (f *fakeMCPService) AppendEvent(_ context.Context, req app.AppendEventRequest) (app.LedgerEvent, error) {
	if f.appendEventErrByType != nil {
		if err := f.appendEventErrByType[req.EventType]; err != nil {
			return app.LedgerEvent{}, err
		}
	}
	f.events = append(f.events, req)
	event := app.LedgerEvent{
		EventID:          req.EventID,
		MissionID:        req.MissionID,
		Sequence:         int64(len(f.ledgerEvents) + 1),
		EventType:        req.EventType,
		Producer:         req.Producer,
		CausationEventID: req.CausationEventID,
		CorrelationID:    req.CorrelationID,
		Payload:          req.Payload,
		CreatedAt:        time.Now().UTC(),
	}
	f.ledgerEvents = append(f.ledgerEvents, event)
	return event, nil
}

func (f *fakeMCPService) CreateEvidenceProposal(
	ctx context.Context,
	req app.CreateEvidenceProposalRequest,
) (app.EvidenceProposalResult, error) {
	evidenceEvent, err := f.AppendEvent(ctx, req.EvidenceEvent)
	if err != nil {
		return app.EvidenceProposalResult{}, err
	}
	proposalEvent, err := f.AppendEvent(ctx, req.ProposalEvent)
	if err != nil {
		return app.EvidenceProposalResult{}, err
	}
	evidence, err := f.CreateEvidenceRecord(ctx, req.Evidence)
	if err != nil {
		return app.EvidenceProposalResult{}, err
	}
	proposal, err := f.CreateProposalBundle(ctx, req.Proposal)
	if err != nil {
		return app.EvidenceProposalResult{}, err
	}
	return app.EvidenceProposalResult{
		Evidence:      evidence,
		Proposal:      proposal,
		EvidenceEvent: evidenceEvent,
		ProposalEvent: proposalEvent,
	}, nil
}

func (f *fakeMCPService) CreateQuestionProposal(
	ctx context.Context,
	req app.CreateQuestionProposalRequest,
) (app.QuestionProposalResult, error) {
	questionEvent, err := f.AppendEvent(ctx, req.QuestionEvent)
	if err != nil {
		return app.QuestionProposalResult{}, err
	}
	proposalEvent, err := f.AppendEvent(ctx, req.ProposalEvent)
	if err != nil {
		return app.QuestionProposalResult{}, err
	}
	question, err := f.CreateQuestionRecord(ctx, req.Question)
	if err != nil {
		return app.QuestionProposalResult{}, err
	}
	proposal, err := f.CreateProposalBundle(ctx, req.Proposal)
	if err != nil {
		return app.QuestionProposalResult{}, err
	}
	return app.QuestionProposalResult{
		Question:      question,
		Proposal:      proposal,
		QuestionEvent: questionEvent,
		ProposalEvent: proposalEvent,
	}, nil
}

func (f *fakeMCPService) CreateClaimProposal(
	ctx context.Context,
	req app.CreateClaimProposalRequest,
) (app.ClaimProposalResult, error) {
	claimEvent, err := f.AppendEvent(ctx, req.ClaimEvent)
	if err != nil {
		return app.ClaimProposalResult{}, err
	}
	proposalEvent, err := f.AppendEvent(ctx, req.ProposalEvent)
	if err != nil {
		return app.ClaimProposalResult{}, err
	}
	claim, err := f.CreateClaimRecord(ctx, req.Claim)
	if err != nil {
		return app.ClaimProposalResult{}, err
	}
	proposal, err := f.CreateProposalBundle(ctx, req.Proposal)
	if err != nil {
		return app.ClaimProposalResult{}, err
	}
	return app.ClaimProposalResult{
		Claim:         claim,
		Proposal:      proposal,
		ClaimEvent:    claimEvent,
		ProposalEvent: proposalEvent,
	}, nil
}

func (f *fakeMCPService) UpdateClaimConfidence(
	ctx context.Context,
	req app.UpdateClaimConfidenceRequest,
) (app.LedgerEvent, error) {
	f.confidenceRequests = append(f.confidenceRequests, req)
	return f.AppendEvent(ctx, app.AppendEventRequest{
		EventID:   req.EventID,
		MissionID: req.MissionID,
		EventType: app.ClaimConfidenceUpdatedEvent,
		Producer:  req.Producer,
		Payload:   mustJSON(takeConfidencePayload(req)),
	})
}

func takeConfidencePayload(req app.UpdateClaimConfidenceRequest) map[string]any {
	return map[string]any{
		"claim_id":           req.ClaimID,
		"confidence":         req.Confidence,
		"basis_evidence_ids": req.BasisEvidenceIDs,
		"origin":             req.Origin,
	}
}

func (f *fakeMCPService) SubmitProposal(
	ctx context.Context,
	req app.SubmitProposalRequest,
) (app.SubmitProposalResult, error) {
	proposalEvent, err := f.AppendEvent(ctx, req.ProposalEvent)
	if err != nil {
		return app.SubmitProposalResult{}, err
	}
	proposal, err := f.CreateProposalBundle(ctx, req.Proposal)
	if err != nil {
		return app.SubmitProposalResult{}, err
	}
	return app.SubmitProposalResult{Proposal: proposal, ProposalEvent: proposalEvent}, nil
}

func (f *fakeMCPService) CreateEvidenceRecord(
	_ context.Context,
	req app.CreateEvidenceRecordRequest,
) (app.EvidenceRecord, error) {
	f.evidenceRequests = append(f.evidenceRequests, req)
	record := app.EvidenceRecord{
		ObjectKind:     app.EvidenceRecordObjectKind,
		EvidenceID:     req.EvidenceID,
		MissionID:      req.MissionID,
		State:          req.State,
		Summary:        req.Summary,
		EvidenceType:   req.EvidenceType,
		SnapshotRefs:   req.SnapshotRefs,
		Confidence:     req.Confidence,
		Producer:       req.Producer,
		CreatedEventID: req.CreatedEventID,
	}
	f.evidence = append(f.evidence, record)
	return record, nil
}

func (f *fakeMCPService) CreateQuestionRecord(
	_ context.Context,
	req app.CreateQuestionRecordRequest,
) (app.QuestionRecord, error) {
	f.questionRequests = append(f.questionRequests, req)
	record := app.QuestionRecord{
		ObjectKind:         app.QuestionRecordObjectKind,
		QuestionID:         req.QuestionID,
		MissionID:          req.MissionID,
		State:              req.State,
		Text:               req.Text,
		Priority:           req.Priority,
		Blocking:           req.Blocking,
		RelatedEvidenceIDs: req.RelatedEvidenceIDs,
		RelatedClaimIDs:    req.RelatedClaimIDs,
		CreatedEventID:     req.CreatedEventID,
	}
	f.questions = append(f.questions, record)
	return record, nil
}

func (f *fakeMCPService) CreateClaimRecord(
	_ context.Context,
	req app.CreateClaimRecordRequest,
) (app.ClaimRecord, error) {
	f.claimRequests = append(f.claimRequests, req)
	record := app.ClaimRecord{
		ObjectKind:            app.ClaimRecordObjectKind,
		ClaimID:               req.ClaimID,
		MissionID:             req.MissionID,
		State:                 req.State,
		Text:                  req.Text,
		ClaimType:             req.ClaimType,
		SupportingEvidenceIDs: req.SupportingEvidenceIDs,
		OpposingEvidenceIDs:   req.OpposingEvidenceIDs,
		DependsOnQuestionIDs:  req.DependsOnQuestionIDs,
		UserAssertionEventID:  req.UserAssertionEventID,
		Confidence:            req.Confidence,
		Approval:              req.Approval,
		CreatedEventID:        req.CreatedEventID,
	}
	f.claims = append(f.claims, record)
	return record, nil
}

func (f *fakeMCPService) CreateProposalBundle(
	_ context.Context,
	req app.CreateProposalBundleRequest,
) (app.ProposalBundle, error) {
	if req.RequestedDecision != "approve" {
		return app.ProposalBundle{}, errors.New("unsupported requested decision")
	}
	f.proposalRequests = append(f.proposalRequests, req)
	return app.ProposalBundle{
		ObjectKind:        app.ProposalBundleObjectKind,
		ProposalID:        req.ProposalID,
		MissionID:         req.MissionID,
		State:             req.State,
		Title:             req.Title,
		ObjectRefs:        req.ObjectRefs,
		RequestedDecision: req.RequestedDecision,
		CreatedEventID:    req.CreatedEventID,
	}, nil
}

type fakeMCPConnector struct{}

func (fakeMCPConnector) SearchLiquid2Sources(
	context.Context,
	app.Liquid2SourceSearchRequest,
) (app.Liquid2SourceSearchResult, error) {
	return app.Liquid2SourceSearchResult{}, nil
}

func (fakeMCPConnector) ReadLiquid2Source(
	context.Context,
	app.Liquid2SourceReadRequest,
) (app.Liquid2SourceDocument, error) {
	return app.Liquid2SourceDocument{}, nil
}

type fakeMCPConfluenceConnector struct{}

func (fakeMCPConfluenceConnector) SearchConfluenceSources(
	context.Context,
	app.ConfluenceSourceSearchRequest,
) (app.ConfluenceSourceSearchResult, error) {
	return app.ConfluenceSourceSearchResult{}, nil
}

func (fakeMCPConfluenceConnector) ReadConfluenceSource(
	context.Context,
	app.ConfluenceSourceReadRequest,
) (app.ConfluenceSourcePage, error) {
	return app.ConfluenceSourcePage{}, nil
}
