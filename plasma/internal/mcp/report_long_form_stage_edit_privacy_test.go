package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/app"
	"github.com/c86j224s/liquid2/plasma/internal/reporting"
)

func TestReportLongFormGateRejectsMissingAndRawFindingsWithoutTraceLeak(t *testing.T) {
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeDisabled)
	reader := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageReader, reporting.FinalEditReaderSourceArtifactID(finalBinding.PlanEventID, finalBinding.PartArtifactIDs), "art_reader_edit", "ses_reader", "provider-reader", "provider-plan", "provider-plan")
	if result := runStageEdit(t, service, reader, []string{ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit}, ToolReportLongFormReaderEditStart, ToolReportLongFormReaderEditPatch, ToolReportLongFormReaderEditSubmit, "rfe_reader_privacy", "Alpha body.", "Alpha privacy reviewed.", nil); result.Error != nil {
		t.Fatalf("reader setup failed: %#v", result.Error)
	}

	finalBinding.ToolSessionID = "ses_gate"
	finalBinding.ProviderSessionID = "provider-gate"
	finalBinding.PreviousProviderSessionID = "provider-reader"
	finalBinding.Producer = app.Producer{Type: "agent_session", ID: "provider-gate"}
	gate := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageGate, "art_reader_edit", finalBinding.ArtifactID, "ses_gate", "provider-gate", "provider-reader", "provider-plan")
	server := NewServer(service,
		WithBinding(stageMCPBinding(gate)),
		WithFinalEditStageBinding(gate),
		WithLongFormFinalizeBinding(finalBinding),
		WithEnabledTools([]string{ToolReportLongFormEditStart, ToolReportLongFormEditSubmit}),
	)
	draftID := "rfe_gate_validation"
	if start := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditStart, Arguments: mustArgs(t, stageStartArgs(gate, draftID, "gate-validation-start"))}); start.Error != nil {
		t.Fatalf("gate validation start failed: %#v", start.Error)
	}
	submitArgs := stageCommonArgs(gate)
	submitArgs["idempotency_key"] = "gate-validation-missing"
	submitArgs["draft_id"] = draftID
	submitArgs["pending_event_id"] = gate.PendingEventID
	submitArgs["plan_event_id"] = gate.PlanEventID
	if missing := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditSubmit, Arguments: mustArgs(t, submitArgs)}); missing.Error == nil {
		t.Fatalf("gate submit accepted missing findings: %#v", missing)
	}

	rawStatement := "This raw passage must stay out of the trace."
	submitArgs["idempotency_key"] = "gate-validation-raw"
	submitArgs["gate_findings"] = []map[string]any{{
		"statement": rawStatement, "classification": reporting.FinalEditGateClassRhetoricalConstruction,
		"raw_passage": "forbidden raw passage",
	}}
	submitArgs["semantic_acceptance"] = []map[string]any{}
	if raw := server.Call(context.Background(), ToolCall{Name: ToolReportLongFormEditSubmit, Arguments: mustArgs(t, submitArgs)}); raw.Error == nil {
		t.Fatalf("gate submit accepted raw passage metadata: %#v", raw)
	}
	for _, event := range service.events {
		if strings.Contains(string(event.Payload), rawStatement) {
			t.Fatalf("raw gate statement leaked into MCP trace payload: %s", event.Payload)
		}
	}
}

