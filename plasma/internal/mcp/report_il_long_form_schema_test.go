package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestReportILLongFormPlanSubmitSchemaOwnsLanguageAndSectionRoles(t *testing.T) {
	var schema struct {
		AdditionalProperties bool `json:"additionalProperties"`
		Required             []string
		Properties           map[string]json.RawMessage
	}
	if err := json.Unmarshal(schemaReportILLongFormPlanSubmit, &schema); err != nil {
		t.Fatal(err)
	}
	if schema.AdditionalProperties || !reflect.DeepEqual(schema.Required, []string{"title", "summary", "parts"}) {
		t.Fatalf("long-form plan schema envelope = %#v", schema)
	}
	if _, ok := schema.Properties["language"]; ok {
		t.Fatal("provider-controlled language remains in the long-form plan schema")
	}

	var parts struct {
		Items struct {
			Properties map[string]json.RawMessage
		}
	}
	if err := json.Unmarshal(schema.Properties["parts"], &parts); err != nil {
		t.Fatal(err)
	}
	var sections struct {
		Items struct {
			AdditionalProperties bool `json:"additionalProperties"`
			Required             []string
			Properties           map[string]json.RawMessage
		}
	}
	if err := json.Unmarshal(parts.Items.Properties["sections"], &sections); err != nil {
		t.Fatal(err)
	}
	if sections.Items.AdditionalProperties || !reflect.DeepEqual(sections.Items.Required, []string{"title", "purpose", "representations", "evidence_source_keys"}) {
		t.Fatalf("long-form Section schema = %#v", sections.Items)
	}
	if _, ok := sections.Items.Properties["role"]; ok {
		t.Fatal("provider-controlled Section role remains in the long-form plan schema")
	}
	var representations struct {
		Type        string `json:"type"`
		UniqueItems bool   `json:"uniqueItems"`
		Items       struct {
			Enum []string `json:"enum"`
		} `json:"items"`
	}
	if err := json.Unmarshal(sections.Items.Properties["representations"], &representations); err != nil {
		t.Fatal(err)
	}
	if representations.Type != "array" || !representations.UniqueItems || !reflect.DeepEqual(representations.Items.Enum, []string{
		"table", "code", "equation", "worked_example", "benchmark", "checklist", "diagram",
	}) {
		t.Fatalf("long-form representations schema = %#v", representations)
	}
}
