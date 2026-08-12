package research

import "encoding/json"

var (
	schemaOutline         = objectSchema([]string{"mission_id"}, map[string]any{"mission_id": prefixedStringSchema("mis_")})
	schemaChanges         = objectSchema([]string{"mission_id", "after_sequence"}, researchChangesProperties())
	schemaList            = researchListSchema(false)
	schemaListLegacy      = researchListSchema(true)
	schemaRead            = researchReadSchema(false)
	schemaReadLegacy      = researchReadSchema(true)
	schemaGrep            = objectSchema([]string{"mission_id", "query"}, researchGrepProperties())
	schemaRefs            = researchRefsSchema(false)
	schemaRefsLegacy      = researchRefsSchema(true)
	schemaEvidencePropose = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "evidence_id", "event_id", "proposal_id", "proposal_event_id", "summary", "evidence_type", "snapshot_refs", "producer"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"evidence_id":       prefixedStringSchema("evd_"),
			"event_id":          prefixedStringSchema("evt_"),
			"proposal_id":       prefixedStringSchema("prp_"),
			"proposal_event_id": prefixedStringSchema("evt_"),
			"proposal_title":    stringSchema(),
			"summary":           stringSchema(),
			"evidence_type":     enumSchema("quote", "fact", "table_row", "statistic", "observation", "interpretation", "reaction", "rumor", "controversy", "market_signal", "code", "formula", "benchmark", "open_question"),
			"snapshot_refs":     arraySchema(snapshotRefSchema()),
			"confidence":        confidenceSchema(),
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

func researchChangesProperties() map[string]any {
	return map[string]any{
		"mission_id":     prefixedStringSchema("mis_"),
		"after_sequence": map[string]any{"type": "integer", "minimum": 0},
		"limit":          map[string]any{"type": "integer", "minimum": 1, "maximum": 100},
	}
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

func researchListProperties(legacy bool) map[string]any {
	return map[string]any{"mission_id": prefixedStringSchema("mis_"), "object_kind": researchObjectKindSchema(legacy), "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 100}, "cursor": stringSchema(), "legacy": map[string]any{"type": "boolean"}}
}

func researchReadProperties(legacy bool) map[string]any {
	return map[string]any{"mission_id": prefixedStringSchema("mis_"), "object_kind": researchObjectKindSchema(legacy), "object_id": stringSchema(), "offset": map[string]any{"type": "integer", "minimum": 0}, "max_bytes": map[string]any{"type": "integer", "minimum": 1, "maximum": 32768}, "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 100}, "cursor": stringSchema(), "legacy": map[string]any{"type": "boolean"}}
}

func researchGrepProperties() map[string]any {
	return map[string]any{"mission_id": prefixedStringSchema("mis_"), "query": stringSchema(), "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 100}, "cursor": stringSchema(), "legacy": map[string]any{"type": "boolean"}}
}

func researchRefsProperties(legacy bool) map[string]any {
	return map[string]any{"mission_id": prefixedStringSchema("mis_"), "object_kind": researchObjectKindSchema(legacy), "object_id": stringSchema(), "limit": map[string]any{"type": "integer", "minimum": 1, "maximum": 100}, "cursor": stringSchema(), "legacy": map[string]any{"type": "boolean"}}
}

func researchObjectKindSchema(legacy bool) map[string]any {
	kinds := []string{"source_snapshot", "raw_artifact", "ledger_event"}
	if legacy {
		kinds = append(kinds, "evidence_record", "claim_record", "question_record", "option_record", "proposal_bundle", "report", "report_version", "report_block")
	}
	return enumSchema(kinds...)
}

func commonMutatingProperties() map[string]any {
	return map[string]any{"mission_id": prefixedStringSchema("mis_"), "session_id": prefixedStringSchema("ses_"), "idempotency_key": stringSchema(), "producer": objectSchemaValue([]string{"type", "id"}, map[string]any{"type": map[string]any{"type": "string", "const": "agent_session"}, "id": prefixedStringSchema("ses_")})}
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

func objectSchema(required []string, properties map[string]any) json.RawMessage {
	schema := objectSchemaValue(required, properties)
	encoded, err := json.Marshal(schema)
	if err != nil {
		panic(err)
	}
	return encoded
}

func objectSchemaValue(required []string, properties map[string]any) map[string]any {
	schema := map[string]any{"type": "object", "additionalProperties": false, "properties": properties}
	if len(required) > 0 {
		schema["required"] = required
	}
	return schema
}

func stringSchema() map[string]any { return map[string]any{"type": "string"} }

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

func arraySchema(items any) map[string]any { return map[string]any{"type": "array", "items": items} }

func snapshotRefSchema() map[string]any {
	return objectSchemaValue([]string{"snapshot_id", "artifact_id"}, map[string]any{"snapshot_id": prefixedStringSchema("src_"), "artifact_id": prefixedStringSchema("art_"), "locator": map[string]any{"type": "object"}})
}

func confidenceSchema() map[string]any {
	return objectSchemaValue(nil, map[string]any{"level": enumSchema("low", "medium", "high", "unknown"), "rationale": stringSchema(), "open_risks": arraySchema(stringSchema()), "needs_verification": map[string]any{"type": "boolean"}})
}

func objectRefSchema() map[string]any {
	return objectSchemaValue([]string{"object_kind", "object_id"}, map[string]any{"object_kind": enumSchema("evidence_record", "claim_record", "question_record", "option_record"), "object_id": stringSchema()})
}