func TestReadOnlyValidationStagesExposeOnlyReadAndSubmitContracts(t *testing.T) {
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeEnabled)
	for index := range service.ledgerEvents {
		if service.ledgerEvents[index].EventID != finalBinding.PlanEventID {
			continue
		}
		payload := finalEditStageTestPayload(t, service.ledgerEvents[index])
		payload["final_edit_pipeline"] = reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
		service.ledgerEvents[index].Payload = mustJSON(payload)
	}
	styleValidation := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageStyleSemanticValidation, "art_style", "art_style_validated", "ses_style_semantic_validation", "provider-style-semantic-validation", "provider-style", "provider-plan")
	styleValidation.FinalEditPipeline = reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	evidenceGate := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageEvidenceGate, "art_style_validated", finalBinding.ArtifactID, "ses_evidence_gate", "provider-evidence-gate", "provider-style-semantic-validation", "provider-plan")
	evidenceGate.FinalEditPipeline = reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3

	for name, tc := range map[string]struct {
		binding reporting.FinalEditStageBinding
		want    []string
		forbid  []string
	}{
		"style semantic validation": {
			binding: styleValidation,
			want:    []string{ToolReportLongFormStyleSemanticValidationRead, ToolReportLongFormStyleSemanticValidationSubmit},
			forbid:  []string{ToolReportLongFormStyleEditStart, ToolReportLongFormStyleEditPatch, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit},
		},
		"evidence gate": {
			binding: evidenceGate,
			want:    []string{ToolReportLongFormEvidenceGateRead, ToolReportLongFormEvidenceGateSubmit},
			forbid:  []string{ToolReportLongFormEditStart, ToolReportLongFormEditPatch, ToolReportLongFormEditSubmit, ToolReportLongFormStyleReviewRead},
		},
	} {
		t.Run(name, func(t *testing.T) {
			options := []Option{WithBinding(stageMCPBinding(tc.binding)), WithFinalEditStageBinding(tc.binding), WithEnabledTools(append(append([]string{}, tc.want...), tc.forbid...))}
			if tc.binding.Stage == reporting.FinalEditStageEvidenceGate {
				options = append(options, WithLongFormFinalizeBinding(gateFinalBindingForStage(tc.binding)))
			}
			server := NewServer(service, options...)
			if server.finalEditConfigErr != nil {
				t.Fatalf("read-only stage server config closed: %v", server.finalEditConfigErr)
			}
			tools := map[string]ToolDefinition{}
			for _, tool := range server.ListTools() {
				tools[tool.Name] = tool
			}
			for _, want := range tc.want {
				if _, ok := tools[want]; !ok {
					t.Fatalf("missing read-only tool %s in %#v", want, tools)
				}
			}
			for _, forbidden := range tc.forbid {
				if _, ok := tools[forbidden]; ok {
					t.Fatalf("forbidden mutating/legacy tool exposed: %s", forbidden)
				}
			}
			if tc.binding.Stage == reporting.FinalEditStageEvidenceGate {
				for _, expected := range []string{"runner-provided draft_id", "tool_session_id as session_id", "returned next_offset", "continuation content"} {
					if !strings.Contains(tools[ToolReportLongFormEvidenceGateRead].Description, expected) {
						t.Fatalf("evidence read description missing %q: %s", expected, tools[ToolReportLongFormEvidenceGateRead].Description)
					}
				}
				for _, expected := range []string{"exactly once", "same draft_id", "bound session_id"} {
					if !strings.Contains(tools[ToolReportLongFormEvidenceGateSubmit].Description, expected) {
						t.Fatalf("evidence submit description missing %q: %s", expected, tools[ToolReportLongFormEvidenceGateSubmit].Description)
					}
				}
			}
		})
	}

	for _, item := range []struct {
		name      string
		schema    json.RawMessage
		forbidden []string
		required  []string
	}{
		{
			name:      "style validation submit",
			schema:    schemaReportLongFormStyleSemanticValidationSubmit,
			forbidden: []string{"final_paragraph_ordinal", "repaired_by_gate", "replacement", "operation", "patch", "manuscript_markdown", "repair_action"},
			required:  []string{"accepted_equivalent", "rejected_revert_to_reader"},
		},
		{
			name:      "evidence gate submit",
			schema:    schemaReportLongFormEvidenceGateSubmit,
			forbidden: []string{"statement\"", "raw_passage", "candidate", "repair_action", "replacement", "operation", "semantic_acceptance", "manuscript_markdown"},
			required:  []string{"statement_sha256", "classification", "evidence_ids"},
		},
	} {
		t.Run(item.name, func(t *testing.T) {
			raw := string(item.schema)
			for _, forbidden := range item.forbidden {
				if strings.Contains(raw, forbidden) {
					t.Fatalf("schema contains forbidden field/literal %q: %s", forbidden, raw)
				}
			}
			for _, required := range item.required {
				if !strings.Contains(raw, required) {
					t.Fatalf("schema missing required literal %q: %s", required, raw)
				}
			}
		})
	}
}

