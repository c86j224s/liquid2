package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/c86j224s/liquid2/plasma/internal/app"
)

type LongFormFinalizationStore interface {
	ListEvents(context.Context, string) ([]app.LedgerEvent, error)
	GetRawArtifact(context.Context, string) (app.RawArtifact, error)
	CreateRawArtifactWithEventConditionally(context.Context, app.CreateRawArtifactRequest, func([]app.LedgerEvent, app.RawArtifact) (app.AppendEventRequest, app.LedgerEvent, bool, error)) (app.RawArtifact, app.LedgerEvent, bool, error)
}

func validateLongFormLineage(events []app.LedgerEvent, binding LongFormFinalizeBinding) error {
	acceptedPending, err := longFormPendingLineage(events, binding.PendingEventID)
	if err != nil {
		return err
	}
	if !acceptedPending[binding.PendingEventID] {
		return fmt.Errorf("%w: bound report pending event does not exist", app.ErrConflict)
	}
	planCount := 0
	parts := make([]string, len(binding.PartArtifactIDs))
	partSeen := make([]bool, len(parts))
	sectionsByIndex := map[[2]int]string{}
	for _, event := range events {
		payload := eventPayload(event)
		pending, _ := payload["pending_event_id"].(string)
		if !acceptedPending[pending] {
			continue
		}
		if pending == binding.PendingEventID && (event.EventType == "report.draft.failed" || event.EventType == "report.final.failed" || event.EventType == "report.artifact.created") {
			return fmt.Errorf("%w: report finalization is terminal", app.ErrConflict)
		}
		if event.EventType == "report.plan.created" {
			planCount++
			if event.EventID != binding.PlanEventID || payload["report_mode"] != ModeLongForm || payload["artifact_id"] != binding.ArtifactID {
				return fmt.Errorf("%w: long-form plan lineage differs from binding", app.ErrConflict)
			}
		}
		if event.EventType == "report.part.created" {
			if payload["plan_event_id"] != binding.PlanEventID {
				return fmt.Errorf("%w: long-form part plan lineage differs from binding", app.ErrConflict)
			}
			index := jsonInt(payload["part_index"])
			if index < 1 || index > len(parts) || partSeen[index-1] {
				return fmt.Errorf("%w: duplicate or out-of-range long-form part index", app.ErrConflict)
			}
			partSeen[index-1] = true
			parts[index-1], _ = payload["artifact_id"].(string)
		}
		if event.EventType == "report.section.created" {
			if payload["plan_event_id"] != binding.PlanEventID {
				return fmt.Errorf("%w: long-form section plan lineage differs from binding", app.ErrConflict)
			}
			partIndex, sectionIndex := jsonInt(payload["part_index"]), jsonInt(payload["section_index"])
			key := [2]int{partIndex, sectionIndex}
			if partIndex < 1 || partIndex > len(parts) || sectionIndex < 1 || sectionsByIndex[key] != "" {
				return fmt.Errorf("%w: duplicate or out-of-range long-form section index", app.ErrConflict)
			}
			sectionsByIndex[key], _ = payload["artifact_id"].(string)
		}
	}
	keys := make([][2]int, 0, len(sectionsByIndex))
	for key := range sectionsByIndex {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i][0] < keys[j][0] || (keys[i][0] == keys[j][0] && keys[i][1] < keys[j][1])
	})
	sections := make([]string, 0, len(keys))
	lastByPart := map[int]int{}
	for _, key := range keys {
		if key[1] != lastByPart[key[0]]+1 {
			return fmt.Errorf("%w: long-form section indexes are not contiguous", app.ErrConflict)
		}
		lastByPart[key[0]] = key[1]
		sections = append(sections, sectionsByIndex[key])
	}
	if planCount != 1 {
		return fmt.Errorf("%w: long-form finalization plan count differs from binding", app.ErrConflict)
	}
	if !equalStrings(parts, binding.PartArtifactIDs) {
		return fmt.Errorf("%w: long-form finalization part lineage differs from binding", app.ErrConflict)
	}
	if !equalStrings(sections, binding.SectionArtifactIDs) {
		return fmt.Errorf("%w: long-form finalization section lineage differs from binding", app.ErrConflict)
	}
	return nil
}

