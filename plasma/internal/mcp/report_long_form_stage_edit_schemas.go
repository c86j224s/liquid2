package mcp

var (
	schemaReportLongFormStageEditStart = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "pending_event_id", "plan_event_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":         prefixedStringSchema("rfe_"),
			"pending_event_id": prefixedStringSchema("evt_"),
			"plan_event_id":    prefixedStringSchema("evt_"),
		}),
	)
	schemaReportLongFormStageEditRead = objectSchema(
		[]string{"mission_id", "session_id", "draft_id"},
		map[string]any{
			"mission_id": prefixedStringSchema("mis_"),
			"session_id": prefixedStringSchema("ses_"),
			"draft_id":   prefixedStringSchema("rfe_"),
			"offset":     map[string]any{"type": "integer", "minimum": 0},
			"max_bytes":  map[string]any{"type": "integer", "minimum": 1, "maximum": 65536},
		},
	)
	schemaReportLongFormStageEditPatch = objectSchema(
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
	schemaReportLongFormStageEditSubmit = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "pending_event_id", "plan_event_id"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":         prefixedStringSchema("rfe_"),
			"pending_event_id": prefixedStringSchema("evt_"),
			"plan_event_id":    prefixedStringSchema("evt_"),
		}),
	)
	schemaReportLongFormGateEditSubmit = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "pending_event_id", "plan_event_id", "gate_findings"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":         prefixedStringSchema("rfe_"),
			"pending_event_id": prefixedStringSchema("evt_"),
			"plan_event_id":    prefixedStringSchema("evt_"),
			"gate_findings": arraySchema(objectSchemaValue(
				[]string{"statement", "classification"},
				map[string]any{
					"statement": stringSchema(),
					"classification": enumSchema(
						"mission_source_grounded",
						"session_grounded",
						"derived_synthesis",
						"rhetorical_construction",
						"unverified_external_fact",
					),
					"repair_action": enumSchema(
						"attach_approved_evidence",
						"qualify_inference_or_uncertainty",
						"retain_with_footnote",
						"remove",
					),
					"evidence_ids": arraySchema(prefixedStringSchema("evd_")),
				},
			)),
			"semantic_acceptance": arraySchema(objectSchemaValue(
				[]string{"paragraph_ordinal", "final_paragraph_ordinal", "verdict"},
				map[string]any{
					"paragraph_ordinal":       map[string]any{"type": "integer", "minimum": 1},
					"final_paragraph_ordinal": map[string]any{"type": "integer", "minimum": 1},
					"verdict":                 enumSchema("accepted_equivalent", "reverted_to_reader", "repaired_by_gate"),
				},
			)),
		}),
	)
	schemaReportLongFormStyleSemanticValidationSubmit = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "pending_event_id", "plan_event_id", "semantic_acceptance"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":         prefixedStringSchema("rfe_"),
			"pending_event_id": prefixedStringSchema("evt_"),
			"plan_event_id":    prefixedStringSchema("evt_"),
			"semantic_acceptance": arraySchema(objectSchemaValue(
				[]string{"paragraph_ordinal", "verdict"},
				map[string]any{
					"paragraph_ordinal": map[string]any{"type": "integer", "minimum": 1},
					"verdict":           enumSchema("accepted_equivalent", "rejected_revert_to_reader"),
				},
			)),
		}),
	)
	schemaReportLongFormEvidenceGateSubmit = objectSchema(
		[]string{"mission_id", "session_id", "idempotency_key", "producer", "draft_id", "pending_event_id", "plan_event_id", "gate_findings"},
		mergeProperties(commonMutatingProperties(), map[string]any{
			"draft_id":         prefixedStringSchema("rfe_"),
			"pending_event_id": prefixedStringSchema("evt_"),
			"plan_event_id":    prefixedStringSchema("evt_"),
			"gate_findings": arraySchema(objectSchemaValue(
				[]string{"statement_sha256", "classification"},
				map[string]any{
					"statement_sha256": stringSchema(),
					"classification": enumSchema(
						"mission_source_grounded",
						"session_grounded",
						"derived_synthesis",
						"rhetorical_construction",
						"unverified_external_fact",
					),
					"evidence_ids": arraySchema(prefixedStringSchema("evd_")),
				},
			)),
		}),
	)
)