func TestReadOnlyValidationStagesRequireCompleteContiguousReadBeforeSubmit(t *testing.T) {
	ctx := context.Background()
	service, finalBinding, styleValidation := seededV3ReadOnlyValidationMCPStage(t, ctx)
	styleServer := NewServer(service,
		WithBinding(stageMCPBinding(styleValidation)),
		WithFinalEditStageBinding(styleValidation),
		WithEnabledTools([]string{ToolReportLongFormStyleSemanticValidationRead, ToolReportLongFormStyleSemanticValidationSubmit}),
	)
	styleSubmit := stageCommonArgs(styleValidation)
	styleSubmit["idempotency_key"] = "style-validation-submit"
	styleSubmit["draft_id"] = "rfe_read_only_style"
	styleSubmit["pending_event_id"] = styleValidation.PendingEventID
	styleSubmit["plan_event_id"] = styleValidation.PlanEventID
	styleSubmit["semantic_acceptance"] = []map[string]any{{"paragraph_ordinal": 1, "verdict": reporting.FinalEditSemanticAcceptedEquivalent}}
	if result := styleServer.Call(ctx, ToolCall{Name: ToolReportLongFormStyleSemanticValidationSubmit, Arguments: mustArgs(t, styleSubmit)}); result.Error == nil {
		t.Fatalf("style semantic validation submit succeeded before read: %#v", result)
	}
	styleRead := map[string]any{"mission_id": styleValidation.MissionID, "session_id": styleValidation.ToolSessionID, "draft_id": "rfe_read_only_style", "offset": 0, "max_bytes": 32}
	first := styleServer.Call(ctx, ToolCall{Name: ToolReportLongFormStyleSemanticValidationRead, Arguments: mustArgs(t, styleRead)})
	if first.Error != nil {
		t.Fatalf("style read first page failed: %#v", first.Error)
	}
	firstPage := first.Content.(map[string]any)
	if firstPage["truncated"] != true {
		t.Fatalf("style read was not forced to paginate: %#v", firstPage)
	}
	if result := styleServer.Call(ctx, ToolCall{Name: ToolReportLongFormStyleSemanticValidationSubmit, Arguments: mustArgs(t, styleSubmit)}); result.Error == nil {
		t.Fatalf("style semantic validation submit succeeded after partial read: %#v", result)
	}
	if wrong := styleServer.Call(ctx, ToolCall{Name: ToolReportLongFormStyleSemanticValidationRead, Arguments: mustArgs(t, styleRead)}); wrong.Error == nil {
		t.Fatalf("style read accepted non-contiguous offset: %#v", wrong)
	}
	stylePacket := firstPage["content"].(string) + readOnlyValidationPacketRest(t, ctx, styleServer, ToolReportLongFormStyleSemanticValidationRead, styleRead, firstPage)
	var comparisons []map[string]any
	if err := json.Unmarshal([]byte(stylePacket), &comparisons); err != nil || len(comparisons) == 0 {
		t.Fatalf("style packet decode len=%d err=%v packet=%s", len(comparisons), err, stylePacket)
	}
	styleSubmit["semantic_acceptance"] = []map[string]any{{"paragraph_ordinal": int(comparisons[0]["paragraph_ordinal"].(float64)), "verdict": reporting.FinalEditSemanticAcceptedEquivalent}}
	styleResult := styleServer.Call(ctx, ToolCall{Name: ToolReportLongFormStyleSemanticValidationSubmit, Arguments: mustArgs(t, styleSubmit)})
	if styleResult.Error != nil {
		t.Fatalf("style semantic validation submit after full read failed: %#v", styleResult.Error)
	}
	styleContent := styleResult.Content.(map[string]any)

	evidenceSourceID := styleContent["artifact_id"].(string)
	finalBinding.ToolSessionID = "ses_evidence_gate"
	finalBinding.ProviderSessionID = "provider-evidence-gate"
	finalBinding.PreviousProviderSessionID = styleValidation.ProviderSessionID
	finalBinding.Producer = app.Producer{Type: "agent_session", ID: finalBinding.ProviderSessionID}
	evidenceGate := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageEvidenceGate, evidenceSourceID, finalBinding.ArtifactID, "ses_evidence_gate", "provider-evidence-gate", styleValidation.ProviderSessionID, finalBinding.ReportPlanSessionID)
	evidenceGate.FinalEditPipeline = reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	evidenceServer := NewServer(service,
		WithBinding(stageMCPBinding(evidenceGate)),
		WithFinalEditStageBinding(evidenceGate),
		WithLongFormFinalizeBinding(finalBinding),
		WithEnabledTools([]string{ToolReportLongFormEvidenceGateRead, ToolReportLongFormEvidenceGateSubmit}),
	)
	evidenceSubmit := stageCommonArgs(evidenceGate)
	evidenceSubmit["idempotency_key"] = "evidence-validation-submit"
	evidenceSubmit["draft_id"] = "rfe_read_only_evidence"
	evidenceSubmit["pending_event_id"] = evidenceGate.PendingEventID
	evidenceSubmit["plan_event_id"] = evidenceGate.PlanEventID
	evidenceSubmit["gate_findings"] = []map[string]any{}
	if result := evidenceServer.Call(ctx, ToolCall{Name: ToolReportLongFormEvidenceGateSubmit, Arguments: mustArgs(t, evidenceSubmit)}); result.Error == nil {
		t.Fatalf("evidence gate submit succeeded before read: %#v", result)
	} else {
		assertReadOnlyValidationContinuation(t, result, "rfe_read_only_evidence", evidenceGate.ToolSessionID, 0, "read")
	}
	evidenceRead := map[string]any{"mission_id": evidenceGate.MissionID, "session_id": evidenceGate.ToolSessionID, "draft_id": "rfe_read_only_evidence", "offset": 0, "max_bytes": 32}
	firstEvidence := evidenceServer.Call(ctx, ToolCall{Name: ToolReportLongFormEvidenceGateRead, Arguments: mustArgs(t, evidenceRead)})
	if firstEvidence.Error != nil {
		t.Fatalf("evidence read first page failed: %#v", firstEvidence.Error)
	}
	firstEvidencePage := firstEvidence.Content.(map[string]any)
	if firstEvidencePage["truncated"] != true {
		t.Fatalf("evidence read was not forced to paginate: %#v", firstEvidencePage)
	}
	nextEvidenceOffset := firstEvidencePage["next_offset"].(int)
	if result := evidenceServer.Call(ctx, ToolCall{Name: ToolReportLongFormEvidenceGateSubmit, Arguments: mustArgs(t, evidenceSubmit)}); result.Error == nil {
		t.Fatalf("evidence gate submit succeeded after partial read: %#v", result)
	} else {
		assertReadOnlyValidationContinuation(t, result, "rfe_read_only_evidence", evidenceGate.ToolSessionID, nextEvidenceOffset, "read")
	}
	wrongDraftRead := map[string]any{"mission_id": evidenceGate.MissionID, "session_id": evidenceGate.ToolSessionID, "draft_id": "rfe_other_evidence", "offset": 0, "max_bytes": 32}
	if result := evidenceServer.Call(ctx, ToolCall{Name: ToolReportLongFormEvidenceGateRead, Arguments: mustArgs(t, wrongDraftRead)}); result.Error == nil {
		t.Fatalf("evidence gate accepted a second draft: %#v", result)
	} else {
		assertReadOnlyValidationContinuation(t, result, "rfe_read_only_evidence", evidenceGate.ToolSessionID, nextEvidenceOffset, "read")
	}
	if len(evidenceServer.readOnlyValidationDrafts) != 1 {
		t.Fatalf("evidence gate created multiple read-only drafts: %#v", evidenceServer.readOnlyValidationDrafts)
	}
	wrongSessionRead := map[string]any{"mission_id": evidenceGate.MissionID, "session_id": "ses_other_evidence", "draft_id": "rfe_read_only_evidence", "offset": nextEvidenceOffset, "max_bytes": 32}
	if result := evidenceServer.Call(ctx, ToolCall{Name: ToolReportLongFormEvidenceGateRead, Arguments: mustArgs(t, wrongSessionRead)}); result.Error == nil {
		t.Fatalf("evidence gate accepted a different MCP session: %#v", result)
	} else {
		assertReadOnlyValidationContinuation(t, result, "rfe_read_only_evidence", evidenceGate.ToolSessionID, nextEvidenceOffset, "read")
	}
	wrongOffsetRead := map[string]any{"mission_id": evidenceGate.MissionID, "session_id": evidenceGate.ToolSessionID, "draft_id": "rfe_read_only_evidence", "offset": 0, "max_bytes": 32}
	if result := evidenceServer.Call(ctx, ToolCall{Name: ToolReportLongFormEvidenceGateRead, Arguments: mustArgs(t, wrongOffsetRead)}); result.Error == nil {
		t.Fatalf("evidence gate accepted a non-contiguous offset: %#v", result)
	} else {
		assertReadOnlyValidationContinuation(t, result, "rfe_read_only_evidence", evidenceGate.ToolSessionID, nextEvidenceOffset, "read")
	}
	evidencePacket := firstEvidencePage["content"].(string) + readOnlyValidationPacketRest(t, ctx, evidenceServer, ToolReportLongFormEvidenceGateRead, evidenceRead, firstEvidencePage)
	var packet struct {
		Passages []struct {
			StatementSHA256 string `json:"statement_sha256"`
		} `json:"passages"`
	}
	if err := json.Unmarshal([]byte(evidencePacket), &packet); err != nil || len(packet.Passages) == 0 {
		t.Fatalf("evidence packet decode len=%d err=%v packet=%s", len(packet.Passages), err, evidencePacket)
	}
	wrongDraftSubmit := map[string]any{}
	for key, value := range evidenceSubmit {
		wrongDraftSubmit[key] = value
	}
	wrongDraftSubmit["draft_id"] = "rfe_other_evidence"
	if result := evidenceServer.Call(ctx, ToolCall{Name: ToolReportLongFormEvidenceGateSubmit, Arguments: mustArgs(t, wrongDraftSubmit)}); result.Error == nil {
		t.Fatalf("evidence gate accepted submit from a different draft: %#v", result)
	} else {
		assertReadOnlyValidationContinuation(t, result, "rfe_read_only_evidence", evidenceGate.ToolSessionID, 0, "submit_once")
	}
	wrongHashSubmit := map[string]any{}
	for key, value := range evidenceSubmit {
		wrongHashSubmit[key] = value
	}
	wrongHashSubmit["gate_findings"] = []map[string]any{{"statement_sha256": strings.Repeat("f", 64), "classification": reporting.FinalEditGateClassDerivedSynthesis}}
	if result := evidenceServer.Call(ctx, ToolCall{Name: ToolReportLongFormEvidenceGateSubmit, Arguments: mustArgs(t, wrongHashSubmit)}); result.Error == nil {
		t.Fatalf("evidence gate accepted an unknown statement hash: %#v", result)
	} else {
		assertReadOnlyValidationContinuation(t, result, "rfe_read_only_evidence", evidenceGate.ToolSessionID, 0, "submit_once")
	}
	evidenceSubmit["gate_findings"] = []map[string]any{{"statement_sha256": packet.Passages[0].StatementSHA256, "classification": reporting.FinalEditGateClassDerivedSynthesis}}
	evidenceResult := evidenceServer.Call(ctx, ToolCall{Name: ToolReportLongFormEvidenceGateSubmit, Arguments: mustArgs(t, evidenceSubmit)})
	if evidenceResult.Error != nil {
		t.Fatalf("evidence gate submit after full read failed: %#v", evidenceResult.Error)
	}
}

