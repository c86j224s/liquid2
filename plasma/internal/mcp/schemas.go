package mcp

import "encoding/json"

var schemaReportPlanSubmit = json.RawMessage(`{
  "type":"object",
  "additionalProperties":false,
  "required":["mission_id","session_id","pending_event_id","report_mode","idempotency_key","producer","plan"],
  "properties":{
    "mission_id":{"type":"string"},
    "session_id":{"type":"string"},
    "pending_event_id":{"type":"string"},
    "report_mode":{"enum":["planned","long_form"]},
    "idempotency_key":{"type":"string"},
    "producer":{"$ref":"#/$defs/producer"},
    "plan":{"oneOf":[{"$ref":"#/$defs/planned_plan"},{"$ref":"#/$defs/long_form_plan"}]}
  },
  "$defs":{
    "producer":{"type":"object","additionalProperties":false,"required":["type","id"],"properties":{"type":{"const":"agent_session"},"id":{"type":"string"}}},
    "target_refs":{"type":"object","additionalProperties":false,"properties":{"claim_ids":{"type":"array","items":{"type":"string"}},"evidence_ids":{"type":"array","items":{"type":"string"}},"snapshot_ids":{"type":"array","items":{"type":"string"}},"question_ids":{"type":"array","items":{"type":"string"}},"option_ids":{"type":"array","items":{"type":"string"}}}},
    "writing_contract":{"type":"object","additionalProperties":false,"required":["central_question","reader_takeaway","reading_path","must_keep","visual_role","tone_and_shape"],"properties":{"central_question":{"type":"string"},"reader_takeaway":{"type":"string"},"reading_path":{"type":"array","items":{"type":"string"}},"must_keep":{"type":"array","items":{"type":"string"}},"can_summarize":{"type":"array","items":{"type":"string"}},"move_to_supporting_layer":{"type":"array","items":{"type":"string"}},"visual_role":{"type":"string"},"tone_and_shape":{"type":"string"}}},
    "planned_section":{"type":"object","additionalProperties":false,"properties":{"title":{"type":"string"},"purpose":{"type":"string"},"target_refs":{"$ref":"#/$defs/target_refs"}}},
    "long_form_section":{"type":"object","additionalProperties":false,"required":["title"],"properties":{"title":{"type":"string"},"purpose":{"type":"string"},"target_refs":{"$ref":"#/$defs/target_refs"}}},
    "part":{"type":"object","additionalProperties":false,"required":["title","sections"],"properties":{"title":{"type":"string"},"purpose":{"type":"string"},"sections":{"type":"array","items":{"$ref":"#/$defs/long_form_section"}}}},
    "planned_plan":{"type":"object","additionalProperties":false,"anyOf":[{"required":["summary"]},{"required":["sections"]}],"properties":{"summary":{"type":"string"},"sections":{"type":"array","items":{"$ref":"#/$defs/planned_section"}},"coverage_notes":{"type":"array","items":{"type":"string"}},"planned_omissions":{"type":"array","items":{"type":"string"}},"writing_contract":{"$ref":"#/$defs/writing_contract"}}},
    "long_form_plan":{"type":"object","additionalProperties":false,"required":["parts"],"properties":{"summary":{"type":"string"},"parts":{"type":"array","items":{"$ref":"#/$defs/part"}},"coverage_notes":{"type":"array","items":{"type":"string"}},"planned_omissions":{"type":"array","items":{"type":"string"}},"writing_contract":{"$ref":"#/$defs/writing_contract"}}}
  }
}`)

