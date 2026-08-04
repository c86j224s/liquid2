package mission

import (
	"encoding/json"
	"strings"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
)

func (b *projectionBuildState) applyMissionMetadataUpdated(event ledger.Event) {
	if event.Producer.Type != "user" {
		b.markNeedsReview("non-user mission.metadata.updated is invalid")
		return
	}
	var body struct {
		Title     *string `json:"title"`
		Objective *string `json:"objective"`
		Scope     *Scope  `json:"scope"`
	}
	if json.Unmarshal(event.Payload, &body) != nil || (body.Title == nil && body.Objective == nil && body.Scope == nil) {
		b.markNeedsReview("invalid mission.metadata.updated payload")
		return
	}
	if body.Title != nil {
		value := strings.TrimSpace(*body.Title)
		if value == "" {
			b.markNeedsReview("invalid mission.metadata.updated title")
			return
		}
		b.projection.Title = value
	}
	if body.Objective != nil {
		b.projection.Objective = strings.TrimSpace(*body.Objective)
	}
	if body.Scope != nil {
		b.projection.Scope = normalizeScope(*body.Scope)
	}
}

func (b *projectionBuildState) applyMissionCreated(event ledger.Event) {
	var body struct {
		Title     string `json:"title"`
		Objective string `json:"objective"`
		Scope     *Scope `json:"scope"`
	}
	if json.Unmarshal(event.Payload, &body) != nil {
		b.markNeedsReview("invalid mission.created payload")
		return
	}
	if body.Title != "" {
		b.projection.Title = body.Title
	}
	if body.Objective != "" {
		b.projection.Objective = body.Objective
		b.objectiveOwner = event.Producer.ID
	}
	if body.Scope != nil {
		b.projection.Scope = *body.Scope
		b.scopeOwner = event.Producer.ID
	}
}

func (b *projectionBuildState) applyMissionSteered(event ledger.Event) {
	if !approvalProducer(event.Producer) {
		b.markNeedsReview("non-user mission.steered requires approval")
		return
	}
	var body struct {
		Objective string `json:"objective"`
		Scope     *Scope `json:"scope"`
	}
	if json.Unmarshal(event.Payload, &body) != nil {
		b.markNeedsReview("invalid mission.steered payload")
		return
	}
	if body.Objective != "" {
		b.applyObjectiveChange(event, body.Objective)
	}
	if body.Scope != nil {
		b.applyScopeChange(event, *body.Scope)
	}
}

func (b *projectionBuildState) applyObjectiveChange(event ledger.Event, objective string) {
	if b.projection.Objective != "" && b.projection.Objective != objective && b.objectiveOwner != "" && b.objectiveOwner != event.Producer.ID {
		b.markNeedsReview("conflicting mission.steered objective requires approval")
		return
	}
	b.projection.Objective = objective
	b.objectiveOwner = event.Producer.ID
}

func (b *projectionBuildState) applyScopeChange(event ledger.Event, scope Scope) {
	if !emptyScope(b.projection.Scope) && !equalScopes(b.projection.Scope, scope) && b.scopeOwner != "" && b.scopeOwner != event.Producer.ID {
		b.markNeedsReview("conflicting mission.steered scope requires approval")
		return
	}
	b.projection.Scope = scope
	b.scopeOwner = event.Producer.ID
}

func (b *projectionBuildState) acceptApprovedTransition(event ledger.Event, reason string) bool {
	if approvalProducer(event.Producer) {
		return true
	}
	b.markNeedsReview(reason)
	return false
}

func (b *projectionBuildState) markNeedsReview(reason string) {
	b.projection.NeedsReview = true
	addUnique(&b.projection.NeedsReviewReasons, reason)
}

func (b *projectionBuildState) payloadRequiredString(event ledger.Event, field, reason string) (string, bool) {
	var body map[string]string
	if json.Unmarshal(event.Payload, &body) != nil {
		b.markNeedsReview(reason)
		return "", false
	}
	value := strings.TrimSpace(body[field])
	if value == "" {
		b.markNeedsReview(reason)
		return "", false
	}
	return value, true
}
