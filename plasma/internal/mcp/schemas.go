package mcp

import "encoding/json"

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
	schemaResearchOutline    = objectSchema([]string{"mission_id"}, map[string]any{"mission_id": prefixedStringSchema("mis_")})
	schemaResearchList       = researchListSchema(false)
	schemaResearchListLegacy = researchListSchema(true)
	schemaResearchRead       = researchReadSchema(false)
	schemaResearchReadLegacy = researchReadSchema(true)
	schemaResearchGrep       = objectSchema([]string{"mission_id", "query"}, researchGrepProperties())
	schemaResearchRefs       = researchRefsSchema(false)
	schemaResearchRefsLegacy = researchRefsSchema(true)
	schemaWorkflowStart      = objectSchema([]string{"mission_id", "instruction"}, workflowStartProperties())
	schemaWorkflowStatus     = objectSchema([]string{"mission_id"}, workflowStatusProperties())
	schemaWorkflowStop       = objectSchema([]string{"mission_id", "workflow_run_id"}, workflowStopProperties())
	schemaReportPatchStart   = objectSchema(
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
	schemaEvidencePropose = objectSchema(
		[]string{
			"mission_id",
			"session_id",
			"idempotency_key",
			"evidence_id",
			"event_id",
			"proposal_id",
			"proposal_event_id",
			"summary",
			"evidence_type",
			"snapshot_refs",
			"producer",
		},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"evidence_id":       prefixedStringSchema("evd_"),
			"event_id":          prefixedStringSchema("evt_"),
			"proposal_id":       prefixedStringSchema("prp_"),
			"proposal_event_id": prefixedStringSchema("evt_"),
			"proposal_title":    stringSchema(),
			"summary":           stringSchema(),
			"evidence_type": enumSchema(
				"quote",
				"fact",
				"table_row",
				"statistic",
				"observation",
				"interpretation",
				"reaction",
				"rumor",
				"controversy",
				"market_signal",
				"code",
				"formula",
				"benchmark",
				"open_question",
			),
			"snapshot_refs": arraySchema(snapshotRefSchema()),
			"confidence":    confidenceSchema(),
		}),
	)
	schemaQuestionsPropose = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "question_id", "event_id", "proposal_id", "proposal_event_id", "text"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"question_id":          prefixedStringSchema("qst_"),
			"event_id":             prefixedStringSchema("evt_"),
			"proposal_id":          prefixedStringSchema("prp_"),
			"proposal_event_id":    prefixedStringSchema("evt_"),
			"proposal_title":       stringSchema(),
			"text":                 stringSchema(),
			"priority":             enumSchema("low", "medium", "high"),
			"blocking":             map[string]any{"type": "boolean"},
			"related_evidence_ids": arraySchema(prefixedStringSchema("evd_")),
			"related_claim_ids":    arraySchema(prefixedStringSchema("clm_")),
		}),
	)
	schemaClaimsPropose = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "claim_id", "event_id", "proposal_id", "proposal_event_id", "text"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"claim_id":                prefixedStringSchema("clm_"),
			"event_id":                prefixedStringSchema("evt_"),
			"proposal_id":             prefixedStringSchema("prp_"),
			"proposal_event_id":       prefixedStringSchema("evt_"),
			"proposal_title":          stringSchema(),
			"text":                    stringSchema(),
			"claim_type":              enumSchema("descriptive", "evaluative", "recommendation", "risk", "decision"),
			"supporting_evidence_ids": arraySchema(prefixedStringSchema("evd_")),
			"opposing_evidence_ids":   arraySchema(prefixedStringSchema("evd_")),
			"depends_on_question_ids": arraySchema(prefixedStringSchema("qst_")),
			"user_assertion_event_id": prefixedStringSchema("evt_"),
			"confidence":              confidenceSchema(),
		}),
	)
	schemaClaimConfidence = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "claim_id", "event_id", "confidence"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"claim_id":           prefixedStringSchema("clm_"),
			"event_id":           prefixedStringSchema("evt_"),
			"confidence":         confidenceSchema(),
			"basis_evidence_ids": arraySchema(prefixedStringSchema("evd_")),
			"causation_event_id": prefixedStringSchema("evt_"),
			"correlation_id":     stringSchema(),
		}),
	)
	schemaProposalsSubmit = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "proposal_id", "event_id", "object_refs"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"proposal_id": prefixedStringSchema("prp_"),
			"event_id":    prefixedStringSchema("evt_"),
			"title":       stringSchema(),
			"object_refs": arraySchema(objectRefSchema()),
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

