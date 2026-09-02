package reportilphase0

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidateFlowResponseSchemaAcceptsValidResponse(t *testing.T) {
	value := validFlowResponse()
	if err := validateFlowResponseSchema(value); err != nil {
		t.Fatalf("valid flow response rejected: %v", err)
	}
}

func TestValidateFlowResponseSchemaRejectsUnknownFields(t *testing.T) {
	tests := []struct {
		name string
		edit func(map[string]any)
	}{
		{name: "root", edit: func(value map[string]any) { value["unexpected"] = true }},
		{name: "document", edit: func(value map[string]any) { value["document"].(map[string]any)["unexpected"] = true }},
		{name: "block", edit: func(value map[string]any) {
			value["document"].(map[string]any)["blocks"].([]any)[0].(map[string]any)["unexpected"] = true
		}},
		{name: "table", edit: func(value map[string]any) {
			value["document"].(map[string]any)["blocks"].([]any)[1].(map[string]any)["table"].(map[string]any)["unexpected"] = true
		}},
		{name: "attestation", edit: func(value map[string]any) { value["flow_attestation"].(map[string]any)["unexpected"] = true }},
		{name: "finding", edit: func(value map[string]any) {
			value["flow_attestation"].(map[string]any)["accidental_repetition_findings"] = []any{map[string]any{"node_id": "n1", "detail": "finding", "unexpected": true}}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validFlowResponse()
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			var generic map[string]any
			if err := json.Unmarshal(raw, &generic); err != nil {
				t.Fatal(err)
			}
			test.edit(generic)
			mutated, err := json.Marshal(generic)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := strictDecode[flowResponse](mutated)
			if err == nil {
				err = validateFlowResponseSchema(decoded)
			}
			if err == nil {
				t.Fatal("unknown field accepted")
			}
		})
	}
}

func TestValidateFlowResponseSchemaRejectsNoncanonicalDocumentEnums(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Document)
	}{
		{name: "semantic role", edit: func(value *Document) { value.Blocks[0].SemanticRole = "provider-invented-role" }},
		{name: "presentation intent", edit: func(value *Document) { value.Blocks[0].PresentationIntent = "provider-invented-intent" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validFlowResponse()
			test.edit(&value.Document)
			if err := validateFlowResponseSchema(value); err == nil {
				t.Fatal("noncanonical Flow document enum accepted")
			}
		})
	}
}

func TestValidateFlowResponseSchemaRejectsExcludedDocumentSurfaces(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Document)
	}{
		{name: "unbound assets", edit: func(value *Document) {
			value.Assets = []Asset{{AssetID: "asset1", MediaType: "image/png", SHA256: strings.Repeat("a", 64), LicenseStatus: "allowed"}}
		}},
		{name: "figure payload on prose", edit: func(value *Document) {
			value.Blocks[0].Figure = &Figure{AssetID: "asset1", Caption: "caption", Alt: "alt"}
		}},
		{name: "raw", edit: func(value *Document) { value.Blocks[0].Kind = "raw" }},
		{name: "extensions", edit: func(value *Document) { value.Extensions = map[string]json.RawMessage{"x": json.RawMessage(`true`)} }},
		{name: "block extension", edit: func(value *Document) { value.Blocks[0].Extension = json.RawMessage(`true`) }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			value := validFlowResponse()
			test.edit(&value.Document)
			if err := validateProductDocumentShape(value.Document); err == nil {
				t.Fatal("excluded surface accepted")
			}
		})
	}
}

func validFlowResponse() flowResponse {
	return flowResponse{
		Document: Document{
			SchemaVersion: DocumentSchemaVersion, PipelineFamily: PipelineFamily,
			DocumentID: "doc1", RevisionID: "rev1", NarrativeContractID: "narr1",
			Title: "Title", Language: "en",
			Blocks: []Block{
				{NodeID: "n1", Kind: "prose", Prose: "Authored prose."},
				{NodeID: "n2", Kind: "table", Table: &Table{Columns: []string{"A"}, Rows: [][]string{{"B"}}}},
			},
			Provenance: map[string]string{"source": "test"},
		},
		Attestation: FlowAttestation{
			SchemaVersion: AttestationSchemaVersion, DocumentID: "doc1", RevisionID: "rev1",
			LinearProjectionSHA256: strings.Repeat("a", 64), Reviewer: "reviewer",
			ReviewedFullManuscript: true, CentralThreadPreserved: true,
			SectionHandoffsResolved: true, OpenLoopsResolved: true,
			TerminologyContinuous: true, Verdict: "accept",
		},
	}
}