var schemaReportRequirementsSubmit = objectSchema(
	[]string{"mission_id", "session_id", "pending_event_id", "plan_event_id", "idempotency_key", "producer", "requirement_map"},
	mergeProperties(commonMutatingProperties(), map[string]any{
		"pending_event_id": prefixedStringSchema("evt_"),
		"plan_event_id":    prefixedStringSchema("evt_"),
		"requirement_map": objectSchemaValue([]string{"reviewed_event_ids", "requirements"}, map[string]any{
			"reviewed_event_ids": arraySchema(prefixedStringSchema("evt_")),
			"requirements": arraySchema(objectSchemaValue([]string{"requirement_id", "instruction", "source_event_ids"}, map[string]any{
				"requirement_id":   prefixedStringSchema("req_"),
				"instruction":      stringSchema(),
				"source_event_ids": arraySchema(prefixedStringSchema("evt_")),
				"owner": objectSchemaValue([]string{"part_index", "section_index"}, map[string]any{
					"part_index":    map[string]any{"type": "integer", "minimum": 1},
					"section_index": map[string]any{"type": "integer", "minimum": 1},
				}),
				"unmapped_reason": stringSchema(),
			})),
		}),
	}),
)

var schemaReportLongFormFinalize = json.RawMessage(`{
  "type":"object",
  "additionalProperties":false,
  "required":["mission_id","session_id","pending_event_id","plan_event_id","idempotency_key","producer","opening_markdown","closing_markdown"],
  "properties":{
    "mission_id":{"type":"string"},
    "session_id":{"type":"string"},
    "pending_event_id":{"type":"string"},
    "plan_event_id":{"type":"string"},
    "idempotency_key":{"type":"string"},
    "producer":{"type":"object","additionalProperties":false,"required":["type","id"],"properties":{"type":{"const":"agent_session"},"id":{"type":"string"}}},
    "opening_markdown":{"type":"string"},
    "closing_markdown":{"type":"string"}
  }
}`)

var (
	schemaReportLongFormEditStart = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "pending_event_id", "plan_event_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":         prefixedStringSchema("rfe_"),
			"pending_event_id": prefixedStringSchema("evt_"),
			"plan_event_id":    prefixedStringSchema("evt_"),
		}),
	)
	schemaReportLongFormEditRead = objectSchema(
		[]string{"mission_id", "session_id", "draft_id"},
		map[string]any{
			"mission_id": prefixedStringSchema("mis_"),
			"session_id": prefixedStringSchema("ses_"),
			"draft_id":   prefixedStringSchema("rfe_"),
			"offset":     map[string]any{"type": "integer", "minimum": 0},
			"max_bytes":  map[string]any{"type": "integer", "minimum": 1, "maximum": 65536},
		},
	)
	schemaReportLongFormEditPatch = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "operation", "replacement"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":    prefixedStringSchema("rfe_"),
			"operation":   enumSchema("replace", "insert_after", "append"),
			"match_text":  stringSchema(),
			"replacement": stringSchema(),
			"occurrence":  map[string]any{"type": "integer", "minimum": 0},
			"replace_all": map[string]any{"type": "boolean"},
			"summary":     stringSchema(),
		}),
	)
	schemaReportLongFormEditSubmit = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "pending_event_id", "plan_event_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":         prefixedStringSchema("rfe_"),
			"pending_event_id": prefixedStringSchema("evt_"),
			"plan_event_id":    prefixedStringSchema("evt_"),
		}),
	)
	schemaReportPartEditStart = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "pending_event_id", "plan_event_id", "part_index", "source_artifact_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":           prefixedStringSchema("rpe_"),
			"pending_event_id":   prefixedStringSchema("evt_"),
			"plan_event_id":      prefixedStringSchema("evt_"),
			"part_index":         map[string]any{"type": "integer", "minimum": 1},
			"source_artifact_id": prefixedStringSchema("art_"),
		}),
	)
	schemaReportPartEditRead = objectSchema(
		[]string{"mission_id", "session_id", "draft_id"},
		map[string]any{
			"mission_id": prefixedStringSchema("mis_"),
			"session_id": prefixedStringSchema("ses_"),
			"draft_id":   prefixedStringSchema("rpe_"),
			"offset":     map[string]any{"type": "integer", "minimum": 0},
			"max_bytes":  map[string]any{"type": "integer", "minimum": 1, "maximum": 65536},
		},
	)
	schemaReportPartEditPatch = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "operation", "replacement"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":    prefixedStringSchema("rpe_"),
			"operation":   enumSchema("replace", "insert_after", "append"),
			"match_text":  stringSchema(),
			"replacement": stringSchema(),
			"occurrence":  map[string]any{"type": "integer", "minimum": 0},
			"replace_all": map[string]any{"type": "boolean"},
			"summary":     stringSchema(),
		}),
	)
	schemaReportPartEditSubmit = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "pending_event_id", "plan_event_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":         prefixedStringSchema("rpe_"),
			"pending_event_id": prefixedStringSchema("evt_"),
			"plan_event_id":    prefixedStringSchema("evt_"),
		}),
	)
)

