package research

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/c86j224s/liquid2/plasma/internal/mcp/wire"
)

func TestReadErrorsPreservePreExtractionStrings(t *testing.T) {
	handler := NewHandler(readerOnlyService{}, "", true)
	tests := []struct {
		name      string
		call      wire.ToolCall
		wantError string
	}{
		{
			name:      "outline input type name",
			call:      wire.ToolCall{Name: "plasma.research.outline", Arguments: json.RawMessage(`{"mission_id":1}`)},
			wantError: "json: cannot unmarshal number into Go struct field researchOutlineInput.mission_id of type string",
		},
		{
			name:      "list input type name",
			call:      wire.ToolCall{Name: "plasma.research.list", Arguments: json.RawMessage(`{"mission_id":"mis_1","object_kind":1}`)},
			wantError: "json: cannot unmarshal number into Go struct field researchListInput.object_kind of type string",
		},
		{
			name:      "read input type name",
			call:      wire.ToolCall{Name: "plasma.research.read", Arguments: json.RawMessage(`{"mission_id":"mis_1","object_kind":1,"object_id":"evt_1"}`)},
			wantError: "json: cannot unmarshal number into Go struct field researchReadInput.object_kind of type string",
		},
		{
			name:      "grep input type name",
			call:      wire.ToolCall{Name: "plasma.research.grep", Arguments: json.RawMessage(`{"mission_id":"mis_1","query":1}`)},
			wantError: "json: cannot unmarshal number into Go struct field researchGrepInput.query of type string",
		},
		{
			name:      "references input type name",
			call:      wire.ToolCall{Name: "plasma.research.references", Arguments: json.RawMessage(`{"mission_id":"mis_1","object_kind":1,"object_id":"evt_1"}`)},
			wantError: "json: cannot unmarshal number into Go struct field researchReferencesInput.object_kind of type string",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := callReadTool(handler, test.call)
			if result.Error == nil || result.Error.Message != "invalid input: decode tool arguments: "+test.wantError {
				t.Fatalf("error = %#v, want %q", result.Error, "invalid input: decode tool arguments: "+test.wantError)
			}
		})
	}
}

func TestLegacyReadMissingLegacyReaderMessages(t *testing.T) {
	handler := NewHandler(readerOnlyService{}, "", true)
	tests := []struct {
		name string
		call wire.ToolCall
	}{
		{name: "outline", call: wire.ToolCall{Name: "plasma.research.outline", Arguments: json.RawMessage(`{"mission_id":"mis_1","legacy":true}`)}},
		{name: "list", call: wire.ToolCall{Name: "plasma.research.list", Arguments: json.RawMessage(`{"mission_id":"mis_1","object_kind":"evidence_record","legacy":true}`)}},
		{name: "grep", call: wire.ToolCall{Name: "plasma.research.grep", Arguments: json.RawMessage(`{"mission_id":"mis_1","query":"q","legacy":true}`)}},
		{name: "references", call: wire.ToolCall{Name: "plasma.research.references", Arguments: json.RawMessage(`{"mission_id":"mis_1","object_kind":"evidence_record","object_id":"evd_1","legacy":true}`)}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := callReadTool(handler, test.call)
			if result.Error == nil || result.Error.Message != "legacy research reader is not available" {
				t.Fatalf("error = %#v, want legacy reader message without invalid-input prefix", result.Error)
			}
		})
	}
}

func TestDirectMutationCallsFailClosedBeforeWrites(t *testing.T) {
	writer := &countingProposalWriter{}
	disabled := NewHandler(writer, "mis_1", false).CallEvidencePropose(context.Background(), evidenceCall())
	if disabled.Error == nil || disabled.Error.Message != "legacy research mutation tool is disabled in the default C1 loop" {
		t.Fatalf("disabled mutation error = %#v", disabled.Error)
	}
	if writer.count != 0 {
		t.Fatalf("disabled mutation wrote through writer")
	}

	boundMismatch := NewHandler(writer, "mis_other", true).CallEvidencePropose(context.Background(), evidenceCall())
	if boundMismatch.Error == nil || boundMismatch.Error.Message != "invalid input: tool call mission_id is outside this MCP session" {
		t.Fatalf("bound mismatch error = %#v", boundMismatch.Error)
	}
	if writer.count != 0 {
		t.Fatalf("bound mismatch wrote through writer")
	}

	missingWriter := NewHandler(readerOnlyService{}, "mis_1", true)
	for _, test := range []struct {
		name string
		call func(context.Context, wire.ToolCall) wire.ToolResult
		args wire.ToolCall
	}{
		{name: "evidence", call: missingWriter.CallEvidencePropose, args: evidenceCall()},
		{name: "questions", call: missingWriter.CallQuestionsPropose, args: questionsCall()},
		{name: "claims", call: missingWriter.CallClaimsPropose, args: claimsCall()},
		{name: "confidence", call: missingWriter.CallClaimConfidence, args: confidenceCall()},
		{name: "submit", call: missingWriter.CallProposalsSubmit, args: submitCall()},
	} {
		t.Run(test.name, func(t *testing.T) {
			result := test.call(context.Background(), test.args)
			if result.Error == nil || result.Error.Message != "research proposal writer is not available" {
				t.Fatalf("missing writer error = %#v", result.Error)
			}
		})
	}
}

func TestLegacyMutationCommonInputErrorsPreservePreExtractionStrings(t *testing.T) {
	handler := NewHandler(&countingProposalWriter{}, "mis_1", true)
	tests := []struct {
		name      string
		call      func(context.Context, wire.ToolCall) wire.ToolResult
		args      wire.ToolCall
		inputType string
	}{
		{name: "evidence", call: handler.CallEvidencePropose, args: evidenceCall(), inputType: "evidenceProposeInput"},
		{name: "questions", call: handler.CallQuestionsPropose, args: questionsCall(), inputType: "questionsProposeInput"},
		{name: "claims", call: handler.CallClaimsPropose, args: claimsCall(), inputType: "claimsProposeInput"},
		{name: "confidence", call: handler.CallClaimConfidence, args: confidenceCall(), inputType: "claimConfidenceInput"},
		{name: "submit", call: handler.CallProposalsSubmit, args: submitCall(), inputType: "proposalsSubmitInput"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := mutationCallWithCommonField(test.args, "mission_id", 1)
			result := test.call(context.Background(), args)
			want := "invalid input: decode tool arguments: json: cannot unmarshal number into Go struct field " + test.inputType + ".CommonMutatingInput.mission_id of type string"
			if result.Error == nil || result.Error.Message != want {
				t.Fatalf("error = %#v, want %q", result.Error, want)
			}
		})
	}
}
