package app

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

type ProjectionStore interface {
	SaveMissionProjection(context.Context, MissionProjection) error
	GetMissionProjection(context.Context, string) (MissionProjection, error)
}

func (s *Service) RebuildProjection(ctx context.Context, missionID string) (MissionProjection, error) {
	events, err := s.ListEvents(ctx, missionID)
	if err != nil {
		return MissionProjection{}, err
	}
	projection, err := BuildProjection(missionID, events)
	if err != nil {
		return MissionProjection{}, err
	}
	if err := s.store.SaveMissionProjection(ctx, projection); err != nil {
		return MissionProjection{}, err
	}
	return projection, nil
}

func (s *Service) GetProjection(ctx context.Context, missionID string) (MissionProjection, error) {
	if err := validateID("mis_", missionID); err != nil {
		return MissionProjection{}, err
	}
	return s.store.GetMissionProjection(ctx, missionID)
}

func BuildProjection(missionID string, events []LedgerEvent) (MissionProjection, error) {
	if err := validateID("mis_", missionID); err != nil {
		return MissionProjection{}, err
	}
	builder := projectionBuildState{
		projection: MissionProjection{
			MissionID:        missionID,
			LifecycleState:   MissionLifecycleActive,
			Scope:            MissionScope{},
			ActiveSessionIDs: []string{},
			AcceptedClaimIDs: []string{},
			OpenQuestionIDs:  []string{},
		},
	}

	for _, event := range events {
		if event.MissionID != missionID {
			return MissionProjection{}, fmt.Errorf("%w: event mission mismatch", ErrInvalidInput)
		}
		builder.projection.LastEventID = event.EventID
		builder.projection.LastSequence = event.Sequence

		switch event.EventType {
		case "mission.created":
			builder.applyMissionCreated(event)
		case "mission.steered":
			builder.applyMissionSteered(event)
		case "mission.metadata.updated":
			builder.applyMissionMetadataUpdated(event)
		case MissionArchivedEvent:
			builder.projection.LifecycleState = MissionLifecycleArchived
		case MissionRestoredEvent:
			builder.projection.LifecycleState = MissionLifecycleActive
		case "session.attached":
			if sessionID, ok := builder.payloadRequiredString(event, "session_id", "session.attached requires session_id"); ok {
				addUnique(&builder.projection.ActiveSessionIDs, sessionID)
			}
		case "claim.approved":
			if builder.acceptApprovedTransition(event, "claim.approved requires user approval") {
				if claimID, ok := builder.payloadRequiredString(event, "claim_id", "claim.approved requires claim_id"); ok {
					addUnique(&builder.projection.AcceptedClaimIDs, claimID)
				}
			}
		case "question.proposed":
			if questionID, ok := builder.payloadRequiredString(event, "question_id", "question.proposed requires question_id"); ok {
				addUnique(&builder.projection.OpenQuestionIDs, questionID)
			}
		case "question.answered", "question.rejected":
			if builder.acceptApprovedTransition(event, event.EventType+" requires user approval") {
				if questionID, ok := builder.payloadRequiredString(event, "question_id", event.EventType+" requires question_id"); ok {
					removeValue(&builder.projection.OpenQuestionIDs, questionID)
				}
			}
		case "report.promoted":
			if builder.acceptApprovedTransition(event, "report.promoted requires user approval") {
				if reportVersionID, ok := builder.payloadRequiredString(event, "report_version_id", "report.promoted requires report_version_id"); ok {
					builder.projection.ActiveReportVersionID = reportVersionID
				}
			}
		}
	}

	return builder.projection, nil
}

func (b *projectionBuildState) applyMissionMetadataUpdated(event LedgerEvent) {
	if event.Producer.Type != "user" {
		b.markNeedsReview("non-user mission.metadata.updated is invalid")
		return
	}
	var body struct {
		Title     *string       `json:"title"`
		Objective *string       `json:"objective"`
		Scope     *MissionScope `json:"scope"`
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
		b.projection.Scope = normalizeMissionScope(*body.Scope)
	}
}

type projectionBuildState struct {
	projection     MissionProjection
	objectiveOwner string
	scopeOwner     string
}

func (b *projectionBuildState) applyMissionCreated(event LedgerEvent) {
	var body struct {
		Title     string        `json:"title"`
		Objective string        `json:"objective"`
		Scope     *MissionScope `json:"scope"`
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

func (b *projectionBuildState) applyMissionSteered(event LedgerEvent) {
	if !isApprovalProducer(event.Producer) {
		b.markNeedsReview("non-user mission.steered requires approval")
		return
	}

	var body struct {
		Objective string        `json:"objective"`
		Scope     *MissionScope `json:"scope"`
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

func (b *projectionBuildState) applyObjectiveChange(event LedgerEvent, objective string) {
	if b.projection.Objective != "" && b.projection.Objective != objective && b.objectiveOwner != "" && b.objectiveOwner != event.Producer.ID {
		b.markNeedsReview("conflicting mission.steered objective requires approval")
		return
	}
	b.projection.Objective = objective
	b.objectiveOwner = event.Producer.ID
}

func (b *projectionBuildState) applyScopeChange(event LedgerEvent, scope MissionScope) {
	if !isEmptyScope(b.projection.Scope) && !equalScope(b.projection.Scope, scope) && b.scopeOwner != "" && b.scopeOwner != event.Producer.ID {
		b.markNeedsReview("conflicting mission.steered scope requires approval")
		return
	}
	b.projection.Scope = scope
	b.scopeOwner = event.Producer.ID
}

func (b *projectionBuildState) acceptApprovedTransition(event LedgerEvent, reason string) bool {
	if isApprovalProducer(event.Producer) {
		return true
	}
	b.markNeedsReview(reason)
	return false
}

func (b *projectionBuildState) markNeedsReview(reason string) {
	b.projection.NeedsReview = true
	addUnique(&b.projection.NeedsReviewReasons, reason)
}

func (b *projectionBuildState) payloadRequiredString(event LedgerEvent, field string, reason string) (string, bool) {
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

func isApprovalProducer(producer Producer) bool {
	return producer.Type == "user" || producer.Type == "steering_chat"
}

func isEmptyScope(scope MissionScope) bool {
	return len(scope.Included) == 0 && len(scope.Excluded) == 0
}

func equalScope(left, right MissionScope) bool {
	return slices.Equal(left.Included, right.Included) && slices.Equal(left.Excluded, right.Excluded)
}

func addUnique(values *[]string, value string) {
	if value == "" {
		return
	}
	for _, existing := range *values {
		if existing == value {
			return
		}
	}
	*values = append(*values, value)
}

func removeValue(values *[]string, value string) {
	if value == "" {
		return
	}
	next := (*values)[:0]
	for _, existing := range *values {
		if existing != value {
			next = append(next, existing)
		}
	}
	*values = next
}