var (
	schemaMissionGet              = objectSchema([]string{"mission_id"}, baseProperties())
	schemaMissionUpdate           = missionUpdateSchema()
	schemaSourcesList             = objectSchema([]string{"mission_id"}, map[string]any{"mission_id": prefixedStringSchema("mis_"), "include_removed": map[string]any{"type": "boolean"}, "include_superseded": map[string]any{"type": "boolean"}})
	schemaSourcesRead             = objectSchema([]string{"mission_id", "snapshot_id"}, sourceReadProperties())
	schemaSourcesTree             = objectSchema([]string{"mission_id", "snapshot_id"}, sourceTreeProperties())
	schemaSourcesGrep             = objectSchema([]string{"mission_id", "snapshot_id", "query"}, sourceGrepProperties())
	schemaSourcesSearch           = objectSchema([]string{"mission_id", "query"}, sourceSearchProperties())
	schemaSourceCandidatesPropose = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "candidates"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"candidates": arraySchema(objectSchemaValue([]string{"url", "reason"}, map[string]any{
				"url":    stringSchema(),
				"title":  stringSchema(),
				"reason": stringSchema(),
			})),
		}),
	)
	schemaSourceCandidatesRead = objectSchema(
		[]string{"mission_id"},
		map[string]any{
			"mission_id":        prefixedStringSchema("mis_"),
			"url":               stringSchema(),
			"proposal_event_id": prefixedStringSchema("evt_"),
			"staging_event_id":  prefixedStringSchema("evt_"),
			"artifact_id":       prefixedStringSchema("art_"),
			"offset":            map[string]any{"type": "integer", "minimum": 0},
			"max_bytes":         map[string]any{"type": "integer", "minimum": 1, "maximum": 50000},
		},
	)
	schemaLocalPathRoots  = objectSchema([]string{}, map[string]any{"mission_id": prefixedStringSchema("mis_")})
	schemaLocalPathTree   = objectSchema([]string{"mission_id", "root_id"}, localPathTreeProperties())
	schemaLocalPathAttach = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "root_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"snapshot_id":   prefixedStringSchema("src_"),
			"root_id":       stringSchema(),
			"relative_path": stringSchema(),
			"title":         stringSchema(),
			"restore":       map[string]any{"type": "boolean"},
		}),
	)
	schemaSourcesRemove = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "snapshot_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"snapshot_id": prefixedStringSchema("src_"),
			"reason":      stringSchema(),
		}),
	)
	schemaSourcesRestore = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "snapshot_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"snapshot_id": prefixedStringSchema("src_"),
		}),
	)
	schemaMermaidValidate  = objectSchema([]string{"mission_id", "source"}, map[string]any{"mission_id": prefixedStringSchema("mis_"), "source": map[string]any{"type": "string", "maxLength": 50000}})
	schemaWorkflowStart    = objectSchema([]string{"mission_id", "instruction"}, workflowStartProperties())
	schemaWorkflowStatus   = objectSchema([]string{"mission_id"}, workflowStatusProperties())
	schemaWorkflowStop     = objectSchema([]string{"mission_id", "workflow_run_id"}, workflowStopProperties())
	schemaReportPatchStart = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "base_artifact_id", "instruction"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"patch_id":         prefixedStringSchema("rptp_"),
			"base_artifact_id": prefixedStringSchema("art_"),
			"title":            stringSchema(),
			"instruction":      stringSchema(),
		}),
	)
	schemaReportPatchRead = objectSchema(
		[]string{"mission_id", "session_id", "patch_id"},
		map[string]any{
			"mission_id": prefixedStringSchema("mis_"),
			"session_id": prefixedStringSchema("ses_"),
			"patch_id":   prefixedStringSchema("rptp_"),
			"offset":     map[string]any{"type": "integer", "minimum": 0},
			"max_bytes":  map[string]any{"type": "integer", "minimum": 1, "maximum": 65536},
		},
	)
	schemaReportPatchApply = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "patch_id", "operation", "replacement"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"patch_id":    prefixedStringSchema("rptp_"),
			"operation":   enumSchema("replace", "insert_after", "append"),
			"match_text":  stringSchema(),
			"replacement": stringSchema(),
			"occurrence":  map[string]any{"type": "integer", "minimum": 0},
			"replace_all": map[string]any{"type": "boolean"},
			"summary":     stringSchema(),
		}),
	)
	schemaReportPatchFinalize = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "patch_id", "pending_event_id", "agent_executor", "report_session_id", "report_session_policy", "report_session_policy_selection"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"patch_id":                        prefixedStringSchema("rptp_"),
			"artifact_id":                     prefixedStringSchema("art_"),
			"filename":                        stringSchema(),
			"title":                           stringSchema(),
			"patch_summary":                   stringSchema(),
			"expected_sha256":                 stringSchema(),
			"pending_event_id":                prefixedStringSchema("evt_"),
			"agent_executor":                  stringSchema(),
			"agent_model":                     stringSchema(),
			"agent_reasoning_effort":          stringSchema(),
			"mcp_mode":                        stringSchema(),
			"agent_session_id":                stringSchema(),
			"previous_agent_session_id":       stringSchema(),
			"returned_agent_session_id":       stringSchema(),
			"report_session_id":               stringSchema(),
			"fork_source_agent_session_id":    stringSchema(),
			"report_session_policy":           stringSchema(),
			"report_session_policy_selection": stringSchema(),
			"session_chain_kind":              stringSchema(),
		}),
	)
	schemaReportPartAssemblyStart = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "pending_event_id", "plan_event_id", "part_index", "section_count"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":         prefixedStringSchema("rpa_"),
			"pending_event_id": prefixedStringSchema("evt_"),
			"plan_event_id":    prefixedStringSchema("evt_"),
			"part_index":       map[string]any{"type": "integer", "minimum": 1},
			"section_count":    map[string]any{"type": "integer", "minimum": 1},
		}),
	)
	schemaReportPartAssemblyRead = objectSchema(
		[]string{"mission_id", "session_id", "draft_id"},
		map[string]any{
			"mission_id": prefixedStringSchema("mis_"),
			"session_id": prefixedStringSchema("ses_"),
			"draft_id":   prefixedStringSchema("rpa_"),
		},
	)
	schemaReportPartSectionRead = objectSchema(
		[]string{"mission_id", "session_id", "section_index"},
		map[string]any{
			"mission_id":    prefixedStringSchema("mis_"),
			"session_id":    prefixedStringSchema("ses_"),
			"section_index": map[string]any{"type": "integer", "minimum": 1},
			"offset":        map[string]any{"type": "integer", "minimum": 0},
			"max_bytes":     map[string]any{"type": "integer", "minimum": 1, "maximum": 65536},
		},
	)
	schemaReportPartAssemblyPatch = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "field", "markdown"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":            prefixedStringSchema("rpa_"),
			"field":               enumSchema("intro", "transition", "closing"),
			"after_section_index": map[string]any{"type": "integer", "minimum": 1},
			"markdown":            stringSchema(),
			"summary":             stringSchema(),
		}),
	)
	schemaReportPartAssemblySubmit = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "pending_event_id", "plan_event_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":         prefixedStringSchema("rpa_"),
			"pending_event_id": prefixedStringSchema("evt_"),
			"plan_event_id":    prefixedStringSchema("evt_"),
		}),
	)
	schemaExperimentReportCreate = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id": prefixedStringSchema("rpd_"),
			"title":    stringSchema(),
		}),
	)
	schemaExperimentReportAppend = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "content"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id": prefixedStringSchema("rpd_"),
			"content":  stringSchema(),
		}),
	)
	schemaExperimentReportRead = objectSchema(
		[]string{"mission_id", "session_id", "draft_id"},
		map[string]any{
			"mission_id": prefixedStringSchema("mis_"),
			"session_id": prefixedStringSchema("ses_"),
			"draft_id":   prefixedStringSchema("rpd_"),
			"offset":     map[string]any{"type": "integer", "minimum": 0},
			"max_bytes":  map[string]any{"type": "integer", "minimum": 1, "maximum": 65536},
		},
	)
	schemaExperimentReportFinalize = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":        prefixedStringSchema("rpd_"),
			"artifact_id":     prefixedStringSchema("art_"),
			"filename":        stringSchema(),
			"title":           stringSchema(),
			"expected_sha256": stringSchema(),
		}),
	)
	schemaSourcesSnapshot = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "connector", "artifact_id", "snapshot_id", "event_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"connector":   connectorSchema(),
			"artifact_id": prefixedStringSchema("art_"),
			"snapshot_id": prefixedStringSchema("src_"),
			"event_id":    prefixedStringSchema("evt_"),
			"ranges":      arraySchema(rangeSchema()),
			"reason":      stringSchema(),
		}),
	)
)