func assertReadOnlyValidationContinuation(t *testing.T, result ToolResult, draftID, sessionID string, nextOffset int, nextAction string) {
	t.Helper()
	content, ok := result.Content.(map[string]any)
	if !ok || content["draft_id"] != draftID || content["session_id"] != sessionID || content["next_offset"] != nextOffset || content["next_action"] != nextAction {
		t.Fatalf("read-only validation continuation mismatch: %#v", result)
	}
}

func TestReadOnlyValidationSubmitRejectsForbiddenKeyPresence(t *testing.T) {
	ctx := context.Background()
	service, finalBinding, styleValidation := seededV3ReadOnlyValidationMCPStage(t, ctx)
	finalBinding.ToolSessionID = "ses_evidence_forbidden"
	finalBinding.ProviderSessionID = "provider-evidence-forbidden"
	finalBinding.PreviousProviderSessionID = styleValidation.ProviderSessionID
	finalBinding.Producer = app.Producer{Type: "agent_session", ID: finalBinding.ProviderSessionID}
	evidenceGate := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageEvidenceGate, styleValidation.SourceArtifactID, finalBinding.ArtifactID, finalBinding.ToolSessionID, finalBinding.ProviderSessionID, styleValidation.ProviderSessionID, finalBinding.ReportPlanSessionID)
	evidenceGate.FinalEditPipeline = reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	base := func(binding reporting.FinalEditStageBinding) map[string]any {
		return map[string]any{
			"mission_id":       binding.MissionID,
			"session_id":       binding.ToolSessionID,
			"idempotency_key":  "submit-key",
			"producer":         map[string]any{"type": "agent_session", "id": binding.ToolSessionID},
			"draft_id":         "rfe_forbidden_keys",
			"pending_event_id": binding.PendingEventID,
			"plan_event_id":    binding.PlanEventID,
			"gate_findings":    []map[string]any{},
			"semantic_acceptance": []map[string]any{
				{"paragraph_ordinal": 1, "verdict": reporting.FinalEditSemanticAcceptedEquivalent},
			},
		}
	}
	for _, tc := range []struct {
		name    string
		tool    string
		binding reporting.FinalEditStageBinding
		args    map[string]any
	}{
		{
			name:    "style_final_paragraph_ordinal_zero",
			tool:    ToolReportLongFormStyleSemanticValidationSubmit,
			binding: styleValidation,
			args: func() map[string]any {
				args := base(styleValidation)
				delete(args, "gate_findings")
				args["semantic_acceptance"] = []map[string]any{{"paragraph_ordinal": 1, "final_paragraph_ordinal": 0, "verdict": reporting.FinalEditSemanticAcceptedEquivalent}}
				return args
			}(),
		},
		{
			name:    "style_gate_findings_empty_array",
			tool:    ToolReportLongFormStyleSemanticValidationSubmit,
			binding: styleValidation,
			args: func() map[string]any {
				args := base(styleValidation)
				args["gate_findings"] = []map[string]any{}
				return args
			}(),
		},
		{
			name:    "evidence_statement_empty_string",
			tool:    ToolReportLongFormEvidenceGateSubmit,
			binding: evidenceGate,
			args: func() map[string]any {
				args := base(evidenceGate)
				delete(args, "semantic_acceptance")
				args["gate_findings"] = []map[string]any{{"statement_sha256": strings.Repeat("a", 64), "classification": reporting.FinalEditGateClassDerivedSynthesis, "statement": ""}}
				return args
			}(),
		},
		{
			name:    "evidence_statement_null",
			tool:    ToolReportLongFormEvidenceGateSubmit,
			binding: evidenceGate,
			args: func() map[string]any {
				args := base(evidenceGate)
				delete(args, "semantic_acceptance")
				args["gate_findings"] = []map[string]any{{"statement_sha256": strings.Repeat("a", 64), "classification": reporting.FinalEditGateClassDerivedSynthesis, "statement": nil}}
				return args
			}(),
		},
		{
			name:    "evidence_repair_action_empty_string",
			tool:    ToolReportLongFormEvidenceGateSubmit,
			binding: evidenceGate,
			args: func() map[string]any {
				args := base(evidenceGate)
				delete(args, "semantic_acceptance")
				args["gate_findings"] = []map[string]any{{"statement_sha256": strings.Repeat("a", 64), "classification": reporting.FinalEditGateClassDerivedSynthesis, "repair_action": ""}}
				return args
			}(),
		},
		{
			name:    "evidence_repair_action_null",
			tool:    ToolReportLongFormEvidenceGateSubmit,
			binding: evidenceGate,
			args: func() map[string]any {
				args := base(evidenceGate)
				delete(args, "semantic_acceptance")
				args["gate_findings"] = []map[string]any{{"statement_sha256": strings.Repeat("a", 64), "classification": reporting.FinalEditGateClassDerivedSynthesis, "repair_action": nil}}
				return args
			}(),
		},
		{
			name:    "evidence_semantic_acceptance_empty_array",
			tool:    ToolReportLongFormEvidenceGateSubmit,
			binding: evidenceGate,
			args: func() map[string]any {
				args := base(evidenceGate)
				args["semantic_acceptance"] = []map[string]any{}
				return args
			}(),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			options := []Option{WithBinding(stageMCPBinding(tc.binding)), WithFinalEditStageBinding(tc.binding), WithEnabledTools([]string{tc.tool})}
			if tc.binding.Stage == reporting.FinalEditStageEvidenceGate {
				options = append(options, WithLongFormFinalizeBinding(finalBinding))
			}
			server := NewServer(service, options...)
			result := server.Call(context.Background(), ToolCall{Name: tc.tool, Arguments: mustArgs(t, tc.args)})
			if result.Error == nil || result.Error.ErrorKind != "validation" || !strings.Contains(result.Error.Message, "not allowed") {
				if result.Error != nil {
					t.Fatalf("forbidden key was not rejected: kind=%s message=%s", result.Error.ErrorKind, result.Error.Message)
				}
				t.Fatalf("forbidden key was not rejected: %#v", result)
			}
		})
	}
}

