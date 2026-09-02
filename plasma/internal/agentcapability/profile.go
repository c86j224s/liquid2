// Package agentcapability defines the provider-session capability profiles that
// product callers may select. Provider adapters own the concrete CLI mapping;
// prompts and user text never choose a profile.
package agentcapability

import (
	"fmt"
	"strings"
)

// ProfileID identifies one versioned provider-session capability contract.
type ProfileID string

const (
	// ProfileLegacyV1 preserves the pre-profile provider behavior for existing
	// sessions and callers that have not adopted an explicit profile.
	ProfileLegacyV1 ProfileID = "legacy.v1"
	// ProfileResearchV1 keeps Plasma research tools while removing supported
	// provider ambient capabilities that the product does not use.
	ProfileResearchV1 ProfileID = "research.v1"
	// ProfileGoalDraftV1 is the request-local workflow-goal drafting profile.
	ProfileGoalDraftV1 ProfileID = "workflow_goal_draft.v1"
	// ProfileReportILV1 is the original tool-free report IL product profile.
	ProfileReportILV1 ProfileID = "report_il.v1"
	// ProfileReportILSourceV1 isolates report IL to frozen source access and the
	// request-local server-owned authoring workspace.
	ProfileReportILSourceV1 ProfileID = "report_il_source.v1"
	// ProfileReportUnverifiedV1 isolates one unverified report call to mission
	// source list/read tools without adding an authoring or validation workflow.
	ProfileReportUnverifiedV1 ProfileID = "report_unverified.v1"

	RevisionV1 = "1"
)

// Profile is the validated capability identity persisted with provider sessions.
type Profile struct {
	ID       ProfileID
	Revision string
}

// Resolve validates a profile pair. The zero value maps to the immutable legacy
// profile so existing report and compatibility callers keep their current behavior.
func Resolve(id ProfileID, revision string) (Profile, error) {
	if id == "" {
		id = ProfileLegacyV1
	}
	revision = strings.TrimSpace(revision)
	if revision == "" {
		revision = RevisionV1
	}
	switch id {
	case ProfileLegacyV1, ProfileResearchV1, ProfileGoalDraftV1, ProfileReportILV1, ProfileReportILSourceV1, ProfileReportUnverifiedV1:
	default:
		return Profile{}, fmt.Errorf("unknown agent capability profile %q", id)
	}
	if revision != RevisionV1 {
		return Profile{}, fmt.Errorf("unsupported agent capability profile revision %q for %s", revision, id)
	}
	return Profile{ID: id, Revision: revision}, nil
}

// Research returns the profile selected for a new conversation or workflow session.
func Research() Profile {
	return Profile{ID: ProfileResearchV1, Revision: RevisionV1}
}

// GoalDraft returns the request-local workflow-goal drafting profile.
func GoalDraft() Profile {
	return Profile{ID: ProfileGoalDraftV1, Revision: RevisionV1}
}

// ReportIL returns the original tool-free report IL product profile.
func ReportIL() Profile {
	return Profile{ID: ProfileReportILV1, Revision: RevisionV1}
}

// ReportILSource returns the report IL frozen source-read-only profile.
func ReportILSource() Profile {
	return Profile{ID: ProfileReportILSourceV1, Revision: RevisionV1}
}

// ReportUnverified returns the isolated unverified report source-read profile.
func ReportUnverified() Profile {
	return Profile{ID: ProfileReportUnverifiedV1, Revision: RevisionV1}
}