func missionUpdateSchema() json.RawMessage {
	properties := commonMutatingProperties()
	properties["producer"] = objectSchemaValue([]string{"type", "id"}, map[string]any{"type": map[string]any{"type": "string", "const": "user"}, "id": stringSchema()})
	properties["title"] = stringSchema()
	properties["objective"] = stringSchema()
	properties["scope"] = objectSchemaValue([]string{"included", "excluded"}, map[string]any{"included": arraySchema(stringSchema()), "excluded": arraySchema(stringSchema())})
	value := map[string]any{
		"type": "object", "additionalProperties": false,
		"required":   []string{"mission_id", "session_id", "idempotency_key", "producer"},
		"properties": properties,
		"anyOf":      []any{map[string]any{"required": []string{"title"}}, map[string]any{"required": []string{"objective"}}, map[string]any{"required": []string{"scope"}}},
	}
	encoded, _ := json.Marshal(value)
	return encoded
}

func objectSchema(required []string, properties map[string]any) json.RawMessage {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	encoded, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return encoded
}

func baseProperties() map[string]any {
	return map[string]any{
		"mission_id": prefixedStringSchema("mis_"),
		"include":    arraySchema(stringSchema()),
	}
}

func sourceSearchProperties() map[string]any {
	return map[string]any{
		"mission_id":    prefixedStringSchema("mis_"),
		"query":         stringSchema(),
		"connectors":    arraySchema(stringSchema()),
		"connection_id": prefixedStringSchema("cnf_"),
		"cloud_id":      stringSchema(),
		"space_key":     stringSchema(),
		"limit":         map[string]any{"type": "integer", "minimum": 1, "maximum": 100},
		"cursor":        stringSchema(),
	}
}

