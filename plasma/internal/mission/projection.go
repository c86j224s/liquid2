package mission

import (
	"fmt"

	"github.com/c86j224s/liquid2/plasma/internal/ledger"
	"github.com/c86j224s/liquid2/plasma/internal/producterror"
)

// BuildProjection derives the mission read model from its durable event
// sequence without storage or external side effects.
func BuildProjection(missionID string, events []ledger.Event) (Projection, error) {
	if err := validateMissionID(missionID); err != nil {
		return Projection{}, err
	}
	builder := projectionBuildState{
		projection: Projection{
			MissionID:        missionID,
			LifecycleState:   LifecycleActive,
			Scope:            Scope{},
			ActiveSessionIDs: []string{},
			AcceptedClaimIDs: []string{},
			OpenQuestionIDs:  []string{},
		},
	}

	for _, event := range events {
		if event.MissionID != missionID {
			return Projection{}, fmt.Errorf("%w: event mission mismatch", producterror.ErrInvalidInput)
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
		case ArchivedEvent:
			builder.projection.LifecycleState = LifecycleArchived
		case RestoredEvent:
			builder.projection.LifecycleState = LifecycleActive
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
				if versionID, ok := builder.payloadRequiredString(event, "report_version_id", "report.promoted requires report_version_id"); ok {
					builder.projection.ActiveReportVersionID = versionID
				}
			}
		}
	}
	return builder.projection, nil
}

type projectionBuildState struct {
	projection     Projection
	objectiveOwner string
	scopeOwner     string
}
