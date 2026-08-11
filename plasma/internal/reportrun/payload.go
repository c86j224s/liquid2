package reportrun

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parsePayload(event Event) (eventPayload, error) {
	payload := eventPayload{}
	if len(event.Payload) == 0 {
		return payload, nil
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return nil, fmt.Errorf("invalid report payload: %w", err)
	}
	return payload, nil
}

func isReportEvent(eventType string) bool {
	eventType = strings.TrimSpace(eventType)
	switch eventType {
	case "report.promoted", "report.exported":
		return false
	default:
		return strings.HasPrefix(eventType, "report.")
	}
}

func eventRole(eventType string, payload eventPayload) string {
	switch strings.TrimSpace(eventType) {
	case "report.draft.pending":
		return "draft_pending"
	case "report.design.pending", "report.humanize.pending", "report.patch.pending":
		return "operation_pending"
	case "report.artifact.created":
		return "final"
	case "report.draft.failed", "report.design.failed", "report.humanize.failed", "report.humanize.skipped", "report.patch.failed":
		if payloadString(payload, "kind") == "report_draft_canceled" ||
			strings.Contains(payloadString(payload, "kind"), "_canceled") ||
			payloadBool(payload, "canceled") {
			return "canceled"
		}
		return "terminal"
	case "report.artifact.exported":
		return "derivative"
	case "report.patch.finalized":
		return "patch_finalized"
	default:
		return "stage"
	}
}

func isCreatedIntermediateEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "report.section.created",
		"report.part.created",
		"report.part.edited",
		"report.final_assembly.created",
		"report.final_edit.writer.submitted",
		"report.final_edit.reader.submitted",
		"report.final_edit.style.submitted",
		"report.final_edit.gate.submitted",
		"report.final_edit.style_semantic_validation.submitted",
		"report.final_edit.evidence_gate.submitted",
		"report.patch.finalized":
		return true
	default:
		return false
	}
}

func isKnownNonCreatorReportEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case "report.drafted",
		"report.plan.created",
		"report.plan.submitted",
		"report.plan.section_repair.completed",
		"report.requirements.started",
		"report.requirements.mapped",
		"report.section.started",
		"report.section.evidence_gap",
		"report.part_plan.created",
		"report.part_assembly.submitted",
		"report.part_edit.started",
		"report.final_edit.writer.started",
		"report.final_edit.reader.started",
		"report.final_edit.style.started",
		"report.final_edit.gate.started",
		"report.final_edit.style_semantic_validation.started",
		"report.final_edit.evidence_gate.started",
		"report.draft.pending",
		"report.design.pending",
		"report.humanize.pending",
		"report.patch.pending",
		"report.draft.failed",
		"report.design.failed",
		"report.humanize.failed",
		"report.patch.failed",
		"report.artifact.failed",
		"report.final.failed",
		"report.part.failed",
		"report.part_edit.failed",
		"report.part_plan.failed",
		"report.plan.failed",
		"report.requirements.failed",
		"report.section.failed",
		"report.humanize.skipped",
		"report.patch.rejected":
		return true
	default:
		return false
	}
}

func payloadBool(payload eventPayload, key string) bool {
	value, _ := payload[key].(bool)
	return value
}

func artifactRoleRank(role string) int {
	switch role {
	case ArtifactRoleFinal:
		return 4
	case ArtifactRoleDerivative:
		return 3
	case ArtifactRoleIntermediate:
		return 2
	case ArtifactRoleInput:
		return 1
	default:
		return 0
	}
}

func payloadString(payload eventPayload, key string) string {
	value, _ := payload[key].(string)
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func artifactIDs(payload eventPayload) []string {
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}
	for _, key := range []string{"artifact_id", "source_artifact_id", "base_artifact_id", "content_model_artifact_id"} {
		add(payloadString(payload, key))
	}
	for _, key := range []string{"artifact_ids", "section_artifact_ids", "part_artifact_ids", "source_artifact_ids"} {
		for _, id := range payloadStringSlice(payload[key]) {
			add(id)
		}
	}
	return ids
}

func referencedArtifactIDs(payload eventPayload) []string {
	seen := map[string]bool{}
	var ids []string
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}
	for _, key := range []string{"source_artifact_id", "base_artifact_id"} {
		add(payloadString(payload, key))
	}
	for _, key := range []string{"source_artifact_ids", "section_artifact_ids", "part_artifact_ids"} {
		for _, id := range payloadStringSlice(payload[key]) {
			add(id)
		}
	}
	return ids
}

func payloadStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && strings.TrimSpace(text) != "" {
			out = append(out, strings.TrimSpace(text))
		}
	}
	return out
}