func sourceReadProperties() map[string]any {
	return map[string]any{
		"mission_id":  prefixedStringSchema("mis_"),
		"snapshot_id": prefixedStringSchema("src_"),
		"artifact_id": prefixedStringSchema("art_"),
		"subpath":     stringSchema(),
		"offset":      map[string]any{"type": "integer", "minimum": 0},
		"max_bytes":   map[string]any{"type": "integer", "minimum": 1, "maximum": 50000},
	}
}

func sourceTreeProperties() map[string]any {
	return map[string]any{
		"mission_id":  prefixedStringSchema("mis_"),
		"snapshot_id": prefixedStringSchema("src_"),
		"subpath":     stringSchema(),
		"depth":       map[string]any{"type": "integer", "minimum": 0, "maximum": 8},
		"limit":       map[string]any{"type": "integer", "minimum": 1, "maximum": 500},
	}
}

func sourceGrepProperties() map[string]any {
	return map[string]any{
		"mission_id":   prefixedStringSchema("mis_"),
		"snapshot_id":  prefixedStringSchema("src_"),
		"subpath":      stringSchema(),
		"query":        stringSchema(),
		"max_snippets": map[string]any{"type": "integer", "minimum": 1, "maximum": 100},
	}
}

func localPathTreeProperties() map[string]any {
	return map[string]any{
		"mission_id":    prefixedStringSchema("mis_"),
		"root_id":       stringSchema(),
		"relative_path": stringSchema(),
		"depth":         map[string]any{"type": "integer", "minimum": 0, "maximum": 8},
		"limit":         map[string]any{"type": "integer", "minimum": 1, "maximum": 500},
	}
}