func readOnlyValidationPacketRest(t *testing.T, ctx context.Context, server *Server, tool string, args map[string]any, page map[string]any) string {
	t.Helper()
	var out strings.Builder
	for page["truncated"] == true {
		args["offset"] = int(page["next_offset"].(int))
		result := server.Call(ctx, ToolCall{Name: tool, Arguments: mustArgs(t, args)})
		if result.Error != nil {
			t.Fatalf("%s next page failed: %#v", tool, result.Error)
		}
		page = result.Content.(map[string]any)
		out.WriteString(page["content"].(string))
	}
	return out.String()
}

func seededV3ReadOnlyValidationMCPStage(t *testing.T, ctx context.Context) (*fakeMCPService, reporting.LongFormFinalizeBinding, reporting.FinalEditStageBinding) {
	t.Helper()
	service, finalBinding := seededFinalEditStageService(t, reporting.FinalEditHumanizeEnabled)
	for index := range service.ledgerEvents {
		if service.ledgerEvents[index].EventID != finalBinding.PlanEventID {
			continue
		}
		payload := finalEditStageTestPayload(t, service.ledgerEvents[index])
		payload["final_edit_pipeline"] = reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
		service.ledgerEvents[index].Payload = mustJSON(payload)
	}
	assemblyID := reporting.FinalEditAssemblyArtifactID(finalBinding.PlanEventID, finalBinding.PartArtifactIDs)
	writer := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageWriter, assemblyID, "art_writer_read_only", "ses_writer_read_only", "provider-writer-read-only", finalBinding.ReportPlanSessionID, finalBinding.ReportPlanSessionID)
	writer.FinalEditPipeline = reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, _, err := reporting.EnsureFinalEditAssembly(ctx, service, "evt_read_only_assembly", writer); err != nil {
		t.Fatal(err)
	}
	assembly, err := service.GetRawArtifact(ctx, assemblyID)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := reporting.StartFinalEditStage(ctx, service, "evt_writer_read_only_start", writer); err != nil {
		t.Fatal(err)
	}
	writerResult, err := reporting.SubmitFinalEditStage(ctx, service, writer, "evt_writer_read_only_submit", string(assembly.Content), 0)
	if err != nil {
		t.Fatal(err)
	}
	reader := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageReader, writerResult.Artifact.ArtifactID, "art_reader_read_only", "ses_reader_read_only", "provider-reader-read-only", finalBinding.ReportPlanSessionID, finalBinding.ReportPlanSessionID)
	reader.FinalEditPipeline = reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, _, err := reporting.StartFinalEditStage(ctx, service, "evt_reader_read_only_start", reader); err != nil {
		t.Fatal(err)
	}
	readerResult, err := reporting.SubmitFinalEditStage(ctx, service, reader, "evt_reader_read_only_submit", string(writerResult.Artifact.Content), 0)
	if err != nil {
		t.Fatal(err)
	}
	style := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageStyle, readerResult.Artifact.ArtifactID, "art_style_read_only", "ses_style_read_only", "provider-style-read-only", reader.ProviderSessionID, reader.ProviderSessionID)
	style.FinalEditPipeline = reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	if _, _, err := reporting.StartFinalEditStage(ctx, service, "evt_style_read_only_start", style); err != nil {
		t.Fatal(err)
	}
	styleMarkdown := strings.Replace(string(readerResult.Artifact.Content), "Alpha body.", "Alpha body!", 1)
	styleResult, err := reporting.SubmitFinalEditStyleStage(ctx, service, style, "evt_style_read_only_submit", styleMarkdown, 1, []reporting.FinalEditStyleOperationDiagnosis{{
		OperationOrdinal: 1, Category: "unnatural_collocation", Reason: "awkward local phrasing",
		MatchText: "Alpha body.", Replacement: "Alpha body!", Occurrence: 1,
	}})
	if err != nil {
		t.Fatal(err)
	}
	styleValidation := testFinalEditStageBinding(finalBinding, reporting.FinalEditStageStyleSemanticValidation, styleResult.Artifact.ArtifactID, "art_style_validation_read_only", "ses_style_validation_read_only", "provider-style-validation-read-only", style.ProviderSessionID, finalBinding.ReportPlanSessionID)
	styleValidation.FinalEditPipeline = reporting.FinalEditPipelineAssemblyWriterReaderStyleValidationEvidenceGateV3
	return service, finalBinding, styleValidation
}