func researchListSchema(legacy bool) json.RawMessage {
	return objectSchema([]string{"mission_id", "object_kind"}, researchListProperties(legacy))
}

func researchReadSchema(legacy bool) json.RawMessage {
	return objectSchema([]string{"mission_id", "object_kind", "object_id"}, researchReadProperties(legacy))
}

func researchRefsSchema(legacy bool) json.RawMessage {
	return objectSchema([]string{"mission_id", "object_kind", "object_id"}, researchRefsProperties(legacy))
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

func researchListProperties(legacy bool) map[string]any {
	return map[string]any{
		"mission_id":  prefixedStringSchema("mis_"),
		"object_kind": researchObjectKindSchema(legacy),
		"limit":       map[string]any{"type": "integer", "minimum": 1, "maximum": 100},
		"cursor":      stringSchema(),
		"legacy":      map[string]any{"type": "boolean"},
	}
}

func researchReadProperties(legacy bool) map[string]any {
	return map[string]any{
		"mission_id":  prefixedStringSchema("mis_"),
		"object_kind": researchObjectKindSchema(legacy),
		"object_id":   stringSchema(),
		"offset":      map[string]any{"type": "integer", "minimum": 0},
		"max_bytes":   map[string]any{"type": "integer", "minimum": 1, "maximum": 32768},
		"limit":       map[string]any{"type": "integer", "minimum": 1, "maximum": 100},
		"cursor":      stringSchema(),
		"legacy":      map[string]any{"type": "boolean"},
	}
}

func researchGrepProperties() map[string]any {
	return map[string]any{
		"mission_id": prefixedStringSchema("mis_"),
		"query":      stringSchema(),
		"limit":      map[string]any{"type": "integer", "minimum": 1, "maximum": 100},
		"cursor":     stringSchema(),
		"legacy":     map[string]any{"type": "boolean"},
	}
}

func researchRefsProperties(legacy bool) map[string]any {
	return map[string]any{
		"mission_id":  prefixedStringSchema("mis_"),
		"object_kind": researchObjectKindSchema(legacy),
		"object_id":   stringSchema(),
		"limit":       map[string]any{"type": "integer", "minimum": 1, "maximum": 100},
		"cursor":      stringSchema(),
		"legacy":      map[string]any{"type": "boolean"},
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

func researchObjectKindSchema(legacy bool) map[string]any {
	kinds := []string{
		"source_snapshot",
		"raw_artifact",
		"ledger_event",
	}
	if legacy {
		kinds = append(kinds,
			"evidence_record",
			"claim_record",
			"question_record",
			"option_record",
			"proposal_bundle",
			"report",
			"report_version",
			"report_block",
		)
	}
	return enumSchema(kinds...)
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

func snapshotRefSchema() map[string]any {
	return objectSchemaValue([]string{"snapshot_id", "artifact_id"}, map[string]any{
		"snapshot_id": prefixedStringSchema("src_"),
		"artifact_id": prefixedStringSchema("art_"),
		"locator":     map[string]any{"type": "object"},
	})
}

func confidenceSchema() map[string]any {
	return objectSchemaValue(nil, map[string]any{
		"level":              enumSchema("low", "medium", "high", "unknown"),
		"rationale":          stringSchema(),
		"open_risks":         arraySchema(stringSchema()),
		"needs_verification": map[string]any{"type": "boolean"},
	})
}

func objectRefSchema() map[string]any {
	return objectSchemaValue([]string{"object_kind", "object_id"}, map[string]any{
		"object_kind": enumSchema("evidence_record", "claim_record", "question_record", "option_record"),
		"object_id":   stringSchema(),
	})
}