func workflowStartProperties() map[string]any {
	return map[string]any{
		"mission_id":                   prefixedStringSchema("mis_"),
		"instruction":                  stringSchema(),
		"workflow_run_id":              prefixedStringSchema("wfr_"),
		"step_instruction_mode":        enumSchema("layered"),
		"user_instruction_raw":         stringSchema(),
		"run_goal":                     stringSchema(),
		"agent_executor":               stringSchema(),
		"mcp_mode":                     enumSchema("auto", "explicit"),
		"max_steps":                    map[string]any{"type": "integer", "minimum": 1, "maximum": 20},
		"max_duration_ms":              map[string]any{"type": "integer", "minimum": 0, "maximum": 86400000},
		"stop_condition":               stringSchema(),
		"start_after_event_id":         prefixedStringSchema("evt_"),
		"requested_by_tool_session_id": prefixedStringSchema("ses_"),
	}
}

func workflowStatusProperties() map[string]any {
	return map[string]any{
		"mission_id":      prefixedStringSchema("mis_"),
		"workflow_run_id": prefixedStringSchema("wfr_"),
	}
}

func workflowStopProperties() map[string]any {
	return map[string]any{
		"mission_id":      prefixedStringSchema("mis_"),
		"workflow_run_id": prefixedStringSchema("wfr_"),
		"reason":          stringSchema(),
	}
}

func commonMutatingProperties() map[string]any {
	return map[string]any{
		"mission_id":      prefixedStringSchema("mis_"),
		"session_id":      prefixedStringSchema("ses_"),
		"idempotency_key": stringSchema(),
		"producer": objectSchemaValue([]string{"type", "id"}, map[string]any{
			"type": map[string]any{"type": "string", "const": "agent_session"},
			"id":   prefixedStringSchema("ses_"),
		}),
	}
}

func mergeProperties(base map[string]any, extra map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(extra))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range extra {
		merged[key] = value
	}
	return merged
}

func objectSchemaValue(required []string, properties map[string]any) map[string]any {
	schema := map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"properties":           properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema() map[string]any {
	return map[string]any{"type": "string"}
}

func enumSchema(values ...string) map[string]any {
	enum := make([]any, 0, len(values))
	for _, value := range values {
		enum = append(enum, value)
	}
	return map[string]any{"type": "string", "enum": enum}
}

func prefixedStringSchema(prefix string) map[string]any {
	return map[string]any{"type": "string", "pattern": "^" + prefix}
}

func arraySchema(items any) map[string]any {
	return map[string]any{"type": "array", "items": items}
}

func connectorSchema() map[string]any {
	return objectSchemaValue([]string{"connector_id", "external_source_id"}, map[string]any{
		"connector_id":       map[string]any{"type": "string", "const": "liquid2"},
		"connector_type":     stringSchema(),
		"external_source_id": stringSchema(),
		"external_uri":       stringSchema(),
		"external_version":   stringSchema(),
		"connector_version":  stringSchema(),
	})
}

func rangeSchema() map[string]any {
	return objectSchemaValue([]string{"content_id", "start", "end"}, map[string]any{
		"content_id": stringSchema(),
		"start":      map[string]any{"type": "integer", "minimum": 0},
		"end":        map[string]any{"type": "integer", "minimum": 0},
	})
}