func longFormPendingLineage(events []app.LedgerEvent, pendingID string) (map[string]bool, error) {
	type link struct{ origin, parent, strategy string }
	links := map[string]link{}
	for _, event := range events {
		if event.EventType != "report.draft.pending" {
			continue
		}
		payload := eventPayload(event)
		origin, _ := payload["origin_pending_event_id"].(string)
		parent, _ := payload["retry_of_pending_event_id"].(string)
		strategy, _ := payload["retry_strategy"].(string)
		if origin == "" {
			origin = event.EventID
		}
		links[event.EventID] = link{origin: origin, parent: parent, strategy: strategy}
	}
	accepted := map[string]bool{}
	current, ok := links[pendingID]
	if !ok {
		return accepted, fmt.Errorf("%w: bound report pending event does not exist", app.ErrConflict)
	}
	origin := current.origin
	seen := map[string]bool{}
	for depth := 0; depth < 64; depth++ {
		if seen[pendingID] {
			return nil, fmt.Errorf("%w: long-form retry lineage cycle", app.ErrConflict)
		}
		seen[pendingID] = true
		item, ok := links[pendingID]
		if !ok || item.origin != origin {
			return nil, fmt.Errorf("%w: long-form retry lineage differs", app.ErrConflict)
		}
		accepted[pendingID] = true
		if item.strategy == "restart" {
			if item.parent == "" {
				return nil, fmt.Errorf("%w: long-form restart lineage is incomplete", app.ErrConflict)
			}
			parent, ok := links[item.parent]
			if !ok || parent.origin != origin {
				return nil, fmt.Errorf("%w: long-form restart lineage differs", app.ErrConflict)
			}
			return accepted, nil
		}
		if item.parent == "" {
			if item.origin != pendingID {
				return nil, fmt.Errorf("%w: long-form retry origin differs", app.ErrConflict)
			}
			return accepted, nil
		}
		if item.strategy != "resume_failed" {
			return nil, fmt.Errorf("%w: unsupported long-form retry lineage", app.ErrConflict)
		}
		pendingID = item.parent
	}
	return nil, fmt.Errorf("%w: long-form retry lineage is too deep", app.ErrConflict)
}

func loadLongFormParts(ctx context.Context, store LongFormFinalizationStore, binding LongFormFinalizeBinding) ([]string, error) {
	parts := make([]string, 0, len(binding.PartArtifactIDs))
	for _, artifactID := range binding.PartArtifactIDs {
		artifact, err := store.GetRawArtifact(ctx, artifactID)
		if err != nil {
			return nil, err
		}
		if artifact.MissionID != binding.MissionID || artifact.MediaType != "text/markdown; charset=utf-8" {
			return nil, fmt.Errorf("%w: bound part artifact is foreign or not Markdown", app.ErrConflict)
		}
		parts = append(parts, string(artifact.Content))
	}
	return parts, nil
}

func eventPayload(event app.LedgerEvent) map[string]any {
	value := map[string]any{}
	_ = json.Unmarshal(event.Payload, &value)
	return value
}

func payloadString(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}
func payloadBool(payload map[string]any, key string) bool {
	value, _ := payload[key].(bool)
	return value
}
func jsonInt(value any) int { number, _ := value.(float64); return int(number) }
func duplicateStrings(values []string) bool {
	seen := map[string]bool{}
	for _, value := range values {
		if value == "" || seen[value] {
			return true
		}
		seen[value] = true
	}
	return false
}
func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
func equalJSONStrings(value any, expected []string) bool {
	if value == nil && len(expected) == 0 {
		return true
	}
	values, ok := value.([]any)
	if !ok || len(values) != len(expected) {
		return false
	}
	for index := range values {
		if values[index] != expected[index] {
			return false
		}
	}
	return true
}
func maxReportingInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
